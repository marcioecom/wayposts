package timeline

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"github.com/marciojunior/wayposts/internal/model"
)

type stubPosts struct {
	byID  map[int64]model.Post
	err   error
	calls int
}

func (s *stubPosts) GetMany(_ context.Context, ids []int64, _ int64) ([]model.Post, error) {
	s.calls++
	if s.err != nil {
		return nil, s.err
	}
	out := make([]model.Post, 0, len(ids))
	for _, id := range ids {
		if p, ok := s.byID[id]; ok {
			out = append(out, p)
		}
	}
	return out, nil
}

type stubUsers struct {
	byID    map[int64]model.AuthorSummary
	err     error
	calls   int
	lastIDs []int64
}

func (s *stubUsers) Users(_ context.Context, ids []int64) ([]model.AuthorSummary, error) {
	s.calls++
	s.lastIDs = append([]int64(nil), ids...)
	if s.err != nil {
		return nil, s.err
	}
	out := make([]model.AuthorSummary, 0, len(ids))
	for _, id := range ids {
		if u, ok := s.byID[id]; ok {
			out = append(out, u)
		}
	}
	return out, nil
}

// slowUsers respeita o prazo do contexto, como o cliente HTTP real.
type slowUsers struct{}

func (s *slowUsers) Users(ctx context.Context, _ []int64) ([]model.AuthorSummary, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}

func testRedis(t *testing.T) *redis.Client {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	t.Cleanup(mr.Close)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	return rdb
}

func testLog() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestHomeHydratesAuthorsInOneBatch(t *testing.T) {
	ctx := context.Background()
	rdb := testRedis(t)

	posts := &stubPosts{byID: map[int64]model.Post{
		101: {ID: 101, AuthorID: 1, Body: "a"},
		102: {ID: 102, AuthorID: 2, Body: "b"},
		103: {ID: 103, AuthorID: 1, Body: "c"},
	}}
	users := &stubUsers{byID: map[int64]model.AuthorSummary{
		1: {ID: 1, Handle: "ana", DisplayName: "Ana"},
		2: {ID: 2, Handle: "bia", DisplayName: "Bia"},
	}}
	svc := NewService(rdb, posts, users, 100, testLog())

	if err := rdb.ZAdd(ctx, feedKey(10),
		redis.Z{Score: 101, Member: 101},
		redis.Z{Score: 102, Member: 102},
		redis.Z{Score: 103, Member: 103},
	).Err(); err != nil {
		t.Fatalf("seed feed: %v", err)
	}

	page, err := svc.Home(ctx, 10, 0, 20)
	if err != nil {
		t.Fatalf("home: %v", err)
	}
	if len(page.Items) != 3 {
		t.Fatalf("items = %d, want 3", len(page.Items))
	}
	if users.calls != 1 {
		t.Fatalf("buscas de autor = %d, want 1", users.calls)
	}
	if len(users.lastIDs) != 2 {
		t.Fatalf("ids hidratados = %v, want 2 únicos", users.lastIDs)
	}

	for _, item := range page.Items {
		if item.AuthorID == 0 {
			t.Fatalf("author_id preservado: %#v", item)
		}
		want, ok := users.byID[item.AuthorID]
		if !ok {
			t.Fatalf("autor inesperado: %#v", item)
		}
		if item.Author == nil || *item.Author != want {
			t.Fatalf("author = %#v, want %#v", item.Author, want)
		}
	}
}

func TestProfileHydratesAuthors(t *testing.T) {
	ctx := context.Background()
	rdb := testRedis(t)

	posts := &stubPosts{byID: map[int64]model.Post{
		201: {ID: 201, AuthorID: 7, Body: "a"},
		202: {ID: 202, AuthorID: 7, Body: "b"},
	}}
	users := &stubUsers{byID: map[int64]model.AuthorSummary{
		7: {ID: 7, Handle: "caio", DisplayName: "Caio"},
	}}
	svc := NewService(rdb, posts, users, 100, testLog())

	if err := rdb.ZAdd(ctx, authorKey(7),
		redis.Z{Score: 201, Member: 201},
		redis.Z{Score: 202, Member: 202},
	).Err(); err != nil {
		t.Fatalf("seed author: %v", err)
	}

	page, err := svc.Profile(ctx, 10, 7, 0, 20)
	if err != nil {
		t.Fatalf("profile: %v", err)
	}
	if len(page.Items) != 2 {
		t.Fatalf("items = %d, want 2", len(page.Items))
	}
	if users.calls != 1 || len(users.lastIDs) != 1 || users.lastIDs[0] != 7 {
		t.Fatalf("busca de autor = calls %d ids %v", users.calls, users.lastIDs)
	}
	for _, item := range page.Items {
		if item.Author == nil || item.Author.Handle != "caio" {
			t.Fatalf("author = %#v", item.Author)
		}
	}
}

