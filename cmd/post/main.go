package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/marciojunior/wayposts/internal/auth"
	"github.com/marciojunior/wayposts/internal/id"
	"github.com/marciojunior/wayposts/internal/media"
	"github.com/marciojunior/wayposts/internal/platform/env"
	"github.com/marciojunior/wayposts/internal/platform/httpapi"
	"github.com/marciojunior/wayposts/internal/platform/metrics"
	"github.com/marciojunior/wayposts/internal/platform/mq"
	"github.com/marciojunior/wayposts/internal/platform/pg"
	"github.com/marciojunior/wayposts/internal/platform/rds"
	"github.com/marciojunior/wayposts/internal/post"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	dsn := env.Get("DATABASE_URL", "postgres://wayposts:wayposts@localhost:5432/wayposts?sslmode=disable")
	addr := env.Get("POST_ADDR", ":8081")
	redisAddr := env.Get("REDIS_ADDR", "localhost:6379")
	rabbitURL := env.Get("RABBIT_URL", "amqp://guest:guest@localhost:5672/")
	shards := env.Int("FANOUT_SHARDS", 16)

	gen, err := id.New(env.Int64("NODE_ID", 1))
	if err != nil {
		log.Error("id generator", "err", err)
		os.Exit(1)
	}

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

	pub, err := mq.NewPublisher(rabbitURL)
	if err != nil {
		log.Error("rabbit", "err", err)
		os.Exit(1)
	}
	defer pub.Close()
	if err := pub.EnsureTopology(shards); err != nil {
		log.Error("rabbit topology", "err", err)
		os.Exit(1)
	}

	m := metrics.New("post")
	store := post.NewStore(pool, rdb, gen, log, m)

	mediaCfg := media.Config{
		Bucket:    env.Get("MEDIA_BUCKET", ""),
		Region:    env.Get("S3_REGION", "us-east-1"),
		Endpoint:  env.Get("S3_ENDPOINT", ""),
		AccessKey: env.Get("S3_ACCESS_KEY", ""),
		SecretKey: env.Get("S3_SECRET_KEY", ""),
		PathStyle: env.Bool("S3_PATH_STYLE", false),
	}
	var mediaStore media.Store
	if mediaCfg.Enabled() {
		s3store, err := media.NewS3(ctx, mediaCfg)
		if err != nil {
			log.Error("media", "err", err)
			os.Exit(1)
		}
		mediaStore = s3store
		log.Info("mídia habilitada", "bucket", mediaCfg.Bucket, "endpoint", mediaCfg.Endpoint)
	} else {
		log.Warn("mídia desabilitada: MEDIA_BUCKET vazio, posts só com texto")
	}

	handler := post.NewHandler(store, post.MediaOptions{
		Store:      mediaStore,
		MaxBytes:   env.Int64("MEDIA_MAX_BYTES", 10<<20),
		PresignTTL: env.Duration("MEDIA_PRESIGN_TTL", 10*time.Minute),
		PublicBase: env.Get("MEDIA_PUBLIC_BASE", ""),
	}, log)

	var wg sync.WaitGroup
	wg.Add(2)
	relay := post.NewRelay(pool, pub, shards, log)
	go func() {
		defer wg.Done()
		relay.Run(ctx)
	}()
	go func() {
		defer wg.Done()
		store.RunLikes(ctx)
	}()

	r := chi.NewRouter()
	r.Use(m.Middleware, httpapi.Recover(log), httpapi.Logging(log))
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		httpapi.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	r.Handle("/metrics", m.Handler())

	r.Group(func(r chi.Router) {
		r.Use(auth.Dev)
		r.Post("/v1/media/presign", handler.Presign)
		r.Post("/v1/posts", handler.Create)
		r.Get("/v1/posts/{id}", handler.Get)
		r.Delete("/v1/posts/{id}", handler.Delete)
		r.Post("/v1/posts/{id}/likes", handler.Like)
		r.Delete("/v1/posts/{id}/likes", handler.Unlike)
	})

	r.Get("/internal/posts", handler.GetMany)

	serveErr := httpapi.Serve(ctx, addr, r, log)
	// Cancela o contexto e espera o flush final de likes e o relay antes de
	// fechar o pool.
	stop()
	wg.Wait()
	if serveErr != nil {
		log.Error("serve", "err", serveErr)
		os.Exit(1)
	}
}
