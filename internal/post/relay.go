package post

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/marciojunior/wayposts/internal/events"
	"github.com/marciojunior/wayposts/internal/platform/mq"
)

type Relay struct {
	pool     *pgxpool.Pool
	pub      *mq.Publisher
	shards   int
	interval time.Duration
	batch    int
	log      *slog.Logger
}

func NewRelay(pool *pgxpool.Pool, pub *mq.Publisher, shards int, log *slog.Logger) *Relay {
	return &Relay{
		pool:     pool,
		pub:      pub,
		shards:   shards,
		interval: 50 * time.Millisecond,
		batch:    200,
		log:      log,
	}
}

func (r *Relay) Run(ctx context.Context) {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := r.drain(ctx); err != nil {
				r.log.Error("outbox drain", "err", err)
			}
		}
	}
}

type outboxItem struct {
	id      int64
	payload []byte
}

func (r *Relay) drain(ctx context.Context) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("relay: begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	rows, err := tx.Query(ctx, `
		SELECT id, payload
		  FROM post.outbox
		 WHERE published_at IS NULL
		 ORDER BY id
		 LIMIT $1
		 FOR UPDATE SKIP LOCKED
	`, r.batch)
	if err != nil {
		return fmt.Errorf("relay: select: %w", err)
	}

	var items []outboxItem
	for rows.Next() {
		var it outboxItem
		if err := rows.Scan(&it.id, &it.payload); err != nil {
			rows.Close()
			return err
		}
		items = append(items, it)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	if len(items) == 0 {
		return nil
	}

	ids := make([]int64, 0, len(items))
	for _, it := range items {
		var ev events.PostCreated
		if err := json.Unmarshal(it.payload, &ev); err != nil {
			return fmt.Errorf("relay: payload do evento %d: %w", it.id, err)
		}
		key := mq.QueueName(mq.Shard(ev.AuthorID, r.shards))
		if err := r.pub.Publish(ctx, key, it.payload); err != nil {
			return fmt.Errorf("relay: publish %d: %w", it.id, err)
		}
		ids = append(ids, it.id)
	}

	if _, err := tx.Exec(ctx, `UPDATE post.outbox SET published_at = now() WHERE id = ANY($1)`, ids); err != nil {
		return fmt.Errorf("relay: mark: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("relay: commit: %w", err)
	}
	return nil
}
