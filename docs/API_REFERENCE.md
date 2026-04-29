# API Reference (v1)

Base path: `/api/v1`

All responses use the same envelope:

```json
{
  "status": "success",
  "message": "Readable message",
  "data": {},
  "code": "OK",
  "request_id": "uuid",
  "metadata": {
    "timestamp": "RFC3339 UTC"
  }
}
```

Error responses use `status: "error"` and include `errors`:

```json
{
  "status": "error",
  "message": "Validation failed",
  "code": "VALIDATION_ERROR",
  "errors": {
    "email": ["valid email is required"]
  },
  "request_id": "uuid",
  "metadata": {
    "timestamp": "RFC3339 UTC"
  }
}
```

## Authentication

Auth is cookie-first:

- `access_token` cookie (15m, path `/`)
- `refresh_token` cookie (7d, path `/api/v1/auth`)

Authorization header is also accepted:

`Authorization: Bearer <access_token>`

## Pagination Contract

List endpoints accept:

- `page` (default `1`)
- `limit` (default `20`, max `100`)

List responses include:

```json
{
  "meta": {
    "total": 87,
    "page": 1,
    "limit": 20,
    "total_pages": 5,
    "has_next": true,
    "has_prev": false
  }
}
```

## Auth Endpoints

### POST `/auth/signup`

Request body:

```json
{
  "email": "kwame@email.com",
  "password": "password123",
  "display_name": "Kwame"
}
```

Success `201` data:

```json
{
  "user": {
    "id": "uuid",
    "email": "kwame@email.com",
    "display_name": "Kwame",
    "phone": null,
    "role": "customer"
  }
}
```

### POST `/auth/login`

Request body:

```json
{
  "email": "kwame@email.com",
  "password": "password123"
}
```

Success `200` data:

```json
{
  "user": {
    "id": "uuid",
    "email": "kwame@email.com",
    "display_name": "Kwame",
    "phone": null,
    "role": "customer"
  }
}
```

### POST `/auth/refresh`

Credential: `refresh_token` cookie.

Success `200` data:

```json
{
  "access_token": "<jwt>",
  "refresh_token": "<jwt>"
}
```

### POST `/auth/logout`

Auth required.

Success `200` data: `null`

## Profile Endpoints

### GET `/profile`

Auth required.

Success `200` data:

```json
{
  "user": {
    "id": "uuid",
    "email": "kwame@email.com",
    "display_name": "Kwame",
    "phone": "0240000000",
    "role": "customer",
    "created_at": "2026-04-29T12:00:00Z",
    "updated_at": "2026-04-29T12:00:00Z"
  }
}
```

### PATCH `/profile`

Auth required.

Request body may include any subset of:

```json
{
  "display_name": "Kwame Mensah",
  "phone": "0240000000"
}
```

Send `"phone": ""` to clear the saved phone number.

Success `200` data: same `user` object as above.

## Product Endpoints

### GET `/products`

Query:

- `category` optional
- `page` optional
- `limit` optional

Success `200` data:

```json
{
  "products": [
    {
      "id": "uuid",
      "name": "Royal Aroma Rice 5kg",
      "category": "Rice & Grains",
      "price": 120,
      "image_url": "https://...",
      "images": [
        {
          "id": "uuid",
          "product_id": "uuid",
          "image_url": "https://...",
          "created_at": "2026-04-29T12:00:00Z"
        }
      ],
      "unit": "bag",
      "active": true
    }
  ],
  "meta": {
    "total": 125,
    "page": 1,
    "limit": 20,
    "total_pages": 7,
    "has_next": true,
    "has_prev": false
  }
}
```

### GET `/products/categories`

Success `200` data:

```json
{
  "categories": ["Rice & Grains", "Cooking Oil", "Beverages"]
}
```

## Admin Product Endpoints

### POST `/admin/products`

Auth required. Admin only.

Request content type: `application/json`

Fields:

- `name` required
- `category` required
- `price` required, integer greater than 0
- `unit` required
- `active` optional, defaults to `true`

Success `201` data:

```json
{
  "id": "uuid",
  "name": "Royal Aroma Rice 5kg",
  "category": "Rice & Grains",
  "price": 120,
  "image_url": "",
  "images": [],
  "unit": "bag",
  "active": true
}
```

### PATCH `/admin/products/:id`

Auth required. Admin only.

Request content type: `application/json`

Any subset of fields may be provided:

- `name` optional
- `category` optional
- `price` optional, integer greater than 0
- `unit` optional
- `active` optional, `true` or `false`

Success `200` data:

```json
{
  "id": "uuid",
  "name": "Royal Aroma Rice 5kg",
  "category": "Rice & Grains",
  "price": 125,
  "image_url": "https://res.cloudinary.com/...",
  "images": [
    {
      "id": "uuid",
      "product_id": "uuid",
      "image_url": "https://res.cloudinary.com/...",
      "created_at": "2026-04-29T12:00:00Z"
    }
  ],
  "unit": "bag",
  "active": true
}
```

### GET `/admin/products/:id/images`

Auth required. Admin only.

Success `200` data:

```json
{
  "images": [
    {
      "id": "uuid",
      "product_id": "uuid",
      "image_url": "https://res.cloudinary.com/...",
      "created_at": "2026-04-29T12:00:00Z"
    }
  ]
}
```

### POST `/admin/products/:id/images`

Auth required. Admin only.

