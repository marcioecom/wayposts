package gateway

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/redis/go-redis/v9"

	"github.com/marciojunior/wayposts/internal/platform/metrics"
)

type Nudge struct {
	Type  string `json:"type"`
	Count int64  `json:"count"`
	Head  int64  `json:"head"`
}

type Hub struct {
	shards   []*shard
	rdb      *redis.Client
	m        *metrics.Metrics
	interval time.Duration
	log      *slog.Logger
}

type shard struct {
	mu      sync.RWMutex
	clients map[*client]struct{}
}

type client struct {
	userID   int64
	lastSeen atomic.Int64
	send     chan Nudge
	conn     *websocket.Conn
}

func NewHub(rdb *redis.Client, interval time.Duration, m *metrics.Metrics, log *slog.Logger) *Hub {
	if interval <= 0 {
		interval = 30 * time.Second
	}
	const n = 64
	h := &Hub{
		shards:   make([]*shard, n),
		rdb:      rdb,
		m:        m,
		interval: interval,
		log:      log,
	}
	for i := range h.shards {
		h.shards[i] = &shard{clients: make(map[*client]struct{})}
	}
	return h
}

func (h *Hub) Run(ctx context.Context) {
	ticker := time.NewTicker(h.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			h.Sweep(ctx)
		}
	}
}

func (h *Hub) shardFor(userID int64) *shard {
	return h.shards[int(userID%int64(len(h.shards)))]
}

func (h *Hub) add(c *client) {
	sh := h.shardFor(c.userID)
	sh.mu.Lock()
	sh.clients[c] = struct{}{}
	sh.mu.Unlock()
	h.m.WSConnections.Inc()
}

func (h *Hub) remove(c *client) {
	sh := h.shardFor(c.userID)
	sh.mu.Lock()
	if _, ok := sh.clients[c]; ok {
		delete(sh.clients, c)
		h.m.WSConnections.Dec()
	}
	sh.mu.Unlock()
}

func (h *Hub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	uid := devUserID(r)
	if uid <= 0 {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	lastSeen, _ := strconv.ParseInt(r.URL.Query().Get("last_seen"), 10, 64)
	if lastSeen < 0 {
		lastSeen = 0
	}

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		// Laboratório: qualquer origem pode abrir a conexão. Em produção a
		// origem seria validada junto com o JWT.
		InsecureSkipVerify: true,
	})
	if err != nil {
		h.log.Warn("ws accept", "user", uid, "err", err)
		return
	}
	conn.SetReadLimit(1024)

	c := &client{
		userID: uid,
		send:   make(chan Nudge, 4),
		conn:   conn,
	}
	c.lastSeen.Store(lastSeen)
	h.add(c)
	defer h.remove(c)

	ctx := conn.CloseRead(r.Context())
	go c.writeLoop(ctx)
	<-ctx.Done()
}

func (c *client) writeLoop(ctx context.Context) {
	ping := time.NewTicker(25 * time.Second)
	defer ping.Stop()

	for {
		select {
		case msg := <-c.send:
			writeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			err := wsjson.Write(writeCtx, c.conn, msg)
			cancel()
			if err != nil {
				_ = c.conn.Close(websocket.StatusInternalError, "write failed")
				return
			}
		case <-ping.C:
			pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			err := c.conn.Ping(pingCtx)
			cancel()
			if err != nil {
				_ = c.conn.Close(websocket.StatusGoingAway, "ping timeout")
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

func (h *Hub) Sweep(ctx context.Context) {
	start := time.Now()

	var wg sync.WaitGroup
	sem := make(chan struct{}, 16)
	for _, sh := range h.shards {
		sh.mu.RLock()
		if len(sh.clients) == 0 {
			sh.mu.RUnlock()
			continue
		}
		clients := make([]*client, 0, len(sh.clients))
		for c := range sh.clients {
			clients = append(clients, c)
		}
		sh.mu.RUnlock()

		wg.Add(1)
		sem <- struct{}{}
		go func(clients []*client) {
			defer wg.Done()
			defer func() { <-sem }()
			h.sweepClients(ctx, clients)
		}(clients)
	}
	wg.Wait()

	h.m.WSSweep.Observe(time.Since(start).Seconds())
}

type pendingNudge struct {
	c    *client
	head int64
}

func (h *Hub) sweepClients(ctx context.Context, clients []*client) {
	pipe := h.rdb.Pipeline()
	heads := make([]*redis.ZSliceCmd, len(clients))
	for i, c := range clients {
		heads[i] = pipe.ZRevRangeWithScores(ctx, feedKey(c.userID), 0, 0)
	}
	if _, err := pipe.Exec(ctx); err != nil && err != redis.Nil {
		h.log.Warn("ws sweep heads", "err", err)
		return
	}

	var need []pendingNudge
	for i, c := range clients {
		zs := heads[i].Val()
		if len(zs) == 0 {
			continue
		}
		head := int64(zs[0].Score)
		last := c.lastSeen.Load()
		if last == 0 {
			c.lastSeen.Store(head)
			continue
		}
		if head <= last {
			continue
		}
		need = append(need, pendingNudge{c: c, head: head})
	}
	if len(need) == 0 {
		return
	}

	counts := make([]*redis.IntCmd, len(need))
	pipe = h.rdb.Pipeline()
	for i, p := range need {
		counts[i] = pipe.ZCount(ctx, feedKey(p.c.userID),
			"("+strconv.FormatInt(p.c.lastSeen.Load(), 10), "+inf")
	}
	if _, err := pipe.Exec(ctx); err != nil && err != redis.Nil {
		h.log.Warn("ws sweep counts", "err", err)
		return
	}

	for i, p := range need {
		count := counts[i].Val()
		select {
		case p.c.send <- Nudge{Type: "new_posts", Count: count, Head: p.head}:
			p.c.lastSeen.Store(p.head)
			h.m.WSNudges.Inc()
		default:
			// Canal cheio: o próximo ciclo tenta de novo.
		}
	}
}

func feedKey(userID int64) string {
	return "feed:" + strconv.FormatInt(userID, 10)
}

func devUserID(r *http.Request) int64 {
	raw := r.URL.Query().Get("token")
	if raw == "" {
		const prefix = "Bearer "
		h := r.Header.Get("Authorization")
		if strings.HasPrefix(h, prefix) {
			raw = strings.TrimSpace(strings.TrimPrefix(h, prefix))
		}
	}
	uid, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || uid <= 0 {
		return 0
	}
	return uid
}
