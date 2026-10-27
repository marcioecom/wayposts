package post

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/marciojunior/wayposts/internal/events"
	"github.com/marciojunior/wayposts/internal/id"
	"github.com/marciojunior/wayposts/internal/model"
	"github.com/marciojunior/wayposts/internal/platform/metrics"
)

const (
	postCacheTTL = 24 * time.Hour
	authorCap    = 200
)

type Store struct {
	pool  *pgxpool.Pool
	rdb   *redis.Client
	gen   *id.Generator
	likes *LikeBuffer
	m     *metrics.Metrics
	log   *slog.Logger
}

func NewStore(pool *pgxpool.Pool, rdb *redis.Client, gen *id.Generator, log *slog.Logger, m *metrics.Metrics) *Store {
	return &Store{
		pool:  pool,
		rdb:   rdb,
		gen:   gen,
		likes: NewLikeBuffer(pool, log, m),
		m:     m,
		log:   log,
	}
}

func (s *Store) RunLikes(ctx context.Context) {
	s.likes.Run(ctx)
}

func postKey(postID int64) string {
	return "post:" + strconv.FormatInt(postID, 10)
}

func likesKey(postID int64) string {
	return "likes:" + strconv.FormatInt(postID, 10)
}

func authorKey(userID int64) string {
	return "author:" + strconv.FormatInt(userID, 10)
}

