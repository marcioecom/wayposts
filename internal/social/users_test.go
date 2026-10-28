package social

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"

	"github.com/marciojunior/wayposts/internal/model"
	"github.com/marciojunior/wayposts/internal/platform/metrics"
	"github.com/marciojunior/wayposts/internal/platform/pg"
)

func TestUsersHandlerRejectsInvalidIDs(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := NewHandler(nil, log)

	for _, raw := range []string{"0", "-1", "abc", "1,abc"} {
		req := httptest.NewRequest(http.MethodGet, "/internal/users?ids="+raw, nil)
		rec := httptest.NewRecorder()
		handler.Users(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("ids=%q: status = %d, want %d", raw, rec.Code, http.StatusBadRequest)
		}
	}
}

func TestUsersHandlerWithoutIDsReturnsEmptyList(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := NewHandler(nil, log)

	req := httptest.NewRequest(http.MethodGet, "/internal/users", nil)
	rec := httptest.NewRecorder()
	handler.Users(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var out struct {
		Items []model.AuthorSummary `json:"items"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out.Items == nil || len(out.Items) != 0 {
		t.Fatalf("items = %#v, want lista vazia", out.Items)
	}
}

// TestUsersBatchCacheAndFallback roda só com TEST_DATABASE_URL apontando para
// um banco com as migrations aplicadas.
func TestUsersBatchCacheAndFallback(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL não definido")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := pg.New(ctx, dsn, 4)
	if err != nil {
		t.Fatalf("postgres: %v", err)
	}
	defer pool.Close()

	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	defer mr.Close()
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	store := NewStore(pool, rdb, 100, metrics.New("test"), log)

	const (
		author1 = int64(9_100_001)
		author2 = int64(9_100_002)
		author3 = int64(9_100_003)
		missing = int64(9_100_099)
	)
	users := []int64{author1, author2, author3}
	cleanup := func() {
		_, _ = pool.Exec(ctx, `DELETE FROM social.users WHERE id = ANY($1)`, users)
	}
	cleanup()
	defer cleanup()

	for _, id := range users {
		if _, err := pool.Exec(ctx, `
			INSERT INTO social.users (id, handle, display_name)
			VALUES ($1, $2, $3)
		`, id, "u"+strconv.FormatInt(id, 10), "User "+strconv.FormatInt(id, 10)); err != nil {
			t.Fatalf("insert user %d: %v", id, err)
		}
	}

	got, err := store.GetUsers(ctx, []int64{author1, author2, author1, author3, missing})
	if err != nil {
		t.Fatalf("get users: %v", err)
	}
	want := []model.AuthorSummary{
		{ID: author1, Handle: "u9100001", DisplayName: "User 9100001"},
		{ID: author2, Handle: "u9100002", DisplayName: "User 9100002"},
		{ID: author3, Handle: "u9100003", DisplayName: "User 9100003"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("resumos: got %v, want %v", got, want)
	}

	// O primeiro item veio do banco e já deve estar no hash user:{id}.
	if raw, err := rdb.HGetAll(ctx, userKey(author1)).Result(); err != nil {
		t.Fatalf("cache: %v", err)
	} else if raw["handle"] != "u9100001" {
		t.Fatalf("cache do autor = %#v", raw)
	}

	// Autor ausente fica marcado por pouco tempo, para não consultar o banco
	// em toda página.
	if raw, err := rdb.HGetAll(ctx, userKey(missing)).Result(); err != nil {
		t.Fatalf("cache do ausente: %v", err)
	} else if raw[userMissingField] != "1" {
		t.Fatalf("sentinela do ausente = %#v", raw)
	}

	// Enquanto o sentinela vale, um usuário criado depois não aparece.
	if _, err := pool.Exec(ctx, `
		INSERT INTO social.users (id, handle, display_name)
		VALUES ($1, $2, $3)
	`, missing, "late"+strconv.FormatInt(missing, 10), "Late"); err != nil {
		t.Fatalf("insert late: %v", err)
	}
	users = append(users, missing)
	got, err = store.GetUsers(ctx, []int64{missing})
	if err != nil {
		t.Fatalf("sentinela: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("sentinela não evitou a leitura: %v", got)
	}

	// Alterar direto no banco não muda a leitura enquanto o cache vale.
	if _, err := pool.Exec(ctx, `UPDATE social.users SET handle = $2 WHERE id = $1`, author1, "changed"); err != nil {
		t.Fatalf("update: %v", err)
	}
	got, err = store.GetUsers(ctx, []int64{author1})
	if err != nil {
		t.Fatalf("cache hit: %v", err)
	}
	if len(got) != 1 || got[0].Handle != "u9100001" {
		t.Fatalf("cache hit: got %v", got)
	}

	// Sem a chave, a leitura volta ao Postgres e reaquece o cache.
	if err := rdb.Del(ctx, userKey(author1)).Err(); err != nil {
		t.Fatalf("del cache: %v", err)
	}
	got, err = store.GetUsers(ctx, []int64{author1})
	if err != nil {
		t.Fatalf("cache miss: %v", err)
	}
	if len(got) != 1 || got[0].Handle != "changed" {
		t.Fatalf("cache miss: got %v", got)
	}

	// O handler devolve o mesmo contrato consumido pela timeline.
	handler := NewHandler(store, log)
	r := chi.NewRouter()
	r.Get("/internal/users", handler.Users)
	req := httptest.NewRequest(http.MethodGet,
		"/internal/users?ids="+strconv.FormatInt(author1, 10)+","+strconv.FormatInt(author2, 10), nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("handler status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var out struct {
		Items []model.AuthorSummary `json:"items"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatalf("decode handler: %v", err)
	}
	if len(out.Items) != 2 {
		t.Fatalf("handler items = %v", out.Items)
	}

	// Redis fora do ar não pode derrubar a leitura: cai no Postgres.
	if err := rdb.Close(); err != nil {
		t.Fatalf("close redis: %v", err)
	}
	got, err = store.GetUsers(ctx, []int64{author1, author2})
	if err != nil {
		t.Fatalf("fallback sem redis: %v", err)
	}
	if len(got) != 2 || got[0].ID != author1 || got[1].ID != author2 {
		t.Fatalf("fallback sem redis: got %v", got)
	}
}
