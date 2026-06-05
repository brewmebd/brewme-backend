package handler

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"brewme/internal/database"
	"brewme/internal/middleware"
	"brewme/internal/model"
	"brewme/internal/utils"

	"github.com/go-sql-driver/mysql"
)

const (
	avatarUploadDir = "uploads/avatar"
	maxAvatarBytes  = 2 << 20 // 2MB
)

// allowedAvatarTypes maps an accepted image content-type to its file extension.
var allowedAvatarTypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/gif":  ".gif",
}

// Register handles POST /api/v1/auth/register.
//
// The request is multipart/form-data so it can carry the avatar file. Expected
// fields (matching the SignUp form): name, email, password, confirmPassword,
// url (page username), bio, category (slug), avatar (optional image file).
func Register(w http.ResponseWriter, r *http.Request) {
	// Parse the multipart body (a little headroom above the 2MB image cap).
	if err := r.ParseMultipartForm(maxAvatarBytes + (1 << 20)); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	fullName := middleware.SanitizeHTML(r.FormValue("name"))
	email := strings.ToLower(strings.TrimSpace(r.FormValue("email")))
	password := r.FormValue("password")
	confirm := r.FormValue("confirmPassword")
	username := strings.TrimSpace(middleware.SanitizeHTML(r.FormValue("url")))
	bio := middleware.SanitizeHTML(r.FormValue("bio"))
	categorySlug := strings.TrimSpace(r.FormValue("category"))

	// Validate input.
	fields := map[string]string{}
	if fullName == "" {
		fields["name"] = "full name is required"
	}
	if username == "" {
		fields["url"] = "page URL is required"
	}
	if email == "" {
		fields["email"] = "email is required"
	}
	if len(password) < 8 {
		fields["password"] = "password must be at least 8 characters"
	}
	if password != confirm {
		fields["confirmPassword"] = "passwords do not match"
	}
	if len(fields) > 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"status": false,
			"error":  "validation_error",
			"fields": fields,
		})
		return
	}

	// Resolve the category slug to its id (NULL when not provided).
	var categoryID sql.NullInt64
	if categorySlug != "" {
		var id int64
		err := database.DB.QueryRow(`SELECT id FROM categories WHERE slug = ?`, categorySlug).Scan(&id)
		switch {
		case errors.Is(err, sql.ErrNoRows):
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"status": false,
				"error":  "invalid_category",
			})
			return
		case err != nil:
			http.Error(w, "Server error", http.StatusInternalServerError)
			return
		}
		categoryID = sql.NullInt64{Int64: id, Valid: true}
	}

	// Email uniqueness pre-check. ErrNoRows means the email is available.
	var exists int
	err := database.DB.QueryRow(`SELECT 1 FROM users WHERE email = ?`, email).Scan(&exists)
	switch {
	case err == nil:
		writeJSON(w, http.StatusConflict, map[string]any{
			"status": false,
			"error":  "email already exists",
		})
		return
	case !errors.Is(err, sql.ErrNoRows):
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	// Hash the password.
	passwordHash, err := utils.HashPassword(password)
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	// Handle the optional avatar file.
	var avatarURL sql.NullString
	savedAvatarPath := ""
	if file, header, ferr := r.FormFile("avatar"); ferr == nil {
		defer file.Close()
		url, path, serr := saveAvatar(file, header)
		if serr != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"status": false,
				"error":  serr.Error(),
			})
			return
		}
		avatarURL = sql.NullString{String: url, Valid: true}
		savedAvatarPath = path
	} else if !errors.Is(ferr, http.ErrMissingFile) {
		http.Error(w, "Invalid avatar upload", http.StatusBadRequest)
		return
	}

	// Insert the user (avatar_url set in the same insert).
	const q = `INSERT INTO users (full_name, username, email, password_hash, bio, category_id, avatar_url)
	           VALUES (?, ?, ?, ?, ?, ?, ?)`
	res, err := database.DB.Exec(q, fullName, username, email, passwordHash,
		sql.NullString{String: bio, Valid: bio != ""}, categoryID, avatarURL)
	if err != nil {
		// Roll back the saved file if the row could not be created.
		if savedAvatarPath != "" {
			_ = os.Remove(savedAvatarPath)
		}
		var myErr *mysql.MySQLError
		if errors.As(err, &myErr) && myErr.Number == 1062 {
			writeJSON(w, http.StatusConflict, map[string]any{
				"status": false,
				"error":  "email or username already exists",
			})
			return
		}
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
	id, _ := res.LastInsertId()

	writeJSON(w, http.StatusCreated, map[string]any{
		"status":  true,
		"message": "User registration successful",
		"user": map[string]any{
			"id":         id,
			"full_name":  fullName,
			"username":   username,
			"email":      email,
			"avatar_url": avatarURL.String,
		},
	})
}

