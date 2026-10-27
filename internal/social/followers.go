package social

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	followerChunkSize = 1000
	// O meta expira antes dos chunks, então um conjunto parcialmente expirado
	// nunca é lido.
	followerCacheTTL = 24 * time.Hour
	followerMetaTTL  = 23 * time.Hour
	followerGenTTL   = 7 * 24 * time.Hour
	rebuildAttempts  = 3
)

func followersMetaKey(authorID int64) string {
	return "followers_meta:" + strconv.FormatInt(authorID, 10)
}

func followersChunkKey(authorID int64, chunk int) string {
	return "followers:" + strconv.FormatInt(authorID, 10) + ":" + strconv.Itoa(chunk)
}

func followersGenKey(authorID int64) string {
	return "followers_gen:" + strconv.FormatInt(authorID, 10)
}

// FollowersAfter devolve seguidores em ordem crescente de ID. A paginação é
// aplicada em memória sobre a lista do cache.
func (s *Store) FollowersAfter(ctx context.Context, authorID, after int64, limit int) ([]int64, error) {
	if limit <= 0 || limit > 5000 {
		limit = 1000
	}

	all, err := s.allFollowers(ctx, authorID)
	if err != nil {
		return nil, err
	}

	start := sort.Search(len(all), func(i int) bool { return all[i] > after })
	if start >= len(all) {
		return nil, nil
	}
	end := start + limit
	if end > len(all) {
		end = len(all)
	}
	out := make([]int64, end-start)
	copy(out, all[start:end])
	return out, nil
}

func (s *Store) allFollowers(ctx context.Context, authorID int64) ([]int64, error) {
	if ids, ok := s.cachedFollowers(ctx, authorID); ok {
		s.m.FollowerCache.WithLabelValues("hit").Inc()
		return ids, nil
	}
	s.m.FollowerCache.WithLabelValues("miss").Inc()

	res, err, _ := s.sf.Do(followersMetaKey(authorID), func() (any, error) {
		if ids, ok := s.cachedFollowers(ctx, authorID); ok {
			return ids, nil
		}
		return s.rebuildFollowers(ctx, authorID)
	})
	if err != nil {
		s.m.FollowerCache.WithLabelValues("error").Inc()
		return nil, err
	}
	return res.([]int64), nil
}

