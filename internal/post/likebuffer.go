package post

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/marciojunior/wayposts/internal/platform/metrics"
)

type likeEvent struct {
	userID int64
	postID int64
	liked  bool
}

type LikeBuffer struct {
	mu          sync.Mutex
	events      []likeEvent
	pool        *pgxpool.Pool
	m           *metrics.Metrics
	interval    time.Duration
	maxBatch    int
	maxBuffered int
	log         *slog.Logger
}

func NewLikeBuffer(pool *pgxpool.Pool, log *slog.Logger, m *metrics.Metrics) *LikeBuffer {
	return &LikeBuffer{
		pool:        pool,
		m:           m,
		interval:    100 * time.Millisecond,
		maxBatch:    5000,
		maxBuffered: 50_000,
		log:         log,
	}
}

// Add descarta o evento quando o buffer estoura. O SLO é do caminho de leitura,
// e um flush travado não pode derrubar o serviço.
func (b *LikeBuffer) Add(userID, postID int64, liked bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if len(b.events) >= b.maxBuffered {
		b.log.Warn("like buffer cheio", "user", userID, "post", postID, "liked", liked)
		return
	}
	b.events = append(b.events, likeEvent{userID: userID, postID: postID, liked: liked})
}

func (b *LikeBuffer) Run(ctx context.Context) {
	ticker := time.NewTicker(b.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			flushCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			if err := b.Flush(flushCtx); err != nil {
				b.log.Error("like flush final", "err", err)
			}
			cancel()
			return
		case <-ticker.C:
			if err := b.Flush(ctx); err != nil {
				b.log.Error("like flush", "err", err)
			}
		}
	}
}

// Flush coalesce por (user, post) com a última escrita vencendo, o que preserva
// a ordem like seguido de unlike na mesma janela.
func (b *LikeBuffer) Flush(ctx context.Context) error {
	for {
		events := b.take(b.maxBatch)
		if len(events) == 0 {
			return nil
		}
		if err := b.persist(ctx, events); err != nil {
			b.m.LikeFlushErrors.Inc()
			b.requeue(events)
			return err
		}
		if len(events) < b.maxBatch {
			return nil
		}
	}
}

func (b *LikeBuffer) take(n int) []likeEvent {
	b.mu.Lock()
	defer b.mu.Unlock()
	if len(b.events) == 0 {
		return nil
	}
	if n > len(b.events) {
		n = len(b.events)
	}
	out := make([]likeEvent, n)
	copy(out, b.events[:n])
	b.events = b.events[n:]
	return out
}

func (b *LikeBuffer) requeue(events []likeEvent) {
	b.mu.Lock()
	defer b.mu.Unlock()
	room := b.maxBuffered - len(b.events)
	if room <= 0 {
		b.log.Error("like buffer sem espaço para requeue", "descartados", len(events))
		return
	}
	if len(events) > room {
		events = events[len(events)-room:]
	}
	// Eventos antigos voltam para a frente.
	b.events = append(events, b.events...)
}

func coalesce(events []likeEvent) (inserts, deletes []likeEvent) {
	type key struct {
		userID int64
		postID int64
	}
	last := make(map[key]likeEvent, len(events))
	order := make([]key, 0, len(events))
	for _, e := range events {
		k := key{e.userID, e.postID}
		if _, ok := last[k]; !ok {
			order = append(order, k)
		}
		last[k] = e
	}
	for _, k := range order {
		e := last[k]
		if e.liked {
			inserts = append(inserts, e)
		} else {
			deletes = append(deletes, e)
		}
	}
	return inserts, deletes
}

func (b *LikeBuffer) persist(ctx context.Context, events []likeEvent) error {
	inserts, deletes := coalesce(events)
	if len(inserts) == 0 && len(deletes) == 0 {
		return nil
	}

	tx, err := b.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	deltas := make(map[int64]int64, len(events))

	if len(inserts) > 0 {
		users := make([]int64, len(inserts))
		posts := make([]int64, len(inserts))
		for i, e := range inserts {
			users[i] = e.userID
			posts[i] = e.postID
		}
		rows, err := tx.Query(ctx, `
			WITH ins AS (
				INSERT INTO post.likes (user_id, post_id)
				SELECT * FROM unnest($1::bigint[], $2::bigint[])
				ON CONFLICT DO NOTHING
				RETURNING post_id
			)
			SELECT post_id, count(*)::bigint FROM ins GROUP BY post_id
		`, users, posts)
		if err != nil {
			return err
		}
		for rows.Next() {
			var pid, n int64
			if err := rows.Scan(&pid, &n); err != nil {
				rows.Close()
				return err
			}
			deltas[pid] += n
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}
	}

	if len(deletes) > 0 {
		users := make([]int64, len(deletes))
		posts := make([]int64, len(deletes))
		for i, e := range deletes {
			users[i] = e.userID
			posts[i] = e.postID
		}
		rows, err := tx.Query(ctx, `
			WITH del AS (
				DELETE FROM post.likes l
				USING unnest($1::bigint[], $2::bigint[]) AS d(user_id, post_id)
				WHERE l.user_id = d.user_id AND l.post_id = d.post_id
				RETURNING l.post_id
			)
			SELECT post_id, count(*)::bigint FROM del GROUP BY post_id
		`, users, posts)
		if err != nil {
			return err
		}
		for rows.Next() {
			var pid, n int64
			if err := rows.Scan(&pid, &n); err != nil {
				rows.Close()
				return err
			}
			deltas[pid] -= n
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}
	}

	if len(deltas) > 0 {
		ids := make([]int64, 0, len(deltas))
		ds := make([]int64, 0, len(deltas))
		for pid, delta := range deltas {
			if delta == 0 {
				continue
			}
			ids = append(ids, pid)
			ds = append(ds, delta)
		}
		if len(ids) > 0 {
			if _, err := tx.Exec(ctx, `
				UPDATE post.posts p
				   SET like_count = GREATEST(p.like_count + d.delta, 0)
				  FROM unnest($1::bigint[], $2::bigint[]) AS d(post_id, delta)
				 WHERE p.id = d.post_id
			`, ids, ds); err != nil {
				return err
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}
	b.m.LikeFlushBatch.Observe(float64(len(events)))
	return nil
}
