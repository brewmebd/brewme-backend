package model

import "time"

type GetCreatorProfile struct {
	CreatorImage    string   `json:"creator_image"`
	CreatorName     string   `json:"creator_name"`
	CreatorCategory string   `json:"creator_category"`
	CreatorBio      string   `json:"creator_bio"`
	CreatorLinks    []string `json:"creator_links"`
	TotalCups       int64    `json:"total_cups"`
}

type GetSupportersFeed struct {
	SupportType      string `json:"support_type"`
	SupporterName    string `json:"supporter_name"`
	SupporterMessage string `json:"supporter_message"` // (or SupporterMassage if you kept it!)
	SupporterCups    int64  `json:"supporter_cups"`

	// FIX: Change this to float64 so it can hold decimals like 15.00
	TotalAmounts float64 `json:"total_amount"`

	SupporterReplied bool      `json:"support_replied"`
	CreatedAt        time.Time `json:"created_at"`
}

type GetCreatorPosts struct {
	
}