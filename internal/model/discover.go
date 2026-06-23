package model

type GetAllCreator struct {
	CreatorID             int64   `json:"creator_id"`
	CreatorUsername       string  `json:"creator_username"`
	CreatorProfilePicture *string `json:"creator_profile_picture"` // Changed to pointer
	CreatorName           string  `json:"creator_name"`
	CreatorCategory       *string `json:"creator_category"`    // Changed to pointer
	CreatorDescription    *string `json:"creator_description"` // Changed to pointer
	TotalSupportersCup    int64   `json:"total_supporters_cup"`
}
