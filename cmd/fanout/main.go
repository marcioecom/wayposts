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

	amqp "github.com/rabbitmq/amqp091-go"

	socialclient "github.com/marciojunior/wayposts/internal/clients/social"
	"github.com/marciojunior/wayposts/internal/fanout"
	"github.com/marciojunior/wayposts/internal/platform/env"
	"github.com/marciojunior/wayposts/internal/platform/httpapi"
	"github.com/marciojunior/wayposts/internal/platform/metrics"
	"github.com/marciojunior/wayposts/internal/platform/mq"
	"github.com/marciojunior/wayposts/internal/platform/rds"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	redisAddr := env.Get("REDIS_ADDR", "localhost:6379")
	rabbitURL := env.Get("RABBIT_URL", "amqp://guest:guest@localhost:5672/")
	socialURL := env.Get("SOCIAL_PUBLIC_URL", "http://localhost:8083")
	healthAddr := env.Get("FANOUT_ADDR", ":8084")
	shards := env.Int("FANOUT_SHARDS", 16)
	threshold := env.Int("FANOUT_THRESHOLD", 10000)
	feedCap := env.Int("FEED_CAP", 500)
	prefetch := env.Int("FANOUT_PREFETCH", 50)

	rdb, err := rds.New(redisAddr, "", 100)
	if err != nil {
		log.Error("redis", "err", err)
		os.Exit(1)
	}
	defer rdb.Close()

	social := socialclient.New(socialURL)
	m := metrics.New("fanout")
	worker := fanout.NewWorker(rdb, social, threshold, feedCap, 1000, m, log)

	conn, err := mq.Dial(rabbitURL)
	if err != nil {
		log.Error("rabbit", "err", err)
		os.Exit(1)
	}
	defer conn.Close()

	setup, err := mq.DeclareFanout(conn, shards)
	if err != nil {
		log.Error("declare", "err", err)
		os.Exit(1)
	}
	_ = setup.Close()

	var wg sync.WaitGroup
	for i := 0; i < shards; i++ {
		queue := mq.QueueName(i)
		ch, err := conn.Channel()
		if err != nil {
			log.Error("channel", "queue", queue, "err", err)
			os.Exit(1)
		}
		if err := ch.Qos(prefetch, 0, false); err != nil {
			log.Error("qos", "queue", queue, "err", err)
			os.Exit(1)
		}
		deliveries, err := ch.Consume(queue, "", false, false, false, false, nil)
		if err != nil {
			log.Error("consume", "queue", queue, "err", err)
			os.Exit(1)
		}

		wg.Add(1)
		go func(queue string, ch *amqp.Channel, deliveries <-chan amqp.Delivery) {
			defer wg.Done()
			defer ch.Close()
			consume(queue, ch, deliveries, worker, m, log, stop)
		}(queue, ch, deliveries)
	}

	health := http.NewServeMux()
	health.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		httpapi.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	health.Handle("/metrics", m.Handler())
	go func() {
		if err := httpapi.Serve(ctx, healthAddr, health, log); err != nil {
			log.Error("health server", "err", err)
		}
	}()

	log.Info("fanout consumindo", "shards", shards, "threshold", threshold)
	<-ctx.Done()
	wg.Wait()
}

func consume(queue string, ch *amqp.Channel, deliveries <-chan amqp.Delivery, worker *fanout.Worker, m *metrics.Metrics, log *slog.Logger, stop context.CancelFunc) {
	for {
		msg, ok := <-deliveries
		if !ok {
			log.Error("consumer encerrado", "queue", queue)
			stop()
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		err := worker.Handle(ctx, msg.Body)
		cancel()

		if err != nil {
			if !msg.Redelivered {
				m.FanoutMessages.WithLabelValues("retry").Inc()
				log.Warn("fanout retry", "queue", queue, "err", err)
				_ = msg.Nack(false, true)
				continue
			}
			m.FanoutMessages.WithLabelValues("dlq").Inc()
			log.Error("fanout para DLQ", "queue", queue, "err", err)
			_ = msg.Nack(false, false)
			continue
		}
		m.FanoutMessages.WithLabelValues("ok").Inc()
		if err := msg.Ack(false); err != nil {
			log.Error("ack", "queue", queue, "err", err)
		}
	}
}