func (s *Store) cachedFollowers(ctx context.Context, authorID int64) ([]int64, bool) {
	raw, err := s.rdb.Get(ctx, followersMetaKey(authorID)).Result()
	if err != nil {
		return nil, false
	}
	chunks, err := strconv.Atoi(raw)
	if err != nil || chunks < 0 {
		return nil, false
	}
	if chunks == 0 {
		return []int64{}, true
	}

	pipe := s.rdb.Pipeline()
	cmds := make([]*redis.StringSliceCmd, chunks)
	for i := 0; i < chunks; i++ {
		cmds[i] = pipe.SMembers(ctx, followersChunkKey(authorID, i))
	}
	if _, err := pipe.Exec(ctx); err != nil && err != redis.Nil {
		return nil, false
	}

	ids := make([]int64, 0, chunks*followerChunkSize)
	for _, cmd := range cmds {
		for _, item := range cmd.Val() {
			fid, err := strconv.ParseInt(item, 10, 64)
			if err != nil || fid <= 0 {
				continue
			}
			ids = append(ids, fid)
		}
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids, true
}

func (s *Store) rebuildFollowers(ctx context.Context, authorID int64) ([]int64, error) {
	for attempt := 0; attempt < rebuildAttempts; attempt++ {
		genBefore, _ := s.rdb.Get(ctx, followersGenKey(authorID)).Result()

		ids, err := s.followersFromDB(ctx, authorID)
		if err != nil {
			return nil, err
		}

		oldChunks, _ := s.rdb.Get(ctx, followersMetaKey(authorID)).Int()
		chunks := (len(ids) + followerChunkSize - 1) / followerChunkSize

		pipe := s.rdb.Pipeline()
		for i := 0; i < chunks; i++ {
			end := (i + 1) * followerChunkSize
			if end > len(ids) {
				end = len(ids)
			}
			members := make([]any, 0, end-i*followerChunkSize)
			for _, fid := range ids[i*followerChunkSize : end] {
				members = append(members, fid)
			}
			key := followersChunkKey(authorID, i)
			pipe.Del(ctx, key)
			pipe.SAdd(ctx, key, members...)
			pipe.Expire(ctx, key, followerCacheTTL)
		}
		for i := chunks; i < oldChunks; i++ {
			pipe.Del(ctx, followersChunkKey(authorID, i))
		}
		if _, err := pipe.Exec(ctx); err != nil {
			return nil, fmt.Errorf("social: followers cache: %w", err)
		}

		genAfter, _ := s.rdb.Get(ctx, followersGenKey(authorID)).Result()
		if genAfter != genBefore {
			s.m.FollowerCache.WithLabelValues("stale_retry").Inc()
			continue
		}

		if err := s.rdb.Set(ctx, followersMetaKey(authorID), chunks, followerMetaTTL).Err(); err != nil {
			return nil, fmt.Errorf("social: followers meta: %w", err)
		}
		s.m.FollowerCache.WithLabelValues("rebuild").Inc()
		return ids, nil
	}
	return nil, fmt.Errorf("social: followers de %d mudaram durante %d reconstruções", authorID, rebuildAttempts)
}

func (s *Store) followersFromDB(ctx context.Context, authorID int64) ([]int64, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT follower_id
		  FROM social.follows
		 WHERE followee_id = $1
		 ORDER BY follower_id
	`, authorID)
	if err != nil {
		return nil, fmt.Errorf("social: followers: %w", err)
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var fid int64
		if err := rows.Scan(&fid); err != nil {
			return nil, err
		}
		ids = append(ids, fid)
	}
	return ids, rows.Err()
}

// invalidateFollowerCache incrementa a geração antes de tocar nos chunks, para
// que uma reconstrução concorrente perceba a corrida e tente de novo.
func (s *Store) invalidateFollowerCache(ctx context.Context, authorID int64) {
	key := followersGenKey(authorID)
	pipe := s.rdb.Pipeline()
	pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, followerGenTTL)
	if _, err := pipe.Exec(ctx); err != nil {
		s.log.Warn("followers gen", "author", authorID, "err", err)
	}
}

// cacheAddFollower não faz nada se o autor não está em cache: a próxima leitura
// reconstrói a partir do Postgres, que já tem a aresta.
func (s *Store) cacheAddFollower(ctx context.Context, authorID, followerID int64) {
	raw, err := s.rdb.Get(ctx, followersMetaKey(authorID)).Result()
	if err != nil {
		return
	}
	chunks, err := strconv.Atoi(raw)
	if err != nil || chunks < 0 {
		return
	}

	pipe := s.rdb.Pipeline()
	if chunks == 0 {
		key := followersChunkKey(authorID, 0)
		pipe.SAdd(ctx, key, followerID)
		pipe.Expire(ctx, key, followerCacheTTL)
		pipe.Set(ctx, followersMetaKey(authorID), 1, followerMetaTTL)
	} else {
		last := chunks - 1
		key := followersChunkKey(authorID, last)
		size, err := s.rdb.SCard(ctx, key).Result()
		if err != nil {
			return
		}
		if size >= followerChunkSize {
			key = followersChunkKey(authorID, chunks)
			pipe.SAdd(ctx, key, followerID)
			pipe.Expire(ctx, key, followerCacheTTL)
			pipe.Set(ctx, followersMetaKey(authorID), chunks+1, followerMetaTTL)
		} else {
			pipe.SAdd(ctx, key, followerID)
		}
	}
	if _, err := pipe.Exec(ctx); err != nil {
		s.log.Warn("followers cache add", "author", authorID, "follower", followerID, "err", err)
	}
}

func (s *Store) cacheRemoveFollower(ctx context.Context, authorID, followerID int64) {
	raw, err := s.rdb.Get(ctx, followersMetaKey(authorID)).Result()
	if err != nil {
		return
	}
	chunks, err := strconv.Atoi(raw)
	if err != nil || chunks <= 0 {
		return
	}

	pipe := s.rdb.Pipeline()
	for i := 0; i < chunks; i++ {
		pipe.SRem(ctx, followersChunkKey(authorID, i), followerID)
	}
	if _, err := pipe.Exec(ctx); err != nil {
		s.log.Warn("followers cache remove", "author", authorID, "follower", followerID, "err", err)
	}
}
