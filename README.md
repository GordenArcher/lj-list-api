# LJ-List API

Backend API for LJ-List, a 3-month bulk grocery hire-purchase platform for government workers in Ghana.

Customers browse products, submit applications, and chat with admin. Payments are handled offline after manual review.

## API Version

All application endpoints are versioned under:

`/api/v1`

Runtime documentation endpoints:

- `GET /`
- `GET /api/v1/docs`

Repository docs:

- [API Reference](docs/API_REFERENCE.md)
- [Operations Guide](docs/OPERATIONS.md)

## Stack

- Go 1.22+
- Gin
- PostgreSQL (pgx v5)
- golang-migrate
- bcrypt + JWT (httpOnly cookies)
- Cloudinary (media)
- Hubtel (SMS)

## Setup

```bash
git clone https://github.com/GordenArcher/lj-list-api.git
cd lj-list-api
cp .env.example .env
go mod download
make run
```

## Environment Variables

| Variable | Description | Default |
|---|---|---|
| `PORT` | HTTP port | `8080` |
| `DATABASE_URL` | Postgres DSN | required |
| `JWT_SECRET` | JWT signing secret | required |
| `RATE_LIMIT_PER_MINUTE` | Global per-IP limit for `/api/*` | `120` |
| `AUTH_RATE_LIMIT_PER_MINUTE` | Per-IP auth route limit (`/api/v1/auth/*`) | `30` |
| `CORS_ALLOWED_ORIGINS` | Comma-separated CORS allowlist | `http://localhost:5173` |
| `COOKIE_SECURE` | Set auth cookies as `Secure` | `false` |
| `COOKIE_DOMAIN` | Cookie domain (`""` means host-only) | `""` |
| `COOKIE_SAME_SITE` | `Lax`, `Strict`, or `None` | `Lax` |
| `CLOUDINARY_CLOUD_NAME` | Cloudinary cloud name | required |
| `CLOUDINARY_API_KEY` | Cloudinary key | required |
| `CLOUDINARY_API_SECRET` | Cloudinary secret | required |
| `HUBTEL_CLIENT_ID` | Hubtel SMS client ID | required |
| `HUBTEL_CLIENT_SECRET` | Hubtel SMS client secret | required |
| `HUBTEL_SMS_URL` | Hubtel SMS send endpoint | `https://smsc.hubtel.com/v1/messages/send` |
| `HUBTEL_SENDER_ID` | SMS sender ID | `LJList` |
| `ADMIN_EMAIL` | Email auto-promoted to admin on signup | required |
| `ADMIN_NUMBER` | Admin phone number for SMS alerts | required for SMS notifications |
| `MIN_ORDER` | Minimum order amount in GHS | `549` |

## Response Envelope

Every endpoint returns one shape:

```json
{
  "status": "success",
  "message": "Products retrieved",
  "data": {},
  "code": "OK",
  "request_id": "a7de1f8a-6f6f-4de9-b78e-ff18c5de9e16",
  "metadata": {
    "timestamp": "2026-04-28T12:00:00Z"
  }
}
```

Error responses use the same envelope with `status = "error"` and `errors` map.

## Auth and Cookies

- Access token cookie: `access_token`
  - Path: `/`
  - TTL: 15 minutes
- Refresh token cookie: `refresh_token`
  - Path: `/api/v1/auth/refresh`
  - TTL: 7 days

`POST /api/v1/auth/refresh` is intentionally public because refresh cookie is the credential.

## Rate Limiting

- Global limiter applies to `/api/*`.
- Auth limiter applies to `/api/v1/auth/*`.

Headers returned:

- `X-RateLimit-Limit`
- `X-RateLimit-Remaining`
- `X-RateLimit-Reset`
- `X-RateLimit-Scope`
- `Retry-After` (when limited)

## CORS

Only origins in `CORS_ALLOWED_ORIGINS` are accepted. Default local frontend origin is:

`http://localhost:5173`

## Route Groups

Public:

- `POST /api/v1/auth/signup`
- `POST /api/v1/auth/login`
- `POST /api/v1/auth/refresh`
- `GET /api/v1/products`
- `GET /api/v1/products/categories`

Authenticated customer:

- `POST /api/v1/auth/logout`
- `GET /api/v1/profile`
- `PATCH /api/v1/profile`
- `POST /api/v1/applications`
- `GET /api/v1/applications`
- `GET /api/v1/applications/:id`
- `GET /api/v1/conversations`
- `POST /api/v1/conversations`
- `GET /api/v1/conversations/:id/messages`
- `POST /api/v1/conversations/:id/messages`

Authenticated admin:

- `GET /api/v1/admin/users`
- `PATCH /api/v1/admin/users/:id`
- `POST /api/v1/admin/products`
- `PATCH /api/v1/admin/products/:id`
- `GET /api/v1/admin/products/:id/images`
- `POST /api/v1/admin/products/:id/images`
- `DELETE /api/v1/admin/products/:id/images/:imageId`
- `GET /api/v1/admin/applications`
- `PATCH /api/v1/admin/applications/:id`
- `GET /api/v1/admin/conversations`
- `POST /api/v1/admin/conversations/:id/messages`

## Project Structure

```text
cmd/server/main.go
internal/config
internal/database
internal/handlers
internal/middleware
internal/models
internal/repositories
internal/routes
internal/services
internal/utils
docs/
```

## License

Proprietary. All rights reserved.
