package events

type PostCreated struct {
	PostID   int64 `json:"post_id"`
	AuthorID int64 `json:"author_id"`
}

const TopicPostCreated = "PostCreated"
