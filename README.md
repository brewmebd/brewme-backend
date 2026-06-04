<div align="center">

# ☕ BrewMe — Backend

### **The API that pours the coffee.**

The **Go** REST API powering BrewMe accounts, creator pages, coffee tips,
memberships, posts, and the creator dashboard.

![Go](https://img.shields.io/badge/Go-1.25-1A1A1A?style=for-the-badge&logo=go&logoColor=F5C518)
![chi](https://img.shields.io/badge/chi-router-1A1A1A?style=for-the-badge&logoColor=F5C518)
![MySQL](https://img.shields.io/badge/MySQL-8.0-1A1A1A?style=for-the-badge&logo=mysql&logoColor=F5C518)

</div>

---

## Overview

A pragmatic MVP backend. Each request flows through a thin, easy-to-follow
pipeline — no heavy framework, just `net/http` + [chi](https://github.com/go-chi/chi):

```
request → router → middleware (cors, logging, throttle) → handler → MySQL
                                                              ↑
                                            validation + SQL live here
```

Handlers own validation and SQL directly (the service/repository layers were
collapsed for MVP speed). The database pool is a shared package global
(`database.DB`).

---

## Tech stack

- **Go** with the standard `net/http`
- **chi** router + CORS middleware (logger, recoverer, real-IP, 60s timeout, throttle 100)
- **MySQL** via `go-sql-driver/mysql`
- **bcrypt** (`golang.org/x/crypto`) for password hashing
- **golang.org/x/net/html** for input sanitization
- **godotenv** for local `.env` loading

---

## Project structure

```
brewme-backend/
├── cmd/
│   └── server/
│       └── main.go          # entrypoint: load .env → open DB → start server (:8080)
├── internal/
│   ├── database/            # MySQL pool (shared global DB, 10 max conns)
│   ├── router/              # routes + middleware, serves /uploads/*
│   ├── handler/             # HTTP handlers (validation + SQL inline)
│   ├── middleware/          # HTML sanitization helpers
│   ├── model/               # domain structs (User, RegisterRequest, …)
│   ├── utils/               # helpers (bcrypt password hashing)
│   └── health/              # GET /health
├── database/
│   └── brew.sql             # full schema + seed data (DROPs & recreates the DB)
├── migrations/              # reserved for versioned migrations
├── uploads/avatar/          # stored avatar images (gitignored)
├── API.md                   # full REST API specification
├── BACKEND_ROADMAP.md       # build plan & status
├── BrewMe.postman_collection.json
├── .env.example
└── go.mod                   # module brewme (Go 1.25)
```

---

## Getting started

### 1. Database

Load the schema (creates the `brewme` database, tables, views, and seed data):

```bash
mysql -u root -p < database/brewme.sql
```

> ⚠️ `brew.sql` drops and recreates the database — don't run it against data you
> want to keep.

### 2. Configure

```bash
cp .env.example .env
```

```ini
APP_ENV=development
PORT=8080
DATABASE_DSN=root:yourpassword@tcp(127.0.0.1:3306)/brewme?parseTime=true&charset=utf8mb4
JWT_SECRET=change-me
CORS_ALLOW_ORIGIN=http://localhost:5173
```

> `DATABASE_DSN` is the only required variable — the server exits if it's unset.
> The server currently binds `:8080` directly (the `PORT` var is reserved for later).

### 3. Run

Run from the **backend root** so `.env` and `uploads/` resolve correctly
(`uploads/` is created relative to the working directory):

```bash
go run ./cmd/server
```

Health check → **http://localhost:8080/health** → `{"message": "success"}`

---

## API

Base URL: **`/api/v1`** · full spec in [`API.md`](API.md) · ready-to-import requests in [`BrewMe.postman_collection.json`](BrewMe.postman_collection.json)

### Implemented today

| Method | Path | Auth | Notes |
|---|---|---|---|
| `GET` | `/health` | 🔓 | Liveness probe |
| `POST` | `/api/v1/auth/register` | 🔓 | **`multipart/form-data`** signup with optional avatar |
| `GET` | `/uploads/*` | 🔓 | Serves uploaded avatars from local disk |

> The rest of the surface (login, creator pages, donations, dashboard,
> memberships, explore) is specified in [`API.md`](API.md) and built out phase by
> phase — see the [roadmap](#roadmap).

### `POST /api/v1/auth/register` fields (`multipart/form-data`)

| Field | Required | Notes |
|---|:---:|---|
| `name` | ✅ | Full name (HTML-sanitized) |
| `email` | ✅ | Unique, lowercased |
| `password` | ✅ | Min 8 characters |
| `confirmPassword` | ✅ | Must match `password` |
| `url` | ✅ | Page username (`brewme.com/{url}`) |
| `bio` | | Optional (HTML-sanitized) |
| `category` | | Category **slug**, e.g. `digital-art` |
| `avatar` | | JPG / PNG / GIF, ≤ 2 MB |

**Success — `201 Created`**
```json
{
  "status": true,
  "message": "User registration successful",
  "user": { "id": 1, "full_name": "Sarah Chen", "username": "sarahchen", "email": "sarah@example.com", "avatar_url": "/uploads/avatar/<file>.png" }
}
```

**Validation error — `400`**
```json
{ "status": false, "error": "validation_error", "fields": { "email": "email is required" } }
```

Duplicate email/username returns `409` (`status: false`).

---

## Avatar uploads

Uploaded avatars are validated by sniffing the real content type (first 512
bytes) and capping size at 2 MB, then stored under
`uploads/avatar/<random>.ext`. The public path `/uploads/avatar/<file>` is saved
to `users.avatar_url`. Build the absolute URL on the client as
`http://localhost:8080{avatar_url}`. If the row insert fails after a file is
saved, the file is rolled back from disk.

---

## Roadmap

Build order (see [`BACKEND_ROADMAP.md`](BACKEND_ROADMAP.md)):

1. **Auth** — signup ✅ / login
2. **Public profile** + **Explore** (read path)
3. **Donations** + payments (the money loop)
4. **Dashboard** — overview & supporters
5. **Posts** & **Settings**
6. **Memberships** — recurring support

---

<div align="center">

Brewed with ☕ in Go.

</div>
