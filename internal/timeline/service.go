package timeline

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strconv"

	"github.com/redis/go-redis/v9"

	postclient "github.com/marciojunior/wayposts/internal/clients/post"
	"github.com/marciojunior/wayposts/internal/id"
	"github.com/marciojunior/wayposts/internal/model"
)

type Page struct {
	Items      []model.Post `json:"items"`
	NextCursor int64        `json:"next_cursor"`
}

type Service struct {
	rdb       *redis.Client
	posts     *postclient.Client
	maxCelebs int
	log       *slog.Logger
}

func NewService(rdb *redis.Client, posts *postclient.Client, maxCelebs int, log *slog.Logger) *Service {
	if maxCelebs <= 0 {
		maxCelebs = 100
	}
	return &Service{rdb: rdb, posts: posts, maxCelebs: maxCelebs, log: log}
}

func feedKey(userID int64) string {
	return "feed:" + strconv.FormatInt(userID, 10)
}

func authorKey(userID int64) string {
	return "author:" + strconv.FormatInt(userID, 10)
}

func celebKey(userID int64) string {
	return "celeb_follows:" + strconv.FormatInt(userID, 10)
}

func (s *Service) Home(ctx context.Context, userID, before int64, limit int) (Page, error) {
	ids, err := s.candidates(ctx, userID, before, limit)
	if err != nil {
		return Page{}, err
	}
	items, err := s.hydrate(ctx, userID, userID, ids)
	if err != nil {
		return Page{}, err
	}
	return pageOf(items), nil
}

func (s *Service) Profile(ctx context.Context, viewerID, authorID, before int64, limit int) (Page, error) {
	by := &redis.ZRangeBy{Min: "-inf", Max: maxScore(before), Offset: 0, Count: int64(limit)}
	raw, err := s.rdb.ZRevRangeByScore(ctx, authorKey(authorID), by).Result()
	if err != nil && err != redis.Nil {
		return Page{}, fmt.Errorf("timeline: profile: %w", err)
	}

	ids := make([]int64, 0, len(raw))
	for _, item := range raw {
		if pid, err := strconv.ParseInt(item, 10, 64); err == nil && pid > 0 {
			ids = append(ids, pid)
		}
	}
	items, err := s.hydrate(ctx, viewerID, 0, ids)
	if err != nil {
		return Page{}, err
	}
	return pageOf(items), nil
}

func pageOf(items []model.Post) Page {
	next := int64(0)
	if len(items) > 0 {
		next = items[len(items)-1].ID
	}
	return Page{Items: items, NextCursor: next}
}

func maxScore(before int64) string {
	if before > 0 {
		return "(" + strconv.FormatInt(before, 10)
	}
	return "+inf"
}

func (s *Service) candidates(ctx context.Context, userID, before int64, limit int) ([]int64, error) {
	by := &redis.ZRangeBy{Min: "-inf", Max: maxScore(before), Offset: 0, Count: int64(limit * 2)}

	seen := make(map[int64]struct{}, limit*3)
	ids := make([]int64, 0, limit*3)
	add := func(raw string) {
		pid, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || pid <= 0 {
			return
		}
		if _, ok := seen[pid]; ok {
			return
		}
		seen[pid] = struct{}{}
		ids = append(ids, pid)
	}

	feed, err := s.rdb.ZRevRangeByScore(ctx, feedKey(userID), by).Result()
	if err != nil && err != redis.Nil {
		return nil, fmt.Errorf("timeline: feed: %w", err)
	}
	for _, raw := range feed {
		add(raw)
	}

	// Read-your-writes: o próprio autor aparece mesmo antes do fanout terminar.
	own, err := s.rdb.ZRevRangeByScore(ctx, authorKey(userID), by).Result()
	if err != nil && err != redis.Nil {
		return nil, fmt.Errorf("timeline: own: %w", err)
	}
	for _, raw := range own {
		add(raw)
	}

	// Celebridades não são materializadas: são lidas e mescladas aqui.
	celebs, err := s.rdb.ZRangeWithScores(ctx, celebKey(userID), 0, int64(s.maxCelebs-1)).Result()
	if err != nil && err != redis.Nil {
		return nil, fmt.Errorf("timeline: celebs: %w", err)
	}
	for _, celeb := range celebs {
		followAt := int64(celeb.Score)
		celebID, err := strconv.ParseInt(fmt.Sprint(celeb.Member), 10, 64)
		if err != nil {
			continue
		}
		list, err := s.rdb.ZRevRangeByScore(ctx, authorKey(celebID), by).Result()
		if err != nil && err != redis.Nil {
			return nil, fmt.Errorf("timeline: celeb feed: %w", err)
		}
		for _, raw := range list {
			pid, err := strconv.ParseInt(raw, 10, 64)
			if err != nil {
				continue
			}
			if id.Time(pid).UnixMilli() < followAt {
				continue
			}
			add(raw)
		}
	}

	sort.Slice(ids, func(i, j int) bool { return ids[i] > ids[j] })
	if len(ids) > limit {
		ids = ids[:limit]
	}
	return ids, nil
}

func (s *Service) hydrate(ctx context.Context, viewerID, feedOwner int64, ids []int64) ([]model.Post, error) {
	if len(ids) == 0 {
		return []model.Post{}, nil
	}

	posts, err := s.posts.GetMany(ctx, ids, viewerID)
	if err != nil {
		return nil, err
	}
	byID := make(map[int64]model.Post, len(posts))
	for _, p := range posts {
		byID[p.ID] = p
	}

	out := make([]model.Post, 0, len(ids))
	var deleted []int64
	for _, pid := range ids {
		p, ok := byID[pid]
		if !ok {
			continue
		}
		if p.Deleted {
			deleted = append(deleted, pid)
			continue
		}
		out = append(out, p)
	}

	if len(deleted) > 0 {
		pipe := s.rdb.Pipeline()
		for _, pid := range deleted {
			if feedOwner > 0 {
				pipe.ZRem(ctx, feedKey(feedOwner), pid)
			}
			// O índice do autor é o do autor do post, que pode não ser o dono
			// da timeline quando o item veio do merge de celebridades.
			if p, ok := byID[pid]; ok {
				pipe.ZRem(ctx, authorKey(p.AuthorID), pid)
			}
		}
		if _, err := pipe.Exec(ctx); err != nil {
			s.log.Warn("timeline lazy delete", "err", err)
		}
	}
	return out, nil
}
