package model

// PayoutItem is a single row in the payout history table.
type PayoutItem struct {
	ID     string `json:"id"`     // reference, e.g. "PO-001"
	Amount string `json:"amount"` // "450.00"
	Date   string `json:"date"`   // "Sep 1, 2026"
	Status string `json:"status"` // "Completed"
}

// EarningsResponse is the dashboard Earnings payload. Money fields are
// pre-formatted strings ($ + thousands separators) to match the frontend.
type EarningsResponse struct {
	TotalEarned      string       `json:"total_earned"`
	TotalChange      string       `json:"total_change"`
	AvailableBalance string       `json:"available_balance"`
	TotalPayoutsSum  string       `json:"total_payouts_sum"`
	ChartData        []ChartEntry `json:"chart_data"`
	Payouts          []PayoutItem `json:"payouts"`
}

// PayoutRequest is the body for requesting a payout.
type PayoutRequest struct {
	Amount float64 `json:"amount"`
	Method string  `json:"method"`
}
