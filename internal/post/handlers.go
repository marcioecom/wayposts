package post

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/marciojunior/wayposts/internal/auth"
	"github.com/marciojunior/wayposts/internal/media"
	"github.com/marciojunior/wayposts/internal/model"
	"github.com/marciojunior/wayposts/internal/platform/httpapi"
)

const (
	maxBodyLen    = 2000
	maxMediaCount = 4
)

type MediaOptions struct {
	Store      media.Store
	MaxBytes   int64
	PresignTTL time.Duration
	PublicBase string
}

type Handler struct {
	store *Store
	media MediaOptions
	log   *slog.Logger
}

func NewHandler(store *Store, opts MediaOptions, log *slog.Logger) *Handler {
	if opts.MaxBytes <= 0 {
		opts.MaxBytes = 10 << 20
	}
	if opts.PresignTTL <= 0 {
		opts.PresignTTL = 10 * time.Minute
	}
	return &Handler{store: store, media: opts, log: log}
}

func (h *Handler) withMediaURLs(p model.Post) model.Post {
	if h.media.PublicBase == "" || len(p.Media) == 0 {
		return p
	}
	out := make([]string, len(p.Media))
	for i, key := range p.Media {
		out[i] = strings.TrimSuffix(h.media.PublicBase, "/") + "/" + key
	}
	p.Media = out
	return p
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	uid, ok := auth.UserID(r.Context())
	if !ok {
		httpapi.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req struct {
		Body  string   `json:"body"`
		Media []string `json:"media"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpapi.Error(w, http.StatusBadRequest, "invalid json")
		return
	}
	req.Body = strings.TrimSpace(req.Body)
	if req.Body == "" {
		httpapi.Error(w, http.StatusBadRequest, "body is required")
		return
	}
	if len([]rune(req.Body)) > maxBodyLen {
		httpapi.Error(w, http.StatusBadRequest, "body too long")
		return
	}
	if len(req.Media) > maxMediaCount {
		httpapi.Error(w, http.StatusBadRequest, "too many media items")
		return
	}
	if err := h.validateMedia(r.Context(), uid, req.Media); err != nil {
		var bad *mediaValidationError
		if errors.As(err, &bad) {
			httpapi.Error(w, http.StatusBadRequest, bad.reason)
			return
		}
		h.log.Error("validate media", "err", err)
		httpapi.Error(w, http.StatusInternalServerError, "internal error")
		return
	}

	p, err := h.store.Create(r.Context(), uid, req.Body, req.Media)
	if err != nil {
		h.log.Error("create post", "err", err)
		httpapi.Error(w, http.StatusInternalServerError, "internal error")
		return
	}
	httpapi.JSON(w, http.StatusCreated, h.withMediaURLs(p))
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	pid, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || pid <= 0 {
		httpapi.Error(w, http.StatusBadRequest, "invalid post id")
		return
	}
	uid, _ := auth.UserID(r.Context())
	p, err := h.store.GetWithViewer(r.Context(), pid, uid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpapi.Error(w, http.StatusNotFound, "post not found")
			return
		}
		h.log.Error("get post", "err", err)
		httpapi.Error(w, http.StatusInternalServerError, "internal error")
		return
	}
	if p.Deleted {
		httpapi.Error(w, http.StatusNotFound, "post not found")
		return
	}
	httpapi.JSON(w, http.StatusOK, h.withMediaURLs(p))
}

func (h *Handler) Like(w http.ResponseWriter, r *http.Request) {
	uid, ok := auth.UserID(r.Context())
	if !ok {
		httpapi.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	pid, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || pid <= 0 {
		httpapi.Error(w, http.StatusBadRequest, "invalid post id")
		return
	}
	count, err := h.store.Like(r.Context(), uid, pid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpapi.Error(w, http.StatusNotFound, "post not found")
			return
		}
		h.log.Error("like", "err", err)
		httpapi.Error(w, http.StatusInternalServerError, "internal error")
		return
	}
	httpapi.JSON(w, http.StatusOK, map[string]any{"post_id": pid, "like_count": count, "liked": true})
}

func (h *Handler) Unlike(w http.ResponseWriter, r *http.Request) {
	uid, ok := auth.UserID(r.Context())
	if !ok {
		httpapi.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	pid, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || pid <= 0 {
		httpapi.Error(w, http.StatusBadRequest, "invalid post id")
		return
	}
	count, err := h.store.Unlike(r.Context(), uid, pid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpapi.Error(w, http.StatusNotFound, "post not found")
			return
		}
		h.log.Error("unlike", "err", err)
		httpapi.Error(w, http.StatusInternalServerError, "internal error")
		return
	}
	httpapi.JSON(w, http.StatusOK, map[string]any{"post_id": pid, "like_count": count, "liked": false})
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	uid, ok := auth.UserID(r.Context())
	if !ok {
		httpapi.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	pid, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || pid <= 0 {
		httpapi.Error(w, http.StatusBadRequest, "invalid post id")
		return
	}
	if err := h.store.Delete(r.Context(), pid, uid); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpapi.Error(w, http.StatusNotFound, "post not found")
			return
		}
		h.log.Error("delete post", "err", err)
		httpapi.Error(w, http.StatusInternalServerError, "internal error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) GetMany(w http.ResponseWriter, r *http.Request) {
	raw := r.URL.Query().Get("ids")
	viewerID, _ := strconv.ParseInt(r.URL.Query().Get("viewer_id"), 10, 64)
	if viewerID < 0 {
		viewerID = 0
	}
	if raw == "" {
		httpapi.JSON(w, http.StatusOK, map[string]any{"items": []model.Post{}})
		return
	}

	parts := strings.Split(raw, ",")
	if len(parts) > 200 {
		httpapi.Error(w, http.StatusBadRequest, "too many ids")
		return
	}
	ids := make([]int64, 0, len(parts))
	for _, part := range parts {
		pid, err := strconv.ParseInt(strings.TrimSpace(part), 10, 64)
		if err != nil || pid <= 0 {
			httpapi.Error(w, http.StatusBadRequest, "invalid id")
			return
		}
		ids = append(ids, pid)
	}

	posts, err := h.store.GetMany(r.Context(), ids, viewerID)
	if err != nil {
		h.log.Error("get many posts", "err", err)
		httpapi.Error(w, http.StatusInternalServerError, "internal error")
		return
	}
	items := make([]model.Post, len(posts))
	for i, p := range posts {
		items[i] = h.withMediaURLs(p)
	}
	httpapi.JSON(w, http.StatusOK, map[string]any{"items": items})
}
