package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"brewme/internal/database"
	"brewme/internal/middleware"

	"github.com/resend/resend-go/v2"
)

type RequestEmailChangeRequest struct {
	NewEmail string `json:"new_email"`
}

type VerifyEmailChangeRequest struct {
	Code string `json:"code"`
}

func RequestEmailChange(w http.ResponseWriter, r *http.Request) {
	userID, err := getAuthenticatedUserID(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req RequestEmailChangeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"status": false, "error": "invalid payload"})
		return
	}

	newEmail := strings.ToLower(strings.TrimSpace(middleware.SanitizeHTML(req.NewEmail)))
	if newEmail == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"status": false, "error": "new email is required"})
		return
	}

	// Get current email
	var currentEmail string
	err = database.DB.QueryRow(`SELECT email FROM users WHERE id = $1`, userID).Scan(&currentEmail)
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	if newEmail == currentEmail {
		writeJSON(w, http.StatusBadRequest, map[string]any{"status": false, "error": "new email must be different"})
		return
	}

	// Check if new email is already taken
	var exists int
	err = database.DB.QueryRow(`SELECT 1 FROM users WHERE email = $1`, newEmail).Scan(&exists)
	if err == nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"status": false, "error": "email is already in use"})
		return
	}

	code, err := generateRecoveryCode() // reusing from auth_recovery.go
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	// Store in Redis: key: email_change:{userID}, value: {newEmail}:{code}
	redisKey := fmt.Sprintf("email_change:%d", userID)
	redisValue := fmt.Sprintf("%s:%s", newEmail, code)
	err = database.Redis.Set(context.Background(), redisKey, redisValue, 15*time.Minute).Err()
	if err != nil {
		http.Error(w, "Server error storing code", http.StatusInternalServerError)
		return
	}

	resendKey := os.Getenv("RESEND_API_KEY")
	if resendKey != "" {
		client := resend.NewClient(resendKey)
		fromEmail := os.Getenv("RESEND_FROM_EMAIL")
		if fromEmail == "" {
			fromEmail = "BrewMe <onboarding@resend.dev>"
		}

		// 1. Send code to NEW email
		newEmailHtml := fmt.Sprintf(`
			<div style="font-family: sans-serif; text-align: center; padding: 20px;">
				<h2>Verify your new email</h2>
				<p>Your verification code is:</p>
				<h1 style="letter-spacing: 5px; background: #f4f4f4; padding: 15px; border-radius: 8px;">%s</h1>
				<p>This code will expire in 15 minutes.</p>
			</div>
		`, code)
		_, errNew := client.Emails.Send(&resend.SendEmailRequest{
			From:    fromEmail,
			To:      []string{newEmail},
			Subject: "BrewMe - Verify your new email address",
			Html:    newEmailHtml,
		})
		if errNew != nil {
			fmt.Printf("Resend failed to send to NEW email (%s): %v\n", newEmail, errNew)
			fmt.Printf("Fallback: Verification code for %s is %s\n", newEmail, code)
		}

		// 2. Send alert to OLD email
		oldEmailHtml := fmt.Sprintf(`
			<div style="font-family: sans-serif; padding: 20px;">
				<h2>Security Alert: Email Change Requested</h2>
				<p>Someone (hopefully you) requested to change the email address associated with your BrewMe account to <b>%s</b>.</p>
				<p>If this was you, you can safely ignore this email.</p>
				<p style="color: red; font-weight: bold;">If you did NOT request this, please change your password immediately and contact support.</p>
			</div>
		`, newEmail)
		_, errOld := client.Emails.Send(&resend.SendEmailRequest{
			From:    fromEmail,
			To:      []string{currentEmail},
			Subject: "BrewMe Security - Email Change Requested",
			Html:    oldEmailHtml,
		})
		if errOld != nil {
			fmt.Printf("Resend failed to send alert to OLD email (%s): %v\n", currentEmail, errOld)
		}
	} else {
		fmt.Printf("Warning: RESEND_API_KEY missing. Email change code for %s is %s\n", newEmail, code)
	}

	writeJSON(w, http.StatusOK, map[string]any{"status": true, "message": "Verification code sent to new email"})
}

func VerifyEmailChange(w http.ResponseWriter, r *http.Request) {
	userID, err := getAuthenticatedUserID(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req VerifyEmailChangeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"status": false, "error": "invalid payload"})
		return
	}

	code := strings.TrimSpace(req.Code)
	if code == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"status": false, "error": "code is required"})
		return
	}

	redisKey := fmt.Sprintf("email_change:%d", userID)
	storedValue, err := database.Redis.Get(context.Background(), redisKey).Result()
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"status": false, "error": "invalid or expired code"})
		return
	}

	parts := strings.SplitN(storedValue, ":", 2)
	if len(parts) != 2 {
		http.Error(w, "Server error reading session", http.StatusInternalServerError)
		return
	}
	newEmail, storedCode := parts[0], parts[1]

	if !strings.EqualFold(storedCode, code) {
		writeJSON(w, http.StatusBadRequest, map[string]any{"status": false, "error": "incorrect code"})
		return
	}

	// Update the email in the DB
	res, err := database.DB.Exec(`UPDATE users SET email = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2`, newEmail, userID)
	if err != nil {
		http.Error(w, "Server error updating email", http.StatusInternalServerError)
		return
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"status": false, "error": "user not found"})
		return
	}

	database.Redis.Del(context.Background(), redisKey)

	writeJSON(w, http.StatusOK, map[string]any{"status": true, "message": "Email updated successfully", "new_email": newEmail})
}
