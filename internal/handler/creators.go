package handler

import (
	"brewme/internal/database"
	"brewme/internal/middleware"
	"brewme/internal/model"
	"brewme/internal/utils"
	"database/sql"
	"encoding/json"
	"errors"
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
			COALESCE(SUM(d.cups), 0) AS total_cups,
			(SELECT COUNT(*) FROM supporters WHERE user_id = u.id) AS total_supporters
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
		&profile.TotalSupporters,
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

	// Query membership tiers for the creator
	tiersQuery := `SELECT id, name, price FROM membership_tiers WHERE user_id = ? AND is_active = 1 ORDER BY sort_order ASC, id ASC`
	rowsTiers, err := database.DB.Query(tiersQuery, userID)
	if err != nil {
		log.Printf("Error querying membership tiers: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer rowsTiers.Close()

	profile.Tiers = make([]model.CreatorMembershipTier, 0)
	for rowsTiers.Next() {
		var tier model.CreatorMembershipTier
		if err := rowsTiers.Scan(&tier.ID, &tier.Name, &tier.Price); err != nil {
			log.Printf("Error scanning tier row: %v", err)
			continue
		}

		// Query perks for this tier
		perksQuery := `SELECT perk_text FROM tier_perks WHERE tier_id = ? ORDER BY sort_order ASC, id ASC`
		rowsPerks, err := database.DB.Query(perksQuery, tier.ID)
		if err != nil {
			log.Printf("Error querying perks: %v", err)
			continue
		}
		defer rowsPerks.Close()

		tier.Perks = make([]string, 0)
		for rowsPerks.Next() {
			var perk string
			if err := rowsPerks.Scan(&perk); err == nil {
				tier.Perks = append(tier.Perks, perk)
			}
		}
		profile.Tiers = append(profile.Tiers, tier)
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

func GetCreatorPublicPosts(w http.ResponseWriter, r *http.Request) {
	// 1. Extract and sanitize the username from the path
	rawUsername := chi.URLParam(r, "username")
	username := middleware.SanitizeHTML(rawUsername)

	if username == "" {
		http.Error(w, "Username is required", http.StatusBadRequest)
		return
	}

	// 2. Extract and parse the limit from the query string (e.g., ?limit=10)
	limitStr := r.URL.Query().Get("limit")
	limit := 10 // Default limit if not provided

	if limitStr != "" {
		parsedLimit, err := strconv.Atoi(limitStr)
		if err == nil && parsedLimit > 0 {
			// Cap the maximum limit to prevent excessive data loading
			if parsedLimit > 50 {
				limit = 50
			} else {
				limit = parsedLimit
			}
		}
	}

	// 3. Resolve Creator User ID
	var creatorUserID int64
	err := database.DB.QueryRow(`SELECT id FROM users WHERE username = ?`, username).Scan(&creatorUserID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "Creator not found", http.StatusNotFound)
			return
		}
		log.Printf("Error resolving creator ID: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// 4. Resolve Viewer Authentication & Membership Status
	var viewerEmail string
	var viewerUserID int64
	var isMember bool

	token, err := utils.GetTokenFromHeader(r)
	if err == nil {
		if viewerID, err := utils.GetUserIDFromToken(token); err == nil {
			viewerUserID = viewerID
			_ = database.DB.QueryRow(`SELECT email FROM users WHERE id = ?`, viewerUserID).Scan(&viewerEmail)
		}
	}

	if viewerUserID > 0 {
		if viewerUserID == creatorUserID {
			isMember = true
		} else if viewerEmail != "" {
			var count int
			err := database.DB.QueryRow(`
				SELECT COUNT(*)
				FROM memberships m
				JOIN supporters s ON m.supporter_id = s.id
				WHERE m.user_id = ? AND s.email = ? AND m.status = 'active'`,
				creatorUserID, viewerEmail,
			).Scan(&count)
			if err == nil && count > 0 {
				isMember = true
			}
		}
	}

	// 5. Query both public and member-only published posts
	query := `
		SELECT 
			p.id, 
			p.title, 
			p.preview, 
			p.likes_count, 
			p.comments_count, 
			p.published_at,
			p.visibility
		FROM posts p
		WHERE p.user_id = ? 
		  AND p.status = 'published' 
		  AND p.visibility IN ('public', 'members')
		ORDER BY p.published_at DESC
		LIMIT ?
	`

	rows, err := database.DB.Query(query, creatorUserID, limit)
	if err != nil {
		log.Printf("Error querying public posts: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	// Initialize empty slice so JSON returns [] instead of null for creators with no posts
	posts := make([]model.PublicPostItem, 0)

	for rows.Next() {
		var item model.PublicPostItem
		var nullPreview sql.NullString
		var nullPublishedAt sql.NullTime
		var visibility string

		err := rows.Scan(
			&item.ID,
			&item.Title,
			&nullPreview,
			&item.LikesCount,
			&item.CommentsCount,
			&nullPublishedAt,
			&visibility,
		)

		if err != nil {
			log.Printf("Error scanning post row: %v", err)
			continue // Skip broken rows but keep processing the rest
		}

		// Handle the nullable fields safely
		item.Preview = nullPreview.String
		if nullPublishedAt.Valid {
			item.PublishedAt = nullPublishedAt.Time
		}

		// If it's a members-only post and the viewer is not a member, mark as MembersOnly (blurred)
		if visibility == "members" {
			item.MembersOnly = !isMember
		} else {
			item.MembersOnly = false
		}

		posts = append(posts, item)
	}

	if err = rows.Err(); err != nil {
		log.Printf("Error iterating post rows: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// 6. Return the JSON response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(posts); err != nil {
		log.Printf("Error encoding posts response: %v", err)
	}
}
