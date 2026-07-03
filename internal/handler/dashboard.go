package handler

import (
	"brewme/internal/database"
	"brewme/internal/model"
	"brewme/internal/utils"
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi"
)

func getAuthenticatedUserID(r *http.Request) (int64, error) {
	token, err := utils.GetTokenFromHeader(r)
	if err != nil {
		return 0, err
	}

	return utils.GetUserIDFromToken(token)
}

func loadMembershipPerks(tierID int64) ([]string, error) {
	rows, err := database.DB.Query(`
		SELECT perk_text
		FROM tier_perks
		WHERE tier_id = $1
		ORDER BY sort_order ASC, id ASC`, tierID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	perks := make([]string, 0)
	for rows.Next() {
		var perk string
		if err := rows.Scan(&perk); err != nil {
			return nil, err
		}
		perks = append(perks, perk)
	}

	return perks, rows.Err()
}

func loadMembershipTier(userID, tierID int64) (*model.DashboardMembershipTier, error) {
	var tier model.DashboardMembershipTier
	err := database.DB.QueryRow(`
		SELECT id, name, price, is_active
		FROM membership_tiers
		WHERE id = $1 AND user_id = $2`, tierID, userID).Scan(
		&tier.ID,
		&tier.Name,
		&tier.Price,
		&tier.IsActive,
	)
	if err != nil {
		return nil, err
	}

	err = database.DB.QueryRow(`
		SELECT COUNT(*)
		FROM memberships
		WHERE user_id = $1 AND tier_id = $2 AND status = 'active'`, userID, tierID).Scan(&tier.SubscriberCount)
	if err != nil {
		return nil, err
	}

	perks, err := loadMembershipPerks(tierID)
	if err != nil {
		return nil, err
	}
	tier.Perks = perks

	return &tier, nil
}

func GetDashboardMemberships(w http.ResponseWriter, r *http.Request) {
	userID, err := getAuthenticatedUserID(r)
	if err != nil {
		http.Error(w, "Invalid to get user from the token", http.StatusUnauthorized)
		return
	}

	var response model.DashboardMembershipsResponse
	if err := database.DB.QueryRow(`
		SELECT
			COUNT(*),
			COALESCE(SUM(amount), 0)
		FROM memberships
		WHERE user_id = $1 AND status = 'active'`, userID).Scan(&response.Summary.TotalMembers, &response.Summary.MonthlyRevenue); err != nil {
		http.Error(w, "Error fetching membership summary", http.StatusInternalServerError)
		return
	}

	if err := database.DB.QueryRow(`
		SELECT COUNT(*)
		FROM membership_tiers
		WHERE user_id = $1 AND is_active = TRUE`, userID).Scan(&response.Summary.ActiveTiers); err != nil {
		http.Error(w, "Error fetching active tiers", http.StatusInternalServerError)
		return
	}

	rows, err := database.DB.Query(`
		SELECT id
		FROM membership_tiers
		WHERE user_id = $1 AND is_active = TRUE
		ORDER BY sort_order ASC, id ASC`, userID)
	if err != nil {
		http.Error(w, "Error fetching tiers", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	response.Tiers = make([]model.DashboardMembershipTier, 0)
	for rows.Next() {
		var tierID int64
		if err := rows.Scan(&tierID); err != nil {
			http.Error(w, "Error reading tiers", http.StatusInternalServerError)
			return
		}

		tier, err := loadMembershipTier(userID, tierID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				continue
			}
			http.Error(w, "Error loading tier details", http.StatusInternalServerError)
			return
		}

		response.Tiers = append(response.Tiers, *tier)
	}

	if err := rows.Err(); err != nil {
		http.Error(w, "Error iterating tiers", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, response)
}

func CreateDashboardMembershipTier(w http.ResponseWriter, r *http.Request) {
	userID, err := getAuthenticatedUserID(r)
	if err != nil {
		http.Error(w, "Invalid to get user from the token", http.StatusUnauthorized)
		return
	}

	var req model.CreateDashboardMembershipTierRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" || req.Price <= 0 || len(req.Perks) == 0 {
		http.Error(w, "Name, price and perks are required", http.StatusBadRequest)
		return
	}

	var activeCount int64
	if err := database.DB.QueryRow(`
		SELECT COUNT(*)
		FROM membership_tiers
		WHERE user_id = $1 AND is_active = TRUE`, userID).Scan(&activeCount); err != nil {
		http.Error(w, "Error checking membership limit", http.StatusInternalServerError)
		return
	}
	if activeCount >= 3 {
		http.Error(w, "You can only create up to 3 membership tiers", http.StatusConflict)
		return
	}

	tx, err := database.DB.Begin()
	if err != nil {
		http.Error(w, "Error starting transaction", http.StatusInternalServerError)
		return
	}
	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()

	var sortOrder int64
	if err := tx.QueryRow(`
		SELECT COALESCE(MAX(sort_order), -1) + 1
		FROM membership_tiers
		WHERE user_id = $1`, userID).Scan(&sortOrder); err != nil {
		http.Error(w, "Error calculating tier order", http.StatusInternalServerError)
		return
	}

	var tierID int64
	err = tx.QueryRow(`
		INSERT INTO membership_tiers (user_id, name, price, billing_period, sort_order, is_active)
		VALUES ($1, $2, $3, 'monthly', $4, TRUE) RETURNING id`, userID, req.Name, req.Price, sortOrder).Scan(&tierID)
	if err != nil {
		log.Printf("Error creating tier: %v\n", err)
		http.Error(w, "Error creating tier", http.StatusInternalServerError)
		return
	}

	perkIndex := 0
	for _, perk := range req.Perks {
		perk = strings.TrimSpace(perk)
		if perk == "" {
			continue
		}

		if _, err := tx.Exec(`
			INSERT INTO tier_perks (tier_id, perk_text, sort_order)
			VALUES ($1, $2, $3)`, tierID, perk, perkIndex); err != nil {
			log.Printf("Error saving perks: %v\n", err)
			http.Error(w, "Error saving perks", http.StatusInternalServerError)
			return
		}
		perkIndex++
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, "Error saving tier", http.StatusInternalServerError)
		return
	}
	tx = nil

	tier, err := loadMembershipTier(userID, tierID)
	if err != nil {
		http.Error(w, "Error loading created tier", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, tier)
}

func UpdateDashboardMembershipTier(w http.ResponseWriter, r *http.Request) {
	userID, err := getAuthenticatedUserID(r)
	if err != nil {
		http.Error(w, "Invalid to get user from the token", http.StatusUnauthorized)
		return
	}

	tierID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || tierID <= 0 {
		http.Error(w, "Invalid tier id", http.StatusBadRequest)
		return
	}

	var req model.CreateDashboardMembershipTierRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" || req.Price <= 0 || len(req.Perks) == 0 {
		http.Error(w, "Name, price and perks are required", http.StatusBadRequest)
		return
	}

	tx, err := database.DB.Begin()
	if err != nil {
		http.Error(w, "Error starting transaction", http.StatusInternalServerError)
		return
	}
	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()

	var existing int64
	if err := tx.QueryRow(`
		SELECT id
		FROM membership_tiers
		WHERE id = $1 AND user_id = $2`, tierID, userID).Scan(&existing); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "Tier not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Error loading tier", http.StatusInternalServerError)
		return
	}

	if _, err := tx.Exec(`
		UPDATE membership_tiers
		SET name = $1, price = $2, is_active = TRUE
		WHERE id = $3 AND user_id = $4`, req.Name, req.Price, tierID, userID); err != nil {
		http.Error(w, "Error updating tier", http.StatusInternalServerError)
		return
	}

	if _, err := tx.Exec(`DELETE FROM tier_perks WHERE tier_id = $1`, tierID); err != nil {
		http.Error(w, "Error resetting perks", http.StatusInternalServerError)
		return
	}

	perkIndex := 0
	for _, perk := range req.Perks {
		perk = strings.TrimSpace(perk)
		if perk == "" {
			continue
		}

		if _, err := tx.Exec(`
			INSERT INTO tier_perks (tier_id, perk_text, sort_order)
			VALUES ($1, $2, $3)`, tierID, perk, perkIndex); err != nil {
			http.Error(w, "Error saving perks", http.StatusInternalServerError)
			return
		}
		perkIndex++
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, "Error saving tier", http.StatusInternalServerError)
		return
	}
	tx = nil

	tier, err := loadMembershipTier(userID, tierID)
	if err != nil {
		http.Error(w, "Error loading updated tier", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, tier)
}

