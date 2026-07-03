package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"brewme/internal/database"

	"github.com/go-chi/chi"
	stripe "github.com/stripe/stripe-go/v81"
	stripecheckout "github.com/stripe/stripe-go/v81/checkout/session"
	stripewebhook "github.com/stripe/stripe-go/v81/webhook"
)

const pricePerCupCents int64 = 500

func getStripeWebhookSecret() string {
	return strings.TrimSpace(os.Getenv("STRIPE_WEBHOOK_SECRET"))
}

func getPlatformFeePercent() int64 {
	value := strings.TrimSpace(os.Getenv("PLATFORM_FEE_PERCENT"))
	if value == "" {
		return 10
	}
	percent, err := strconv.ParseInt(value, 10, 64)
	if err != nil || percent < 0 {
		return 10
	}
	if percent > 100 {
		return 100
	}
	return percent
}

func getFrontendBaseURL(r *http.Request) string {
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin != "" {
		return strings.TrimRight(origin, "/")
	}
	if value := strings.TrimSpace(os.Getenv("FRONTEND_BASE_URL")); value != "" {
		return strings.TrimRight(value, "/")
	}
	return "http://localhost:5173"
}

func loadCreatorByUsername(username string) (id int64, fullName string, err error) {
	err = database.DB.QueryRow(`SELECT id, full_name FROM users WHERE username = ?`, username).Scan(&id, &fullName)
	return
}

func loadStripeAccountForUser(userID int64) (accountID string, connected bool, err error) {
	return loadStripePayoutAccount(userID)
}

