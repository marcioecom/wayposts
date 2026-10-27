package main

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/marciojunior/wayposts/internal/auth"
	"github.com/marciojunior/wayposts/internal/gateway"
	"github.com/marciojunior/wayposts/internal/platform/env"
	"github.com/marciojunior/wayposts/internal/platform/httpapi"
	"github.com/marciojunior/wayposts/internal/platform/metrics"
	"github.com/marciojunior/wayposts/internal/platform/rds"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	addr := env.Get("GATEWAY_ADDR", ":8080")
	postURL := env.Get("POST_PUBLIC_URL", "http://localhost:8081")
	timelineURL := env.Get("TIMELINE_PUBLIC_URL", "http://localhost:8082")
	socialURL := env.Get("SOCIAL_PUBLIC_URL", "http://localhost:8083")

	postProxy := proxy(postURL, log)
	timelineProxy := proxy(timelineURL, log)
	socialProxy := proxy(socialURL, log)
	m := metrics.New("gateway")

	redisAddr := env.Get("REDIS_ADDR", "localhost:6379")
	rdb, err := rds.New(redisAddr, "", 20)
	if err != nil {
		log.Error("redis", "err", err)
		os.Exit(1)
	}
	defer rdb.Close()
	hub := gateway.NewHub(rdb, env.Duration("NUDGE_INTERVAL", 30*time.Second), m, log)
	go hub.Run(ctx)

	r := chi.NewRouter()
	r.Use(m.Middleware, httpapi.Recover(log), httpapi.Logging(log))
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		httpapi.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	r.Handle("/metrics", m.Handler())
	r.Get("/v1/ws", hub.ServeHTTP)

	r.Route("/v1", func(r chi.Router) {
		r.Use(auth.Dev)

		r.Handle("/posts", postProxy)
		r.Handle("/posts/*", postProxy)
		r.Handle("/media", postProxy)
		r.Handle("/media/*", postProxy)

		r.Handle("/timeline", timelineProxy)
		r.Handle("/timeline/*", timelineProxy)

		r.Handle("/follow", socialProxy)
		r.Handle("/follow/*", socialProxy)
		r.Handle("/users", socialProxy)
		r.Handle("/users/*", socialProxy)
	})

	if err := httpapi.Serve(ctx, addr, r, log); err != nil {
		log.Error("serve", "err", err)
		os.Exit(1)
	}
}

func proxy(target string, log *slog.Logger) http.Handler {
	u, err := url.Parse(target)
	if err != nil {
		log.Error("proxy url", "target", target, "err", err)
		os.Exit(1)
	}
	p := httputil.NewSingleHostReverseProxy(u)
	p.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Error("upstream", "target", target, "path", r.URL.Path, "err", err)
		httpapi.Error(w, http.StatusBadGateway, "upstream unavailable")
	}
	return p
}
