package handler

import (
	"brewme/internal/database"
	"brewme/internal/model"
	"brewme/internal/utils"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
)

func DashboardStats(w http.ResponseWriter, r *http.Request) {
	// 1. Get the authenticated user's ID.
	token, err := utils.GetTokenFromHeader(r)
	if err != nil {
		http.Error(w, "Invalid to get token", http.StatusUnauthorized)
		return
	}

	// FIX 1: Changed user_id to userID to match the variable used in queries
	userID, err := utils.GetUserIDFromToken(token)
	if err != nil {
		http.Error(w, "Invalid to get user from the token", http.StatusUnauthorized)
		return
	}

	var stats model.DashboardStats

	// --- 1. Fetch Total Earnings & Supporters (Using the 'creator_earnings' View) ---
	var totalEarned float64
	var totalSupporters int64

	// FIX 2: Changed := to = because err is already declared above
	err = database.DB.QueryRow(`
		SELECT total_earned, supporter_count 
		FROM creator_earnings 
		WHERE user_id = $1`,
		userID,
	).Scan(&totalEarned, &totalSupporters)

	if err != nil && err != sql.ErrNoRows {
		http.Error(w, "Error fetching totals", http.StatusInternalServerError)
		log.Println("Database error (Totals):", err)
		return
	}

	// --- 2. Fetch Total Published Posts ---
	var totalPosts int64
	err = database.DB.QueryRow(`
		SELECT COUNT(*) 
		FROM posts 
		WHERE user_id = $1 AND status = 'published'`,
		userID,
	).Scan(&totalPosts)

	if err != nil {
		http.Error(w, "Error fetching posts", http.StatusInternalServerError)
		log.Println("Database error (Posts):", err)
		return
	}

	// --- 3. Fetch Current Month's Earnings (Donations + Memberships) ---
	var monthlyEarned float64
	err = database.DB.QueryRow(`
		SELECT 
			COALESCE((SELECT SUM(amount) FROM donations WHERE user_id = $1 AND status = 'succeeded' AND EXTRACT(MONTH FROM created_at) = EXTRACT(MONTH FROM CURRENT_DATE) AND EXTRACT(YEAR FROM created_at) = EXTRACT(YEAR FROM CURRENT_DATE)), 0) +
			COALESCE((SELECT SUM(amount) FROM memberships WHERE user_id = $2 AND status = 'active' AND EXTRACT(MONTH FROM started_at) = EXTRACT(MONTH FROM CURRENT_DATE) AND EXTRACT(YEAR FROM started_at) = EXTRACT(YEAR FROM CURRENT_DATE)), 0)
		`, userID, userID).Scan(&monthlyEarned)

	if err != nil {
		http.Error(w, "Error fetching monthly earnings", http.StatusInternalServerError)
		log.Println("Database error (Monthly):", err)
		return
	}

	// --- 4. Fetch Chart Data (Earnings by Month for the current year) ---
	// Note: := is valid here because 'rows' is a completely new variable
	rows, err := database.DB.Query(`
		SELECT 
			TO_CHAR(date_col, 'Mon') as month_name,
			SUM(amount) as monthly_total
		FROM (
			SELECT created_at as date_col, amount FROM donations WHERE user_id = $1 AND status = 'succeeded' AND EXTRACT(YEAR FROM created_at) = EXTRACT(YEAR FROM CURRENT_DATE)
			UNION ALL
			SELECT started_at as date_col, amount FROM memberships WHERE user_id = $2 AND status = 'active' AND EXTRACT(YEAR FROM started_at) = EXTRACT(YEAR FROM CURRENT_DATE)
		) as combined_earnings
		GROUP BY EXTRACT(MONTH FROM date_col), month_name
		ORDER BY EXTRACT(MONTH FROM date_col) ASC`,
		userID, userID,
	)

	if err != nil {
		http.Error(w, "Error fetching chart data", http.StatusInternalServerError)
		log.Println("Database error (Chart):", err)
		return
	}
	defer rows.Close()

	var chartData []model.ChartEntry
	for rows.Next() {
		var entry model.ChartEntry
		if err := rows.Scan(&entry.Month, &entry.Earnings); err != nil {
			continue // Skip problematic rows or handle the error more strictly if needed
		}
		chartData = append(chartData, entry)
	}

	// Check for errors encountered during iteration
	if err = rows.Err(); err != nil {
		http.Error(w, "Error reading chart data", http.StatusInternalServerError)
		log.Println("Database error (Chart Rows):", err)
		return
	}

	// --- 5. Calculate Deltas / Dummy Data for Changes ---
	// Note: For real percentage changes, you would calculate the difference between the current month and the previous month.
	// These are now native numbers instead of formatted strings.
	earningsChange := 12.5
	monthlyChange := 8.3
	supportersChange := int64(23)
	postsChange := int64(3)

	// --- 6. Direct Assignment ---
	stats.TotalEarned = totalEarned
	stats.EarningsChange = earningsChange
	stats.MonthlyEarned = monthlyEarned
	stats.MonthlyChange = monthlyChange
	stats.TotalSupporters = totalSupporters
	stats.SupportersChange = supportersChange
	stats.TotalPosts = totalPosts
	stats.PostsChange = postsChange

	// Ensure chartData is at least an empty slice `[]` instead of `null` in JSON if there are no records
	if chartData == nil {
		chartData = make([]model.ChartEntry, 0)
	}
	stats.ChartData = chartData

	// --- 7. Send JSON Response ---
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(stats); err != nil {
		log.Println("JSON encoding error:", err)
	}
}

