package model

import "time"

type GetAllCategory struct {
	CategoryID   int64     `json:"category_id"`
	CategoryName string    `json:"category_name"`
	CategorySlug string    `json:"category_slug"`
	CreatedAt    time.Time `json:"created_at"`
}
