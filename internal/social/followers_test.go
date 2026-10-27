package social

import (
	"context"
	"io"
	"log/slog"
	"os"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"github.com/marciojunior/wayposts/internal/platform/metrics"
	"github.com/marciojunior/wayposts/internal/platform/pg"
)

// TestFollowerCacheAgainstPostgres roda só com TEST_DATABASE_URL apontando para
// um banco com as migrations aplicadas. Usa IDs altos e limpa o que criou.
func TestFollowerCacheAgainstPostgres(t *testing.T) {
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
	defer rdb.Close()

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	store := NewStore(pool, rdb, 100, metrics.New("test"), log)

	const (
		author    = int64(9_000_001)
		follower1 = int64(9_000_010)
		follower2 = int64(9_000_011)
		follower3 = int64(9_000_012)
		follower4 = int64(9_000_013)
	)
	users := []int64{author, follower1, follower2, follower3, follower4}
	cleanup := func() {
		_, _ = pool.Exec(ctx, `DELETE FROM social.follows WHERE followee_id = $1 OR follower_id = $1`, author)
		_, _ = pool.Exec(ctx, `DELETE FROM social.users WHERE id = ANY($1)`, users)
	}
	cleanup()
	defer cleanup()

	for _, id := range users {
		if _, err := pool.Exec(ctx, `
			INSERT INTO social.users (id, handle, display_name)
			VALUES ($1, $2, $3)
		`, id, "it"+strconv.FormatInt(id, 10), "it"+strconv.FormatInt(id, 10)); err != nil {
			t.Fatalf("insert user %d: %v", id, err)
		}
	}

	for _, fid := range []int64{follower3, follower1, follower2} {
		if err := store.Follow(ctx, fid, author); err != nil {
			t.Fatalf("follow %d: %v", fid, err)
		}
	}

	got, err := store.FollowersAfter(ctx, author, 0, 100)
	if err != nil {
		t.Fatalf("followers: %v", err)
	}
	want := []int64{follower1, follower2, follower3}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("primeira leitura: got %v, want %v", got, want)
	}

	if n, err := rdb.Exists(ctx, followersMetaKey(author)).Result(); err != nil || n != 1 {
		t.Fatalf("meta no cache: n=%d err=%v", n, err)
	}

	if err := store.Follow(ctx, follower4, author); err != nil {
		t.Fatalf("follow novo: %v", err)
	}
	got, err = store.FollowersAfter(ctx, author, 0, 100)
	if err != nil {
		t.Fatalf("followers após follow: %v", err)
	}
	want = []int64{follower1, follower2, follower3, follower4}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("após follow: got %v, want %v", got, want)
	}

	if err := store.Unfollow(ctx, follower2, author); err != nil {
		t.Fatalf("unfollow: %v", err)
	}
	got, err = store.FollowersAfter(ctx, author, 0, 100)
	if err != nil {
		t.Fatalf("followers após unfollow: %v", err)
	}
	want = []int64{follower1, follower3, follower4}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("após unfollow: got %v, want %v", got, want)
	}

	got, err = store.FollowersAfter(ctx, author, follower1, 2)
	if err != nil {
		t.Fatalf("paginado: %v", err)
	}
	want = []int64{follower3, follower4}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("paginado: got %v, want %v", got, want)
	}

	empty, err := store.FollowersAfter(ctx, follower1, 0, 100)
	if err != nil {
		t.Fatalf("sem seguidores: %v", err)
	}
	if len(empty) != 0 {
		t.Fatalf("esperava lista vazia, veio %v", empty)
	}
	if raw, err := rdb.Get(ctx, followersMetaKey(follower1)).Result(); err != nil || raw != "0" {
		t.Fatalf("meta de lista vazia: %q err=%v", raw, err)
	}
}
