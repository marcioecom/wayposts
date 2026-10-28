package socialclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/marciojunior/wayposts/internal/model"
)

type Client struct {
	base string
	http *http.Client
}

func New(base string) *Client {
	return &Client{
		base: base,
		http: &http.Client{Timeout: 5 * time.Second},
	}
}

func (c *Client) FollowerCount(ctx context.Context, authorID int64) (int, error) {
	var out struct {
		Count int `json:"count"`
	}
	if err := c.get(ctx, fmt.Sprintf("/internal/followers/%d/count", authorID), &out); err != nil {
		return 0, err
	}
	return out.Count, nil
}

func (c *Client) Followers(ctx context.Context, authorID, after int64, limit int) ([]int64, error) {
	var out struct {
		Followers []int64 `json:"followers"`
		NextAfter int64   `json:"next_after"`
	}
	q := url.Values{}
	q.Set("after", strconv.FormatInt(after, 10))
	q.Set("limit", strconv.Itoa(limit))
	path := fmt.Sprintf("/internal/followers/%d?%s", authorID, q.Encode())
	if err := c.get(ctx, path, &out); err != nil {
		return nil, err
	}
	return out.Followers, nil
}

// Users busca o resumo de vários autores em uma chamada.
func (c *Client) Users(ctx context.Context, ids []int64) ([]model.AuthorSummary, error) {
	if len(ids) == 0 {
		return []model.AuthorSummary{}, nil
	}
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = strconv.FormatInt(id, 10)
	}
	q := url.Values{}
	q.Set("ids", strings.Join(parts, ","))

	var out struct {
		Items []model.AuthorSummary `json:"items"`
	}
	if err := c.get(ctx, "/internal/users?"+q.Encode(), &out); err != nil {
		return nil, err
	}
	return out.Items, nil
}

func (c *Client) get(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+path, nil)
	if err != nil {
		return err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("social: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("social: %s respondeu %d", path, resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}