Request content type: `multipart/form-data`

Fields:

- `images` one or more image files

You may also send a single `image` field for one file.

Success `201` data:

```json
{
  "images": [
    {
      "id": "uuid",
      "product_id": "uuid",
      "image_url": "https://res.cloudinary.com/...",
      "created_at": "2026-04-29T12:00:00Z"
    }
  ]
}
```

### DELETE `/admin/products/:id/images/:imageId`

Auth required. Admin only.

Deletes the product image record and destroys the underlying Cloudinary asset.

## Application Endpoints

### POST `/applications`

Auth required.

Request body:

```json
{
  "package_type": "custom",
  "package_name": "",
  "cart_items": [
    { "product_id": "uuid", "quantity": 2 },
    { "product_id": "uuid", "quantity": 5 }
  ],
  "staff_number": "GS123456",
  "mandate_number": "MND-001",
  "institution": "Ghana Health Service",
  "ghana_card_number": "GHA-000-0000-0"
}
```

Success `201` data: full `application` object (`id`, `user_id`, `package_type`, `cart_items`, `total_amount`, `monthly_amount`, `status`, `staff_number`, `mandate_number`, `institution`, `ghana_card_number`, `created_at`, `updated_at`).

### GET `/applications`

Auth required. Paginated list for the logged-in user.

Success `200` data:

```json
{
  "applications": [ { "id": "uuid", "status": "pending" } ],
  "meta": {
    "total": 12,
    "page": 1,
    "limit": 20,
    "total_pages": 1,
    "has_next": false,
    "has_prev": false
  }
}
```

### GET `/applications/:id`

Auth required. Returns one application owned by current user.

## Conversation Endpoints

### POST `/conversations`

Auth required.

Request body:

```json
{
  "message": "I need help with my application"
}
```

Creates the conversation and stores the initial message only the first time.
If the conversation already exists, this endpoint returns the existing
conversation and does not append the message again. Use
`POST /conversations/:id/messages` for follow-up messages.

Success `201` data: conversation object when newly created.

Success `200` data: existing conversation object when it already exists.

### GET `/conversations`

Auth required. Paginated list for current user.

### GET `/conversations/:id/messages`

Auth required. Paginated messages in chronological order.

### POST `/conversations/:id/messages`

Auth required.

Request body:

```json
{
  "content": "Thank you for the update"
}
```

Success `201` data: message object (`id`, `conversation_id`, `sender_id`, `content`, `read_at`, `created_at`).

## Admin Endpoints

All admin endpoints require authenticated user with role `admin`.

### GET `/admin/users`

Query:

- `role` optional, `customer` or `admin`
- `page` optional
- `limit` optional

Success `200` data:

```json
{
  "users": [
    {
      "id": "uuid",
      "email": "kwame@email.com",
      "display_name": "Kwame",
      "phone": "0240000000",
      "role": "customer",
      "created_at": "2026-04-29T12:00:00Z",
      "updated_at": "2026-04-29T12:00:00Z"
    }
  ],
  "meta": {
    "total": 1,
    "page": 1,
    "limit": 20,
    "total_pages": 1,
    "has_next": false,
    "has_prev": false
  }
}
```

### PATCH `/admin/users/:id`

Request body may include any subset of:

```json
{
  "display_name": "Kwame Mensah",
  "phone": "0240000000",
  "role": "customer"
}
```

Allowed roles: `customer`, `admin`.

Success `200` data:

```json
{
  "user": {
    "id": "uuid",
    "email": "kwame@email.com",
    "display_name": "Kwame Mensah",
    "phone": "0240000000",
    "role": "customer",
    "created_at": "2026-04-29T12:00:00Z",
    "updated_at": "2026-04-29T12:30:00Z"
  }
}
```

### GET `/admin/applications`

Query:

- `status` optional
- `page` optional
- `limit` optional

Returns paginated applications across all users. Unlike the customer-facing
application list, each item also includes a `customer` object so the admin UI
can show who submitted it without making extra user lookups.

Success `200` data:

```json
{
  "applications": [
    {
      "id": "uuid",
      "user_id": "uuid",
      "customer": {
        "id": "uuid",
        "email": "kwame@email.com",
        "display_name": "Kwame",
        "phone": "0240000000",
        "role": "customer"
      },
      "package_type": "custom",
      "package_name": "",
      "cart_items": [],
      "total_amount": 650,
      "monthly_amount": 217,
      "status": "pending",
      "staff_number": "GS123456",
      "mandate_number": "MND-001",
      "institution": "Ghana Health Service",
      "ghana_card_number": "GHA-000-0000-0",
      "created_at": "2026-04-29T12:00:00Z",
      "updated_at": "2026-04-29T12:00:00Z"
    }
  ],
  "meta": {
    "total": 1,
    "page": 1,
    "limit": 20,
    "total_pages": 1,
    "has_next": false,
    "has_prev": false
  }
}
```

### PATCH `/admin/applications/:id`

Request body:

```json
{
  "status": "approved"
}
```

Allowed statuses: `pending`, `reviewed`, `approved`, `declined`.

### GET `/admin/conversations`

Paginated conversation list for admin.

### POST `/admin/conversations/:id/messages`

Request body:

```json
{
  "content": "Your application has been approved"
}
```

## Documentation Endpoints

- `GET /` returns generated API docs.
- `GET /api/v1/docs` returns generated API docs under versioned path.