func createPendingDonation(userID int64, supporterName string, message string, isAnonymous bool, cups int64, amountCents int64) (int64, error) {
	name := sql.NullString{String: strings.TrimSpace(supporterName), Valid: strings.TrimSpace(supporterName) != ""}
	msg := sql.NullString{String: strings.TrimSpace(message), Valid: strings.TrimSpace(message) != ""}

	res, err := database.DB.Exec(`
		INSERT INTO donations (user_id, display_name, is_anonymous, cups, amount, currency, message, status)
		VALUES (?, ?, ?, ?, ?, 'USD', ?, 'pending')`,
		userID, name, isAnonymous, cups, float64(amountCents)/100, msg,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func deletePendingDonation(donationID int64) {
	_, _ = database.DB.Exec(`DELETE FROM donations WHERE id = ? AND status = 'pending'`, donationID)
}

// CreateDonationCheckout starts the supporter checkout flow for a creator.
func CreateDonationCheckout(w http.ResponseWriter, r *http.Request) {
	username := strings.TrimSpace(chi.URLParam(r, "username"))
	if username == "" {
		http.Error(w, "Username is required", http.StatusBadRequest)
		return
	}

	var req struct {
		Cups          int64  `json:"cups"`
		SupporterName string `json:"supporter_name"`
		Message       string `json:"message"`
		IsAnonymous   bool   `json:"is_anonymous"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if req.Cups <= 0 || req.Cups > 1000 {
		http.Error(w, "Cup count must be between 1 and 1000", http.StatusBadRequest)
		return
	}

	secretKey := getStripeSecretKey()
	if secretKey == "" {
		http.Error(w, "Stripe is not configured", http.StatusServiceUnavailable)
		return
	}

	creatorID, creatorName, err := loadCreatorByUsername(username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "Creator not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Error loading creator", http.StatusInternalServerError)
		return
	}

	accountID, connected, err := loadStripeAccountForUser(creatorID)
	if err != nil {
		http.Error(w, "Error loading creator payout setup", http.StatusInternalServerError)
		return
	}
	if accountID == "" || !connected {
		http.Error(w, "Creator has not finished Stripe setup yet", http.StatusConflict)
		return
	}

	amountCents := req.Cups * pricePerCupCents
	if amountCents <= 0 {
		http.Error(w, "Invalid amount", http.StatusBadRequest)
		return
	}

	donationID, err := createPendingDonation(creatorID, req.SupporterName, req.Message, req.IsAnonymous, req.Cups, amountCents)
	if err != nil {
		http.Error(w, "Error creating donation", http.StatusInternalServerError)
		return
	}

	frontendBase := getFrontendBaseURL(r)
	successURL := fmt.Sprintf("%s/success?slug=%s&creator=%s&amount=%.2f", frontendBase, url.QueryEscape(username), url.QueryEscape(creatorName), float64(amountCents)/100)
	cancelURL := fmt.Sprintf("%s/%s", frontendBase, url.PathEscape(username))
	platformFee := amountCents * getPlatformFeePercent() / 100

	stripe.Key = secretKey
	session, err := stripecheckout.New(&stripe.CheckoutSessionParams{
		Mode:              stripe.String("payment"),
		SuccessURL:        stripe.String(successURL),
		CancelURL:         stripe.String(cancelURL),
		ClientReferenceID: stripe.String(fmt.Sprintf("donation_%d", donationID)),
		LineItems: []*stripe.CheckoutSessionLineItemParams{{
			Quantity: stripe.Int64(1),
			PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
				Currency:   stripe.String("usd"),
				UnitAmount: stripe.Int64(amountCents),
				ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
					Name:        stripe.String(fmt.Sprintf("Support for %s", creatorName)),
					Description: stripe.String(fmt.Sprintf("%d coffee%s on BrewMe", req.Cups, map[bool]string{true: "", false: "s"}[req.Cups == 1])),
				},
			},
		}},
		PaymentIntentData: &stripe.CheckoutSessionPaymentIntentDataParams{
			ApplicationFeeAmount: stripe.Int64(platformFee),
			TransferData: &stripe.CheckoutSessionPaymentIntentDataTransferDataParams{
				Destination: stripe.String(accountID),
			},
			Metadata: map[string]string{
				"donation_id":      fmt.Sprintf("%d", donationID),
				"creator_user_id":  fmt.Sprintf("%d", creatorID),
				"creator_username": username,
			},
		},
		Metadata: map[string]string{
			"donation_id":      fmt.Sprintf("%d", donationID),
			"creator_user_id":  fmt.Sprintf("%d", creatorID),
			"creator_username": username,
		},
	})
	if err != nil {
		deletePendingDonation(donationID)
		http.Error(w, "Error creating checkout session", http.StatusBadGateway)
		return
	}

	_, _ = database.DB.Exec(`
		UPDATE donations
		SET stripe_charge_id = ?
		WHERE id = ? AND status = 'pending'`, session.ID, donationID)

	writeJSON(w, http.StatusOK, map[string]any{
		"status":      true,
		"donation_id": donationID,
		"url":         session.URL,
	})
}

func markDonationSucceeded(donationID int64, paymentIntentID string, customerEmail string, customerName string) error {
	tx, err := database.DB.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()

	var userID int64
	var amount float64
	var cups int64
	var currentStatus string
	var formDisplayName sql.NullString
	var isAnonymous bool
	if err := tx.QueryRow(`
		SELECT user_id, amount, cups, status, display_name, is_anonymous 
		FROM donations 
		WHERE id = ? FOR UPDATE`, donationID).Scan(&userID, &amount, &cups, &currentStatus, &formDisplayName, &isAnonymous); err != nil {
		return err
	}
	if currentStatus == "succeeded" {
		return tx.Commit()
	}

	var supporterID sql.NullInt64
	if customerEmail != "" {
		var existingSupporterID int64
		err = tx.QueryRow(`
			SELECT id 
			FROM supporters 
			WHERE user_id = ? AND email = ?`, userID, customerEmail).Scan(&existingSupporterID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				// Create new supporter
				displayName := strings.TrimSpace(customerName)
				if formDisplayName.Valid && strings.TrimSpace(formDisplayName.String) != "" {
					displayName = strings.TrimSpace(formDisplayName.String)
				}
				if displayName == "" {
					displayName = "Anonymous"
				}
				res, err := tx.Exec(`
					INSERT INTO supporters (user_id, display_name, email, is_anonymous)
					VALUES (?, ?, ?, ?)`,
					userID, displayName, customerEmail, isAnonymous,
				)
				if err != nil {
					return err
				}
				lastID, err := res.LastInsertId()
				if err != nil {
					return err
				}
				supporterID = sql.NullInt64{Int64: lastID, Valid: true}
			} else {
				return err
			}
		} else {
			// Supporter already exists, update their name if a new one is provided in form
			if formDisplayName.Valid && strings.TrimSpace(formDisplayName.String) != "" {
				_, _ = tx.Exec(`
					UPDATE supporters
					SET display_name = ?
					WHERE id = ?`, strings.TrimSpace(formDisplayName.String), existingSupporterID)
			}
			supporterID = sql.NullInt64{Int64: existingSupporterID, Valid: true}
		}
	}

	if _, err := tx.Exec(`
		UPDATE donations
		SET status = 'succeeded',
		    stripe_charge_id = COALESCE(NULLIF(?, ''), stripe_charge_id),
		    supporter_id = ?
		WHERE id = ?`, paymentIntentID, supporterID, donationID); err != nil {
		return err
	}

	if _, err := tx.Exec(`
		UPDATE goals
		SET current_amount = current_amount + ?
		WHERE user_id = ? AND is_active = 1`, amount, userID); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	tx = nil
	return nil
}

func markMembershipSucceeded(creatorID int64, tierID int64, subscriptionID string, customerEmail string, customerName string, supporterName string) error {
	tx, err := database.DB.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()

	// 1. Idempotency Check: check if subscription already exists
	var existingID int64
	err = tx.QueryRow(`SELECT id FROM memberships WHERE stripe_subscription_id = ?`, subscriptionID).Scan(&existingID)
	if err == nil {
		return tx.Commit() // Already processed
	} else if !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	// 2. Fetch or Create Supporter
	var supporterID int64
	err = tx.QueryRow(`SELECT id FROM supporters WHERE user_id = ? AND email = ?`, creatorID, customerEmail).Scan(&supporterID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			displayName := strings.TrimSpace(supporterName)
			if displayName == "" {
				displayName = strings.TrimSpace(customerName)
			}
			if displayName == "" {
				displayName = "Anonymous"
			}
			res, err := tx.Exec(`
				INSERT INTO supporters (user_id, display_name, email, is_anonymous)
				VALUES (?, ?, ?, 0)`,
				creatorID, displayName, customerEmail,
			)
			if err != nil {
				return err
			}
			supporterID, err = res.LastInsertId()
			if err != nil {
				return err
			}
		} else {
			return err
		}
	} else {
		// Update display name if a new one is provided in form
		displayName := strings.TrimSpace(supporterName)
		if displayName != "" {
			_, _ = tx.Exec(`UPDATE supporters SET display_name = ? WHERE id = ?`, displayName, supporterID)
		}
	}

	// 3. Fetch Tier Price
	var tierPrice float64
	err = tx.QueryRow(`SELECT price FROM membership_tiers WHERE id = ?`, tierID).Scan(&tierPrice)
	if err != nil {
		return err
	}

	// 4. Create active membership record
	displayName := strings.TrimSpace(supporterName)
	if displayName == "" {
		displayName = strings.TrimSpace(customerName)
	}
	if displayName == "" {
		displayName = "Anonymous"
	}
	_, err = tx.Exec(`
		INSERT INTO memberships (user_id, tier_id, supporter_id, display_name, amount, status, stripe_subscription_id, started_at)
		VALUES (?, ?, ?, ?, ?, 'active', ?, CURRENT_TIMESTAMP)`,
		creatorID, tierID, supporterID, displayName, tierPrice, subscriptionID,
	)
	if err != nil {
		return err
	}

	// 5. Update Goals
	if _, err := tx.Exec(`
		UPDATE goals
		SET current_amount = current_amount + ?
		WHERE user_id = ? AND is_active = 1`, tierPrice, creatorID); err != nil {
		return err
	}

	return tx.Commit()
}

func handleSubscriptionRenewal(subscriptionID string, amountPaid float64) error {
	tx, err := database.DB.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()

	var creatorID int64
	err = tx.QueryRow(`
		SELECT user_id 
		FROM memberships 
		WHERE stripe_subscription_id = ?`, subscriptionID).Scan(&creatorID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil // Subscription doesn't exist locally yet
		}
		return err
	}

	// Update period and set to active
	_, err = tx.Exec(`
		UPDATE memberships 
		SET current_period_end = DATE_ADD(CURRENT_TIMESTAMP, INTERVAL 1 MONTH), status = 'active'
		WHERE stripe_subscription_id = ?`, subscriptionID)
	if err != nil {
		return err
	}

	// Update active goal
	_, err = tx.Exec(`
		UPDATE goals 
		SET current_amount = current_amount + ? 
		WHERE user_id = ? AND is_active = 1`, amountPaid, creatorID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// StripeWebhook receives payment events and finalizes successful donations.
func StripeWebhook(w http.ResponseWriter, r *http.Request) {
	secret := getStripeWebhookSecret()
	if secret == "" {
		http.Error(w, "Stripe webhook secret is not configured", http.StatusServiceUnavailable)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Unable to read webhook body", http.StatusBadRequest)
		return
	}

	event, err := stripewebhook.ConstructEventWithOptions(body, r.Header.Get("Stripe-Signature"), secret, stripewebhook.ConstructEventOptions{
		IgnoreAPIVersionMismatch: true,
	})
	if err != nil {
		fmt.Printf("[ERROR] Stripe Webhook signature verification failed: %v\n", err)
		http.Error(w, fmt.Sprintf("Invalid webhook signature: %v", err), http.StatusBadRequest)
		return
	}

	switch event.Type {
	case "checkout.session.completed":
		var session stripe.CheckoutSession
		if err := json.Unmarshal(event.Data.Raw, &session); err != nil {
			http.Error(w, "Invalid checkout payload", http.StatusBadRequest)
			return
		}
		if strings.ToLower(string(session.PaymentStatus)) != "paid" {
			w.WriteHeader(http.StatusOK)
			return
		}

		email := ""
		name := ""
		if session.CustomerDetails != nil {
			email = session.CustomerDetails.Email
			name = session.CustomerDetails.Name
		}

		// Check if this is a membership subscription checkout session
		if tierIDStr, ok := session.Metadata["tier_id"]; ok && tierIDStr != "" {
			tierID, err := strconv.ParseInt(tierIDStr, 10, 64)
			if err != nil || tierID <= 0 {
				http.Error(w, "Invalid tier metadata", http.StatusBadRequest)
				return
			}
			creatorIDStr := session.Metadata["creator_user_id"]
			creatorID, err := strconv.ParseInt(creatorIDStr, 10, 64)
			if err != nil || creatorID <= 0 {
				http.Error(w, "Invalid creator metadata", http.StatusBadRequest)
				return
			}
			supporterName := session.Metadata["supporter_name"]
			subscriptionID := ""
			if session.Subscription != nil {
				subscriptionID = session.Subscription.ID
			}

			if err := markMembershipSucceeded(creatorID, tierID, subscriptionID, email, name, supporterName); err != nil {
				http.Error(w, "Failed to update membership", http.StatusInternalServerError)
				return
			}
		} else {
			// Otherwise process as standard donation checkout session
			donationIDStr := session.Metadata["donation_id"]
			donationID, err := strconv.ParseInt(donationIDStr, 10, 64)
			if err != nil || donationID <= 0 {
				http.Error(w, "Missing donation metadata", http.StatusBadRequest)
				return
			}

			paymentIntentID := ""
			if session.PaymentIntent != nil {
				paymentIntentID = session.PaymentIntent.ID
			}

			if err := markDonationSucceeded(donationID, paymentIntentID, email, name); err != nil {
				http.Error(w, "Failed to update donation", http.StatusInternalServerError)
				return
			}
		}

	case "invoice.payment_succeeded":
		var invoice stripe.Invoice
		if err := json.Unmarshal(event.Data.Raw, &invoice); err != nil {
			http.Error(w, "Invalid invoice payload", http.StatusBadRequest)
			return
		}
		if invoice.Subscription == nil {
			w.WriteHeader(http.StatusOK)
			return
		}
		amountPaid := float64(invoice.AmountPaid) / 100.0
		if err := handleSubscriptionRenewal(invoice.Subscription.ID, amountPaid); err != nil {
			http.Error(w, "Failed to update subscription renewal", http.StatusInternalServerError)
			return
		}

	case "customer.subscription.deleted":
		var subscription stripe.Subscription
		if err := json.Unmarshal(event.Data.Raw, &subscription); err != nil {
			http.Error(w, "Invalid subscription payload", http.StatusBadRequest)
			return
		}
		_, err := database.DB.Exec(`
			UPDATE memberships
			SET status = 'canceled', canceled_at = CURRENT_TIMESTAMP
			WHERE stripe_subscription_id = ?`, subscription.ID)
		if err != nil {
			log.Printf("Failed to cancel subscription in webhook: %v", err)
			http.Error(w, "Failed to update subscription status", http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
}
