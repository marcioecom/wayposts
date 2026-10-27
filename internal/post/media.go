package post

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/marciojunior/wayposts/internal/auth"
	"github.com/marciojunior/wayposts/internal/media"
	"github.com/marciojunior/wayposts/internal/platform/httpapi"
)

type mediaValidationError struct {
	reason string
}

func (e *mediaValidationError) Error() string { return e.reason }

func invalidMedia(format string, args ...any) error {
	return &mediaValidationError{reason: fmt.Sprintf(format, args...)}
}

func (h *Handler) Presign(w http.ResponseWriter, r *http.Request) {
	uid, ok := auth.UserID(r.Context())
	if !ok {
		httpapi.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if h.media.Store == nil {
		httpapi.Error(w, http.StatusServiceUnavailable, "media upload is disabled")
		return
	}

	var req struct {
		Items []struct {
			ContentType string `json:"content_type"`
			Size        int64  `json:"size"`
		} `json:"items"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpapi.Error(w, http.StatusBadRequest, "invalid json")
		return
	}
	if len(req.Items) == 0 {
		httpapi.Error(w, http.StatusBadRequest, "items is required")
		return
	}
	if len(req.Items) > maxMediaCount {
		h.store.m.MediaPresign.WithLabelValues("rejected").Inc()
		httpapi.Error(w, http.StatusBadRequest, "too many media items")
		return
	}

	type item struct {
		Key       string            `json:"key"`
		URL       string            `json:"url"`
		Method    string            `json:"method"`
		Headers   map[string]string `json:"headers"`
		ExpiresIn int64             `json:"expires_in"`
	}
	out := make([]item, 0, len(req.Items))
	for _, it := range req.Items {
		contentType := strings.ToLower(strings.TrimSpace(it.ContentType))
		if _, ok := media.AllowedContentType(contentType); !ok {
			h.store.m.MediaPresign.WithLabelValues("rejected").Inc()
			httpapi.Error(w, http.StatusBadRequest, "unsupported content_type: "+contentType)
			return
		}
		if it.Size <= 0 || it.Size > h.media.MaxBytes {
			h.store.m.MediaPresign.WithLabelValues("rejected").Inc()
			httpapi.Error(w, http.StatusBadRequest,
				fmt.Sprintf("size must be between 1 and %d bytes", h.media.MaxBytes))
			return
		}

		key, err := media.Key(uid, h.store.gen.Next(), contentType)
		if err != nil {
			h.store.m.MediaPresign.WithLabelValues("rejected").Inc()
			httpapi.Error(w, http.StatusBadRequest, err.Error())
			return
		}
		url, err := h.media.Store.PresignPut(r.Context(), key, contentType, h.media.PresignTTL)
		if err != nil {
			h.log.Error("presign", "err", err)
			h.store.m.MediaPresign.WithLabelValues("rejected").Inc()
			httpapi.Error(w, http.StatusBadGateway, "storage unavailable")
			return
		}
		h.store.m.MediaPresign.WithLabelValues("ok").Inc()
		out = append(out, item{
			Key:       key,
			URL:       url,
			Method:    http.MethodPut,
			Headers:   map[string]string{"Content-Type": contentType},
			ExpiresIn: int64(h.media.PresignTTL.Seconds()),
		})
	}

	httpapi.JSON(w, http.StatusOK, map[string]any{"items": out})
}

func (h *Handler) validateMedia(ctx context.Context, authorID int64, keys []string) error {
	if len(keys) == 0 {
		return nil
	}
	if h.media.Store == nil {
		h.store.m.MediaValidation.WithLabelValues("disabled").Inc()
		return invalidMedia("media upload is disabled")
	}

	seen := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		key = strings.TrimSpace(key)
		if key == "" {
			return invalidMedia("media key is empty")
		}
		if _, dup := seen[key]; dup {
			return invalidMedia("duplicated media key: %s", key)
		}
		seen[key] = struct{}{}

		if !media.OwnedBy(key, authorID) {
			h.store.m.MediaValidation.WithLabelValues("not_owned").Inc()
			return invalidMedia("media %s does not belong to you", key)
		}

		info, err := h.media.Store.Head(ctx, key)
		if errors.Is(err, media.ErrNotFound) {
			h.store.m.MediaValidation.WithLabelValues("missing").Inc()
			return invalidMedia("media %s was not uploaded", key)
		}
		if err != nil {
			return fmt.Errorf("post: media head %s: %w", key, err)
		}
		if !strings.HasPrefix(strings.ToLower(info.ContentType), "image/") {
			h.store.m.MediaValidation.WithLabelValues("wrong_type").Inc()
			return invalidMedia("media %s is not an image", key)
		}
		if info.Size <= 0 || info.Size > h.media.MaxBytes {
			h.store.m.MediaValidation.WithLabelValues("too_large").Inc()
			return invalidMedia("media %s is larger than %d bytes", key, h.media.MaxBytes)
		}
	}

	h.store.m.MediaValidation.WithLabelValues("ok").Inc()
	return nil
}
