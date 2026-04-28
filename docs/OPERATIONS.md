# Operations Guide

This document covers runtime behavior that impacts deployment and frontend integration.

## Versioning

- API base is `/api/v1`.
- Keep `v1` stable; introduce breaking changes in a new major path (`/api/v2`).

## Security Model

## JWT and Cookie Rules

- Access token cookie:
  - Name: `access_token`
  - Path: `/`
  - TTL: 15 minutes
- Refresh token cookie:
  - Name: `refresh_token`
  - Path: `/api/v1/auth/refresh`
  - TTL: 7 days

Cookies are `httpOnly`. `Secure`, `Domain`, and `SameSite` are controlled by config.

## Role Checks

- `AuthRequired` validates access token and sets user context.
- `AdminRequired` enforces `role == "admin"`.

## Rate Limiting

Two independent per-IP limiters are active:

- Global limiter on `/api/*` using `RATE_LIMIT_PER_MINUTE`.
- Auth limiter on `/api/v1/auth/*` using `AUTH_RATE_LIMIT_PER_MINUTE`.

Headers emitted:

- `X-RateLimit-Limit`
- `X-RateLimit-Remaining`
- `X-RateLimit-Reset`
- `X-RateLimit-Scope` (`global` or `auth`)
- `Retry-After` (on 429)

## CORS

- Incoming `Origin` must be in `CORS_ALLOWED_ORIGINS`.
- Default development origin: `http://localhost:5173`.
- Credentials are enabled (`Access-Control-Allow-Credentials: true`).

## Response Contract

All endpoints return the same top-level envelope:

- `status`
- `message`
- `data` (success)
- `errors` (failure)
- `code`
- `request_id`
- `metadata.timestamp`

This allows one response parser across frontend routes.

## Pagination

Standard query params:

- `page` (default `1`)
- `limit` (default `20`, max `100`)

Standard metadata keys:

- `total`
- `page`
- `limit`
- `total_pages`
- `has_next`
- `has_prev`

## Deployment Checklist

- Set `JWT_SECRET` to a strong random value.
- Set production `CORS_ALLOWED_ORIGINS`.
- Set `COOKIE_SECURE=true` behind HTTPS.
- Set `COOKIE_SAME_SITE=None` only when cross-site cookies are required and HTTPS is enabled.
- Set `ADMIN_EMAIL` before first admin signup.
- Validate `DATABASE_URL` and run migrations on startup.

## Local Testing Checklist

- Start server and verify docs:
  - `GET /`
  - `GET /api/v1/docs`
- Verify auth flow:
  - signup -> login -> refresh -> logout
- Verify throttling:
  - burst auth requests until `429` and confirm `X-RateLimit-Scope: auth`
  - burst API list requests and confirm `X-RateLimit-Scope: global`
