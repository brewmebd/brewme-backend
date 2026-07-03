package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"brewme/internal/database"
	"brewme/internal/model"

	stripe "github.com/stripe/stripe-go/v81"
	stripeaccount "github.com/stripe/stripe-go/v81/account"
	stripeaccountlink "github.com/stripe/stripe-go/v81/accountlink"
	stripeloginlink "github.com/stripe/stripe-go/v81/loginlink"
)

func getStripeReturnURL() string {
	if value := strings.TrimSpace(os.Getenv("STRIPE_CONNECT_RETURN_URL")); value != "" {
		return value
	}
	return "http://localhost:5173/dashboard/settings"
}

func getStripeRefreshURL() string {
	if value := strings.TrimSpace(os.Getenv("STRIPE_CONNECT_REFRESH_URL")); value != "" {
		return value
	}
	return "http://localhost:5173/dashboard/settings"
}

func getStripeSecretKey() string {
	return strings.TrimSpace(os.Getenv("STRIPE_SECRET_KEY"))
}

func loadStripePayoutAccount(userID int64) (accountID string, connected bool, err error) {
	err = database.DB.QueryRow(`
		SELECT COALESCE(external_account_id, ''), COALESCE(is_connected, FALSE)
		FROM payout_accounts
		WHERE user_id = $1 AND provider = 'stripe'`, userID).Scan(&accountID, &connected)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", false, nil
		}
		return "", false, err
	}
	return accountID, connected, nil
}

func upsertStripePayoutAccount(userID int64, accountID string, connected bool) error {
	_, err := database.DB.Exec(`
		INSERT INTO payout_accounts (user_id, provider, external_account_id, is_connected)
		VALUES ($1, 'stripe', $2, $3)
		ON CONFLICT (user_id, provider) DO UPDATE SET
			external_account_id = EXCLUDED.external_account_id,
			is_connected = EXCLUDED.is_connected`, userID, accountID, connected)
	return err
}

func refreshStripePayoutStatus(userID int64, accountID string) (bool, error) {
	secretKey := getStripeSecretKey()
	if secretKey == "" || accountID == "" {
		return false, nil
	}

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("https://api.stripe.com/v1/accounts/%s", accountID), nil)
	if err != nil {
		return false, err
	}
	req.SetBasicAuth(secretKey, "")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return false, fmt.Errorf("stripe account lookup failed: %s", strings.TrimSpace(string(body)))
	}

	var payload struct {
		DetailsSubmitted bool `json:"details_submitted"`
		PayoutsEnabled   bool `json:"payouts_enabled"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return false, err
	}

	connected := payload.DetailsSubmitted && payload.PayoutsEnabled
	if _, err := database.DB.Exec(`
		UPDATE payout_accounts
		SET is_connected = $1
		WHERE user_id = $2 AND provider = 'stripe'`, connected, userID); err != nil {
		return false, err
	}

	return connected, nil
}

// CreateStripeConnectLink creates or refreshes the creator's Stripe Express onboarding flow.
func CreateStripeConnectLink(w http.ResponseWriter, r *http.Request) {
	userID, err := getAuthenticatedUserID(r)
	if err != nil {
		http.Error(w, "Invalid to get user from the token", http.StatusUnauthorized)
		return
	}

	secretKey := getStripeSecretKey()
	if secretKey == "" {
		http.Error(w, "Stripe is not configured", http.StatusServiceUnavailable)
		return
	}

	var fullName, email string
	if err := database.DB.QueryRow(`SELECT full_name, email FROM users WHERE id = $1`, userID).Scan(&fullName, &email); err != nil {
		http.Error(w, "Error loading creator profile", http.StatusInternalServerError)
		return
	}

	accountID, connected, err := loadStripePayoutAccount(userID)
	if err != nil {
		http.Error(w, "Error loading payout account", http.StatusInternalServerError)
		return
	}

	stripe.Key = secretKey
	if accountID == "" {
		account, err := stripeaccount.New(&stripe.AccountParams{
			Type:    stripe.String("express"),
			Country: stripe.String("US"),
			Email:   stripe.String(email),
		})
		if err != nil {
			http.Error(w, "Error creating Stripe account", http.StatusBadGateway)
			return
		}

		accountID = account.ID
		connected = false
		if err := upsertStripePayoutAccount(userID, accountID, connected); err != nil {
			http.Error(w, "Error saving payout account", http.StatusInternalServerError)
			return
		}
	} else {
		if refreshed, err := refreshStripePayoutStatus(userID, accountID); err == nil {
			connected = refreshed
		}
	}

	if connected {
		link, err := stripeloginlink.New(&stripe.LoginLinkParams{
			Account: stripe.String(accountID),
		})
		if err != nil {
			http.Error(w, "Error creating Stripe login link", http.StatusBadGateway)
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"status":     true,
			"account_id": accountID,
			"connected":  connected,
			"url":        link.URL,
		})
		return
	}

	link, err := stripeaccountlink.New(&stripe.AccountLinkParams{
		Account:    stripe.String(accountID),
		RefreshURL: stripe.String(getStripeRefreshURL()),
		ReturnURL:  stripe.String(getStripeReturnURL()),
		Type:       stripe.String("account_onboarding"),
	})
	if err != nil {
		http.Error(w, "Error creating Stripe onboarding link", http.StatusBadGateway)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":     true,
		"account_id": accountID,
		"connected":  connected,
		"url":        link.URL,
	})
}

// GetStripeConnectStatus returns the cached Stripe connect state, refreshing it from Stripe when possible.
func GetStripeConnectStatus(w http.ResponseWriter, r *http.Request) {
	userID, err := getAuthenticatedUserID(r)
	if err != nil {
		http.Error(w, "Invalid to get user from the token", http.StatusUnauthorized)
		return
	}

	accountID, connected, err := loadStripePayoutAccount(userID)
	if err != nil {
		http.Error(w, "Error loading payout account", http.StatusInternalServerError)
		return
	}

	if accountID != "" {
		if refreshed, err := refreshStripePayoutStatus(userID, accountID); err == nil {
			connected = refreshed
		}
	}

	status := model.DashboardStripeStatus{IsConnected: connected}
	writeJSON(w, http.StatusOK, status)
}
