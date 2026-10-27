package postclient

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

func (c *Client) GetMany(ctx context.Context, ids []int64, viewerID int64) ([]model.Post, error) {
	if len(ids) == 0 {
		return []model.Post{}, nil
	}
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = strconv.FormatInt(id, 10)
	}
	q := url.Values{}
	q.Set("ids", strings.Join(parts, ","))
	if viewerID > 0 {
		q.Set("viewer_id", strconv.FormatInt(viewerID, 10))
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		fmt.Sprintf("%s/internal/posts?%s", c.base, q.Encode()), nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("post: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("post: /internal/posts respondeu %d", resp.StatusCode)
	}

	var out struct {
		Items []model.Post `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out.Items, nil
}