func LatestSupporters(w http.ResponseWriter, r *http.Request) {
	// 1. Authenticate and get User ID
	token, err := utils.GetTokenFromHeader(r)
	if err != nil {
		http.Error(w, "Invalid to get token", http.StatusUnauthorized)
		return
	}
	userID, err := utils.GetUserIDFromToken(token)
	if err != nil {
		http.Error(w, "Invalid to get user from the token", http.StatusUnauthorized)
		return
	}

	// 2. Parse the 'limit' query parameter (default to 5 if not provided or invalid)
	limit := 5
	limitParam := r.URL.Query().Get("limit")
	if limitParam != "" {
		if parsedLimit, err := strconv.Atoi(limitParam); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	// 3. Query the `supporter_feed` View
	// This view safely combines donations and memberships and standardizes their columns
	rows, err := database.DB.Query(`
		SELECT 
			display_name, 
			message, 
			amount, 
			cups, 
			created_at, 
			support_type, 
			replied
		FROM supporter_feed 
		WHERE user_id = $1 
		ORDER BY created_at DESC 
		LIMIT $2`,
		userID, limit,
	)

	if err != nil {
		http.Error(w, "Error fetching recent supporters", http.StatusInternalServerError)
		log.Println("Database error (LatestSupporters):", err)
		return
	}
	defer rows.Close()

	// Initialize as an empty slice so it encodes to `[]` instead of `null` if empty
	supporters := make([]model.RecentSupporter, 0)

	for rows.Next() {
		var supporter model.RecentSupporter

		// Because the message column can be NULL in the database (e.g. memberships have no message),
		// we must scan it into a sql.NullString first to prevent panics.
		var nullMessage sql.NullString

		err := rows.Scan(
			&supporter.SupporterName,
			&nullMessage,
			&supporter.TotalAmount,
			&supporter.SupporterCups,
			&supporter.CreatedAt,
			&supporter.SupportType,
			&supporter.SupportReplied,
		)

		if err != nil {
			log.Println("Error scanning supporter row:", err)
			continue
		}

		// Convert sql.NullString to standard Go string
		// (If it was NULL in DB, .String will safely be an empty string "")
		supporter.SupporterMessage = nullMessage.String

		supporters = append(supporters, supporter)
	}

	if err = rows.Err(); err != nil {
		http.Error(w, "Error reading rows", http.StatusInternalServerError)
		log.Println("Database iteration error:", err)
		return
	}

	// 4. Send JSON Response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(supporters); err != nil {
		log.Println("JSON encoding error:", err)
	}
}
