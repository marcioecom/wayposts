package auth

import (
	"context"
	"net/http"
	"strconv"
	"strings"
)

type ctxKey struct{}

// Dev é a autenticação de desenvolvimento do laboratório: o token é o ID do
// usuário semeado. A interface é a mesma de um JWT para permitir a troca sem
// tocar nos handlers.
func Dev(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		const prefix = "Bearer "
		raw := r.Header.Get("Authorization")
		if !strings.HasPrefix(raw, prefix) {
			unauthorized(w, "missing bearer token")
			return
		}
		uid, err := strconv.ParseInt(strings.TrimSpace(strings.TrimPrefix(raw, prefix)), 10, 64)
		if err != nil || uid <= 0 {
			unauthorized(w, "invalid bearer token")
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, uid)))
	})
}

func unauthorized(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusUnauthorized)
	_, _ = w.Write([]byte(`{"error":"` + msg + `"}`))
}

func UserID(ctx context.Context) (int64, bool) {
	uid, ok := ctx.Value(ctxKey{}).(int64)
	return uid, ok
}
