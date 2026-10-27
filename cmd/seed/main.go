package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"math"
	"math/rand"
	"os"
	"sort"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/marciojunior/wayposts/internal/platform/env"
	"github.com/marciojunior/wayposts/internal/platform/rds"
)

func main() {
	users := flag.Int("users", 2000, "número de usuários")
	follows := flag.Int("follows", 100, "quantos autores cada usuário segue (média)")
	celebs := flag.Int("celebs", 3, "quantos usuários são celebridades")
	celebFollowers := flag.Int("celeb-followers", 600, "seguidores garantidos por celebridade")
	maxFollowers := flag.Int("max-followers", 400, "teto de seguidores para autores comuns; precisa ficar abaixo do threshold")
	reset := flag.Bool("reset", true, "truncar dados e limpar o Redis antes de semear")
	seed := flag.Int64("seed", 42, "semente do gerador aleatório")
	flag.Parse()

	if *users <= 0 || *celebs < 0 || *celebs >= *users {
		fmt.Fprintln(os.Stderr, "parâmetros inválidos: users > celebs >= 0")
		os.Exit(1)
	}
	if *follows >= *users-*celebs {
		fmt.Fprintln(os.Stderr, "follows precisa ser menor que users-celebs")
		os.Exit(1)
	}
	if *celebFollowers > *users-*celebs {
		fmt.Fprintln(os.Stderr, "celeb-followers precisa ser menor que users-celebs")
		os.Exit(1)
	}
	if *maxFollowers <= 0 || *maxFollowers >= *celebFollowers {
		fmt.Fprintln(os.Stderr, "max-followers precisa ser positivo e menor que celeb-followers")
		os.Exit(1)
	}

	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	dsn := env.Get("DATABASE_URL", "postgres://wayposts:wayposts@localhost:5432/wayposts?sslmode=disable")
	redisAddr := env.Get("REDIS_ADDR", "localhost:6379")

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Error("postgres", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	rdb, err := rds.New(redisAddr, "", 20)
	if err != nil {
		log.Error("redis", "err", err)
		os.Exit(1)
	}
	defer rdb.Close()

	if *reset {
		if _, err := pool.Exec(ctx, `
			TRUNCATE post.outbox, post.likes, post.posts, social.follows, social.users
			RESTART IDENTITY CASCADE
		`); err != nil {
			log.Error("truncate", "err", err)
			os.Exit(1)
		}
		if err := rdb.FlushDB(ctx).Err(); err != nil {
			log.Error("flush redis", "err", err)
			os.Exit(1)
		}
		log.Info("dados anteriores removidos")
	}

	if err := insertUsers(ctx, pool, *users); err != nil {
		log.Error("seed users", "err", err)
		os.Exit(1)
	}
	log.Info("usuários inseridos", "count", *users)

	rng := rand.New(rand.NewSource(*seed))
	edges, err := buildEdges(rng, *users, *celebs, *follows, *celebFollowers, *maxFollowers)
	if err != nil {
		log.Error("build edges", "err", err)
		os.Exit(1)
	}
	if err := insertFollows(ctx, pool, edges); err != nil {
		log.Error("seed follows", "err", err)
		os.Exit(1)
	}
	log.Info("follows inseridos", "edges", len(edges))

	if _, err := pool.Exec(ctx, `
		UPDATE social.users u
		   SET follower_count = c.n
		  FROM (
			SELECT followee_id, count(*)::int AS n
			  FROM social.follows
			 GROUP BY followee_id
		  ) c
		 WHERE u.id = c.followee_id
	`); err != nil {
		log.Error("update follower counts", "err", err)
		os.Exit(1)
	}

	if err := seedCelebFollows(ctx, pool, rdb, *celebs); err != nil {
		log.Error("seed celeb_follows", "err", err)
		os.Exit(1)
	}

	printSummary(ctx, pool, *users, *celebs, len(edges))
}

func insertUsers(ctx context.Context, pool *pgxpool.Pool, n int) error {
	const batchSize = 5000
	for start := 1; start <= n; start += batchSize {
		end := min(start+batchSize-1, n)
		rows := make([][]any, 0, end-start+1)
		for i := start; i <= end; i++ {
			rows = append(rows, []any{
				int64(i),
				fmt.Sprintf("u%d", i),
				fmt.Sprintf("User %d", i),
				0,
			})
		}
		if _, err := pool.CopyFrom(ctx,
			pgx.Identifier{"social", "users"},
			[]string{"id", "handle", "display_name", "follower_count"},
			pgx.CopyFromRows(rows),
		); err != nil {
			return err
		}
	}
	return nil
}

type edge struct {
	follower int64
	followee int64
}

func buildEdges(rng *rand.Rand, users, celebs, follows, celebFollowers, maxFollowers int) ([]edge, error) {
	celebIDs := make(map[int64]struct{}, celebs)
	for i := 1; i <= celebs; i++ {
		celebIDs[int64(i)] = struct{}{}
	}

	// Grafo normal: power-law entre não-celebridades, limitado por maxFollowers.
	// O teto mantém autores comuns abaixo do threshold de fanout e permite que a
	// celebridade seja identificada só pela contagem de seguidores.
	normal := make([]int64, 0, users-celebs)
	weights := make([]float64, 0, users-celebs)
	for i := celebs + 1; i <= users; i++ {
		normal = append(normal, int64(i))
		weights = append(weights, 1/math.Pow(float64(i), 0.8))
	}

	cdf := make([]float64, len(weights))
	total := 0.0
	for i, w := range weights {
		total += w
		cdf[i] = total
	}

	followerCount := make(map[int64]int, len(normal))
	edges := make([]edge, 0, users*follows+celebs*celebFollowers)
	for uid := int64(1); uid <= int64(users); uid++ {
		if _, isCeleb := celebIDs[uid]; isCeleb {
			continue
		}

		jitter := 0
		if follows > 4 {
			jitter = rng.Intn(follows/5+1) - follows/10
		}
		k := follows + jitter
		if k < 1 {
			k = 1
		}
		if k > len(normal)-1 {
			k = len(normal) - 1
		}

		chosen := make(map[int64]struct{}, k)
		attempts := 0
		for len(chosen) < k {
			attempts++
			if attempts > k*200 {
				break
			}
			x := rng.Float64() * total
			idx := sort.Search(len(cdf), func(i int) bool { return cdf[i] >= x })
			if idx >= len(normal) {
				idx = len(normal) - 1
			}
			followee := normal[idx]
			if followee == uid {
				continue
			}
			if _, ok := chosen[followee]; ok {
				continue
			}
			if followerCount[followee] >= maxFollowers {
				continue
			}
			chosen[followee] = struct{}{}
			followerCount[followee]++
			edges = append(edges, edge{follower: uid, followee: followee})
		}
	}

	// Celebridades: conjunto explícito de seguidores, para controlar o threshold.
	for c := int64(1); c <= int64(celebs); c++ {
		count := 0
		for _, idx := range rng.Perm(users) {
			follower := int64(idx + 1)
			if follower == c {
				continue
			}
			edges = append(edges, edge{follower: follower, followee: c})
			count++
			if count >= celebFollowers {
				break
			}
		}
	}

	return edges, nil
}

func insertFollows(ctx context.Context, pool *pgxpool.Pool, edges []edge) error {
	const batchSize = 10000
	createdAt := time.Now().Add(-time.Hour).UTC()

	for start := 0; start < len(edges); start += batchSize {
		end := min(start+batchSize, len(edges))
		rows := make([][]any, 0, end-start)
		for _, e := range edges[start:end] {
			rows = append(rows, []any{e.follower, e.followee, createdAt})
		}
		if _, err := pool.CopyFrom(ctx,
			pgx.Identifier{"social", "follows"},
			[]string{"follower_id", "followee_id", "created_at"},
			pgx.CopyFromRows(rows),
		); err != nil {
			return err
		}
	}
	return nil
}

func seedCelebFollows(ctx context.Context, pool *pgxpool.Pool, rdb *redis.Client, celebs int) error {
	if celebs == 0 {
		return nil
	}
	ids := make([]int64, 0, celebs)
	for i := 1; i <= celebs; i++ {
		ids = append(ids, int64(i))
	}

	rows, err := pool.Query(ctx, `
		SELECT followee_id, follower_id, created_at
		  FROM social.follows
		 WHERE followee_id = ANY($1)
	`, ids)
	if err != nil {
		return err
	}
	defer rows.Close()

	pipe := rdb.Pipeline()
	pending := 0
	for rows.Next() {
		var (
			followee, follower int64
			createdAt          time.Time
		)
		if err := rows.Scan(&followee, &follower, &createdAt); err != nil {
			return err
		}
		key := fmt.Sprintf("celeb_follows:%d", follower)
		pipe.ZAdd(ctx, key, redis.Z{Score: float64(createdAt.UnixMilli()), Member: followee})
		pending++
		if pending >= 1000 {
			if _, err := pipe.Exec(ctx); err != nil {
				return err
			}
			pipe = rdb.Pipeline()
			pending = 0
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if pending > 0 {
		_, err = pipe.Exec(ctx)
	}
	return err
}

func printSummary(ctx context.Context, pool *pgxpool.Pool, users, celebs int, edges int) {
	rows, err := pool.Query(ctx, `
		SELECT id, handle, follower_count
		  FROM social.users
		 ORDER BY follower_count DESC, id
		 LIMIT 10
	`)
	if err != nil {
		return
	}
	defer rows.Close()

	fmt.Printf("\nsemeados: users=%d celebs=%d edges=%d\n", users, celebs, edges)
	fmt.Println("top seguidores:")
	for rows.Next() {
		var id int64
		var handle string
		var count int
		if err := rows.Scan(&id, &handle, &count); err != nil {
			return
		}
		fmt.Printf("  id=%-6d handle=%-8s followers=%d\n", id, handle, count)
	}
	fmt.Println("\nexemplo:")
	fmt.Printf("  curl -H 'Authorization: Bearer 10' -X POST localhost:8080/v1/posts -d '{\"body\":\"oi\"}'\n")
	fmt.Printf("  curl -H 'Authorization: Bearer 11' localhost:8080/v1/timeline/home\n")
}
