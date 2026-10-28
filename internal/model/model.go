package model

import "time"

type Post struct {
	ID          int64     `json:"id"`
	AuthorID    int64     `json:"author_id"`
	Body        string    `json:"body"`
	Media       []string  `json:"media"`
	CreatedAt   time.Time `json:"created_at"`
	Deleted     bool      `json:"deleted"`
	LikeCount   int64     `json:"like_count"`
	ViewerLiked bool      `json:"viewer_liked"`
	// Author fica nil quando o social não responde.
	Author *AuthorSummary `json:"author,omitempty"`
}

type User struct {
	ID            int64  `json:"id"`
	Handle        string `json:"handle"`
	DisplayName   string `json:"display_name"`
	FollowerCount int    `json:"follower_count"`
}

// AuthorSummary é a autoria mínima de um post na timeline.
type AuthorSummary struct {
	ID          int64  `json:"id"`
	Handle      string `json:"handle"`
	DisplayName string `json:"display_name"`
}
