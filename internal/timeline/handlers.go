package timeline

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/marciojunior/wayposts/internal/auth"
	"github.com/marciojunior/wayposts/internal/platform/httpapi"
)

type Handler struct {
	service *Service
	log     *slog.Logger
}

func NewHandler(service *Service, log *slog.Logger) *Handler {
	return &Handler{service: service, log: log}
}

func (h *Handler) Home(w http.ResponseWriter, r *http.Request) {
	uid, ok := auth.UserID(r.Context())
	if !ok {
		httpapi.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	before, limit := params(r)

	page, err := h.service.Home(r.Context(), uid, before, limit)
	if err != nil {
		h.log.Error("home timeline", "err", err)
		httpapi.Error(w, http.StatusInternalServerError, "internal error")
		return
	}
	httpapi.JSON(w, http.StatusOK, page)
}

func (h *Handler) Profile(w http.ResponseWriter, r *http.Request) {
	viewer, ok := auth.UserID(r.Context())
	if !ok {
		httpapi.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		httpapi.Error(w, http.StatusBadRequest, "invalid user id")
		return
	}
	before, limit := params(r)

	page, err := h.service.Profile(r.Context(), viewer, id, before, limit)
	if err != nil {
		h.log.Error("profile timeline", "err", err)
		httpapi.Error(w, http.StatusInternalServerError, "internal error")
		return
	}
	httpapi.JSON(w, http.StatusOK, page)
}

func params(r *http.Request) (before int64, limit int) {
	before, _ = strconv.ParseInt(r.URL.Query().Get("before"), 10, 64)
	limit, _ = strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}
	return before, limit
}
