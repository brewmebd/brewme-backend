package model

import "time"

// DashboardPostItem is a post as shown to its own creator in the dashboard.
// Unlike PublicPostItem it includes private fields (visibility, image, status)
// and is returned for every visibility, not just public ones.
type DashboardPostItem struct {
	ID            int64      `json:"id"`
	Title         string     `json:"title"`
	Preview       string     `json:"preview"`
	Body          string     `json:"body"`
	Image         *string    `json:"image"` // nil -> JSON null
	Visibility    string     `json:"visibility"`
	MembersOnly   bool       `json:"membersOnly"`
	Status        string     `json:"status"`
	LikesCount    int64      `json:"likes_count"`
	CommentsCount int64      `json:"comments_count"`
	PublishedAt   *time.Time `json:"published_at"`
	CreatedAt     time.Time  `json:"created_at"`
}