func DeleteDashboardMembershipTier(w http.ResponseWriter, r *http.Request) {
	userID, err := getAuthenticatedUserID(r)
	if err != nil {
		http.Error(w, "Invalid to get user from the token", http.StatusUnauthorized)
		return
	}

	tierID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || tierID <= 0 {
		http.Error(w, "Invalid tier id", http.StatusBadRequest)
		return
	}

	result, err := database.DB.Exec(`
		UPDATE membership_tiers
		SET is_active = FALSE
		WHERE id = $1 AND user_id = $2`, tierID, userID)
	if err != nil {
		http.Error(w, "Error archiving tier", http.StatusInternalServerError)
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		http.Error(w, "Error confirming archive", http.StatusInternalServerError)
		return
	}
	if rowsAffected == 0 {
		http.Error(w, "Tier not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func GetDashboardSettings(w http.ResponseWriter, r *http.Request) {
	userID, err := getAuthenticatedUserID(r)
	if err != nil {
		http.Error(w, "Invalid to get user from the token", http.StatusUnauthorized)
		return
	}

	var response model.DashboardSettingsResponse
	var categoryName sql.NullString
	var avatarURL sql.NullString
	var goalTitle sql.NullString
	var goalAmount sql.NullFloat64
	var newSupporter sql.NullBool
	var newMessage sql.NullBool
	var weeklyReport sql.NullBool
	var marketingEmails sql.NullBool
	var isConnected sql.NullBool
	var cardLast4 sql.NullString

	err = database.DB.QueryRow(`
		SELECT
			u.full_name,
			u.bio,
			u.username,
			u.email,
			u.avatar_url,
			COALESCE(c.name, ''),
			COALESCE(g.label, ''),
			COALESCE(g.target_amount, 0),
			COALESCE(n.new_supporter, TRUE),
			COALESCE(n.new_message, TRUE),
			COALESCE(n.weekly_report, FALSE),
			COALESCE(n.marketing_emails, FALSE),
			COALESCE(p.is_connected, FALSE),
			COALESCE(p.card_last4, '')
		FROM users u
		LEFT JOIN categories c ON c.id = u.category_id
		LEFT JOIN goals g ON g.user_id = u.id AND g.is_active = TRUE
		LEFT JOIN notification_settings n ON n.user_id = u.id
		LEFT JOIN payout_accounts p ON p.user_id = u.id AND p.provider = 'stripe'
		WHERE u.id = $1`, userID).Scan(
		&response.Profile.CreatorName,
		&response.Profile.CreatorBio,
		&response.Profile.CreatorUrl,
		&response.Profile.CreatorEmail,
		&avatarURL,
		&categoryName,
		&goalTitle,
		&goalAmount,
		&newSupporter,
		&newMessage,
		&weeklyReport,
		&marketingEmails,
		&isConnected,
		&cardLast4,
	)
	if err != nil {
		http.Error(w, "Error fetching settings", http.StatusInternalServerError)
		return
	}

	response.Profile.CreatorID = userID
	response.Profile.CreatorImage = avatarURL.String
	response.Profile.CreatorCategory = categoryName.String
	response.Goal.Title = goalTitle.String
	if goalAmount.Valid {
		response.Goal.Amount = goalAmount.Float64
	}
	if newSupporter.Valid {
		response.Notifications.NewSupporter = newSupporter.Bool
	}
	if newMessage.Valid {
		response.Notifications.NewMessage = newMessage.Bool
	}
	if weeklyReport.Valid {
		response.Notifications.WeeklyReport = weeklyReport.Bool
	}
	if marketingEmails.Valid {
		response.Notifications.MarketingEmails = marketingEmails.Bool
	}
	if isConnected.Valid {
		response.Stripe.IsConnected = isConnected.Bool
	}
	response.Stripe.CardLast4 = cardLast4.String

	response.SocialLinks = make([]string, 0)
	linksRows, err := database.DB.Query(`SELECT url FROM social_links WHERE user_id = $1 ORDER BY sort_order ASC, id ASC`, userID)
	if err == nil {
		defer linksRows.Close()
		for linksRows.Next() {
			var urlStr string
			if err := linksRows.Scan(&urlStr); err == nil {
				response.SocialLinks = append(response.SocialLinks, urlStr)
			}
		}
	}

	if accountID, connected, err := loadStripePayoutAccount(userID); err == nil && accountID != "" {
		if refreshed, err := refreshStripePayoutStatus(userID, accountID); err == nil {
			response.Stripe.IsConnected = refreshed
		} else {
			response.Stripe.IsConnected = connected
		}
	}

	writeJSON(w, http.StatusOK, response)
}

func UpdateDashboardProfile(w http.ResponseWriter, r *http.Request) {
	userID, err := getAuthenticatedUserID(r)
	if err != nil {
		http.Error(w, "Invalid to get user from the token", http.StatusUnauthorized)
		return
	}

	var req model.UpdateDashboardProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	req.Name = strings.TrimSpace(req.Name)
	req.Email = strings.TrimSpace(req.Email)
	req.Category = strings.TrimSpace(req.Category)

	if req.Name == "" || req.Email == "" {
		http.Error(w, "Name and email are required", http.StatusBadRequest)
		return
	}

	var categoryID sql.NullInt64
	if req.Category != "" {
		if err := database.DB.QueryRow(`SELECT id FROM categories WHERE name = $1`, req.Category).Scan(&categoryID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				http.Error(w, "Invalid category", http.StatusBadRequest)
				return
			}
			http.Error(w, "Error resolving category", http.StatusInternalServerError)
			return
		}
	}

	result, err := database.DB.Exec(`
		UPDATE users
		SET full_name = $1, bio = $2, email = $3, category_id = $4, updated_at = CURRENT_TIMESTAMP
		WHERE id = $5`, req.Name, req.Bio, req.Email, categoryID, userID)
	if err != nil {
		http.Error(w, "Error saving profile", http.StatusInternalServerError)
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		http.Error(w, "Error confirming profile update", http.StatusInternalServerError)
		return
	}
	if rowsAffected == 0 {
		http.Error(w, "Profile not found", http.StatusNotFound)
		return
	}

	// Sync Social Links
	_, _ = database.DB.Exec(`DELETE FROM social_links WHERE user_id = $1`, userID)
	for idx, link := range req.SocialLinks {
		link = strings.TrimSpace(link)
		if link == "" {
			continue
		}
		platform := "website"
		lowerLink := strings.ToLower(link)
		if strings.Contains(lowerLink, "twitter.com") || strings.Contains(lowerLink, "x.com") {
			platform = "twitter"
		} else if strings.Contains(lowerLink, "instagram.com") {
			platform = "instagram"
		} else if strings.Contains(lowerLink, "youtube.com") {
			platform = "youtube"
		}

		_, err = database.DB.Exec(`
			INSERT INTO social_links (user_id, platform, url, sort_order)
			VALUES ($1, $2, $3, $4)`, userID, platform, link, idx)
		if err != nil {
			log.Printf("Error inserting social link: %v", err)
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{"status": true, "message": "Profile updated successfully"})
}

func UpdateDashboardAvatar(w http.ResponseWriter, r *http.Request) {
	userID, err := getAuthenticatedUserID(r)
	if err != nil {
		http.Error(w, "Invalid to get user from the token", http.StatusUnauthorized)
		return
	}

	if err := r.ParseMultipartForm(maxAvatarBytes + (1 << 20)); err != nil {
		http.Error(w, "Invalid avatar upload", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("avatar")
	if err != nil {
		http.Error(w, "Avatar file is required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	avatarURL, _, err := saveAvatar(file, header)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	result, err := database.DB.Exec(`
		UPDATE users
		SET avatar_url = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2`, avatarURL, userID)
	if err != nil {
		http.Error(w, "Error saving avatar", http.StatusInternalServerError)
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		http.Error(w, "Error confirming avatar update", http.StatusInternalServerError)
		return
	}
	if rowsAffected == 0 {
		http.Error(w, "Profile not found", http.StatusNotFound)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":     true,
		"avatar_url": avatarURL,
	})
}

func UpdateDashboardNotifications(w http.ResponseWriter, r *http.Request) {
	userID, err := getAuthenticatedUserID(r)
	if err != nil {
		http.Error(w, "Invalid to get user from the token", http.StatusUnauthorized)
		return
	}

	var req model.UpdateDashboardNotificationsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	_, err = database.DB.Exec(`
		INSERT INTO notification_settings
			(user_id, new_supporter, new_message, weekly_report, marketing_emails)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id) DO UPDATE SET
			new_supporter = EXCLUDED.new_supporter,
			new_message = EXCLUDED.new_message,
			weekly_report = EXCLUDED.weekly_report,
			marketing_emails = EXCLUDED.marketing_emails,
			updated_at = CURRENT_TIMESTAMP`, userID, req.NewSupporter, req.NewMessage, req.WeeklyReport, req.MarketingEmails)
	if err != nil {
		http.Error(w, "Error saving notifications", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"status": true, "message": "Notification settings updated successfully"})
}

func UpdateDashboardGoal(w http.ResponseWriter, r *http.Request) {
	userID, err := getAuthenticatedUserID(r)
	if err != nil {
		http.Error(w, "Invalid to get user from the token", http.StatusUnauthorized)
		return
	}

	var req model.UpdateDashboardGoalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" || req.Amount <= 0 {
		http.Error(w, "Goal title and amount are required", http.StatusBadRequest)
		return
	}

	tx, err := database.DB.Begin()
	if err != nil {
		http.Error(w, "Error starting transaction", http.StatusInternalServerError)
		return
	}
	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()

	var goalID int64
	err = tx.QueryRow(`
		SELECT id
		FROM goals
		WHERE user_id = $1 AND is_active = TRUE
		ORDER BY id DESC
		LIMIT 1`, userID).Scan(&goalID)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "Error loading goal", http.StatusInternalServerError)
			return
		}
		if _, err := tx.Exec(`
			INSERT INTO goals (user_id, label, target_amount, current_amount, is_active)
			VALUES ($1, $2, $3, 0, TRUE)`, userID, req.Title, req.Amount); err != nil {
			http.Error(w, "Error saving goal", http.StatusInternalServerError)
			return
		}
	} else {
		if _, err := tx.Exec(`
			UPDATE goals
			SET label = $1, target_amount = $2, is_active = TRUE
			WHERE id = $3 AND user_id = $4`, req.Title, req.Amount, goalID, userID); err != nil {
			http.Error(w, "Error saving goal", http.StatusInternalServerError)
			return
		}
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, "Error saving goal", http.StatusInternalServerError)
		return
	}
	tx = nil

	writeJSON(w, http.StatusOK, map[string]any{"status": true, "message": "Goal updated successfully"})
}

func GetDashboardStripeStatus(w http.ResponseWriter, r *http.Request) {
	userID, err := getAuthenticatedUserID(r)
	if err != nil {
		http.Error(w, "Invalid to get user from the token", http.StatusUnauthorized)
		return
	}

	var response model.DashboardStripeStatus
	var connected sql.NullBool
	var cardLast4 sql.NullString

	if err := database.DB.QueryRow(`
		SELECT COALESCE(is_connected, FALSE), COALESCE(card_last4, '')
		FROM payout_accounts
		WHERE user_id = $1 AND provider = 'stripe'`, userID).Scan(&connected, &cardLast4); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, http.StatusOK, response)
			return
		}
		http.Error(w, "Error fetching payout status", http.StatusInternalServerError)
		return
	}

	if connected.Valid {
		response.IsConnected = connected.Bool
	}
	response.CardLast4 = cardLast4.String

	writeJSON(w, http.StatusOK, response)
}
