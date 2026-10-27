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
	postclient "github.com/marciojunior/wayposts/internal/clients/post"
	"github.com/marciojunior/wayposts/internal/platform/env"
	"github.com/marciojunior/wayposts/internal/platform/httpapi"
	"github.com/marciojunior/wayposts/internal/platform/metrics"
	"github.com/marciojunior/wayposts/internal/platform/rds"
	"github.com/marciojunior/wayposts/internal/timeline"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	addr := env.Get("TIMELINE_ADDR", ":8082")
	redisAddr := env.Get("REDIS_ADDR", "localhost:6379")
	postURL := env.Get("POST_PUBLIC_URL", "http://localhost:8081")

	rdb, err := rds.New(redisAddr, "", 50)
	if err != nil {
		log.Error("redis", "err", err)
		os.Exit(1)
	}
	defer rdb.Close()

	posts := postclient.New(postURL)
	svc := timeline.NewService(rdb, posts, env.Int("MAX_CELEBS", 100), log)
	handler := timeline.NewHandler(svc, log)
	m := metrics.New("timeline")

	r := chi.NewRouter()
	r.Use(m.Middleware, httpapi.Recover(log), httpapi.Logging(log))
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		httpapi.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	r.Handle("/metrics", m.Handler())

	r.Group(func(r chi.Router) {
		r.Use(auth.Dev)
		r.Get("/v1/timeline/home", handler.Home)
		r.Get("/v1/timeline/profile/{id}", handler.Profile)
	})

	if err := httpapi.Serve(ctx, addr, r, log); err != nil {
		log.Error("serve", "err", err)
		os.Exit(1)
	}
}
