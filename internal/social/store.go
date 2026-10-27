package social

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/singleflight"

	"github.com/marciojunior/wayposts/internal/model"
	"github.com/marciojunior/wayposts/internal/platform/metrics"
)

var ErrInvalidFollow = errors.New("social: follower e followee não podem ser iguais")

type Store struct {
	pool      *pgxpool.Pool
	rdb       *redis.Client
	threshold int
	m         *metrics.Metrics
	sf        singleflight.Group
	log       *slog.Logger
}

func NewStore(pool *pgxpool.Pool, rdb *redis.Client, threshold int, m *metrics.Metrics, log *slog.Logger) *Store {
	return &Store{pool: pool, rdb: rdb, threshold: threshold, m: m, log: log}
}

func celebFollowsKey(userID int64) string {
	return "celeb_follows:" + strconv.FormatInt(userID, 10)
}

func (s *Store) Follow(ctx context.Context, followerID, followeeID int64) error {
	if followerID == followeeID {
		return ErrInvalidFollow
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("social: begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var count int
	err = tx.QueryRow(ctx, `
		WITH ins AS (
			INSERT INTO social.follows (follower_id, followee_id, created_at)
			VALUES ($1, $2, now())
			ON CONFLICT DO NOTHING
			RETURNING 1
		)
		UPDATE social.users
		   SET follower_count = follower_count + (SELECT count(*) FROM ins)
		 WHERE id = $2
		RETURNING follower_count
	`, followerID, followeeID).Scan(&count)
	if err != nil {
		return fmt.Errorf("social: follow: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("social: commit: %w", err)
	}

	if count > s.threshold {
		// NX preserva o follow_at original em follow duplicado.
		if err := s.rdb.ZAddArgs(ctx, celebFollowsKey(followerID), redis.ZAddArgs{
			NX:      true,
			Members: []redis.Z{{Score: float64(time.Now().UnixMilli()), Member: followeeID}},
		}).Err(); err != nil {
			s.log.Warn("celeb_follows", "follower", followerID, "followee", followeeID, "err", err)
		}
	}

	s.invalidateFollowerCache(ctx, followeeID)
	s.cacheAddFollower(ctx, followeeID, followerID)
	return nil
}

func (s *Store) Unfollow(ctx context.Context, followerID, followeeID int64) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("social: begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	_, err = tx.Exec(ctx, `
		WITH del AS (
			DELETE FROM social.follows
			 WHERE follower_id = $1 AND followee_id = $2
			RETURNING 1
		)
		UPDATE social.users
		   SET follower_count = GREATEST(follower_count - (SELECT count(*) FROM del), 0)
		 WHERE id = $2
	`, followerID, followeeID)
	if err != nil {
		return fmt.Errorf("social: unfollow: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("social: commit: %w", err)
	}

	if err := s.rdb.ZRem(ctx, celebFollowsKey(followerID), followeeID).Err(); err != nil {
		s.log.Warn("celeb_follows remove", "follower", followerID, "followee", followeeID, "err", err)
	}

	s.invalidateFollowerCache(ctx, followeeID)
	s.cacheRemoveFollower(ctx, followeeID, followerID)
	return nil
}

func (s *Store) FollowerCount(ctx context.Context, authorID int64) (int, error) {
	var count int
	err := s.pool.QueryRow(ctx,
		`SELECT follower_count FROM social.users WHERE id = $1`, authorID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("social: follower count: %w", err)
	}
	return count, nil
}

func (s *Store) GetUser(ctx context.Context, id int64) (model.User, error) {
	var u model.User
	err := s.pool.QueryRow(ctx, `
		SELECT id, handle, display_name, follower_count
		  FROM social.users
		 WHERE id = $1
	`, id).Scan(&u.ID, &u.Handle, &u.DisplayName, &u.FollowerCount)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.User{}, err
		}
		return model.User{}, fmt.Errorf("social: get user: %w", err)
	}
	return u, nil
}
