package handler

import (
	"brewme/internal/database"
	"brewme/internal/model"
	"brewme/internal/utils"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi"
)

func SubmitReply(w http.ResponseWriter, r *http.Request) {
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

	// 2. Extract the donation ID from the URL path using Chi
	idStr := chi.URLParam(r, "id")
	if idStr == "" {
		http.Error(w, "Supporter ID is required", http.StatusBadRequest)
		return
	}

	supporterID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid Supporter ID format", http.StatusBadRequest)
		return
	}

	// 2b. Determine which kind of supporter we are replying to. Donations and
	// memberships live in separate tables with independent id sequences, so the
	// same id can exist in both — the type disambiguates which row to update.
	// Defaults to "coffee" (donations) for backwards compatibility.
	table := "donations"
	switch r.URL.Query().Get("type") {
	case "", "coffee":
		table = "donations"
	case "membership":
		table = "memberships"
	default:
		http.Error(w, "Invalid supporter type", http.StatusBadRequest)
		return
	}

	// 3. Decode the incoming JSON Request Body
	var reqBody model.ReplyRequest
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// 4. Validate the input
	if reqBody.Message == "" {
		http.Error(w, "Reply message cannot be empty", http.StatusBadRequest)
		return
	}

	// 5. Update the Database
	// We ensure `user_id = ?` to guarantee a creator can only reply to their own
	// supporters. `table` is chosen from a fixed whitelist above, never user text,
	// so interpolating it here is safe from SQL injection.
	result, err := database.DB.Exec(fmt.Sprintf(`
		UPDATE %s
		SET reply_message = $1, replied_at = CURRENT_TIMESTAMP
		WHERE id = $2 AND user_id = $3`, table),
		reqBody.Message, supporterID, userID,
	)

	if err != nil {
		http.Error(w, "Error saving reply", http.StatusInternalServerError)
		log.Println("Database error (SubmitReply):", err)
		return
	}

	// 6. Verify the record was actually updated
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		http.Error(w, "Error confirming update", http.StatusInternalServerError)
		return
	}

	if rowsAffected == 0 {
		// If 0 rows were affected, either the ID doesn't exist, or it doesn't belong to this user
		http.Error(w, "Supporter record not found or unauthorized", http.StatusNotFound)
		return
	}

	// 7. Prepare and send the Success Response
	response := model.ReplyResponse{
		Status:       true,
		Message:      "Reply sent successfully",
		CreatorReply: reqBody.Message,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Println("JSON encoding error:", err)
	}
}

func GetSupportersList(w http.ResponseWriter, r *http.Request) {
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

	// 2. Parse the 'limit' query parameter (default to 50)
	limit := 50
	limitParam := r.URL.Query().Get("limit")
	if limitParam != "" {
		if parsedLimit, err := strconv.Atoi(limitParam); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	// 3. Query the Database
	// We use a custom UNION ALL query because the default 'supporter_feed' view
	// does not expose the raw 'id' or the actual 'reply_message' text.
	query := `
		SELECT 
			id,
			COALESCE(display_name, 'Anonymous') as supporter_name,
			message as supporter_message,
			amount as total_amount,
			cups as supporter_cups,
			created_at,
			'coffee' as support_type,
			(reply_message IS NOT NULL) as support_replied,
			reply_message as creator_reply
		FROM donations
		WHERE user_id = $1 AND status = 'succeeded'
		
		UNION ALL
		
		SELECT 
			id,
			COALESCE(display_name, 'Anonymous') as supporter_name,
			NULL as supporter_message,
			amount as total_amount,
			0 as supporter_cups,
			started_at as created_at,
			'membership' as support_type,
			(reply_message IS NOT NULL) as support_replied,
			reply_message as creator_reply
		FROM memberships
		WHERE user_id = $2 AND status = 'active'
		
		ORDER BY created_at DESC
		LIMIT $3
	`

	// Notice we pass userID twice (once for donations, once for memberships)
	rows, err := database.DB.Query(query, userID, userID, limit)
	if err != nil {
		http.Error(w, "Error fetching supporters list", http.StatusInternalServerError)
		log.Println("Database error (GetSupportersList):", err)
		return
	}
	defer rows.Close()

	// Initialize as an empty slice so it encodes to `[]` instead of `null` if empty
	supporters := make([]model.SupporterItem, 0)

	for rows.Next() {
		var item model.SupporterItem

		// Use sql.NullString for columns that might be NULL in the DB
		var nullMessage, nullReply sql.NullString

		err := rows.Scan(
			&item.ID,
			&item.SupporterName,
			&nullMessage,
			&item.TotalAmount,
			&item.SupporterCups,
			&item.CreatedAt,
			&item.SupportType,
			&item.SupportReplied,
			&nullReply,
		)

		if err != nil {
			log.Println("Error scanning supporter row:", err)
			continue
		}

		// Map the sql.NullString to our *string pointers in the struct
		// If valid, assign the memory address of the string. Otherwise, leave it as nil (which encodes as null)
		if nullMessage.Valid {
			msg := nullMessage.String
			item.SupporterMessage = &msg
		}

		if nullReply.Valid {
			reply := nullReply.String
			item.CreatorReply = &reply
		}

		supporters = append(supporters, item)
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
