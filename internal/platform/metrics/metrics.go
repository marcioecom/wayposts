package metrics

import (
	"bufio"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Metrics struct {
	registry *prometheus.Registry

	HTTPRequests *prometheus.CounterVec
	HTTPDuration *prometheus.HistogramVec
	HTTPInFlight prometheus.Gauge

	PostsCreated prometheus.Counter
	PostCache    *prometheus.CounterVec
	Likes        *prometheus.CounterVec

	LikeFlushBatch  prometheus.Histogram
	LikeFlushErrors prometheus.Counter

	FanoutMessages *prometheus.CounterVec
	FanoutDuration prometheus.Histogram
	FanoutLag      prometheus.Histogram
	FanoutPushed   prometheus.Counter

	WSConnections prometheus.Gauge
	WSNudges      prometheus.Counter
	WSSweep       prometheus.Histogram

	MediaPresign    *prometheus.CounterVec
	MediaValidation *prometheus.CounterVec

	FollowerCache *prometheus.CounterVec
	UserCache     *prometheus.CounterVec
}

func New(service string) *Metrics {
	m := &Metrics{
		registry: prometheus.NewRegistry(),

		HTTPRequests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "wayposts_http_requests_total",
			Help: "Requisições HTTP por método, rota e status.",
		}, []string{"method", "route", "status"}),
		HTTPDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "wayposts_http_request_duration_seconds",
			Help:    "Latência das requisições HTTP.",
			Buckets: []float64{0.001, 0.0025, 0.005, 0.01, 0.025, 0.05, 0.075, 0.1, 0.15, 0.25, 0.5, 1, 2.5, 5, 10},
		}, []string{"method", "route"}),
		HTTPInFlight: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "wayposts_http_in_flight",
			Help: "Requisições HTTP em andamento.",
		}),

		PostsCreated: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "wayposts_posts_created_total",
			Help: "Posts criados.",
		}),
		PostCache: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "wayposts_post_cache_total",
			Help: "Leituras de corpo de post por resultado.",
		}, []string{"result"}),
		Likes: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "wayposts_likes_total",
			Help: "Operações de like por tipo.",
		}, []string{"action"}),

		LikeFlushBatch: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "wayposts_like_flush_batch_size",
			Help:    "Eventos por flush de likes.",
			Buckets: []float64{1, 10, 50, 100, 250, 500, 1000, 2500, 5000},
		}),
		LikeFlushErrors: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "wayposts_like_flush_errors_total",
			Help: "Flushes de likes que falharam.",
		}),

		FanoutMessages: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "wayposts_fanout_messages_total",
			Help: "Mensagens de fanout por resultado.",
		}, []string{"result"}),
		FanoutDuration: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "wayposts_fanout_duration_seconds",
			Help:    "Tempo de processamento de um evento de fanout.",
			Buckets: []float64{0.001, 0.0025, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30},
		}),
		FanoutLag: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "wayposts_fanout_lag_seconds",
			Help:    "Intervalo entre a criação do post e o fim do fanout.",
			Buckets: []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1, 2, 5, 10, 30, 60},
		}),
		FanoutPushed: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "wayposts_fanout_feed_writes_total",
			Help: "Escritas de post em feeds de seguidores.",
		}),

		WSConnections: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "wayposts_ws_connections",
			Help: "Conexões WebSocket abertas.",
		}),
		WSNudges: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "wayposts_ws_nudges_total",
			Help: "Nudges de novos posts enviados.",
		}),
		WSSweep: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "wayposts_ws_sweep_duration_seconds",
			Help:    "Duração de uma varredura de nudge.",
			Buckets: []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30, 60, 120},
		}),

		MediaPresign: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "wayposts_media_presign_total",
			Help: "URLs de upload emitidas por resultado.",
		}, []string{"result"}),
		MediaValidation: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "wayposts_media_validation_total",
			Help: "Validações de mídia na criação do post por resultado.",
		}, []string{"result"}),

		FollowerCache: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "wayposts_follower_cache_total",
			Help: "Leituras de seguidores por resultado.",
		}, []string{"result"}),
		UserCache: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "wayposts_user_cache_total",
			Help: "Resumos de autor lidos do cache, contados por id.",
		}, []string{"result"}),
	}

	reg := prometheus.WrapRegistererWith(prometheus.Labels{"service": service}, m.registry)
	reg.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		m.HTTPRequests,
		m.HTTPDuration,
		m.HTTPInFlight,
		m.PostsCreated,
		m.PostCache,
		m.Likes,
		m.LikeFlushBatch,
		m.LikeFlushErrors,
		m.FanoutMessages,
		m.FanoutDuration,
		m.FanoutLag,
		m.FanoutPushed,
		m.WSConnections,
		m.WSNudges,
		m.WSSweep,
		m.MediaPresign,
		m.MediaValidation,
		m.FollowerCache,
		m.UserCache,
	)
	// Registra uma série por label para o primeiro scrape já mostrar a métrica.
	m.HTTPRequests.WithLabelValues("GET", "/healthz", "200").Add(0)
	m.PostCache.WithLabelValues("hit").Add(0)
	m.PostCache.WithLabelValues("miss").Add(0)
	m.Likes.WithLabelValues("like").Add(0)
	m.Likes.WithLabelValues("unlike").Add(0)
	m.FanoutMessages.WithLabelValues("ok").Add(0)
	m.FanoutMessages.WithLabelValues("retry").Add(0)
	m.FanoutMessages.WithLabelValues("dlq").Add(0)
	m.MediaPresign.WithLabelValues("ok").Add(0)
	m.MediaPresign.WithLabelValues("rejected").Add(0)
	m.MediaValidation.WithLabelValues("ok").Add(0)
	m.MediaValidation.WithLabelValues("disabled").Add(0)
	m.MediaValidation.WithLabelValues("missing").Add(0)
	m.MediaValidation.WithLabelValues("too_large").Add(0)
	m.MediaValidation.WithLabelValues("wrong_type").Add(0)
	m.MediaValidation.WithLabelValues("not_owned").Add(0)
	m.FollowerCache.WithLabelValues("hit").Add(0)
	m.FollowerCache.WithLabelValues("miss").Add(0)
	m.FollowerCache.WithLabelValues("rebuild").Add(0)
	m.FollowerCache.WithLabelValues("stale_retry").Add(0)
	m.FollowerCache.WithLabelValues("error").Add(0)
	m.UserCache.WithLabelValues("hit").Add(0)
	m.UserCache.WithLabelValues("miss").Add(0)
	m.UserCache.WithLabelValues("absent").Add(0)
	m.UserCache.WithLabelValues("redis_error").Add(0)

	return m
}

