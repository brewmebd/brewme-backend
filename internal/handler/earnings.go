package handler

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"brewme/internal/database"
	"brewme/internal/model"
)

// GetDashboardEarnings returns totals, the monthly earnings chart, and payout
// history for the authenticated creator.
func GetDashboardEarnings(w http.ResponseWriter, r *http.Request) {
	userID, ok := authUserID(w, r)
	if !ok {
		return
	}

	// 1. Lifetime gross + completed payouts from the creator_earnings view.
	var totalEarned, totalPaidOut float64
	err := database.DB.QueryRow(`
		SELECT total_earned, total_paid_out FROM creator_earnings WHERE user_id = ?`,
		userID,
	).Scan(&totalEarned, &totalPaidOut)
	if err != nil && err != sql.ErrNoRows {
		http.Error(w, "Error fetching earnings", http.StatusInternalServerError)
		log.Println("Database error (earnings totals):", err)
		return
	}

	// 2. Pending payouts (money reserved but not yet paid).
	var pendingPayouts float64
	if err := database.DB.QueryRow(
		`SELECT COALESCE(SUM(amount), 0) FROM payouts WHERE user_id = ? AND status = 'pending'`, userID,
	).Scan(&pendingPayouts); err != nil {
		http.Error(w, "Error fetching pending payouts", http.StatusInternalServerError)
		log.Println("Database error (pending payouts):", err)
		return
	}

	available := totalEarned - totalPaidOut - pendingPayouts
	if available < 0 {
		available = 0
	}

	// 3. This-month vs last-month gross, for the change badge.
	thisMonth := monthlyEarnings(userID, 0)
	lastMonth := monthlyEarnings(userID, 1)
	var changePct float64
	switch {
	case lastMonth > 0:
		changePct = (thisMonth - lastMonth) / lastMonth * 100
	case thisMonth > 0:
		changePct = 100
	}

	// 4. Chart: earnings per month for the current year.
	chart, err := earningsChart(userID)
	if err != nil {
		http.Error(w, "Error fetching chart", http.StatusInternalServerError)
		log.Println("Database error (chart):", err)
		return
	}

	// 5. Payout history.
	payouts, err := payoutHistory(userID)
	if err != nil {
		http.Error(w, "Error fetching payouts", http.StatusInternalServerError)
		log.Println("Database error (payout history):", err)
		return
	}

	writeJSON(w, http.StatusOK, model.EarningsResponse{
		TotalEarned:      formatMoney(totalEarned),
		TotalChange:      formatPct(changePct),
		AvailableBalance: formatMoney(available),
		TotalPayoutsSum:  formatMoney(totalPaidOut),
		ChartData:        chart,
		Payouts:          payouts,
	})
}

// RequestPayout records a pending payout against the creator's available balance.
func RequestPayout(w http.ResponseWriter, r *http.Request) {
	userID, ok := authUserID(w, r)
	if !ok {
		return
	}

	var req model.PayoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if req.Amount <= 0 {
		http.Error(w, "Amount must be greater than zero", http.StatusBadRequest)
		return
	}

	method := strings.ToLower(strings.TrimSpace(req.Method))
	switch method {
	case "stripe", "paypal", "bank":
	case "":
		method = "stripe"
	default:
		http.Error(w, "Invalid payout method", http.StatusBadRequest)
		return
	}

	// Recompute available balance server-side (never trust the client).
	var totalEarned, totalPaidOut, pending float64
	if err := database.DB.QueryRow(
		`SELECT total_earned, total_paid_out FROM creator_earnings WHERE user_id = ?`, userID,
	).Scan(&totalEarned, &totalPaidOut); err != nil && err != sql.ErrNoRows {
		http.Error(w, "Error checking balance", http.StatusInternalServerError)
		log.Println("Database error (payout balance):", err)
		return
	}
	if err := database.DB.QueryRow(
		`SELECT COALESCE(SUM(amount), 0) FROM payouts WHERE user_id = ? AND status = 'pending'`, userID,
	).Scan(&pending); err != nil {
		http.Error(w, "Error checking balance", http.StatusInternalServerError)
		return
	}
	available := totalEarned - totalPaidOut - pending
	if req.Amount > available+0.001 { // tiny epsilon for float noise
		http.Error(w, "Amount exceeds available balance", http.StatusBadRequest)
		return
	}

	// Next global reference number (PO-###).
	var nextNum int64
	if err := database.DB.QueryRow(
		`SELECT COALESCE(MAX(CAST(SUBSTRING(reference, 4) AS UNSIGNED)), 0) + 1 FROM payouts WHERE reference LIKE 'PO-%'`,
	).Scan(&nextNum); err != nil {
		http.Error(w, "Error generating reference", http.StatusInternalServerError)
		log.Println("Database error (payout ref):", err)
		return
	}
	reference := fmt.Sprintf("PO-%03d", nextNum)

	if _, err := database.DB.Exec(`
		INSERT INTO payouts (user_id, reference, amount, method, status, payout_date)
		VALUES (?, ?, ?, ?, 'pending', CURDATE())`,
		userID, reference, req.Amount, method,
	); err != nil {
		http.Error(w, "Error creating payout", http.StatusInternalServerError)
		log.Println("Database error (RequestPayout):", err)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"status":  true,
		"message": "Payout request received!",
		"payout": model.PayoutItem{
			ID:     reference,
			Amount: strconv.FormatFloat(req.Amount, 'f', 2, 64),
			Date:   nowDateLabel(),
			Status: "Pending",
		},
	})
}

