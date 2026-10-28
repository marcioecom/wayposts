package social

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/marciojunior/wayposts/internal/model"
)

const (
	// Resumos de autor mudam pouco e são lidos em toda página de timeline.
	// Dez minutos mantêm o custo baixo sem segurar renomeações por muito tempo.
	userCacheTTL = 10 * time.Minute
	// Autor inexistente fica marcado por menos tempo, para que uma conta
	// criada depois apareça logo.
	userMissTTL = 30 * time.Second
)

// userMissingField marca um id que o banco não devolveu.
const userMissingField = "missing"

func userKey(id int64) string {
	return "user:" + strconv.FormatInt(id, 10)
}

// GetUsers busca os resumos no Postgres em uma consulta. Se o Redis falhar, a
// leitura cai no banco.
func (s *Store) GetUsers(ctx context.Context, ids []int64) ([]model.AuthorSummary, error) {
	unique := uniquePositiveIDs(ids)
	if len(unique) == 0 {
		return []model.AuthorSummary{}, nil
	}

	byID, misses := s.cachedUsers(ctx, unique)

	var dbErr error
	if len(misses) > 0 {
		found, err := s.usersFromDB(ctx, misses)
		if err != nil {
			// Devolve o que o cache já tinha para não derrubar a página.
			dbErr = fmt.Errorf("social: users: %w", err)
		} else {
			for _, u := range found {
				byID[u.ID] = u
			}
			s.cacheUsers(ctx, found)
			s.cacheMissingUsers(ctx, missingFrom(found, misses))
		}
	}

	out := make([]model.AuthorSummary, 0, len(unique))
	for _, id := range unique {
		if u, ok := byID[id]; ok {
			out = append(out, u)
		}
	}
	return out, dbErr
}

func (s *Store) cachedUsers(ctx context.Context, ids []int64) (map[int64]model.AuthorSummary, []int64) {
	out := make(map[int64]model.AuthorSummary, len(ids))
	misses := make([]int64, 0, len(ids))

	pipe := s.rdb.Pipeline()
	cmds := make(map[int64]*redis.MapStringStringCmd, len(ids))
	for _, id := range ids {
		cmds[id] = pipe.HGetAll(ctx, userKey(id))
	}
	if _, err := pipe.Exec(ctx); err != nil && err != redis.Nil {
		s.m.UserCache.WithLabelValues("redis_error").Inc()
		s.m.UserCache.WithLabelValues("miss").Add(float64(len(ids)))
		for _, id := range ids {
			misses = append(misses, id)
		}
		return out, misses
	}

	for _, id := range ids {
		raw := cmds[id].Val()
		if raw[userMissingField] == "1" {
			s.m.UserCache.WithLabelValues("absent").Inc()
			continue
		}
		u, ok := fromUserHash(id, raw)
		if !ok {
			misses = append(misses, id)
			continue
		}
		out[id] = u
		s.m.UserCache.WithLabelValues("hit").Inc()
	}
	s.m.UserCache.WithLabelValues("miss").Add(float64(len(misses)))
	return out, misses
}

func fromUserHash(id int64, raw map[string]string) (model.AuthorSummary, bool) {
	if raw["id"] == "" {
		return model.AuthorSummary{}, false
	}
	return model.AuthorSummary{
		ID:          id,
		Handle:      raw["handle"],
		DisplayName: raw["display_name"],
	}, true
}

func (s *Store) usersFromDB(ctx context.Context, ids []int64) ([]model.AuthorSummary, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, handle, display_name
		  FROM social.users
		 WHERE id = ANY($1)
	`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]model.AuthorSummary, 0, len(ids))
	for rows.Next() {
		var u model.AuthorSummary
		if err := rows.Scan(&u.ID, &u.Handle, &u.DisplayName); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func (s *Store) cacheUsers(ctx context.Context, users []model.AuthorSummary) {
	if len(users) == 0 {
		return
	}
	// TxPipeline mantém HSET e EXPIRE juntos. Um pipeline simples pode gravar o
	// hash e falhar antes do TTL, deixando o resumo antigo para sempre.
	pipe := s.rdb.TxPipeline()
	for _, u := range users {
		key := userKey(u.ID)
		pipe.HSet(ctx, key, map[string]any{
			"id":           u.ID,
			"handle":       u.Handle,
			"display_name": u.DisplayName,
		})
		pipe.Expire(ctx, key, userCacheTTL)
	}
	if _, err := pipe.Exec(ctx); err != nil && err != redis.Nil {
		s.m.UserCache.WithLabelValues("redis_error").Inc()
		s.log.Warn("user cache", "err", err)
	}
}

func (s *Store) cacheMissingUsers(ctx context.Context, ids []int64) {
	if len(ids) == 0 {
		return
	}
	pipe := s.rdb.TxPipeline()
	for _, id := range ids {
		key := userKey(id)
		pipe.HSet(ctx, key, userMissingField, "1")
		pipe.Expire(ctx, key, userMissTTL)
	}
	if _, err := pipe.Exec(ctx); err != nil && err != redis.Nil {
		s.m.UserCache.WithLabelValues("redis_error").Inc()
		s.log.Warn("user cache missing", "err", err)
	}
}

func missingFrom(found []model.AuthorSummary, requested []int64) []int64 {
	seen := make(map[int64]struct{}, len(found))
	for _, u := range found {
		seen[u.ID] = struct{}{}
	}
	out := make([]int64, 0, len(requested))
	for _, id := range requested {
		if _, ok := seen[id]; ok {
			continue
		}
		out = append(out, id)
	}
	return out
}

func uniquePositiveIDs(ids []int64) []int64 {
	seen := make(map[int64]struct{}, len(ids))
	out := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}