func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
}

// Middleware usa o padrão de rota do chi, não o path, para não explodir a
// cardinalidade.
func (m *Metrics) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m.HTTPInFlight.Inc()
		defer m.HTTPInFlight.Dec()

		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)

		route := "unmatched"
		if rc := chi.RouteContext(r.Context()); rc != nil && rc.RoutePattern() != "" {
			route = rc.RoutePattern()
		}

		// O handler de WS só retorna quando a conexão fecha; conta o handshake
		// sem jogar a duração da sessão no histograma de HTTP.
		if route == "/v1/ws" {
			m.HTTPRequests.WithLabelValues(r.Method, route, "101").Inc()
			return
		}

		m.HTTPRequests.WithLabelValues(r.Method, route, strconv.Itoa(rec.status)).Inc()
		m.HTTPDuration.WithLabelValues(r.Method, route).Observe(time.Since(start).Seconds())
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status  int
	written bool
}

func (r *statusRecorder) WriteHeader(status int) {
	if r.written {
		return
	}
	r.written = true
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	if !r.written {
		r.WriteHeader(http.StatusOK)
	}
	return r.ResponseWriter.Write(b)
}

func (r *statusRecorder) Unwrap() http.ResponseWriter {
	return r.ResponseWriter
}

// Sem Hijack, o handler de WS recebe 501 Not Implemented.
func (r *statusRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hj, ok := r.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, http.ErrNotSupported
	}
	r.written = true
	r.status = http.StatusSwitchingProtocols
	return hj.Hijack()
}

func (r *statusRecorder) Flush() {
	if f, ok := r.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}