func TestHomeSurvivesAuthorLookupFailure(t *testing.T) {
	ctx := context.Background()
	rdb := testRedis(t)

	posts := &stubPosts{byID: map[int64]model.Post{
		301: {ID: 301, AuthorID: 1, Body: "a"},
	}}
	users := &stubUsers{err: errors.New("social indisponível")}
	svc := NewService(rdb, posts, users, 100, testLog())

	if err := rdb.ZAdd(ctx, feedKey(10), redis.Z{Score: 301, Member: 301}).Err(); err != nil {
		t.Fatalf("seed feed: %v", err)
	}

	page, err := svc.Home(ctx, 10, 0, 20)
	if err != nil {
		t.Fatalf("home não pode falhar por causa do autor: %v", err)
	}
	if len(page.Items) != 1 {
		t.Fatalf("items = %d, want 1", len(page.Items))
	}
	if page.Items[0].Author != nil {
		t.Fatalf("author = %#v, want ausente", page.Items[0].Author)
	}
	if page.Items[0].AuthorID != 1 {
		t.Fatalf("author_id = %d", page.Items[0].AuthorID)
	}
}

func TestHomeStopsWaitingForAuthorsAtTheDeadline(t *testing.T) {
	ctx := context.Background()
	rdb := testRedis(t)

	posts := &stubPosts{byID: map[int64]model.Post{
		501: {ID: 501, AuthorID: 1, Body: "a"},
	}}
	svc := NewService(rdb, posts, &slowUsers{}, 100, testLog())

	if err := rdb.ZAdd(ctx, feedKey(10), redis.Z{Score: 501, Member: 501}).Err(); err != nil {
		t.Fatalf("seed feed: %v", err)
	}

	start := time.Now()
	page, err := svc.Home(ctx, 10, 0, 20)
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("home não pode falhar no prazo: %v", err)
	}
	if len(page.Items) != 1 {
		t.Fatalf("items = %d, want 1", len(page.Items))
	}
	if page.Items[0].Author != nil {
		t.Fatalf("author = %#v, want ausente", page.Items[0].Author)
	}
	if elapsed > 3*authorLookupTimeout {
		t.Fatalf("home esperou %v, orçamento %v", elapsed, authorLookupTimeout)
	}
}

func TestHomeKeepsPostsWithPartialAuthorResult(t *testing.T) {
	ctx := context.Background()
	rdb := testRedis(t)

	posts := &stubPosts{byID: map[int64]model.Post{
		401: {ID: 401, AuthorID: 1, Body: "a"},
		402: {ID: 402, AuthorID: 2, Body: "b"},
	}}
	users := &stubUsers{byID: map[int64]model.AuthorSummary{
		1: {ID: 1, Handle: "ana", DisplayName: "Ana"},
	}}
	svc := NewService(rdb, posts, users, 100, testLog())

	if err := rdb.ZAdd(ctx, feedKey(10),
		redis.Z{Score: 401, Member: 401},
		redis.Z{Score: 402, Member: 402},
	).Err(); err != nil {
		t.Fatalf("seed feed: %v", err)
	}

	page, err := svc.Home(ctx, 10, 0, 20)
	if err != nil {
		t.Fatalf("home: %v", err)
	}
	if len(page.Items) != 2 {
		t.Fatalf("items = %d, want 2", len(page.Items))
	}
	byID := make(map[int64]model.Post, len(page.Items))
	for _, item := range page.Items {
		byID[item.ID] = item
	}
	if byID[401].Author == nil || byID[401].Author.Handle != "ana" {
		t.Fatalf("401 author = %#v", byID[401].Author)
	}
	if byID[402].Author != nil {
		t.Fatalf("402 author = %#v, want ausente", byID[402].Author)
	}
}

func TestPageJSONKeepsAuthorIDAndEmbedsAuthor(t *testing.T) {
	page := Page{Items: []model.Post{{
		ID:       1,
		AuthorID: 2,
		Body:     "oi",
		Author:   &model.AuthorSummary{ID: 2, Handle: "bia", DisplayName: "Bia"},
	}}}

	raw, err := json.Marshal(page)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var out struct {
		Items []map[string]any `json:"items"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	item := out.Items[0]
	if _, ok := item["author_id"]; !ok {
		t.Fatalf("author_id ausente: %s", raw)
	}
	author, ok := item["author"].(map[string]any)
	if !ok {
		t.Fatalf("author ausente ou com formato errado: %s", raw)
	}
	for _, field := range []string{"id", "handle", "display_name"} {
		if _, ok := author[field]; !ok {
			t.Fatalf("author.%s ausente: %s", field, raw)
		}
	}
}
