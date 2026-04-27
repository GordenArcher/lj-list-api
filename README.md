# LJ-List API

Backend API for **LJ-List**, a 3-month bulk grocery hire-purchase platform serving government workers in Ghana.

Customers browse pre-packaged grocery bundles or build custom orders, submit applications, and communicate with the store owner via a one-on-one chat system. No online payment processing — all applications are reviewed and approved manually.

## Stack

| Layer        | Technology                                      |
|-------------|-------------------------------------------------|
| Runtime      | Go 1.22+                                        |
| HTTP         | [Gin](https://github.com/gin-gonic/gin)         |
| Database     | PostgreSQL 15+ via [pgx v5](https://github.com/jackc/pgx) |
| Migrations   | [golang-migrate](https://github.com/golang-migrate/migrate) |
| Auth         | bcrypt + JWT (cookie-based, httpOnly)           |
| File storage | [Cloudinary](https://cloudinary.com)            |
| SMS          | [Hubtel](https://hubtel.com)                    |
| Env loading  | [godenv](https://github.com/GordenArcher/godenv) |
| UUID         | [google/uuid](https://github.com/google/uuid)   |

## Architecture

```
Handler → Service → Repository → Database
```

- **Handlers** parse HTTP requests, call services, return responses.
- **Services** hold business logic. Handlers never touch the database.
- **Repositories** run SQL via pgx. Services never write queries.
- **Models** are plain structs shared across layers.
- **Utils** are reusable, stateless functions (hashing, tokens, validation).

Every response uses a single envelope:

```json
{
  "status": "success",
  "message": "...",
  "data": {},
  "code": "OK",
  "request_id": "uuid",
  "metadata": {
    "timestamp": "2026-04-27T10:30:00Z"
  }
}
```

## Getting Started

### Prerequisites

- Go 1.22+
- PostgreSQL 15+
- make (optional)

### Setup

```bash
git clone https://github.com/GordenArcher/lj-list-api.git
cd lj-list-api

cp .env.example .env
# Fill in your database URL, Cloudinary credentials, Hubtel API key, etc.

go mod download
make run
```

### Environment Variables

| Variable              | Description                              | Default               |
|-----------------------|------------------------------------------|-----------------------|
| `PORT`                | HTTP server port                         | `8080`                |
| `DATABASE_URL`        | PostgreSQL connection string             | *required*            |
| `JWT_SECRET`          | Secret key for signing JWTs              | *required*            |
| `CLOUDINARY_CLOUD_NAME` | Cloudinary cloud name                 | *required*            |
| `CLOUDINARY_API_KEY`  | Cloudinary API key                       | *required*            |
| `CLOUDINARY_API_SECRET` | Cloudinary API secret                  | *required*            |
| `HUBTEL_API_KEY`      | Hubtel SMS API key                       | *required*            |
| `HUBTEL_SENDER_ID`    | SMS sender ID                            | `LJList`              |
| `ADMIN_EMAIL`         | Email to auto-promote as admin on signup | *required*            |
| `MIN_ORDER`           | Minimum order amount in cedis            | `549`                 |

### Migrations

```bash
make migrate-up      # Run pending migrations
make migrate-down    # Rollback last migration
make migrate-create  # Create a new migration file
```

### API Documentation

Once running, visit `GET /` for a self-documenting API reference with curl examples.

## Project Structure

```
lj-list-api/
├── cmd/server/main.go          # Entrypoint
├── internal/
│   ├── config/config.go        # Env loading & config struct
│   ├── database/
│   │   ├── postgres.go         # Connection pool
│   │   └── migrations/         # SQL migration files
│   ├── models/                 # Domain structs & response envelope
│   ├── handlers/               # HTTP handlers (Gin)
│   ├── services/               # Business logic
│   ├── repositories/           # Database queries (pgx)
│   ├── middleware/             # Auth, request ID, CORS
│   ├── routes/routes.go        # Route registration
│   └── utils/                  # Reusable helpers
├── .env.example
├── go.mod
├── go.sum
└── Makefile
```

## Key Design Decisions

**Cookie-based auth, not Bearer tokens.** The JWT lives in an httpOnly, Secure, SameSite cookie. The frontend never touches the token — it can't leak via JavaScript. This also means no token management logic in the React app.

**All IDs are UUIDs.** No auto-increment integers. Predictable, merge-safe, and no information leakage about record counts.

**One envelope for every response.** The frontend writes a single response parser. Error responses have `data` omitted. Success responses have `errors` omitted. The `status` field is always `"success"` or `"error"` — nothing else.

**Cloudinary as default storage.** Product images, chat attachments, and any future file uploads all go through a single `storage` package. Swapping to S3 or local storage later means changing one package, not every handler.

**Fire-and-forget SMS.** Notifications are sent via goroutine. The API does not wait for Hubtel to respond — if SMS fails, it logs and moves on. This keeps response times predictable.

**No payment processing.** This is intentional. LJ-List operates on a manual approval model. Applications are reviewed by the admin, who contacts the customer directly.

## License

Proprietary. All rights reserved.