// ── helpers ──────────────────────────────────────────────────────────────

// monthlyEarnings sums donations + memberships for a month offset back from the
// current month (0 = this month, 1 = last month).
func monthlyEarnings(userID int64, monthsAgo int) float64 {
	var total float64
	_ = database.DB.QueryRow(`
		SELECT
			COALESCE((SELECT SUM(amount) FROM donations
				WHERE user_id = ? AND status = 'succeeded'
				  AND created_at >= DATE_FORMAT(DATE_SUB(CURRENT_DATE(), INTERVAL ? MONTH), '%Y-%m-01')
				  AND created_at <  DATE_FORMAT(DATE_SUB(CURRENT_DATE(), INTERVAL ? MONTH) + INTERVAL 1 MONTH, '%Y-%m-01')), 0) +
			COALESCE((SELECT SUM(amount) FROM memberships
				WHERE user_id = ? AND status = 'active'
				  AND started_at >= DATE_FORMAT(DATE_SUB(CURRENT_DATE(), INTERVAL ? MONTH), '%Y-%m-01')
				  AND started_at <  DATE_FORMAT(DATE_SUB(CURRENT_DATE(), INTERVAL ? MONTH) + INTERVAL 1 MONTH, '%Y-%m-01')), 0)
	`, userID, monthsAgo, monthsAgo, userID, monthsAgo, monthsAgo).Scan(&total)
	return total
}

// earningsChart returns monthly totals (donations + memberships) for the year.
func earningsChart(userID int64) ([]model.ChartEntry, error) {
	rows, err := database.DB.Query(`
		SELECT DATE_FORMAT(date_col, '%b') AS month_name, SUM(amount) AS monthly_total
		FROM (
			SELECT created_at AS date_col, amount FROM donations
				WHERE user_id = ? AND status = 'succeeded' AND YEAR(created_at) = YEAR(CURRENT_DATE())
			UNION ALL
			SELECT started_at AS date_col, amount FROM memberships
				WHERE user_id = ? AND status = 'active' AND YEAR(started_at) = YEAR(CURRENT_DATE())
		) AS combined
		GROUP BY MONTH(date_col), month_name
		ORDER BY MONTH(date_col) ASC`,
		userID, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	chart := make([]model.ChartEntry, 0)
	for rows.Next() {
		var e model.ChartEntry
		if err := rows.Scan(&e.Month, &e.Earnings); err != nil {
			continue
		}
		chart = append(chart, e)
	}
	return chart, rows.Err()
}

// payoutHistory returns the creator's payouts, newest first.
func payoutHistory(userID int64) ([]model.PayoutItem, error) {
	rows, err := database.DB.Query(`
		SELECT reference, amount, payout_date, status
		FROM payouts WHERE user_id = ?
		ORDER BY payout_date DESC, id DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]model.PayoutItem, 0)
	for rows.Next() {
		var ref, status string
		var amount float64
		var date sql.NullTime
		if err := rows.Scan(&ref, &amount, &date, &status); err != nil {
			continue
		}
		label := ""
		if date.Valid {
			label = date.Time.Format("Jan 2, 2006")
		}
		items = append(items, model.PayoutItem{
			ID:     ref,
			Amount: strconv.FormatFloat(amount, 'f', 2, 64),
			Date:   label,
			Status: titleCase(status),
		})
	}
	return items, rows.Err()
}

// formatMoney renders a float as "$2,847.50".
func formatMoney(v float64) string {
	return "$" + humanizeAmount(v)
}

// humanizeAmount renders a float with 2 decimals and thousands separators.
func humanizeAmount(v float64) string {
	s := strconv.FormatFloat(v, 'f', 2, 64)
	neg := ""
	if strings.HasPrefix(s, "-") {
		neg, s = "-", s[1:]
	}
	parts := strings.SplitN(s, ".", 2)
	intPart := parts[0]
	var b strings.Builder
	n := len(intPart)
	for i := 0; i < n; i++ {
		if i > 0 && (n-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteByte(intPart[i])
	}
	return neg + b.String() + "." + parts[1]
}

// formatPct renders a percentage like "+12.5%" / "-3.0%".
func formatPct(v float64) string {
	sign := "+"
	if v < 0 {
		sign = ""
	}
	return sign + strconv.FormatFloat(v, 'f', 1, 64) + "%"
}

func titleCase(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func nowDateLabel() string {
	// Uses the DB's CURDATE for the stored row; for the response label we use the
	// same civil date via a lightweight query to avoid clock/timezone drift.
	var d sql.NullTime
	if err := database.DB.QueryRow(`SELECT CURDATE()`).Scan(&d); err == nil && d.Valid {
		return d.Time.Format("Jan 2, 2006")
	}
	return ""
}
