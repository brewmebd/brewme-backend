package model

// DashboardMembershipTier represents a creator's membership tier with perks and
// live subscriber counts for the dashboard.
type DashboardMembershipTier struct {
	ID              int64    `json:"id"`
	Name            string   `json:"name"`
	Price           float64  `json:"price"`
	SubscriberCount int64    `json:"subscriber_count"`
	Perks           []string `json:"perks"`
	IsActive        bool     `json:"is_active"`
}

// DashboardMembershipSummary provides the headline metrics shown above the
// membership tier cards.
type DashboardMembershipSummary struct {
	TotalMembers   int64   `json:"total_members"`
	MonthlyRevenue float64 `json:"monthly_revenue"`
	ActiveTiers    int64   `json:"active_tiers"`
}

// DashboardMembershipsResponse is the payload returned to the Memberships page.
type DashboardMembershipsResponse struct {
	Summary DashboardMembershipSummary `json:"summary"`
	Tiers   []DashboardMembershipTier  `json:"tiers"`
}

// DashboardNotifications stores the notification preferences surfaced by the
// Settings page.
type DashboardNotifications struct {
	NewSupporter    bool `json:"new_supporter"`
	NewMessage      bool `json:"new_message"`
	WeeklyReport    bool `json:"weekly_report"`
	MarketingEmails bool `json:"marketing_emails"`
}

// DashboardStripeStatus captures the read-only payout connection state.
type DashboardStripeStatus struct {
	IsConnected bool   `json:"is_connected"`
	CardLast4   string `json:"card_last4"`
}

// DashboardGoal stores the active fundraising goal shown in Settings.
type DashboardGoal struct {
	Title  string  `json:"goal_title"`
	Amount float64 `json:"goal_amount"`
}

type DashboardSettingsResponse struct {
	Profile       CreatorProfile         `json:"profile"`
	Notifications DashboardNotifications `json:"notifications"`
	Stripe        DashboardStripeStatus  `json:"stripe"`
	Goal          DashboardGoal          `json:"goal"`
	SocialLinks   []string               `json:"social_links"`
}

// UpdateDashboardProfileRequest updates the public creator identity fields.
type UpdateDashboardProfileRequest struct {
	Name        string   `json:"name"`
	Bio         string   `json:"bio"`
	Email       string   `json:"email"`
	Category    string   `json:"category"`
	SocialLinks []string `json:"social_links"`
}

// UpdateDashboardNotificationsRequest updates notification preferences.
type UpdateDashboardNotificationsRequest struct {
	NewSupporter    bool `json:"new_supporter"`
	NewMessage      bool `json:"new_message"`
	WeeklyReport    bool `json:"weekly_report"`
	MarketingEmails bool `json:"marketing_emails"`
}

// UpdateDashboardGoalRequest updates the active goal record.
type UpdateDashboardGoalRequest struct {
	Title  string  `json:"goal_title"`
	Amount float64 `json:"goal_amount"`
}

// CreateDashboardMembershipTierRequest creates or edits a membership tier.
type CreateDashboardMembershipTierRequest struct {
	Name  string   `json:"name"`
	Price float64  `json:"price"`
	Perks []string `json:"perks"`
}
