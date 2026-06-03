# BrewMe — API Specification

REST API backing the BrewMe creator-support platform. Maps every endpoint to the
React screen and database table it serves (see [`database/brew.sql`](database/brew.sql)).

- **Base URL:** `/api/v1`
- **Auth:** Bearer JWT in `Authorization: Bearer <token>` for creator-only routes.
- **Content type:** `application/json` unless noted (uploads use `multipart/form-data`).
- **Money:** sent/received as decimal strings or numbers in major units (USD), e.g. `15.00`.
- **Timestamps:** ISO 8601 UTC, e.g. `2026-04-22T10:00:00Z`.

### Conventions

| Field | Meaning |
|---|---|
| 🔓 Public | No auth required |
| 🔒 Auth | Requires a logged-in creator (own resources only) |
| `:username` | Creator page slug, e.g. `sarahchen` |

### Standard error shape
```json
{ "error": { "code": "VALIDATION_ERROR", "message": "Email is already in use.", "fields": { "email": "taken" } } }
```
Status codes: `400` validation, `401` unauthenticated, `403` forbidden, `404` not found, `409` conflict, `422` unprocessable, `500` server.

---

## 1. Authentication & Account

Backs **SignUpPage**, **LoginPage**, and the session.

### `POST /auth/signup` 🔓
Create a creator account and claim a page URL.
```json
// Request
{ "full_name": "Sarah Chen", "email": "sarah@example.com", "password": "min8chars", "username": "sarahchen" }
// 201 Response
{ "token": "<jwt>", "user": { "id": 1, "full_name": "Sarah Chen", "username": "sarahchen", "email": "sarah@example.com" } }
```
- Validates `username` is unique (→ `409 USERNAME_TAKEN`) and `email` unique.
- Hashes password (bcrypt/argon2). Creates default `notification_settings` row.

### `POST /auth/login` 🔓
```json
// Request
{ "email": "sarah@example.com", "password": "..." }
// 200 Response
{ "token": "<jwt>", "user": { "id": 1, "username": "sarahchen", "full_name": "Sarah Chen" } }
```

### `POST /auth/logout` 🔒
Invalidates the current token/session. → `204`

### `GET /auth/me` 🔒
Returns the authenticated creator's profile + settings (for dashboard bootstrap).

### `GET /auth/username-available?username=sarahchen` 🔓
Live check for the signup URL field. → `{ "available": true }`

---

## 2. Public Creator Pages

Backs **CreatorProfilePage** (`/:username`) and the support widget.

### `GET /creators/:username` 🔓
Full public profile: header, socials, active goal, tier summary.
```json
// 200 Response
{
  "id": 1,
  "username": "sarahchen",
  "full_name": "Sarah Chen",
  "bio": "Digital artist creating illustrations...",
  "category": "Digital Art",
  "avatar_url": null,
  "supporter_count": 1247,
  "price_per_cup": 5.00,
  "socials": [ { "platform": "twitter", "url": "https://..." } ],
  "goal": { "label": "New iPad Pro for drawing streams", "current_amount": 340.00, "target_amount": 500.00 },
  "tiers": [ { "id": 10, "name": "Coffee Supporter", "price": 5.00, "perks": ["..."] } ]
}
```
→ `404` if the username doesn't exist (the "this brew doesn't exist" page).

### `GET /creators/:username/supporters` 🔓
Recent supporters feed (Supporters tab). Reads the `supporter_feed` view.
- Query: `?limit=20&cursor=<id>`
```json
{ "data": [ { "display_name": "Emily R.", "message": "Love your work! 🎨", "cups": 3, "amount": 15.00, "created_at": "..." } ], "next_cursor": null }
```

### `GET /creators/:username/posts` 🔓
Posts tab. Members-only posts are returned **locked** (no body) for non-members.
- Query: `?limit=10&cursor=<id>`
```json
{ "data": [ { "id": 5, "title": "...", "preview": "...", "visibility": "members", "locked": true, "published_at": "..." } ], "next_cursor": null }
```

---

