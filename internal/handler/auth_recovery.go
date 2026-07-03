package handler

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"strings"
	"time"

	"brewme/internal/database"
	"brewme/internal/middleware"
	"brewme/internal/utils"

	"github.com/resend/resend-go/v2"
)

// ForgotPasswordRequest payload
type ForgotPasswordRequest struct {
	Email string `json:"email"`
}

// ResetPasswordRequest payload
type ResetPasswordRequest struct {
	Email    string `json:"email"`
	Code     string `json:"code"`
	Password string `json:"password"`
}

func generateRecoveryCode() (string, error) {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 6)
	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		b[i] = charset[n.Int64()]
	}
	return string(b), nil
}

func ForgotPassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ForgotPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"status": false, "error": "invalid payload"})
		return
	}

	email := strings.ToLower(strings.TrimSpace(middleware.SanitizeHTML(req.Email)))
	if email == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"status": false, "error": "email required"})
		return
	}

	// 1. Check if user exists
	var exists int
	err := database.DB.QueryRow(`SELECT 1 FROM users WHERE email = $1`, email).Scan(&exists)
	if err != nil {
		if err == sql.ErrNoRows {
			// Don't leak that the email isn't registered, return success anyway
			writeJSON(w, http.StatusOK, map[string]any{"status": true, "message": "If the email exists, a code was sent"})
			return
		}
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	// 2. Generate a 6-character code
	code, err := generateRecoveryCode()
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	// 3. Store code in Redis with 15 minutes expiration
	redisKey := fmt.Sprintf("recovery:%s", email)
	err = database.Redis.Set(context.Background(), redisKey, code, 15*time.Minute).Err()
	if err != nil {
		http.Error(w, "Server error storing code", http.StatusInternalServerError)
		return
	}

	// 4. Send Email via Resend
	resendKey := os.Getenv("RESEND_API_KEY")
	if resendKey == "" {
		fmt.Println("Warning: RESEND_API_KEY is not set. Generated code is:", code)
		// For local testing without API key, still return success
	} else {
		client := resend.NewClient(resendKey)

		// Resend requires a verified domain in the From address (e.g. Acme <onboarding@resend.dev> for testing)
		fromEmail := os.Getenv("RESEND_FROM_EMAIL")
		if fromEmail == "" {
			fromEmail = "BrewMe <onboarding@resend.dev>"
		}

		htmlBody := fmt.Sprintf(`
			<div style="font-family: sans-serif; text-align: center; padding: 20px;">
				<h2>Password Reset</h2>
				<p>Your password reset code is:</p>
				<h1 style="letter-spacing: 5px; background: #f4f4f4; padding: 15px; border-radius: 8px;">%s</h1>
				<p>This code will expire in 15 minutes.</p>
				<p>If you did not request this, please ignore this email.</p>
			</div>
		`, code)

		params := &resend.SendEmailRequest{
			From:    fromEmail,
			To:      []string{email},
			Subject: "BrewMe - Password Reset Code",
			Html:    htmlBody,
		}

		_, err = client.Emails.Send(params)
		if err != nil {
			fmt.Println("Resend Error:", err)
			writeJSON(w, http.StatusInternalServerError, map[string]any{"status": false, "error": "failed to send email"})
			return
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{"status": true, "message": "Code sent successfully"})
}

func ResetPassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ResetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"status": false, "error": "invalid payload"})
		return
	}

	email := strings.ToLower(strings.TrimSpace(middleware.SanitizeHTML(req.Email)))
	code := strings.TrimSpace(req.Code)
	password := req.Password

	if email == "" || code == "" || password == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"status": false, "error": "all fields required"})
		return
	}

	if len(password) < 8 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"status": false, "error": "password must be at least 8 characters"})
		return
	}

	// 1. Verify code in Redis
	redisKey := fmt.Sprintf("recovery:%s", email)
	storedCode, err := database.Redis.Get(context.Background(), redisKey).Result()
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"status": false, "error": "invalid or expired code"})
		return
	}

	// Case-insensitive code comparison
	if !strings.EqualFold(storedCode, code) {
		writeJSON(w, http.StatusBadRequest, map[string]any{"status": false, "error": "incorrect code"})
		return
	}

	// 2. Hash new password
	hash, err := utils.HashPassword(password)
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	// 3. Update database
	res, err := database.DB.Exec(`UPDATE users SET password_hash = $1 WHERE email = $2`, hash, email)
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"status": false, "error": "user not found"})
		return
	}

	// 4. Delete recovery code & expire sessions
	database.Redis.Del(context.Background(), redisKey)
	database.Redis.Del(context.Background(), fmt.Sprintf("session:%s", email)) // force logout from all devices

	writeJSON(w, http.StatusOK, map[string]any{"status": true, "message": "Password updated successfully"})
}
