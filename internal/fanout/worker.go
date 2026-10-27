package fanout

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	socialclient "github.com/marciojunior/wayposts/internal/clients/social"
	"github.com/marciojunior/wayposts/internal/events"
	"github.com/marciojunior/wayposts/internal/id"
	"github.com/marciojunior/wayposts/internal/platform/metrics"
)

type Worker struct {
	rdb       *redis.Client
	social    *socialclient.Client
	m         *metrics.Metrics
	threshold int
	feedCap   int
	chunk     int
	log       *slog.Logger
}

func NewWorker(rdb *redis.Client, social *socialclient.Client, threshold, feedCap, chunk int, m *metrics.Metrics, log *slog.Logger) *Worker {
	if chunk <= 0 {
		chunk = 1000
	}
	return &Worker{rdb: rdb, social: social, m: m, threshold: threshold, feedCap: feedCap, chunk: chunk, log: log}
}

func (w *Worker) Handle(ctx context.Context, body []byte) error {
	start := time.Now()
	defer func() {
		w.m.FanoutDuration.Observe(time.Since(start).Seconds())
	}()

	var ev events.PostCreated
	if err := json.Unmarshal(body, &ev); err != nil {
		return fmt.Errorf("fanout: evento inválido: %w", err)
	}
	if ev.PostID <= 0 || ev.AuthorID <= 0 {
		return fmt.Errorf("fanout: evento sem ids")
	}

	count, err := w.social.FollowerCount(ctx, ev.AuthorID)
	if err != nil {
		return err
	}
	if count > w.threshold {
		// Celebridade: o post é lido no merge, não materializado.
		return nil
	}

	after := int64(0)
	for {
		followers, err := w.social.Followers(ctx, ev.AuthorID, after, w.chunk)
		if err != nil {
			return err
		}
		if len(followers) == 0 {
			break
		}
		if err := w.push(ctx, followers, ev.PostID); err != nil {
			return err
		}
		w.m.FanoutPushed.Add(float64(len(followers)))
		next := followers[len(followers)-1]
		if next <= after {
			break
		}
		after = next
	}

	w.m.FanoutLag.Observe(time.Since(id.Time(ev.PostID)).Seconds())
	return nil
}

func (w *Worker) push(ctx context.Context, followers []int64, postID int64) error {
	pipe := w.rdb.Pipeline()
	for _, uid := range followers {
		key := "feed:" + strconv.FormatInt(uid, 10)
		pipe.ZAdd(ctx, key, redis.Z{Score: id.Float(postID), Member: postID})
		pipe.ZRemRangeByRank(ctx, key, 0, -int64(w.feedCap+1))
	}
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("fanout: push: %w", err)
	}
	return nil
}
