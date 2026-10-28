package social

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/marciojunior/wayposts/internal/auth"
	"github.com/marciojunior/wayposts/internal/model"
	"github.com/marciojunior/wayposts/internal/platform/httpapi"
)

type Handler struct {
	store *Store
	log   *slog.Logger
}

func NewHandler(store *Store, log *slog.Logger) *Handler {
	return &Handler{store: store, log: log}
}

func (h *Handler) Follow(w http.ResponseWriter, r *http.Request) {
	uid, ok := auth.UserID(r.Context())
	if !ok {
		httpapi.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req struct {
		FolloweeID int64 `json:"followee_id"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpapi.Error(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.FolloweeID <= 0 {
		httpapi.Error(w, http.StatusBadRequest, "followee_id is required")
		return
	}

	if err := h.store.Follow(r.Context(), uid, req.FolloweeID); err != nil {
		switch {
		case errors.Is(err, ErrInvalidFollow):
			httpapi.Error(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, pgx.ErrNoRows):
			httpapi.Error(w, http.StatusNotFound, "followee not found")
		default:
			h.log.Error("follow", "err", err)
			httpapi.Error(w, http.StatusInternalServerError, "internal error")
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Unfollow(w http.ResponseWriter, r *http.Request) {
	uid, ok := auth.UserID(r.Context())
	if !ok {
		httpapi.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	followeeID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || followeeID <= 0 {
		httpapi.Error(w, http.StatusBadRequest, "invalid followee id")
		return
	}
	if err := h.store.Unfollow(r.Context(), uid, followeeID); err != nil {
		h.log.Error("unfollow", "err", err)
		httpapi.Error(w, http.StatusInternalServerError, "internal error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) GetUser(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		httpapi.Error(w, http.StatusBadRequest, "invalid user id")
		return
	}
	u, err := h.store.GetUser(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpapi.Error(w, http.StatusNotFound, "user not found")
			return
		}
		h.log.Error("get user", "err", err)
		httpapi.Error(w, http.StatusInternalServerError, "internal error")
		return
	}
	httpapi.JSON(w, http.StatusOK, u)
}

func (h *Handler) Followers(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		httpapi.Error(w, http.StatusBadRequest, "invalid author id")
		return
	}
	after, _ := strconv.ParseInt(r.URL.Query().Get("after"), 10, 64)
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	followers, err := h.store.FollowersAfter(r.Context(), id, after, limit)
	if err != nil {
		h.log.Error("followers", "err", err)
		httpapi.Error(w, http.StatusInternalServerError, "internal error")
		return
	}

	next := after
	if len(followers) > 0 {
		next = followers[len(followers)-1]
	}
	httpapi.JSON(w, http.StatusOK, map[string]any{
		"followers":  followers,
		"next_after": next,
	})
}

// Users resolve o resumo de vários autores em uma chamada. A timeline usa para
// hidratar a página.
func (h *Handler) Users(w http.ResponseWriter, r *http.Request) {
	raw := r.URL.Query().Get("ids")
	if raw == "" {
		httpapi.JSON(w, http.StatusOK, map[string]any{"items": []model.AuthorSummary{}})
		return
	}

	parts := strings.Split(raw, ",")
	if len(parts) > 200 {
		httpapi.Error(w, http.StatusBadRequest, "too many ids")
		return
	}
	ids := make([]int64, 0, len(parts))
	for _, part := range parts {
		id, err := strconv.ParseInt(strings.TrimSpace(part), 10, 64)
		if err != nil || id <= 0 {
			httpapi.Error(w, http.StatusBadRequest, "invalid id")
			return
		}
		ids = append(ids, id)
	}

	users, err := h.store.GetUsers(r.Context(), ids)
	if err != nil {
		// Com erro parcial, devolve o que o cache resolveu. Sem nada, responde 500.
		h.log.Error("get users", "err", err)
		if len(users) == 0 {
			httpapi.Error(w, http.StatusInternalServerError, "internal error")
			return
		}
	}
	httpapi.JSON(w, http.StatusOK, map[string]any{"items": users})
}

func (h *Handler) FollowerCount(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		httpapi.Error(w, http.StatusBadRequest, "invalid author id")
		return
	}
	count, err := h.store.FollowerCount(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpapi.Error(w, http.StatusNotFound, "author not found")
			return
		}
		h.log.Error("follower count", "err", err)
		httpapi.Error(w, http.StatusInternalServerError, "internal error")
		return
	}
	httpapi.JSON(w, http.StatusOK, map[string]int{"count": count})
}
