package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"brewme/internal/database"

	"github.com/go-chi/chi"
	stripe "github.com/stripe/stripe-go/v81"
	stripecheckout "github.com/stripe/stripe-go/v81/checkout/session"
)

// CreateMembershipCheckout starts a Stripe subscription checkout session for a membership tier.
func CreateMembershipCheckout(w http.ResponseWriter, r *http.Request) {
	username := strings.TrimSpace(chi.URLParam(r, "username"))
	if username == "" {
		http.Error(w, "Username is required", http.StatusBadRequest)
		return
	}

	var req struct {
		TierID        int64  `json:"tier_id"`
		SupporterName string `json:"supporter_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if req.TierID <= 0 {
		http.Error(w, "Valid tier ID is required", http.StatusBadRequest)
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

	// Fetch selected membership tier details
	var tierName string
	var tierPrice float64
	err = database.DB.QueryRow(`
		SELECT name, price
		FROM membership_tiers
		WHERE id = ? AND user_id = ? AND is_active = 1`, req.TierID, creatorID).Scan(&tierName, &tierPrice)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "Membership tier not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Error loading membership tier", http.StatusInternalServerError)
		return
	}

	priceCents := int64(tierPrice * 100)
	if priceCents <= 0 {
		http.Error(w, "Invalid membership price", http.StatusBadRequest)
		return
	}

	platformFeePercent := getPlatformFeePercent()
	frontendBase := getFrontendBaseURL(r)
	successURL := fmt.Sprintf("%s/success?slug=%s&creator=%s&amount=%.2f", frontendBase, url.QueryEscape(username), url.QueryEscape(creatorName), tierPrice)
	cancelURL := fmt.Sprintf("%s/%s", frontendBase, url.PathEscape(username))

	stripe.Key = secretKey
	session, err := stripecheckout.New(&stripe.CheckoutSessionParams{
		Mode:       stripe.String("subscription"),
		SuccessURL: stripe.String(successURL),
		CancelURL:  stripe.String(cancelURL),
		LineItems: []*stripe.CheckoutSessionLineItemParams{{
			Quantity: stripe.Int64(1),
			PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
				Currency:   stripe.String("usd"),
				UnitAmount: stripe.Int64(priceCents),
				ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
					Name:        stripe.String(fmt.Sprintf("%s - Support for %s", tierName, creatorName)),
					Description: stripe.String("Recurring monthly membership on BrewMe"),
				},
				Recurring: &stripe.CheckoutSessionLineItemPriceDataRecurringParams{
					Interval: stripe.String("month"),
				},
			},
		}},
		SubscriptionData: &stripe.CheckoutSessionSubscriptionDataParams{
			TransferData: &stripe.CheckoutSessionSubscriptionDataTransferDataParams{
				Destination:   stripe.String(accountID),
				AmountPercent: stripe.Float64(float64(100 - platformFeePercent)),
			},
			Metadata: map[string]string{
				"creator_user_id": fmt.Sprintf("%d", creatorID),
				"tier_id":         fmt.Sprintf("%d", req.TierID),
				"supporter_name":  req.SupporterName,
			},
		},
		Metadata: map[string]string{
			"creator_user_id": fmt.Sprintf("%d", creatorID),
			"tier_id":         fmt.Sprintf("%d", req.TierID),
			"supporter_name":  req.SupporterName,
		},
	})
	if err != nil {
		http.Error(w, "Error creating checkout session", http.StatusBadGateway)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status": true,
		"url":    session.URL,
	})
}