## 3. Donations (one-time coffees)

Backs the **"Buy a coffee"** support widget. No account required for the supporter.

### `POST /creators/:username/donations` 🔓
Create a coffee tip. `amount` is derived server-side as `cups * price_per_cup`.
```json
// Request
{ "cups": 3, "supporter_name": "Emily R.", "message": "Love your work! 🎨", "is_anonymous": false, "payment_method_id": "pm_xxx" }
// 201 Response
{ "id": 101, "amount": 15.00, "status": "succeeded", "client_secret": "pi_xxx_secret" }
```
- Creates/updates a `supporters` row (deduped by email when provided), inserts a `donations` row, updates the active `goals.current_amount`.
- Integrates with Stripe PaymentIntent. `status` may be `pending` until webhook confirms.

### `POST /webhooks/stripe` 🔓 (signed)
Stripe webhook receiver. Confirms `payment_intent.succeeded`, marks donations `succeeded`, advances goal totals, records membership renewals.

---

## 4. Memberships (recurring)

Backs tier subscriptions ("joined Gold tier — $10/mo").

### `POST /creators/:username/memberships` 🔓
Subscribe a supporter to a tier.
```json
// Request
{ "tier_id": 11, "supporter_name": "Aria Patel", "email": "aria@example.com", "payment_method_id": "pm_xxx" }
// 201 Response
{ "id": 55, "tier_id": 11, "amount": 15.00, "status": "active", "current_period_end": "2026-07-18T00:00:00Z" }
```
Creates a Stripe subscription + `memberships` row.

### `DELETE /memberships/:id` 🔒/supporter-token
Cancel a membership → sets `status = 'canceled'`, `canceled_at`.

---

## 5. Dashboard — Overview

Backs **DashboardOverview** (stat cards, chart, recent activity).

### `GET /dashboard/overview` 🔒
```json
{
  "stats": { "total_earned": 2847.50, "this_month": 486.00, "supporters": 1247, "posts": 34,
             "changes": { "total_earned": "+12.5%", "this_month": "+8.3%", "supporters": "+23", "posts": "+3" } },
  "earnings_chart": [ { "month": "Jan", "earnings": 180 }, { "month": "Feb", "earnings": 220 } ],
  "recent_activity": [ { "name": "Emily R.", "action": "bought 3 coffees", "amount": "$15.00", "time": "2h ago" } ]
}
```
Reads the `creator_earnings` view + recent `donations`/`memberships`.

---

## 6. Dashboard — Supporters

Backs **DashboardSupporters** (table + reply action).

### `GET /dashboard/supporters` 🔒
```json
// Query: ?search=emily&limit=20&cursor=<id>
{ "data": [ { "id": 101, "name": "Emily Rodriguez", "message": "...", "amount": "$15.00", "cups": 3, "date": "Apr 22, 2026", "replied": false } ], "next_cursor": null }
```
Searchable by name/message.

### `POST /donations/:id/reply` 🔒
Creator replies to a supporter message.
```json
// Request
{ "message": "Thank you so much!" }
// 200 Response
{ "id": 101, "replied": true, "replied_at": "..." }
```

---

## 7. Dashboard — Earnings

Backs **DashboardEarnings** (summary cards, chart, payout history, CSV export, request payout).

### `GET /dashboard/earnings` 🔒
```json
{
  "summary": { "total_earned": 2847.50, "available_balance": 486.00, "total_payouts": 2361.50 },
  "chart": [ { "month": "Jan", "earnings": 180 } ]
}
```

### `GET /dashboard/payouts` 🔒
Payout history table.
```json
{ "data": [ { "id": 1, "reference": "PO-001", "amount": "$450.00", "date": "Sep 1, 2026", "method": "Stripe", "status": "Completed" } ] }
```

### `POST /dashboard/payouts` 🔒
Request a payout of the available balance → creates a `payouts` row (`status: pending`).
```json
// 201 Response
{ "id": 6, "reference": "PO-006", "amount": "486.00", "status": "pending" }
```

