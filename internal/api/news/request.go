package news

import "time"

type NewsWriteRequest struct {
	Title       string `json:"title" validate:"required"`
	Summary     string `json:"summary" validate:"required"`
	Content     string `json:"content" validate:"required"`
	IsPublished bool   `json:"is_published"`
	ImageURL    string `json:"-"`
}

type PublicNewsItem struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Summary     string     `json:"summary"`
	ImageURL    string     `json:"image_url"`
	PublishedAt *time.Time `json:"published_at"`
}

type PublicNewsDetail struct {
	PublicNewsItem
	Content string `json:"content"`
}

type PublicNewsPage struct {
	Items []PublicNewsItem `json:"items"`
	Page  int              `json:"page"`
	Limit int              `json:"limit"`
	Total int64            `json:"total"`
}
