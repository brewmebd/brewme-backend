package model

import "time"

// ReplyRequest parses the incoming JSON body from the frontend
type ReplyRequest struct {
	Message string `json:"message"`
}

// ReplyResponse structures the JSON sent back to the frontend upon success
type ReplyResponse struct {
	Status       bool   `json:"status"`
	Message      string `json:"message"`
	CreatorReply string `json:"creator_reply"`
}

type SupporterItem struct {
	ID               int64     `json:"id"`
	SupporterName    string    `json:"supporter_name"`
	SupporterMessage *string   `json:"supporter_message"` // Pointer to allow `null` in JSON
	TotalAmount      float64   `json:"total_amount"`
	SupporterCups    int       `json:"supporter_cups"`
	CreatedAt        time.Time `json:"created_at"`
	SupportType      string    `json:"support_type"`
	SupportReplied   bool      `json:"support_replied"`
	CreatorReply     *string   `json:"creator_reply"` // Pointer to allow `null` in JSON
}
