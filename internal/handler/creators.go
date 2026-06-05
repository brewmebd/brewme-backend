package handler

import (
	"brewme/internal/database"
	"brewme/internal/middleware"
	"brewme/internal/model"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi"
)

func GetCreatorProfile(w http.ResponseWriter, r *http.Request) {
	// 1. Extract the username from the URL path using chi
	rawUsername := chi.URLParam(r, "username")

	// Sanitize it just like you had before
	username := middleware.SanitizeHTML(rawUsername)

	if username == "" {
		http.Error(w, "Username is required", http.StatusBadRequest)
		return
	}

	var userID int64
	var profile model.GetCreatorProfile

	// Use NullString for columns that can be NULL in the database
	var avatarURL sql.NullString
	var categoryName sql.NullString
	var bio sql.NullString

	query := `SELECT 
			u.id, 
			u.avatar_url, 
			u.full_name, 
			c.name, 
			u.bio, 
			COALESCE(SUM(d.cups), 0) AS total_cups
		FROM users u
		LEFT JOIN categories c ON u.category_id = c.id
		LEFT JOIN donations d ON u.id = d.user_id AND d.status = 'succeeded'
		WHERE u.username = ?
		GROUP BY u.id, c.name`

	err := database.DB.QueryRow(query, username).Scan(
		&userID,
		&avatarURL,
		&profile.CreatorName,
		&categoryName,
		&bio,
		&profile.TotalCups,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Creator not found", http.StatusNotFound)
			return
		}
		log.Printf("Error querying user profile: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Safely assign the nullable strings to the profile struct
	profile.CreatorImage = avatarURL.String
	profile.CreatorCategory = categoryName.String
	profile.CreatorBio = bio.String

	links_query := `SELECT platform, url FROM social_links WHERE user_id = ? ORDER BY sort_order ASC`

	rows, err := database.DB.Query(links_query, userID)
	if err != nil {
		log.Printf("Error querying social links: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	profile.CreatorLinks = make([]string, 0)

	for rows.Next() {
		var platform, link string
		// Scanning into both platform and link
		if err := rows.Scan(&platform, &link); err == nil {
			profile.CreatorLinks = append(profile.CreatorLinks, link)
		} else {
			log.Printf("Error scanning row: %v", err)
		}
	}

	if err := rows.Err(); err != nil {
		log.Printf("Error iterating over social links: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(profile); err != nil {
		log.Printf("Error encoding profile response: %v", err)
	}
}

func GetSupportersFeed(w http.ResponseWriter, r *http.Request) {
	rawUsername := chi.URLParam(r, "username")
	username := middleware.SanitizeHTML(rawUsername)

	if username == "" {
		http.Error(w, "Username is required", http.StatusBadRequest)
		return
	}

	limitStr := r.URL.Query().Get("limit")
	limit := 20 // Default limit

	if limitStr != "" {
		parsedLimit, err := strconv.Atoi(limitStr)
		if err == nil && parsedLimit > 0 {
			// Cap the maximum limit to prevent massive database queries (e.g., 100)
			if parsedLimit > 100 {
				limit = 100
			} else {
				limit = parsedLimit
			}
		}
	}

	feed := make([]model.GetSupportersFeed, 0)

	// ADDED LIMIT ? HERE
	// We replaced sf.* with the explicit 7 columns matching your Scan()
	query := `SELECT 
				sf.support_type, 
				sf.display_name, 
				sf.message, 
				sf.cups, 
				sf.amount, 
				sf.replied, 
				sf.created_at 
			  FROM supporter_feed sf
			  JOIN users u ON sf.user_id = u.id
			  WHERE u.username = ? 
			  ORDER BY sf.created_at DESC
			  LIMIT ?;`

	// PASSED THE limit VARIABLE HERE
	rows, err := database.DB.Query(query, username, limit)
	if err != nil {
		log.Printf("Error querying supporter feed: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var item model.GetSupportersFeed
		var nullMessage sql.NullString
		err := rows.Scan(
			&item.SupportType,
			&item.SupporterName,
			&nullMessage,
			&item.SupporterCups,
			&item.TotalAmounts,
			&item.SupporterReplied,
			&item.CreatedAt,
		)
		if err != nil {
			log.Printf("Error scanning feed row: %v", err)
			continue // Skip broken rows but keep processing the rest
		}

		// Assign the parsed string (it will be "" if the SQL value was NULL)
		item.SupporterMessage = nullMessage.String

		feed = append(feed, item)
	}

	if err = rows.Err(); err != nil {
		log.Printf("Error iterating feed rows: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(feed); err != nil {
		log.Printf("Error encoding feed response: %v", err)
	}
}
