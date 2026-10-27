package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/marciojunior/wayposts/db/migrations"
	"github.com/marciojunior/wayposts/internal/platform/env"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	dsn := env.Get("DATABASE_URL", "postgres://wayposts:wayposts@localhost:5432/wayposts?sslmode=disable")

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Error("connect", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	if _, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS public.schema_migrations (
			name       text PRIMARY KEY,
			applied_at timestamptz NOT NULL DEFAULT now()
		)`); err != nil {
		log.Error("schema_migrations", "err", err)
		os.Exit(1)
	}

	entries, err := migrations.FS.ReadDir(".")
	if err != nil {
		log.Error("read migrations", "err", err)
		os.Exit(1)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })

	applied := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		var exists bool
		if err := pool.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM public.schema_migrations WHERE name = $1)`,
			entry.Name(),
		).Scan(&exists); err != nil {
			log.Error("check migration", "name", entry.Name(), "err", err)
			os.Exit(1)
		}
		if exists {
			continue
		}

		body, err := migrations.FS.ReadFile(entry.Name())
		if err != nil {
			log.Error("read migration", "name", entry.Name(), "err", err)
			os.Exit(1)
		}

		tx, err := pool.Begin(ctx)
		if err != nil {
			log.Error("begin", "name", entry.Name(), "err", err)
			os.Exit(1)
		}
		if _, err := tx.Exec(ctx, string(body)); err != nil {
			_ = tx.Rollback(ctx)
			log.Error("apply migration", "name", entry.Name(), "err", err)
			os.Exit(1)
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO public.schema_migrations (name) VALUES ($1)`, entry.Name(),
		); err != nil {
			_ = tx.Rollback(ctx)
			log.Error("record migration", "name", entry.Name(), "err", err)
			os.Exit(1)
		}
		if err := tx.Commit(ctx); err != nil {
			log.Error("commit", "name", entry.Name(), "err", err)
			os.Exit(1)
		}

		applied++
		log.Info("migration applied", "name", entry.Name())
	}

	log.Info("migrations up to date", "applied", applied)
}
