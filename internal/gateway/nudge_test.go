package gateway

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"github.com/marciojunior/wayposts/internal/platform/metrics"
)

func newTestHub(t *testing.T) (*Hub, *redis.Client) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(mr.Close)

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	return NewHub(rdb, time.Hour, metrics.New("test"), log), rdb
}

func addClient(h *Hub, userID, lastSeen int64) *client {
	c := &client{userID: userID, send: make(chan Nudge, 4)}
	c.lastSeen.Store(lastSeen)
	h.add(c)
	return c
}

func TestSweepSendsNudgeForNewHead(t *testing.T) {
	h, rdb := newTestHub(t)
	ctx := context.Background()

	if err := rdb.ZAdd(ctx, "feed:11", redis.Z{Score: 100, Member: 100}).Err(); err != nil {
		t.Fatal(err)
	}
	if err := rdb.ZAdd(ctx, "feed:11", redis.Z{Score: 200, Member: 200}).Err(); err != nil {
		t.Fatal(err)
	}
	if err := rdb.ZAdd(ctx, "feed:11", redis.Z{Score: 300, Member: 300}).Err(); err != nil {
		t.Fatal(err)
	}

	c := addClient(h, 11, 200)
	defer h.remove(c)

	h.Sweep(ctx)
	select {
	case msg := <-c.send:
		if msg.Type != "new_posts" || msg.Count != 1 || msg.Head != 300 {
			t.Fatalf("nudge inesperado: %+v", msg)
		}
	default:
		t.Fatal("esperava nudge com head novo")
	}
	if got := c.lastSeen.Load(); got != 300 {
		t.Fatalf("lastSeen: got %d, want 300", got)
	}

	h.Sweep(ctx)
	select {
	case msg := <-c.send:
		t.Fatalf("nudge duplicado: %+v", msg)
	default:
	}
}

func TestSweepAdoptsHeadOnFirstScan(t *testing.T) {
	h, rdb := newTestHub(t)
	ctx := context.Background()

	if err := rdb.ZAdd(ctx, "feed:12", redis.Z{Score: 500, Member: 500}).Err(); err != nil {
		t.Fatal(err)
	}
	c := addClient(h, 12, 0)
	defer h.remove(c)

	h.Sweep(ctx)
	select {
	case msg := <-c.send:
		t.Fatalf("primeira varredura não deveria anunciar: %+v", msg)
	default:
	}
	if got := c.lastSeen.Load(); got != 500 {
		t.Fatalf("lastSeen: got %d, want 500", got)
	}
}

func TestSweepIgnoresEmptyFeed(t *testing.T) {
	h, _ := newTestHub(t)
	ctx := context.Background()

	c := addClient(h, 13, 100)
	defer h.remove(c)

	h.Sweep(ctx)
	select {
	case msg := <-c.send:
		t.Fatalf("feed vazio não deveria gerar nudge: %+v", msg)
	default:
	}
	if got := c.lastSeen.Load(); got != 100 {
		t.Fatalf("lastSeen não deveria mudar: got %d", got)
	}
}
