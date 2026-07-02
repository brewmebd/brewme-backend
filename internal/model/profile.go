package model

type CreatorProfile struct {
	CreatorID       int64  `json:"creator_id"`
	CreatorImage    string `json:"creator_image"`
	CreatorName     string `json:"creator_name"`
	CreatorBio      string `json:"creator_bio"`
	CreatorUrl      string `json:"creator_url"`
	CreatorEmail    string `json:"creator_email"`
	CreatorCategory string `json:"creator_category"`
}
