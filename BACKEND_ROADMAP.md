# BrewMe — Backend Roadmap

Go backend plan for the BrewMe creator-support platform. Maps to
[`database/brew.sql`](database/brew.sql) and [`API.md`](API.md).

> **Status assessment**
> - ✅ **Schema + API spec are solid** — well-designed FKs, indexes, decimal money, views, and a clearly prioritized endpoint list.
> - ⚠️ **Go code is a placeholder, not a structure yet** — `main.go` prints a string, the health handler logs instead of responding, and `go.mod` has an invalid module name (`module main.go`).

---

## Recommended stack (pragmatic Go MVP)

| Concern | Pick | Why |
|---|---|---|
| **Router** | `go-chi/chi/v5` | `net/http`-compatible, lightweight, great middleware. Less magic than Gin, easy to drop later. |
| **DB access** | `sqlc` + `database/sql` + `go-sql-driver/mysql` | Turns the SQL you already wrote into type-safe Go. No ORM surprises, fast. |
| **Migrations** | `golang-migrate` or `goose` | Version the schema instead of re-running `brew.sql` (which `DROP`s the DB). |
| **Auth / JWT** | `golang-jwt/jwt/v5` + `golang.org/x/crypto/bcrypt` | Matches the `Authorization: Bearer` design. |
| **Validation** | `go-playground/validator/v10` | Maps cleanly to the `VALIDATION_ERROR` error shape. |
| **Config** | `caarlos0/env` or `joho/godotenv` | 12-factor env config, no heavy framework. |
| **Payments** | `stripe/stripe-go/v79` | Donations + signed webhook. |
| **Logging** | `log/slog` (stdlib) | Structured, built into Go 1.21+, zero deps. |

**Alternative (speed-of-writing over control):** Gin (router) + GORM (ORM) — more batteries, more magic. Valid choice; this doc assumes **chi + sqlc**.

---

## Project structure

```
backend/
├── go.mod                  # module brewme  (fix the name!)
├── .env.example
├── cmd/
│   └── server/
│       └── main.go         # wires config, db, router, starts http.Server
├── internal/
│   ├── config/             # env loading
│   ├── db/                 # sql.DB connection + sqlc-generated code
│   │   ├── queries/        # *.sql files (sqlc input)
│   │   └── sqlc/           # generated Go (sqlc output)
│   ├── middleware/         # JWT auth, CORS, request logging, recoverer
│   ├── auth/               # signup/login/me + jwt + bcrypt helpers
│   ├── creators/           # public profile, explore, categories
│   ├── donations/          # tips + stripe payment intents
│   ├── memberships/        # tiers + subscriptions
│   ├── posts/              # posts + media
│   ├── dashboard/          # overview, earnings, supporters, settings
│   ├── payments/           # stripe client + webhook handler
│   └── httperr/            # standard { error: {code,message,fields} } shape
├── migrations/             # 0001_init.up.sql (from brew.sql), .down.sql
└── uploads/                # local avatar/media storage for MVP (S3 later)
```

Each domain package gets a `handler.go` (HTTP) and `service.go` (logic). Keep
HTTP concerns out of business logic so it stays testable. One central `httperr`
package keeps every error response consistent with the spec.

---

## Build roadmap

### Phase 0 — Foundation (do first)
- [ ] Fix `go.mod` module name → `brewme`.
- [ ] `cmd/server/main.go`: load config, open MySQL pool (`SetMaxOpenConns`), mount chi router, start `http.Server`.
- [ ] Real health endpoint → `200 {"status":"ok"}`.
- [ ] Migrations from `brew.sql` (split out the `DROP DATABASE` + seed data).
- [ ] Middleware: logging, recoverer, CORS (for the Vite frontend), request-ID.
- [ ] `httperr` package matching the API error shape.

### Phase 1 — Auth (`/auth/*`)
JWT issue/verify, bcrypt hashing, signup creates a default `notification_settings`
row, `username-available` check. Unlocks every `🔒` route.

### Phase 2 — Read path
`GET /creators/:username`, `/explore`, `/categories`. Pure reads, no payments —
gets the frontend rendering real data fast.

### Phase 3 — Money loop
`POST /creators/:username/donations` + Stripe PaymentIntent + **signed**
`/webhooks/stripe`. Riskiest part — isolate in `payments/`, test the webhook with
the Stripe CLI.

### Phase 4 — Dashboard overview + supporters
Reads the `creator_earnings` / `supporter_feed` views + the reply endpoint.

### Phase 5 — Posts + Settings
CRUD, members-only post locking, avatar/media upload (local disk for MVP),
socials / goal / notifications.

### Phase 6 — Memberships
Stripe subscriptions, tier management (enforce max-3 tiers → `409`).

---

## Get-this-right-early checklist

- **Cursor pagination** — `?cursor=<id>` is used everywhere; build one reusable helper.
- **Never trust client amounts** — derive `amount = cups * price_per_cup` server-side.
- **Webhook idempotency** — Stripe retries; dedupe on `stripe_charge_id` before advancing `goals.current_amount`.
- **Auth scoping** — every `🔒` route filters by the JWT's `user_id`; a creator must never touch another's rows.
- **Secrets in env** — JWT secret, Stripe keys, DB DSN. Add `.env` to `.gitignore`, commit only `.env.example`.