func (s *Store) Create(ctx context.Context, authorID int64, body string, media []string) (model.Post, error) {
	if media == nil {
		media = []string{}
	}
	mediaJSON, err := json.Marshal(media)
	if err != nil {
		return model.Post{}, fmt.Errorf("post: media: %w", err)
	}

	pid := s.gen.Next()
	p := model.Post{
		ID:        pid,
		AuthorID:  authorID,
		Body:      body,
		Media:     media,
		CreatedAt: id.Time(pid),
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return model.Post{}, fmt.Errorf("post: begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `
		INSERT INTO post.posts (id, author_id, body, media, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`, p.ID, p.AuthorID, p.Body, mediaJSON, p.CreatedAt); err != nil {
		return model.Post{}, fmt.Errorf("post: insert: %w", err)
	}

	payload, err := json.Marshal(events.PostCreated{PostID: p.ID, AuthorID: p.AuthorID})
	if err != nil {
		return model.Post{}, fmt.Errorf("post: event: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO post.outbox (event_type, payload) VALUES ($1, $2)
	`, events.TopicPostCreated, payload); err != nil {
		return model.Post{}, fmt.Errorf("post: outbox: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return model.Post{}, fmt.Errorf("post: commit: %w", err)
	}

	s.m.PostsCreated.Inc()
	s.cache(ctx, p)
	return p, nil
}

// cache escreve o corpo e o índice do autor. É best-effort: o Postgres é a
// fonte da verdade e o read path tem fallback.
func (s *Store) cache(ctx context.Context, p model.Post) {
	mediaJSON, _ := json.Marshal(p.Media)
	pipe := s.rdb.Pipeline()
	pipe.HSet(ctx, postKey(p.ID), map[string]any{
		"author_id":  p.AuthorID,
		"body":       p.Body,
		"media":      string(mediaJSON),
		"created_at": p.CreatedAt.Format(time.RFC3339Nano),
		"deleted":    "0",
		"like_count": p.LikeCount,
	})
	pipe.Expire(ctx, postKey(p.ID), postCacheTTL)
	pipe.ZAdd(ctx, authorKey(p.AuthorID), redis.Z{Score: id.Float(p.ID), Member: p.ID})
	pipe.ZRemRangeByRank(ctx, authorKey(p.AuthorID), 0, -(authorCap + 1))
	if _, err := pipe.Exec(ctx); err != nil {
		s.log.Warn("post cache", "post", p.ID, "err", err)
	}
}

func (s *Store) Get(ctx context.Context, postID int64) (model.Post, error) {
	if p, ok := s.fromCache(ctx, postID); ok {
		s.m.PostCache.WithLabelValues("hit").Inc()
		return p, nil
	}
	s.m.PostCache.WithLabelValues("miss").Inc()
	return s.fromDB(ctx, postID)
}

func (s *Store) GetMany(ctx context.Context, ids []int64, viewerID int64) ([]model.Post, error) {
	if len(ids) == 0 {
		return []model.Post{}, nil
	}

	pipe := s.rdb.Pipeline()
	hashCmds := make(map[int64]*redis.MapStringStringCmd, len(ids))
	likeCmds := make(map[int64]*redis.BoolSliceCmd, len(ids))
	existsCmds := make(map[int64]*redis.IntCmd, len(ids))
	for _, pid := range ids {
		hashCmds[pid] = pipe.HGetAll(ctx, postKey(pid))
		if viewerID > 0 {
			likeCmds[pid] = pipe.SMIsMember(ctx, likesKey(pid), viewerID)
			existsCmds[pid] = pipe.Exists(ctx, likesKey(pid))
		}
	}
	_, _ = pipe.Exec(ctx)
	for _, pid := range ids {
		if len(hashCmds[pid].Val()) > 0 {
			s.m.PostCache.WithLabelValues("hit").Inc()
		} else {
			s.m.PostCache.WithLabelValues("miss").Inc()
		}
	}

	liked := make(map[int64]bool, len(ids))
	if viewerID > 0 {
		var missing []int64
		for _, pid := range ids {
			if v := likeCmds[pid].Val(); len(v) > 0 && v[0] {
				liked[pid] = true
				continue
			}
			if existsCmds[pid].Val() == 0 {
				missing = append(missing, pid)
			}
		}
		if len(missing) > 0 {
			rows, err := s.pool.Query(ctx, `
				SELECT post_id FROM post.likes
				 WHERE user_id = $1 AND post_id = ANY($2)
			`, viewerID, missing)
			if err != nil {
				return nil, fmt.Errorf("post: liked fallback: %w", err)
			}
			for rows.Next() {
				var pid int64
				if err := rows.Scan(&pid); err != nil {
					rows.Close()
					return nil, err
				}
				liked[pid] = true
			}
			rows.Close()
			if err := rows.Err(); err != nil {
				return nil, err
			}
		}
	}

	out := make([]model.Post, 0, len(ids))
	var missing []int64
	for _, pid := range ids {
		if p, ok := fromHash(pid, hashCmds[pid].Val()); ok {
			p.ViewerLiked = liked[pid]
			out = append(out, p)
			continue
		}
		missing = append(missing, pid)
	}

	if len(missing) > 0 {
		fromDB, err := s.manyFromDB(ctx, missing)
		if err != nil {
			return nil, err
		}
		for _, p := range fromDB {
			p.ViewerLiked = liked[p.ID]
			out = append(out, p)
		}
	}
	return out, nil
}

func (s *Store) GetWithViewer(ctx context.Context, postID, viewerID int64) (model.Post, error) {
	posts, err := s.GetMany(ctx, []int64{postID}, viewerID)
	if err != nil {
		return model.Post{}, err
	}
	if len(posts) == 0 {
		return model.Post{}, pgx.ErrNoRows
	}
	return posts[0], nil
}

func (s *Store) Delete(ctx context.Context, postID, authorID int64) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE post.posts
		   SET deleted = true
		 WHERE id = $1 AND author_id = $2 AND deleted = false
	`, postID, authorID)
	if err != nil {
		return fmt.Errorf("post: delete: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}

	if n, err := s.rdb.Exists(ctx, postKey(postID)).Result(); err == nil && n > 0 {
		if err := s.rdb.HSet(ctx, postKey(postID), "deleted", "1").Err(); err != nil {
			s.log.Warn("post cache delete", "post", postID, "err", err)
		}
	}
	if err := s.rdb.ZRem(ctx, authorKey(authorID), postID).Err(); err != nil {
		s.log.Warn("author index delete", "post", postID, "err", err)
	}
	return nil
}

func (s *Store) fromCache(ctx context.Context, postID int64) (model.Post, bool) {
	raw, err := s.rdb.HGetAll(ctx, postKey(postID)).Result()
	if err != nil || len(raw) == 0 {
		return model.Post{}, false
	}
	return fromHash(postID, raw)
}

func fromHash(postID int64, raw map[string]string) (model.Post, bool) {
	if len(raw) == 0 {
		return model.Post{}, false
	}
	p := model.Post{ID: postID}
	p.AuthorID, _ = strconv.ParseInt(raw["author_id"], 10, 64)
	p.Body = raw["body"]
	p.Deleted = raw["deleted"] == "1"
	p.LikeCount, _ = strconv.ParseInt(raw["like_count"], 10, 64)
	if raw["media"] != "" {
		_ = json.Unmarshal([]byte(raw["media"]), &p.Media)
	}
	if p.Media == nil {
		p.Media = []string{}
	}
	if t, err := time.Parse(time.RFC3339Nano, raw["created_at"]); err == nil {
		p.CreatedAt = t
	}
	if p.AuthorID == 0 {
		return model.Post{}, false
	}
	return p, true
}

func (s *Store) fromDB(ctx context.Context, postID int64) (model.Post, error) {
	var (
		p         model.Post
		mediaJSON []byte
	)
	err := s.pool.QueryRow(ctx, `
		SELECT id, author_id, body, media, created_at, deleted, like_count
		  FROM post.posts
		 WHERE id = $1
	`, postID).Scan(&p.ID, &p.AuthorID, &p.Body, &mediaJSON, &p.CreatedAt, &p.Deleted, &p.LikeCount)
	if err != nil {
		return model.Post{}, fmt.Errorf("post: get: %w", err)
	}
	_ = json.Unmarshal(mediaJSON, &p.Media)
	if p.Media == nil {
		p.Media = []string{}
	}
	return p, nil
}

func (s *Store) manyFromDB(ctx context.Context, ids []int64) ([]model.Post, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, author_id, body, media, created_at, deleted, like_count
		  FROM post.posts
		 WHERE id = ANY($1)
	`, ids)
	if err != nil {
		return nil, fmt.Errorf("post: get many: %w", err)
	}
	defer rows.Close()

	var out []model.Post
	for rows.Next() {
		var (
			p         model.Post
			mediaJSON []byte
		)
		if err := rows.Scan(&p.ID, &p.AuthorID, &p.Body, &mediaJSON, &p.CreatedAt, &p.Deleted, &p.LikeCount); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(mediaJSON, &p.Media)
		if p.Media == nil {
			p.Media = []string{}
		}
		out = append(out, p)
	}
	return out, rows.Err()
}
