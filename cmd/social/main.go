package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-chi/chi/v5"

	"github.com/marciojunior/wayposts/internal/auth"
	"github.com/marciojunior/wayposts/internal/platform/env"
	"github.com/marciojunior/wayposts/internal/platform/httpapi"
	"github.com/marciojunior/wayposts/internal/platform/metrics"
	"github.com/marciojunior/wayposts/internal/platform/pg"
	"github.com/marciojunior/wayposts/internal/platform/rds"
	"github.com/marciojunior/wayposts/internal/social"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	dsn := env.Get("DATABASE_URL", "postgres://wayposts:wayposts@localhost:5432/wayposts?sslmode=disable")
	addr := env.Get("SOCIAL_ADDR", ":8083")
	redisAddr := env.Get("REDIS_ADDR", "localhost:6379")
	threshold := env.Int("FANOUT_THRESHOLD", 10000)

	pool, err := pg.New(ctx, dsn, int32(env.Int("PG_MAX_CONNS", 20)))
	if err != nil {
		log.Error("postgres", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	rdb, err := rds.New(redisAddr, "", 50)
	if err != nil {
		log.Error("redis", "err", err)
		os.Exit(1)
	}
	defer rdb.Close()

	m := metrics.New("social")
	store := social.NewStore(pool, rdb, threshold, m, log)
	handler := social.NewHandler(store, log)

	r := chi.NewRouter()
	r.Use(m.Middleware, httpapi.Recover(log), httpapi.Logging(log))
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		httpapi.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	r.Handle("/metrics", m.Handler())

	r.Route("/v1", func(r chi.Router) {
		r.Use(auth.Dev)
		r.Post("/follow", handler.Follow)
		r.Delete("/follow/{id}", handler.Unfollow)
		r.Get("/users/{id}", handler.GetUser)
	})

	r.Get("/internal/followers/{id}", handler.Followers)
	r.Get("/internal/followers/{id}/count", handler.FollowerCount)
	r.Get("/internal/users", handler.Users)

	if err := httpapi.Serve(ctx, addr, r, log); err != nil {
		log.Error("serve", "err", err)
		os.Exit(1)
	}
}