### `GET /dashboard/earnings/export.csv` 🔒
Returns `text/csv` of monthly earnings (the "Export CSV" button).

---

## 8. Dashboard — Posts

Backs **DashboardPosts** (list, new post editor, visibility, media).

### `GET /dashboard/posts` 🔒
All of the creator's posts incl. drafts, with like/comment counts.

### `POST /dashboard/posts` 🔒
Create/publish a post.
```json
// Request
{ "title": "Behind the scenes...", "body": "full content", "visibility": "public", "status": "published", "media_ids": [] }
// 201 Response
{ "id": 5, "title": "...", "visibility": "public", "status": "published", "published_at": "..." }
```

### `PATCH /dashboard/posts/:id` 🔒
Update title/body/visibility or publish a draft.

### `DELETE /dashboard/posts/:id` 🔒
Delete a post. → `204`

### `POST /dashboard/posts/media` 🔒 `multipart/form-data`
Upload an image/file for the "Add Media" button. → `{ "id": 9, "url": "https://...", "media_type": "image" }`

---

## 9. Dashboard — Memberships (tier management)

Backs **DashboardMemberships** (max 3 tiers, perks, subscriber counts).

### `GET /dashboard/memberships` 🔒
```json
{
  "summary": { "total_members": 131, "monthly_revenue": 1465.00, "active_tiers": 3 },
  "tiers": [ { "id": 10, "name": "Coffee Supporter", "price": 5.00, "subscribers": 89, "perks": ["..."] } ]
}
```

### `POST /dashboard/memberships/tiers` 🔒
Create a tier (rejected with `409` if the creator already has 3).
```json
{ "name": "Gold Member", "price": 15.00, "perks": ["Exclusive posts & downloads", "Monthly Q&A access"] }
```

### `PATCH /dashboard/memberships/tiers/:id` 🔒
Edit tier name/price/perks (the per-card edit button).

### `DELETE /dashboard/memberships/tiers/:id` 🔒
Archive a tier (`is_active = false`). → `204`

---

## 10. Dashboard — Settings

Backs **DashboardSettings** (profile, avatar, page URL, notifications, Stripe).

### `GET /dashboard/settings` 🔒
Profile fields + notification toggles + payout-account status.

### `PATCH /dashboard/settings/profile` 🔒
```json
{ "full_name": "Sarah Chen", "bio": "...", "username": "sarahchen", "email": "sarah@example.com" }
```
- Changing `username` re-checks uniqueness (→ `409`).

### `POST /dashboard/settings/avatar` 🔒 `multipart/form-data`
Upload a new profile photo (JPG/PNG/GIF, max 2MB). → `{ "avatar_url": "https://..." }`

### `PATCH /dashboard/settings/notifications` 🔒
```json
{ "new_supporter": true, "new_message": true, "weekly_report": false, "marketing_emails": false }
```

### `PUT /dashboard/settings/socials` 🔒
Replace the creator's social links list.
```json
{ "socials": [ { "platform": "twitter", "url": "https://..." }, { "platform": "instagram", "url": "https://..." } ] }
```

### `PUT /dashboard/settings/goal` 🔒
Create/update the active funding goal.
```json
{ "label": "New iPad Pro for drawing streams", "target_amount": 500.00, "is_active": true }
```

### Stripe Connect
- `POST /dashboard/settings/stripe/connect` 🔒 → returns an onboarding URL (Stripe Connect OAuth).
- `GET /dashboard/settings/stripe/status` 🔒 → `{ "is_connected": true, "card_last4": "4242" }`
- `POST /dashboard/settings/stripe/disconnect` 🔒

---

## 11. Explore / Discovery

Backs **ExplorePage** (search + category filter grid).

### `GET /explore` 🔓
```json
// Query: ?search=sarah&category=Digital%20Art&limit=24&cursor=<id>
{ "data": [ { "username": "sarahchen", "full_name": "Sarah Chen", "category": "Digital Art", "bio": "...", "supporter_count": 1247 } ], "next_cursor": null }
```

