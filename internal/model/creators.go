package model

import "time"

type CreatorMembershipTier struct {
	ID    int64    `json:"id"`
	Name  string   `json:"name"`
	Price float64  `json:"price"`
	Perks []string `json:"perks"`
}

type GetCreatorProfile struct {
	CreatorImage    string                  `json:"creator_image"`
	CreatorName     string                  `json:"creator_name"`
	CreatorCategory string                  `json:"creator_category"`
	CreatorBio      string                  `json:"creator_bio"`
	CreatorLinks    []string                `json:"creator_links"`
	TotalCups       int64                   `json:"total_cups"`
	TotalSupporters int64                   `json:"total_supporters"`
	Tiers           []CreatorMembershipTier `json:"tiers"`
	StripeConnected bool                    `json:"stripe_connected"`
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

type PublicPostItem struct {
	ID            int64     `json:"id"`
	Title         string    `json:"title"`
	Preview       string    `json:"preview"`
	LikesCount    int64     `json:"likes_count"`
	CommentsCount int64     `json:"comments_count"`
	PublishedAt   time.Time `json:"published_at"`
	MembersOnly   bool      `json:"membersOnly"`
}
