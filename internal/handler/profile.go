package handler

import (
	"brewme/internal/database"
	"brewme/internal/model"
	"brewme/internal/utils"
	"encoding/json"
	"net/http"
)

func GetUserProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid method", http.StatusMethodNotAllowed)
		return
	}

	token, err := utils.GetTokenFromHeader(r)
	if err != nil {
		http.Error(w, "Invalid to get token", http.StatusUnauthorized)
		return
	}

	user_id, err := utils.GetUserIDFromToken(token)
	if err != nil {
		http.Error(w, "Invalid to get user from the token", http.StatusUnauthorized)
		return
	}

	var info model.CreatorProfile
	query := `SELECT id, avatar_url, full_name, bio, username, email FROM users WHERE id = $1`
	err = database.DB.QueryRow(query, user_id).Scan(&info.CreatorID, &info.CreatorImage, &info.CreatorName, &info.CreatorBio, &info.CreatorUrl, &info.CreatorEmail)
	if err != nil {
		http.Error(w, "Invalid database query", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":       true,
		"profile_info": info,
	})
}

func isUsernameTaken(username string) bool {
	var exists bool
	query := "SELECT EXISTS(SELECT 1 FROM users WHERE username = $1)"
	_ = database.DB.QueryRow(query, username).Scan(&exists)
	return exists
}

func CheckUsername(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid Method", http.StatusMethodNotAllowed)
		return
	}

	username := r.URL.Query().Get("username")

	if username == "" {
		http.Error(w, "Username is required", http.StatusBadRequest)
		return
	}

	available := !isUsernameTaken(username)

	response := map[string]interface{}{
		"username":  username,
		"available": available,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
