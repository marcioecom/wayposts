package post

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"

	"github.com/marciojunior/wayposts/internal/model"
)

const likesTTLSeconds = 7 * 24 * 60 * 60

var likeScript = redis.NewScript(`
if redis.call('EXISTS', KEYS[1]) == 0 then
  return -1
end
local added = redis.call('SADD', KEYS[2], ARGV[1])
if added == 1 then
  redis.call('HINCRBY', KEYS[1], 'like_count', 1)
  if redis.call('TTL', KEYS[2]) < 0 then
    redis.call('EXPIRE', KEYS[2], ARGV[2])
  end
end
return tonumber(redis.call('HGET', KEYS[1], 'like_count') or '0')
`)

var unlikeScript = redis.NewScript(`
if redis.call('EXISTS', KEYS[1]) == 0 then
  return -1
end
local removed = redis.call('SREM', KEYS[2], ARGV[1])
if removed == 1 then
  local n = redis.call('HINCRBY', KEYS[1], 'like_count', -1)
  if n < 0 then
    redis.call('HSET', KEYS[1], 'like_count', 0)
    return 0
  end
  return n
end
return tonumber(redis.call('HGET', KEYS[1], 'like_count') or '0')
`)

func (s *Store) warm(ctx context.Context, postID int64) (model.Post, error) {
	p, err := s.Get(ctx, postID)
	if err != nil {
		return model.Post{}, err
	}
	if p.Deleted {
		return model.Post{}, pgx.ErrNoRows
	}
	if n, err := s.rdb.Exists(ctx, postKey(postID)).Result(); err == nil && n == 0 {
		s.cache(ctx, p)
	}
	return p, nil
}

func (s *Store) Like(ctx context.Context, userID, postID int64) (int64, error) {
	if _, err := s.warm(ctx, postID); err != nil {
		return 0, err
	}

	count, err := s.runLikeScript(ctx, likeScript, userID, postID)
	if errors.Is(err, errPostNotCached) {
		if _, err := s.warm(ctx, postID); err != nil {
			return 0, err
		}
		count, err = s.runLikeScript(ctx, likeScript, userID, postID)
	}
	if err != nil {
		return 0, err
	}

	s.likes.Add(userID, postID, true)
	s.m.Likes.WithLabelValues("like").Inc()
	return count, nil
}

func (s *Store) Unlike(ctx context.Context, userID, postID int64) (int64, error) {
	if _, err := s.warm(ctx, postID); err != nil {
		return 0, err
	}

	count, err := s.runLikeScript(ctx, unlikeScript, userID, postID)
	if errors.Is(err, errPostNotCached) {
		if _, err := s.warm(ctx, postID); err != nil {
			return 0, err
		}
		count, err = s.runLikeScript(ctx, unlikeScript, userID, postID)
	}
	if err != nil {
		return 0, err
	}

	s.likes.Add(userID, postID, false)
	s.m.Likes.WithLabelValues("unlike").Inc()
	return count, nil
}

var errPostNotCached = errors.New("post: corpo não está no cache")

func (s *Store) runLikeScript(ctx context.Context, script *redis.Script, userID, postID int64) (int64, error) {
	count, err := script.Run(ctx, s.rdb,
		[]string{postKey(postID), likesKey(postID)},
		userID, likesTTLSeconds,
	).Int64()
	if err != nil {
		return 0, fmt.Errorf("post: like script: %w", err)
	}
	if count < 0 {
		return 0, errPostNotCached
	}
	return count, nil
}