// saveAvatar validates and stores an uploaded avatar, returning the public URL
// path and the on-disk path (for cleanup on failure).
func saveAvatar(file multipart.File, header *multipart.FileHeader) (url, path string, err error) {
	if header.Size > maxAvatarBytes {
		return "", "", errors.New("avatar exceeds 2MB limit")
	}

	// Sniff the real content type from the first 512 bytes.
	head := make([]byte, 512)
	n, _ := file.Read(head)
	ext, ok := allowedAvatarTypes[http.DetectContentType(head[:n])]
	if !ok {
		return "", "", errors.New("avatar must be a JPG, PNG, or GIF image")
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", "", err
	}

	if err := os.MkdirAll(avatarUploadDir, 0o755); err != nil {
		return "", "", err
	}
	name, err := randomName()
	if err != nil {
		return "", "", err
	}
	filename := name + ext
	fullPath := filepath.Join(avatarUploadDir, filename)

	dst, err := os.Create(fullPath)
	if err != nil {
		return "", "", err
	}
	defer dst.Close()
	if _, err := io.Copy(dst, file); err != nil {
		_ = os.Remove(fullPath)
		return "", "", err
	}

	// URL served by the static file route registered in the router.
	return "/" + avatarUploadDir + "/" + filename, fullPath, nil
}

// randomName returns a 32-char hex string for a collision-free filename.
func randomName() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req model.LoginRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid json", http.StatusBadRequest)
		return
	}

	email := middleware.SanitizeHTML(req.Email)
	password := req.Password

	var password_hash string
	var user_id int64
	query := `SELECT id, password_hash FROM users WHERE email = ?`
	err = database.DB.QueryRow(query, email).Scan(&user_id, &password_hash)
	if err != nil {
		http.Error(w, "Invalid database query", http.StatusInternalServerError)
		fmt.Println(err)
		return
	}

	if !utils.CheckPasswordHash(password, password_hash) {
		http.Error(w, "Invalid credential", http.StatusUnauthorized)
		return
	}

	token, err := utils.CreateToken(email, user_id)
	if err != nil {
		http.Error(w, "Failed to create token", http.StatusInternalServerError)
		return
	}

	// Session key by email (consistent with the JWT claims)
	sessionKey := fmt.Sprintf("session:%s", email)

	// Overwrite any existing session (single session per user)
	database.Redis.Del(r.Context(), sessionKey)

	sessionData, _ := json.Marshal(map[string]interface{}{
		"token":      token,
		"email":      email,
		"user_id":    user_id,
		"created_at": time.Now().Unix(),
		"ip":         r.RemoteAddr,
	})

	if err := database.Redis.Set(r.Context(), sessionKey, sessionData, 24*time.Hour).Err(); err != nil {
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Login successful",
		"token":   token,
	})
}

func UserLogout(w http.ResponseWriter, r *http.Request) {
	// ✅ Use "email" — matches what SessionMiddleware sets in context
	email, ok := r.Context().Value("email").(string)
	if !ok || email == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	sessionKey := fmt.Sprintf("session:%s", email)
	database.Redis.Del(r.Context(), sessionKey)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Logged out successfully",
	})
}
