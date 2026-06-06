package model

import "time"

// ChartEntry remains the same
type ChartEntry struct {
	Month    string  `json:"month"`
	Earnings float64 `json:"earnings"`
}

type DashboardStats struct {
	TotalEarned      float64      `json:"total_earned"`      // e.g., 2847.50
	EarningsChange   float64      `json:"earnings_change"`   // e.g., 12.5 (Frontend adds the '+' and '%')
	MonthlyEarned    float64      `json:"monthly_earned"`    // e.g., 486.00
	MonthlyChange    float64      `json:"monthly_change"`    // e.g., 8.3
	TotalSupporters  int64        `json:"total_supporters"`  // e.g., 1247
	SupportersChange int64        `json:"supporters_change"` // e.g., 23
	TotalPosts       int64        `json:"total_posts"`       // e.g., 34
	PostsChange      int64        `json:"posts_change"`      // e.g., 3
	ChartData        []ChartEntry `json:"chart_data"`
}

type RecentSupporter struct {
	SupporterName    string    `json:"supporter_name"`
	SupporterMessage string    `json:"supporter_message"`
	TotalAmount      float64   `json:"total_amount"`
	SupporterCups    int       `json:"supporter_cups"`
	CreatedAt        time.Time `json:"created_at"`
	SupportType      string    `json:"support_type"`
	SupportReplied   bool      `json:"support_replied"`
}