### `GET /categories` 🔓
Category list for the filter pills. → `[ { "id": 1, "name": "Digital Art", "slug": "digital-art" } ]`

---

## Endpoint summary

| # | Method | Path | Auth | Screen |
|---|--------|------|------|--------|
| 1 | POST | `/auth/signup` | 🔓 | SignUp |
| 1 | POST | `/auth/login` | 🔓 | Login |
| 1 | POST | `/auth/logout` | 🔒 | — |
| 1 | GET | `/auth/me` | 🔒 | Dashboard boot |
| 1 | GET | `/auth/username-available` | 🔓 | SignUp |
| 2 | GET | `/creators/:username` | 🔓 | Creator profile |
| 2 | GET | `/creators/:username/supporters` | 🔓 | Profile · Supporters tab |
| 2 | GET | `/creators/:username/posts` | 🔓 | Profile · Posts tab |
| 3 | POST | `/creators/:username/donations` | 🔓 | Support widget |
| 3 | POST | `/webhooks/stripe` | 🔓* | Payments |
| 4 | POST | `/creators/:username/memberships` | 🔓 | Tier subscribe |
| 4 | DELETE | `/memberships/:id` | 🔒 | Manage membership |
| 5 | GET | `/dashboard/overview` | 🔒 | Overview |
| 6 | GET | `/dashboard/supporters` | 🔒 | Supporters |
| 6 | POST | `/donations/:id/reply` | 🔒 | Supporters · Reply |
| 7 | GET | `/dashboard/earnings` | 🔒 | Earnings |
| 7 | GET | `/dashboard/payouts` | 🔒 | Earnings |
| 7 | POST | `/dashboard/payouts` | 🔒 | Request payout |
| 7 | GET | `/dashboard/earnings/export.csv` | 🔒 | Export CSV |
| 8 | GET | `/dashboard/posts` | 🔒 | Posts |
| 8 | POST | `/dashboard/posts` | 🔒 | New post |
| 8 | PATCH | `/dashboard/posts/:id` | 🔒 | Edit post |
| 8 | DELETE | `/dashboard/posts/:id` | 🔒 | Delete post |
| 8 | POST | `/dashboard/posts/media` | 🔒 | Add media |
| 9 | GET | `/dashboard/memberships` | 🔒 | Memberships |
| 9 | POST | `/dashboard/memberships/tiers` | 🔒 | Add tier |
| 9 | PATCH | `/dashboard/memberships/tiers/:id` | 🔒 | Edit tier |
| 9 | DELETE | `/dashboard/memberships/tiers/:id` | 🔒 | Remove tier |
| 10 | GET | `/dashboard/settings` | 🔒 | Settings |
| 10 | PATCH | `/dashboard/settings/profile` | 🔒 | Settings · Profile |
| 10 | POST | `/dashboard/settings/avatar` | 🔒 | Settings · Photo |
| 10 | PATCH | `/dashboard/settings/notifications` | 🔒 | Settings · Notifications |
| 10 | PUT | `/dashboard/settings/socials` | 🔒 | Settings · Socials |
| 10 | PUT | `/dashboard/settings/goal` | 🔒 | Settings · Goal |
| 10 | POST | `/dashboard/settings/stripe/connect` | 🔒 | Settings · Payouts |
| 10 | GET | `/dashboard/settings/stripe/status` | 🔒 | Settings · Payouts |
| 11 | GET | `/explore` | 🔓 | Explore |
| 11 | GET | `/categories` | 🔓 | Explore filters |

\* Stripe webhook is unauthenticated but signature-verified.

---

## MVP priority

Build in this order to get a working loop fastest:

1. **Auth** (`/auth/*`) — signup/login.
2. **Public profile** (`GET /creators/:username`) + **Explore** — the read path.
3. **Donations** (`POST .../donations`) + **Stripe webhook** — the core money loop.
4. **Dashboard overview + supporters** — creators see their support.
5. **Posts** + **Settings** — content & profile management.
6. **Memberships** — recurring revenue (can ship after one-time tips work).
