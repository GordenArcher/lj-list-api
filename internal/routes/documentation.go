package routes

import (
	"net/http"

	"github.com/GordenArcher/lj-list-api/internal/models"
)

func buildAPIDocumentation() models.APIResponse {
	// ServiceInfo describes the LJ-List API and its operational constraints.
	// These values are not configuration, they are hard constraints that
	// the frontend must understand to integrate correctly. If min_order
	// changes, both the API and database schema must be updated together.
	doc := models.APIDocumentation{
		Service:     "LJ-List API",
		Version:     "1.0.0",
		Description: "3-month bulk grocery payment plan for government workers in Ghana",
		Routes: []models.RouteDoc{
			// Documentation Endpoints
			{
				Method:      http.MethodGet,
				Path:        "/",
				Description: "Root documentation endpoint that returns generated API docs in the standard response envelope.",
				Auth:        false,
				Example:     `curl http://localhost:8080/`,
			},
			{
				Method:      http.MethodGet,
				Path:        "/api/v1/docs",
				Description: "Versioned documentation endpoint for frontend clients already targeting /api/v1.",
				Auth:        false,
				Example:     `curl http://localhost:8080/api/v1/docs`,
			},

			// Authentication Endpoints
			{
				Method:      http.MethodPost,
				Path:        "/api/v1/auth/signup",
				Description: "Create a new inactive customer or admin account. phone_number, staff_number, institution, ghana_card_number, password, and display_name/name are required. The account remains inactive until the phone owner verifies the OTP sent by SMS. If the submitted phone_number matches ADMIN_PHONE_NUMBER, the created account gets role='admin'.",
				Auth:        false,
				Request: map[string]string{
					"phone_number":      "string (required, valid phone number)",
					"staff_number":      "string (required, unique)",
					"institution":       "string (required)",
					"ghana_card_number": "string (required, unique)",
					"password":          "string (minimum 8 characters)",
					"display_name":      "string (required, 2-100 characters)",
					"name":              "string (optional alias for display_name)",
				},
				ResponseSuccess: models.NewSuccessResponse("req-abc123", "Account created. Activation OTP sent.", map[string]any{
					"user": map[string]any{
						"id":           "550e8400-e29b-41d4-a716-446655440000",
						"display_name": "Kwame Mensah",
						"phone_number": "+233240000000",
						"role":         "customer",
					},
					"verification": map[string]any{
						"phone_number":       "+233240000000",
						"expires_in_minutes": 10,
					},
				}),
				ResponseError: models.NewErrorResponse("req-abc123", "VALIDATION_ERROR", "Validation failed", map[string][]string{
					"phone_number": {"must be a valid phone number"},
					"password":     {"must be at least 8 characters"},
					"display_name": {"must be between 2 and 100 characters"},
				}),
				ErrorExamples: map[string]models.APIResponse{
					"duplicate_phone_number": models.NewErrorResponse("req-abc123", "CONFLICT", "Phone number already registered", map[string][]string{
						"phone_number": {"this phone number is already taken"},
					}),
					"duplicate_staff_number": models.NewErrorResponse("req-abc123", "CONFLICT", "Staff number already registered", map[string][]string{
						"staff_number": {"this staff number is already taken"},
					}),
					"duplicate_ghana_card_number": models.NewErrorResponse("req-abc123", "CONFLICT", "Ghana Card number already registered", map[string][]string{
						"ghana_card_number": {"this Ghana Card number is already taken"},
					}),
					"validation_error": models.NewErrorResponse("req-abc123", "VALIDATION_ERROR", "Validation failed", map[string][]string{
						"phone_number": {"must be a valid phone number"},
						"password":     {"must be at least 8 characters"},
						"display_name": {"must be between 2 and 100 characters"},
					}),
				},
				Example: `curl -X POST http://localhost:8080/api/v1/auth/signup \
  -H "Content-Type: application/json" \
  -d '{
    "phone_number":"+233240000000",
    "staff_number":"GES-2024-0018",
    "institution":"Ghana Education Service",
    "ghana_card_number":"GHA-123456789-0",
    "password":"password123",
    "display_name":"Kwame Mensah"
  }'`,
			},
			{
				Method:      http.MethodPost,
				Path:        "/api/v1/auth/verify-otp",
				Description: "Verify the OTP sent after signup, activate the account, and issue auth cookies.",
				Auth:        false,
				Request: map[string]string{
					"phone_number": "string (required, valid phone number)",
					"otp":          "string (required, 6-digit code)",
				},
				ResponseSuccess: models.NewSuccessResponse("req-abc123", "Account activated successfully", map[string]any{
					"user": map[string]any{
						"id":           "550e8400-e29b-41d4-a716-446655440000",
						"display_name": "Kwame Mensah",
						"phone_number": "+233240000000",
						"role":         "customer",
					},
				}),
				ResponseError: models.NewErrorResponse("req-abc123", "VALIDATION_ERROR", "Invalid or expired OTP", map[string][]string{
					"otp": {"otp is invalid or expired"},
				}),
				Example: `curl -X POST http://localhost:8080/api/v1/auth/verify-otp \
  -H "Content-Type: application/json" \
  -c cookies.txt \
  -d '{
    "phone_number":"+233240000000",
    "otp":"123456"
  }'`,
			},
			{
				Method:      http.MethodPost,
				Path:        "/api/v1/auth/resend-otp",
				Description: "Generate and send a fresh activation OTP for an inactive account.",
				Auth:        false,
				Request: map[string]string{
					"phone_number": "string (required, valid phone number)",
				},
				ResponseSuccess: models.NewSuccessResponse("req-abc123", "Activation OTP sent", map[string]any{
					"verification": map[string]any{
						"phone_number":       "+233240000000",
						"expires_in_minutes": 10,
					},
				}),
				Example: `curl -X POST http://localhost:8080/api/v1/auth/resend-otp \
  -H "Content-Type: application/json" \
  -d '{
    "phone_number":"+233240000000"
  }'`,
			},
			{
				Method:      http.MethodPost,
				Path:        "/api/v1/auth/login",
				Description: "Authenticate an already activated account with phone_number and password. Returns an access_token cookie (15 min expiry) and refresh_token cookie (7 days expiry). Both are httpOnly for XSS protection.",
				Auth:        false,
				Request: map[string]string{
					"phone_number": "string (required, valid phone number)",
					"password":     "string (required)",
				},
				ResponseSuccess: models.NewSuccessResponse("req-abc123", "Login successful", map[string]any{
					"user": map[string]any{
						"id":           "550e8400-e29b-41d4-a716-446655440000",
						"display_name": "Kwame Mensah",
						"phone_number": "+233240000000",
						"role":         "customer",
					},
				}),
				ResponseError: models.NewErrorResponse("req-abc123", "UNAUTHORIZED", "Invalid phone number or password", map[string][]string{
					"auth": {"phone number or password is incorrect"},
				}),
				ErrorExamples: map[string]models.APIResponse{
					"invalid_credentials": models.NewErrorResponse("req-abc123", "UNAUTHORIZED", "Invalid phone number or password", map[string][]string{
						"auth": {"phone number or password is incorrect"},
					}),
					"account_not_activated": models.NewErrorResponse("req-abc123", "FORBIDDEN", "Account not activated", map[string][]string{
						"auth": {"verify the activation OTP sent to your phone number"},
					}),
					"validation_error": models.NewErrorResponse("req-abc123", "VALIDATION_ERROR", "Validation failed", map[string][]string{
						"phone_number": {"must be a valid phone number"},
						"password":     {"required"},
					}),
				},
				Example: `curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -c cookies.txt \
  -d '{
    "phone_number":"+233240000000",
    "password":"password123"
  }'`,
			},
			{
				Method:      http.MethodPost,
				Path:        "/api/v1/admin/auth/login",
				Description: "Authenticate an activated admin account only. Customer accounts receive a forbidden response so the admin frontend has a clear entry point.",
				Auth:        false,
				AdminOnly:   false,
				Request: map[string]string{
					"phone_number": "string (required, valid phone number)",
					"password":     "string (required)",
				},
				Example: `curl -X POST http://localhost:8080/api/v1/admin/auth/login \
  -H "Content-Type: application/json" \
  -c cookies.txt \
  -d '{
    "phone_number":"+233240000000",
    "password":"password123"
  }'`,
			},
			{
				Method:      http.MethodGet,
				Path:        "/api/v1/admin/dashboard",
				Description: "Get admin overview totals and a date-range chart. Supported range values: today, week, month, or custom with from/to dates in YYYY-MM-DD format.",
				Auth:        true,
				AdminOnly:   true,
				Request: map[string]string{
					"range": "string (optional: today, week, month, custom)",
					"from":  "string (required when range=custom, YYYY-MM-DD)",
					"to":    "string (required when range=custom, YYYY-MM-DD)",
				},
				ResponseSuccess: models.NewSuccessResponse("req-abc123", "Dashboard stats retrieved", map[string]any{
					"range":               "month",
					"from":                "2026-04-06",
					"to":                  "2026-05-05",
					"total_users":         142,
					"total_products":      16,
					"total_applications":  38,
					"total_conversations": 21,
					"total_messages":      84,
					"series": []map[string]any{
						{
							"date":          "2026-05-01",
							"users":         2,
							"products":      1,
							"applications":  4,
							"conversations": 3,
							"messages":      9,
						},
						{
							"date":          "2026-05-02",
							"users":         1,
							"products":      0,
							"applications":  2,
							"conversations": 1,
							"messages":      6,
						},
					},
				}),
				Example: `curl 'http://localhost:8080/api/v1/admin/dashboard?range=month' \
  -b cookies.txt`,
			},
			{
				Method:      http.MethodPost,
				Path:        "/api/v1/auth/refresh",
				Description: "Exchange a valid refresh_token cookie for a fresh token pair.",
				Auth:        false,
				Request: map[string]any{
					"cookies": map[string]string{
						"refresh_token": "string (required, httpOnly cookie)",
					},
				},
				ResponseSuccess: models.NewSuccessResponse("req-abc123", "Tokens refreshed", map[string]any{
					"access_token":  "<jwt>",
					"refresh_token": "<jwt>",
				}),
				Example: `curl -X POST http://localhost:8080/api/v1/auth/refresh \
  -b cookies.txt \
  -c cookies.txt`,
			},
			{
				Method:      http.MethodPost,
				Path:        "/api/v1/auth/logout",
				Description: "Clear both access_token and refresh_token cookies. After logout, authenticated endpoints return 401.",
				Auth:        true,
				Example: `curl -X POST http://localhost:8080/api/v1/auth/logout \
  -b cookies.txt`,
			},
			{
				Method:      http.MethodGet,
				Path:        "/api/v1/profile",
				Description: "Get the authenticated user's profile. Returns the editable account fields plus role and timestamps.",
				Auth:        true,
				Response: map[string]any{
					"user": map[string]any{
						"id":                "550e8400-e29b-41d4-a716-446655440000",
						"display_name":      "Kwame",
						"phone_number":      "+233240000000",
						"staff_number":      "GES-2024-0018",
						"institution":       "Ghana Education Service",
						"ghana_card_number": "GHA-123456789-0",
						"role":              "customer",
						"created_at":        "2026-04-29T12:00:00Z",
						"updated_at":        "2026-04-29T12:00:00Z",
					},
				},
				Example: `curl http://localhost:8080/api/v1/profile \
  -b cookies.txt`,
			},
			{
				Method:      http.MethodPatch,
				Path:        "/api/v1/profile",
				Description: "Update the authenticated user's editable account fields. uuid, role, activation state, and OTP metadata are not client-editable.",
				Auth:        true,
				Request: map[string]string{
					"display_name":      "string (optional, 2-100 characters)",
					"phone_number":      "string (optional, valid phone number)",
					"staff_number":      "string (optional, unique, non-empty)",
					"institution":       "string (optional, non-empty)",
					"ghana_card_number": "string (optional, unique, non-empty)",
					"password":          "string (optional, minimum 8 characters)",
				},
				Response: map[string]any{
					"user": map[string]any{
						"id":                "550e8400-e29b-41d4-a716-446655440000",
						"display_name":      "Kwame Mensah",
						"phone_number":      "+233240000000",
						"staff_number":      "GES-2024-0018",
						"institution":       "Ghana Education Service",
						"ghana_card_number": "GHA-123456789-0",
						"role":              "customer",
						"created_at":        "2026-04-29T12:00:00Z",
						"updated_at":        "2026-04-29T12:30:00Z",
					},
				},
				Example: `curl -X PATCH http://localhost:8080/api/v1/profile \
  -H "Content-Type: application/json" \
  -b cookies.txt \
  -d '{
    "display_name":"Kwame Mensah",
    "phone_number":"+233240000000",
    "staff_number":"GES-2024-0018",
    "institution":"Ghana Education Service",
    "ghana_card_number":"GHA-123456789-0",
    "password":"password123"
  }'`,
			},

			// Product Endpoints
			{
				Method:      http.MethodGet,
				Path:        "/api/v1/products",
				Description: "List active products with optional category filtering. category is the category UUID from /api/v1/products/categories, though the backend still resolves category names during the transition. This endpoint is paginated using page and limit query params.",
				Auth:        false,
				Request: map[string]string{
					"category": "string (optional, category UUID; category names are accepted temporarily for compatibility)",
					"page":     "integer (optional, default 1)",
					"limit":    "integer (optional, default 20, max 100)",
				},
				Response: map[string]any{
					"products": []map[string]any{
						{
							"id":          "550e8400-e29b-41d4-a716-446655440001",
							"legacy_id":   101,
							"category_id": "11111111-1111-1111-1111-111111111111",
							"name":        "Royal Aroma Rice 5kg",
							"category":    "Rice, Spaghetti & Grains",
							"price":       120,
							"old_price":   150,
							"tag":         "In Stock",
							"image_url":   "https://res.cloudinary.com/demo/image/upload/v1/rice.jpg",
							"images": []map[string]any{
								{
									"id":         "550e8400-e29b-41d4-a716-446655440101",
									"product_id": "550e8400-e29b-41d4-a716-446655440001",
									"image_url":  "https://res.cloudinary.com/demo/image/upload/v1/rice.jpg",
									"created_at": "2026-04-29T12:00:00Z",
								},
								{
									"id":         "550e8400-e29b-41d4-a716-446655440102",
									"product_id": "550e8400-e29b-41d4-a716-446655440001",
									"image_url":  "https://res.cloudinary.com/demo/image/upload/v1/rice-back.jpg",
									"created_at": "2026-04-29T12:05:00Z",
								},
							},
							"unit":   "bag",
							"active": true,
						},
					},
					"meta": map[string]any{
						"total":       125,
						"page":        1,
						"limit":       20,
						"total_pages": 7,
						"has_next":    true,
						"has_prev":    false,
					},
				},
				Example: `curl 'http://localhost:8080/api/v1/products?category=11111111-1111-1111-1111-111111111111&page=1&limit=20'`,
			},
			{
				Method:      http.MethodGet,
				Path:        "/api/v1/products/:id",
				Description: "Fetch a single active product by UUID. The response includes hydrated images.",
				Auth:        false,
				Response: map[string]any{
					"id":          "550e8400-e29b-41d4-a716-446655440001",
					"legacy_id":   101,
					"category_id": "11111111-1111-1111-1111-111111111111",
					"name":        "Royal Aroma Rice 5kg",
					"category":    "Rice, Spaghetti & Grains",
					"price":       120,
					"old_price":   150,
					"tag":         "In Stock",
					"image_url":   "https://res.cloudinary.com/demo/image/upload/v1/rice.jpg",
					"images":      []map[string]any{},
					"unit":        "bag",
					"active":      true,
				},
				Example: `curl http://localhost:8080/api/v1/products/550e8400-e29b-41d4-a716-446655440001`,
			},
			{
				Method:      http.MethodGet,
				Path:        "/api/v1/products/categories",
				Description: "List all active product categories with their UUIDs and display names. The frontend should use the UUID in product create/update and filters.",
				Auth:        false,
				Response: map[string]any{
					"categories": []map[string]any{
						{
							"id":   "11111111-1111-1111-1111-111111111111",
							"name": "Rice, Spaghetti & Grains",
						},
						{
							"id":   "22222222-2222-2222-2222-222222222222",
							"name": "Cooking Oil",
						},
					},
				},
				Example: `curl http://localhost:8080/api/v1/products/categories`,
			},
			{
				Method:      http.MethodGet,
				Path:        "/api/v1/packages",
				Description: "Get the full package catalog, including fixed packages plus the Provisions and Detergents bundles.",
				Auth:        false,
				Example:     `curl http://localhost:8080/api/v1/packages`,
			},
			{
				Method:      http.MethodGet,
				Path:        "/api/v1/packages/fixed",
				Description: "Get only the fixed grocery bundles used by the storefront package picker.",
				Auth:        false,
				Example:     `curl http://localhost:8080/api/v1/packages/fixed`,
			},
			{
				Method:      http.MethodGet,
				Path:        "/api/v1/packages/provisions",
				Description: "Get the Provisions department bundles.",
				Auth:        false,
				Example:     `curl http://localhost:8080/api/v1/packages/provisions`,
			},
			{
				Method:      http.MethodGet,
				Path:        "/api/v1/packages/detergents",
				Description: "Get the Detergents department bundles.",
				Auth:        false,
				Example:     `curl http://localhost:8080/api/v1/packages/detergents`,
			},

			// Application Endpoints (Customer)
			{
				Method:      http.MethodPost,
				Path:        "/api/v1/applications",
				Description: "Submit a new grocery application. Frontend cart items send product_id and quantity; product_id may be the legacy numeric ID from App.jsx or the product UUID. package_type is 'fixed' for the predefined bundles or 'custom' for a cart built from products. Cart total must exceed GHC 549 (MIN_ORDER config). mandate_number is always required. staff_number, institution, and ghana_card_number are resolved from the request first, then from the authenticated user's profile if omitted.",
				Auth:        true,
				Request: map[string]any{
					"package_type": "string ('fixed' or 'custom')",
					"package_name": "string (required for fixed packages, e.g. 'Abusua Asomdwee')",
					"cart_items": []map[string]any{
						{
							"product_id": "string (required, legacy numeric ID or product UUID)",
							"quantity":   "integer (required, quantity selected in the frontend cart)",
						},
					},
					"staff_number":      "string (optional if already on user profile)",
					"mandate_number":    "string (required, mandate ID)",
					"institution":       "string (optional if already on user profile)",
					"ghana_card_number": "string (optional if already on user profile)",
				},
				Example: `curl -X POST http://localhost:8080/api/v1/applications \
  -H "Content-Type: application/json" \
  -b cookies.txt \
  -d '{
    "package_type":"custom",
    "cart_items":[
      {"product_id":"101","quantity":2},
      {"product_id":"402","quantity":5}
    ],
    "staff_number":"GS123456",
    "mandate_number":"MND-001",
    "institution":"Ghana Health Service",
    "ghana_card_number":"GHA-000-0000-0"
  }'`,
			},
			{
				Method:      http.MethodGet,
				Path:        "/api/v1/applications",
				Description: "List applications submitted by the authenticated user. Results are paginated.",
				Auth:        true,
				Request: map[string]string{
					"page":  "integer (optional, default 1)",
					"limit": "integer (optional, default 20, max 100)",
				},
				Response: map[string]any{
					"applications": []map[string]any{
						{
							"id":           "550e8400-e29b-41d4-a716-446655440010",
							"user_id":      "550e8400-e29b-41d4-a716-446655440000",
							"package_type": "custom",
							"cart_items": []map[string]any{
								{
									"product_id": "550e8400-e29b-41d4-a716-446655440001",
									"name":       "Royal Aroma Rice 5kg",
									"image_url":  "https://res.cloudinary.com/demo/image/upload/v1/rice.jpg",
									"price":      120,
									"quantity":   2,
									"subtotal":   240,
								},
							},
							"total_amount":      650,
							"monthly_amount":    217,
							"status":            "pending",
							"staff_number":      "GS123456",
							"mandate_number":    "MND-001",
							"institution":       "Ghana Health Service",
							"ghana_card_number": "GHA-000-0000-0",
							"created_at":        "2026-04-28T12:00:00Z",
							"updated_at":        "2026-04-28T12:00:00Z",
						},
					},
					"meta": map[string]any{
						"total":       12,
						"page":        1,
						"limit":       20,
						"total_pages": 1,
						"has_next":    false,
						"has_prev":    false,
					},
				},
				Example: `curl 'http://localhost:8080/api/v1/applications?page=1&limit=20' \
  -b cookies.txt`,
			},
			{
				Method:      http.MethodGet,
				Path:        "/api/v1/applications/:id",
				Description: "Get a single application by ID. Returns full order details including cart items, prices, and current status. Users can only view their own applications.",
				Auth:        true,
				Example: `curl http://localhost:8080/api/v1/applications/550e8400-e29b-41d4-a716-446655440000 \
  -b cookies.txt`,
			},

			// Conversation Endpoints (Customer)
			{
				Method:      http.MethodPost,
				Path:        "/api/v1/conversations",
				Description: "Start a new conversation with the support inbox. The first message is routed through a bootstrap admin account, but any admin can later view and reply to the thread. If a conversation already exists with this user, returns the existing conversation instead of creating a duplicate and does not insert the initial message again.",
				Auth:        true,
				Request: map[string]string{
					"message": "string (initial message, required)",
				},
				Example: `curl -X POST http://localhost:8080/api/v1/conversations \
  -H "Content-Type: application/json" \
  -b cookies.txt \
  -d '{"message":"I have a question about my application"}'`,
			},
			{
				Method:      http.MethodGet,
				Path:        "/api/v1/conversations",
				Description: "List conversations for the authenticated user. Admins receive the shared inbox view, which returns all customer threads with other-user profile, last message, and unread count. Results are paginated.",
				Auth:        true,
				Request: map[string]string{
					"page":  "integer (optional, default 1)",
					"limit": "integer (optional, default 20, max 100)",
				},
				Response: map[string]any{
					"conversations": []map[string]any{
						{
							"id": "550e8400-e29b-41d4-a716-446655440020",
							"other_user": map[string]any{
								"id":           "550e8400-e29b-41d4-a716-446655440999",
								"display_name": "LJ List Admin",
								"phone_number": "+233241111111",
								"role":         "admin",
							},
							"last_message": "Your application is under review.",
							"unread_count": 1,
							"created_at":   "2026-04-28T12:00:00Z",
						},
					},
					"meta": map[string]any{
						"total":       3,
						"page":        1,
						"limit":       20,
						"total_pages": 1,
						"has_next":    false,
						"has_prev":    false,
					},
				},
				Example: `curl 'http://localhost:8080/api/v1/conversations?page=1&limit=20' \
  -b cookies.txt`,
			},

			// Message Endpoints (Customer)
			{
				Method:      http.MethodGet,
				Path:        "/api/v1/conversations/:id/messages",
				Description: "List messages in a conversation in chronological order. Results are paginated. Fetching messages marks unread messages from the other participant as read.",
				Auth:        true,
				Request: map[string]string{
					"page":  "integer (optional, default 1)",
					"limit": "integer (optional, default 20, max 100)",
				},
				Response: map[string]any{
					"messages": []map[string]any{
						{
							"id":              "550e8400-e29b-41d4-a716-446655440030",
							"conversation_id": "550e8400-e29b-41d4-a716-446655440020",
							"sender_id":       "550e8400-e29b-41d4-a716-446655440000",
							"content":         "Hello, any update on my application?",
							"read_at":         nil,
							"created_at":      "2026-04-28T12:10:00Z",
						},
					},
					"meta": map[string]any{
						"total":       28,
						"page":        1,
						"limit":       20,
						"total_pages": 2,
						"has_next":    true,
						"has_prev":    false,
					},
				},
				Example: `curl 'http://localhost:8080/api/v1/conversations/550e8400-e29b-41d4-a716-446655440000/messages?page=1&limit=20' \
  -b cookies.txt`,
			},
			{
				Method:      http.MethodPost,
				Path:        "/api/v1/conversations/:id/messages",
				Description: "Send a message in a conversation. The sender_id is set automatically from the authenticated user. Admin and customer can both send messages.",
				Auth:        true,
				Request: map[string]string{
					"content": "string (message text, required)",
				},
				Example: `curl -X POST http://localhost:8080/api/v1/conversations/550e8400-e29b-41d4-a716-446655440000/messages \
  -H "Content-Type: application/json" \
  -b cookies.txt \
  -d '{"content":"Thank you for the update"}'`,
			},

			// Admin Endpoints
			{
				Method:      http.MethodGet,
				Path:        "/api/v1/admin/users",
				Description: "List users for the admin dashboard. Supports optional role filtering and standard pagination.",
				Auth:        true,
				AdminOnly:   true,
				Request: map[string]string{
					"role":  "string (optional, 'customer' or 'admin')",
					"page":  "integer (optional, default 1)",
					"limit": "integer (optional, default 20, max 100)",
				},
				Response: map[string]any{
					"users": []map[string]any{
						{
							"id":           "550e8400-e29b-41d4-a716-446655440000",
							"display_name": "Kwame",
							"phone_number": "+233240000000",
							"role":         "customer",
							"created_at":   "2026-04-29T12:00:00Z",
							"updated_at":   "2026-04-29T12:00:00Z",
						},
					},
					"meta": map[string]any{
						"total":       42,
						"page":        1,
						"limit":       20,
						"total_pages": 3,
						"has_next":    true,
						"has_prev":    false,
					},
				},
				Example: `curl 'http://localhost:8080/api/v1/admin/users?role=customer&page=1&limit=20' \
  -b cookies.txt`,
			},
			{
				Method:      http.MethodGet,
				Path:        "/api/v1/admin/categories",
				Description: "List all categories for admin management, including inactive rows. Categories are the UUID-backed source of truth for product creation.",
				Auth:        true,
				AdminOnly:   true,
				Response: map[string]any{
					"categories": []map[string]any{
						{
							"id":         "11111111-1111-1111-1111-111111111111",
							"name":       "Rice, Spaghetti & Grains",
							"sort_order": 1,
							"active":     true,
						},
					},
				},
				Example: `curl http://localhost:8080/api/v1/admin/categories -b cookies.txt`,
			},
			{
				Method:      http.MethodGet,
				Path:        "/api/v1/admin/categories/:id",
				Description: "Fetch a single category by UUID for editing in the admin UI.",
				Auth:        true,
				AdminOnly:   true,
				Response: map[string]any{
					"id":         "11111111-1111-1111-1111-111111111111",
					"name":       "Rice, Spaghetti & Grains",
					"sort_order": 1,
					"active":     true,
				},
				Example: `curl http://localhost:8080/api/v1/admin/categories/11111111-1111-1111-1111-111111111111 -b cookies.txt`,
			},
			{
				Method:      http.MethodPost,
				Path:        "/api/v1/admin/categories",
				Description: "Create a product category (admin only). The category name is the display label, and the generated UUID is what product creation should use.",
				Auth:        true,
				AdminOnly:   true,
				Request: map[string]any{
					"name":       "string (required, display name such as 'Rice, Spaghetti & Grains')",
					"sort_order": "integer (optional, display order)",
					"active":     "boolean (optional, defaults to true)",
				},
				ResponseSuccess: models.NewSuccessResponse("req-abc123", "Category created successfully", map[string]any{
					"id":         "11111111-1111-1111-1111-111111111111",
					"name":       "Rice, Spaghetti & Grains",
					"sort_order": 1,
					"active":     true,
				}),
				Example: `curl -X POST http://localhost:8080/api/v1/admin/categories \
  -H "Content-Type: application/json" \
  -b cookies.txt \
  -d '{
    "name":"Rice, Spaghetti & Grains",
    "sort_order":1,
    "active":true
  }'`,
			},
			{
				Method:      http.MethodPatch,
				Path:        "/api/v1/admin/categories/:id",
				Description: "Update a product category (admin only). Renaming a category also updates the denormalized category name on linked products.",
				Auth:        true,
				AdminOnly:   true,
				Request: map[string]any{
					"name":       "string (optional)",
					"sort_order": "integer (optional)",
					"active":     "boolean (optional)",
				},
				ResponseSuccess: models.NewSuccessResponse("req-abc123", "Category updated successfully", map[string]any{
					"id":         "11111111-1111-1111-1111-111111111111",
					"name":       "Rice, Spaghetti & Grains",
					"sort_order": 1,
					"active":     true,
				}),
				Example: `curl -X PATCH http://localhost:8080/api/v1/admin/categories/11111111-1111-1111-1111-111111111111 \
  -H "Content-Type: application/json" \
  -b cookies.txt \
  -d '{
    "name":"Rice, Spaghetti & Grains",
    "sort_order":1,
    "active":true
  }'`,
			},
			{
				Method:      http.MethodDelete,
				Path:        "/api/v1/admin/categories/:id",
				Description: "Deactivate a category (admin only). If products already use it, the category is deactivated instead of being removed.",
				Auth:        true,
				AdminOnly:   true,
				ResponseSuccess: models.NewSuccessResponse("req-abc123", "Category deactivated successfully", map[string]any{
					"category": map[string]any{
						"id":         "11111111-1111-1111-1111-111111111111",
						"name":       "Rice, Spaghetti & Grains",
						"sort_order": 1,
						"active":     false,
					},
					"deactivated":  true,
					"soft_deleted": true,
				}),
				Example: `curl -X DELETE http://localhost:8080/api/v1/admin/categories/11111111-1111-1111-1111-111111111111 -b cookies.txt`,
			},
			{
				Method:      http.MethodPatch,
				Path:        "/api/v1/admin/users/:id",
				Description: "Update a user's profile fields or role (admin only). The bootstrap admin account cannot be demoted away from admin.",
				Auth:        true,
				AdminOnly:   true,
				Request: map[string]string{
					"display_name": "string (optional, 2-100 characters)",
					"phone_number": "string (optional, valid phone number)",
					"role":         "string (optional, 'customer' or 'admin')",
				},
				Response: map[string]any{
					"user": map[string]any{
						"id":           "550e8400-e29b-41d4-a716-446655440000",
						"display_name": "Kwame Mensah",
						"phone_number": "+233240000000",
						"role":         "customer",
						"created_at":   "2026-04-29T12:00:00Z",
						"updated_at":   "2026-04-29T12:30:00Z",
					},
				},
				Example: `curl -X PATCH http://localhost:8080/api/v1/admin/users/550e8400-e29b-41d4-a716-446655440000 \
  -H "Content-Type: application/json" \
  -b cookies.txt \
  -d '{
    "display_name":"Kwame Mensah",
    "phone_number":"+233240000000",
    "role":"customer"
  }'`,
			},
			{
				Method:      http.MethodPost,
				Path:        "/api/v1/admin/products",
				Description: "Create product metadata (admin only). This matches the storefront product form: name, category_id, price, old_price, tag, unit, and active. Images are managed separately through the product image endpoints so a product can have many images instead of one.",
				Auth:        true,
				AdminOnly:   true,
				Request: map[string]any{
					"content_type": "application/json",
					"name":         "string (required, e.g. 'Royal Aroma 25kg (5*5)')",
					"category_id":  "string (required, category UUID from /api/v1/products/categories)",
					"price":        "integer (required, current selling price)",
					"old_price":    "integer (optional, strikethrough price shown in the frontend)",
					"tag":          "string (optional, badge text such as 'In Stock', 'Premium', or 'Seasonal')",
					"unit":         "string (required, e.g. 'bag', 'bottle', 'basket')",
					"active":       "boolean (optional, defaults to true)",
				},
				Response: map[string]any{
					"id":          "550e8400-e29b-41d4-a716-446655440001",
					"category_id": "11111111-1111-1111-1111-111111111111",
					"name":        "Royal Aroma Rice 5kg",
					"category":    "Rice, Spaghetti & Grains",
					"price":       120,
					"old_price":   150,
					"tag":         "In Stock",
					"image_url":   "",
					"images":      []map[string]any{},
					"unit":        "bag",
					"active":      true,
				},
				ResponseSuccess: models.NewSuccessResponse("req-abc123", "Product created successfully", map[string]any{
					"id":          "550e8400-e29b-41d4-a716-446655440001",
					"category_id": "11111111-1111-1111-1111-111111111111",
					"name":        "Royal Aroma Rice 5kg",
					"category":    "Rice, Spaghetti & Grains",
					"price":       120,
					"old_price":   150,
					"tag":         "In Stock",
					"image_url":   "",
					"images":      []map[string]any{},
					"unit":        "bag",
					"active":      true,
				}),
				Example: `curl -X POST http://localhost:8080/api/v1/admin/products \
  -H "Content-Type: application/json" \
  -b cookies.txt \
 				  -d '{
    "name":"Royal Aroma Rice 5kg",
    "category_id":"11111111-1111-1111-1111-111111111111",
    "price":120,
    "old_price":150,
    "tag":"In Stock",
    "unit":"bag",
    "active":true
  }'`,
			},
			{
				Method:      http.MethodGet,
				Path:        "/api/v1/admin/products",
				Description: "List products for the admin dashboard, including inactive rows. category accepts a category UUID from /api/v1/products/categories, with temporary name lookup kept for compatibility. Results are paginated.",
				Auth:        true,
				AdminOnly:   true,
				Request: map[string]string{
					"category": "string (optional, category UUID)",
					"page":     "integer (optional, default 1)",
					"limit":    "integer (optional, default 20, max 100)",
				},
				Response: map[string]any{
					"products": []map[string]any{
						{
							"id":          "550e8400-e29b-41d4-a716-446655440001",
							"legacy_id":   101,
							"category_id": "11111111-1111-1111-1111-111111111111",
							"name":        "Royal Aroma Rice 5kg",
							"category":    "Rice, Spaghetti & Grains",
							"price":       120,
							"old_price":   150,
							"tag":         "In Stock",
							"image_url":   "https://res.cloudinary.com/demo/image/upload/v1/rice.jpg",
							"images":      []map[string]any{},
							"unit":        "bag",
							"active":      true,
						},
					},
					"meta": map[string]any{
						"total":       1,
						"page":        1,
						"limit":       20,
						"total_pages": 1,
						"has_next":    false,
						"has_prev":    false,
					},
				},
				Example: `curl 'http://localhost:8080/api/v1/admin/products?page=1&limit=10' -b cookies.txt`,
			},
			{
				Method:      http.MethodGet,
				Path:        "/api/v1/admin/products/:id",
				Description: "Fetch a single product by UUID for admin editing, including inactive rows.",
				Auth:        true,
				AdminOnly:   true,
				Response: map[string]any{
					"id":          "550e8400-e29b-41d4-a716-446655440001",
					"legacy_id":   101,
					"category_id": "11111111-1111-1111-1111-111111111111",
					"name":        "Royal Aroma Rice 5kg",
					"category":    "Rice, Spaghetti & Grains",
					"price":       120,
					"old_price":   150,
					"tag":         "In Stock",
					"image_url":   "https://res.cloudinary.com/demo/image/upload/v1/rice.jpg",
					"images":      []map[string]any{},
					"unit":        "bag",
					"active":      true,
				},
				Example: `curl http://localhost:8080/api/v1/admin/products/550e8400-e29b-41d4-a716-446655440001 -b cookies.txt`,
			},
			{
				Method:      http.MethodPatch,
				Path:        "/api/v1/admin/products/:id",
				Description: "Update catalog product details (admin only). This endpoint edits the same storefront product fields as create: name, category_id, price, old_price, tag, unit, and active. Use the dedicated image endpoints to add or delete gallery images.",
				Auth:        true,
				AdminOnly:   true,
				Request: map[string]any{
					"content_type": "application/json",
					"name":         "string (optional)",
					"category_id":  "string (optional, category UUID)",
					"price":        "integer (optional, current selling price)",
					"old_price":    "integer (optional, strikethrough price shown in the frontend)",
					"tag":          "string (optional, badge text)",
					"unit":         "string (optional)",
					"active":       "boolean (optional)",
				},
				Response: map[string]any{
					"id":          "550e8400-e29b-41d4-a716-446655440001",
					"category_id": "11111111-1111-1111-1111-111111111111",
					"name":        "Royal Aroma Rice 5kg",
					"category":    "Rice, Spaghetti & Grains",
					"price":       125,
					"old_price":   150,
					"tag":         "In Stock",
					"image_url":   "https://res.cloudinary.com/demo/image/upload/v1/rice.jpg",
					"images": []map[string]any{
						{
							"id":         "550e8400-e29b-41d4-a716-446655440101",
							"product_id": "550e8400-e29b-41d4-a716-446655440001",
							"image_url":  "https://res.cloudinary.com/demo/image/upload/v1/rice.jpg",
							"created_at": "2026-04-29T12:00:00Z",
						},
						{
							"id":         "550e8400-e29b-41d4-a716-446655440102",
							"product_id": "550e8400-e29b-41d4-a716-446655440001",
							"image_url":  "https://res.cloudinary.com/demo/image/upload/v1/rice-back.jpg",
							"created_at": "2026-04-29T12:05:00Z",
						},
					},
					"unit":   "bag",
					"active": true,
				},
				ResponseSuccess: models.NewSuccessResponse("req-abc123", "Product updated successfully", map[string]any{
					"id":          "550e8400-e29b-41d4-a716-446655440001",
					"category_id": "11111111-1111-1111-1111-111111111111",
					"name":        "Royal Aroma Rice 5kg",
					"category":    "Rice, Spaghetti & Grains",
					"price":       125,
					"old_price":   150,
					"tag":         "Premium",
					"image_url":   "https://res.cloudinary.com/demo/image/upload/v1/rice.jpg",
					"images": []map[string]any{
						{
							"id":         "550e8400-e29b-41d4-a716-446655440101",
							"product_id": "550e8400-e29b-41d4-a716-446655440001",
							"image_url":  "https://res.cloudinary.com/demo/image/upload/v1/rice.jpg",
							"created_at": "2026-04-29T12:00:00Z",
						},
						{
							"id":         "550e8400-e29b-41d4-a716-446655440102",
							"product_id": "550e8400-e29b-41d4-a716-446655440001",
							"image_url":  "https://res.cloudinary.com/demo/image/upload/v1/rice-back.jpg",
							"created_at": "2026-04-29T12:05:00Z",
						},
					},
					"unit":   "bag",
					"active": true,
				}),
				Example: `curl -X PATCH http://localhost:8080/api/v1/admin/products/550e8400-e29b-41d4-a716-446655440001 \
  -H "Content-Type: application/json" \
  -b cookies.txt \
  -d '{
    "price":125,
    "old_price":150,
    "tag":"Premium",
    "active":true
  }'`,
			},
			{
				Method:      http.MethodDelete,
				Path:        "/api/v1/admin/products/:id",
				Description: "Delete a product (admin only). If the product has existing applications, it is kept and deactivated instead of being deleted.",
				Auth:        true,
				AdminOnly:   true,
				Example: `curl -X DELETE http://localhost:8080/api/v1/admin/products/550e8400-e29b-41d4-a716-446655440001 \
  -b cookies.txt`,
			},
			{
				Method:      http.MethodGet,
				Path:        "/api/v1/admin/products/:id/images",
				Description: "List every image attached to a product (admin only). Images are ordered oldest-first, which also determines the primary compatibility image_url.",
				Auth:        true,
				AdminOnly:   true,
				Response: map[string]any{
					"images": []map[string]any{
						{
							"id":         "550e8400-e29b-41d4-a716-446655440101",
							"product_id": "550e8400-e29b-41d4-a716-446655440001",
							"image_url":  "https://res.cloudinary.com/demo/image/upload/v1/rice.jpg",
							"created_at": "2026-04-29T12:00:00Z",
						},
						{
							"id":         "550e8400-e29b-41d4-a716-446655440102",
							"product_id": "550e8400-e29b-41d4-a716-446655440001",
							"image_url":  "https://res.cloudinary.com/demo/image/upload/v1/rice-back.jpg",
							"created_at": "2026-04-29T12:05:00Z",
						},
					},
				},
				Example: `curl http://localhost:8080/api/v1/admin/products/550e8400-e29b-41d4-a716-446655440001/images \
  -b cookies.txt`,
			},
			{
				Method:      http.MethodPost,
				Path:        "/api/v1/admin/products/:id/images",
				Description: "Upload one or more product gallery images (admin only). Send repeated images fields or a single image field. The first remaining image becomes products.image_url for backward compatibility.",
				Auth:        true,
				AdminOnly:   true,
				Request: map[string]string{
					"content_type": "multipart/form-data",
					"images":       "file (one or more image files)",
					"image":        "file (optional single-file alias)",
				},
				Response: map[string]any{
					"images": []map[string]any{
						{
							"id":         "550e8400-e29b-41d4-a716-446655440101",
							"product_id": "550e8400-e29b-41d4-a716-446655440001",
							"image_url":  "https://res.cloudinary.com/demo/image/upload/v1/rice.jpg",
							"created_at": "2026-04-29T12:00:00Z",
						},
						{
							"id":         "550e8400-e29b-41d4-a716-446655440102",
							"product_id": "550e8400-e29b-41d4-a716-446655440001",
							"image_url":  "https://res.cloudinary.com/demo/image/upload/v1/rice-back.jpg",
							"created_at": "2026-04-29T12:05:00Z",
						},
					},
				},
				Example: `curl -X POST http://localhost:8080/api/v1/admin/products/550e8400-e29b-41d4-a716-446655440001/images \
  -b cookies.txt \
  -F 'images=@/path/to/rice-front.jpg' \
  -F 'images=@/path/to/rice-back.jpg'`,
			},
			{
				Method:      http.MethodDelete,
				Path:        "/api/v1/admin/products/:id/images/:imageId",
				Description: "Delete a single product image by its UUID (admin only). This removes the product_images row and also destroys the underlying Cloudinary asset.",
				Auth:        true,
				AdminOnly:   true,
				Example: `curl -X DELETE http://localhost:8080/api/v1/admin/products/550e8400-e29b-41d4-a716-446655440001/images/550e8400-e29b-41d4-a716-446655440101 \
  -b cookies.txt`,
			},
			{
				Method:      http.MethodGet,
				Path:        "/api/v1/admin/packages",
				Description: "Get the full package catalog for admin editing, including inactive package rows.",
				Auth:        true,
				AdminOnly:   true,
				ResponseSuccess: models.NewSuccessResponse("req-abc123", "Package catalog retrieved", map[string]any{
					"min_order": 549,
					"package_options": []string{
						"ABUSUA ASOMDWEE (GHC569)",
						"MEDAASE MEDO (GHC769)",
					},
					"fixed_packages": []map[string]any{
						{
							"id":           "abusua",
							"name":         "Abusua Asomdwee",
							"tagline":      "Perfect for individuals & small families",
							"price":        "GH₵569",
							"monthly":      "GH₵190/mo",
							"tag":          "Starter",
							"popular":      false,
							"rice_options": "Ginny Viet · Ginny Gold · Everest Viet",
							"items": []map[string]any{
								{
									"product_id": "prod-113",
									"qty":        1,
									"label":      "Rice 25kg (5*5)",
									"emoji":      "🌾",
									"image_url":  "https://res.cloudinary.com/demo/image/upload/v1/ginny.jpg",
									"product": map[string]any{
										"id":        "prod-113",
										"name":      "Ginny Viet 25kg (5*5)",
										"image_url": "https://res.cloudinary.com/demo/image/upload/v1/ginny.jpg",
										"unit":      "bag",
										"active":    true,
									},
								},
							},
						},
					},
					"provisions_packages": []map[string]any{},
					"detergent_packages":  []map[string]any{},
				}),
				Example: `curl http://localhost:8080/api/v1/admin/packages -b cookies.txt`,
			},
			{
				Method:      http.MethodGet,
				Path:        "/api/v1/admin/packages/fixed/:id",
				Description: "Fetch a single fixed package by its frontend slug for admin editing.",
				Auth:        true,
				AdminOnly:   true,
				Response: map[string]any{
					"id":           "abusua",
					"name":         "Abusua Asomdwee",
					"tagline":      "Perfect for individuals & small families",
					"price":        "GH₵569",
					"monthly":      "GH₵190/mo",
					"tag":          "Starter",
					"popular":      false,
					"rice_options": "Ginny Viet · Ginny Gold · Everest Viet",
					"items":        []map[string]any{},
				},
				Example: `curl http://localhost:8080/api/v1/admin/packages/fixed/abusua -b cookies.txt`,
			},
			{
				Method:      http.MethodPost,
				Path:        "/api/v1/admin/packages/fixed",
				Description: "Create a fixed package bundle (admin only). This mirrors the fixed package cards from App.jsx, including the display strings and the item list that is tied back to products by UUID string or legacy product reference.",
				Auth:        true,
				AdminOnly:   true,
				Request: map[string]any{
					"id":           "string (required, frontend slug such as 'abusua')",
					"name":         "string (required, package title shown on the card)",
					"tagline":      "string (required, short descriptive subtitle)",
					"price":        "string (required, display price such as 'GH₵569')",
					"monthly":      "string (required, display installment price such as 'GH₵190/mo')",
					"tag":          "string (required, badge label such as 'Starter' or 'Premium')",
					"popular":      "boolean (required, whether the card gets the popular treatment)",
					"rice_options": "string (optional, comma-separated rice choices shown in the frontend)",
					"items": []map[string]any{
						{
							"product_id": "string (required, product UUID or legacy product reference)",
							"qty":        "integer (required)",
							"label":      "string (required, frontend item label)",
							"emoji":      "string (required, emoji shown on the card)",
							"image_url":  "string (optional, Cloudinary URL for the item image)",
						},
					},
				},
				ResponseSuccess: models.NewSuccessResponse("req-abc123", "Fixed package created successfully", map[string]any{
					"id":           "custom",
					"name":         "Custom",
					"tagline":      "Optional admin bundle",
					"price":        "GH₵0",
					"monthly":      "GH₵0",
					"tag":          "Custom",
					"popular":      false,
					"rice_options": "",
					"items":        []map[string]any{},
				}),
				Example: `curl -X POST http://localhost:8080/api/v1/admin/packages/fixed -b cookies.txt -H "Content-Type: application/json" -d '{
  "id":"custom",
  "name":"Custom",
  "tagline":"Optional admin bundle",
  "price":"GH₵0",
  "monthly":"GH₵0",
  "tag":"Custom",
  "popular":false,
  "rice_options":"",
  "items":[
    {
      "product_id":"550e8400-e29b-41d4-a716-446655440001",
      "qty":1,
      "label":"Rice 25kg (5*5)",
      "emoji":"🌾",
      "image_url":""
    }
  ]
}'`,
			},
			{
				Method:      http.MethodPatch,
				Path:        "/api/v1/admin/packages/fixed/:id",
				Description: "Update a fixed package bundle (admin only). The request body uses the same frontend shape as create.",
				Auth:        true,
				AdminOnly:   true,
				Request: map[string]any{
					"id":           "string (optional, frontend slug)",
					"name":         "string (optional)",
					"tagline":      "string (optional)",
					"price":        "string (optional)",
					"monthly":      "string (optional)",
					"tag":          "string (optional)",
					"popular":      "boolean (optional)",
					"rice_options": "string (optional)",
					"items": []map[string]any{
						{
							"product_id": "string (required, product UUID or legacy product reference)",
							"qty":        "integer (required)",
							"label":      "string (required)",
							"emoji":      "string (required)",
							"image_url":  "string (optional)",
						},
					},
				},
				ResponseSuccess: models.NewSuccessResponse("req-abc123", "Fixed package updated successfully", map[string]any{
					"id":           "abusua",
					"name":         "Abusua Asomdwee",
					"tagline":      "Perfect for individuals & small families",
					"price":        "GH₵569",
					"monthly":      "GH₵190/mo",
					"tag":          "Starter",
					"popular":      false,
					"rice_options": "Ginny Viet · Ginny Gold · Everest Viet",
					"items":        []map[string]any{},
				}),
				Example: `curl -X PATCH http://localhost:8080/api/v1/admin/packages/fixed/abusua -b cookies.txt`,
			},
			{
				Method:          http.MethodDelete,
				Path:            "/api/v1/admin/packages/fixed/:id",
				Description:     "Deactivate a fixed package bundle (admin only).",
				Auth:            true,
				AdminOnly:       true,
				ResponseSuccess: models.NewSuccessResponse("req-abc123", "Fixed package deactivated successfully", nil),
				Example:         `curl -X DELETE http://localhost:8080/api/v1/admin/packages/fixed/abusua -b cookies.txt`,
			},
			{
				Method:      http.MethodGet,
				Path:        "/api/v1/admin/packages/provisions/:id",
				Description: "Fetch a single Provisions package by its frontend slug.",
				Auth:        true,
				AdminOnly:   true,
				Response: map[string]any{
					"id":    "maakye",
					"name":  "Maakye",
					"price": 250,
					"items": "1 Milo tin or 2 strips of 20g Milo",
				},
				Example: `curl http://localhost:8080/api/v1/admin/packages/provisions/maakye -b cookies.txt`,
			},
			{
				Method:      http.MethodPost,
				Path:        "/api/v1/admin/packages/provisions",
				Description: "Create a Provisions department package (admin only). The frontend sends a simple bundle definition: id, name, price, and a descriptive items string.",
				Auth:        true,
				AdminOnly:   true,
				Request: map[string]any{
					"id":    "string (required, frontend slug such as 'maakye')",
					"name":  "string (required, package title)",
					"price": "integer (required, package price)",
					"items": "string (required, human-readable package contents)",
				},
				ResponseSuccess: models.NewSuccessResponse("req-abc123", "Package created successfully", map[string]any{
					"id":    "maakye",
					"name":  "Maakye",
					"price": 250,
					"items": "1 Milo tin or 2 strips of 20g Milo",
				}),
				Example: `curl -X POST http://localhost:8080/api/v1/admin/packages/provisions -b cookies.txt -H "Content-Type: application/json" -d '{
  "id":"maakye",
  "name":"Maakye",
  "price":250,
  "items":"1 Milo tin or 2 strips of 20g Milo"
}'`,
			},
			{
				Method:      http.MethodPatch,
				Path:        "/api/v1/admin/packages/provisions/:id",
				Description: "Update a Provisions department package (admin only).",
				Auth:        true,
				AdminOnly:   true,
				Request: map[string]any{
					"id":    "string (optional)",
					"name":  "string (optional)",
					"price": "integer (optional)",
					"items": "string (optional)",
				},
				ResponseSuccess: models.NewSuccessResponse("req-abc123", "Package updated successfully", map[string]any{
					"id":    "maakye",
					"name":  "Maakye",
					"price": 250,
					"items": "1 Milo tin or 2 strips of 20g Milo",
				}),
				Example: `curl -X PATCH http://localhost:8080/api/v1/admin/packages/provisions/maakye -b cookies.txt -H "Content-Type: application/json" -d '{
  "price":350,
  "items":"2 tins of Milo or 4 strips of 20g Milo"
}'`,
			},
			{
				Method:          http.MethodDelete,
				Path:            "/api/v1/admin/packages/provisions/:id",
				Description:     "Deactivate a Provisions department package (admin only).",
				Auth:            true,
				AdminOnly:       true,
				ResponseSuccess: models.NewSuccessResponse("req-abc123", "Provisions package deactivated successfully", nil),
				Example:         `curl -X DELETE http://localhost:8080/api/v1/admin/packages/provisions/maakye -b cookies.txt`,
			},
			{
				Method:      http.MethodGet,
				Path:        "/api/v1/admin/packages/detergents/:id",
				Description: "Fetch a single Detergents package by its frontend slug.",
				Auth:        true,
				AdminOnly:   true,
				Response: map[string]any{
					"id":    "mawohonte",
					"name":  "Ma Wo Ho Nte",
					"price": 270,
					"items": "Madar / Kleesoft 400g (¼ Box)",
				},
				Example: `curl http://localhost:8080/api/v1/admin/packages/detergents/mawohonte -b cookies.txt`,
			},
			{
				Method:      http.MethodPost,
				Path:        "/api/v1/admin/packages/detergents",
				Description: "Create a Detergents department package (admin only). The frontend uses the same simple bundle shape as Provisions: id, name, price, and items text.",
				Auth:        true,
				AdminOnly:   true,
				Request: map[string]any{
					"id":    "string (required, frontend slug such as 'mawohonte')",
					"name":  "string (required, package title)",
					"price": "integer (required, package price)",
					"items": "string (required, human-readable package contents)",
				},
				ResponseSuccess: models.NewSuccessResponse("req-abc123", "Package created successfully", map[string]any{
					"id":    "mawohonte",
					"name":  "Ma Wo Ho Nte",
					"price": 270,
					"items": "Madar / Kleesoft 400g (¼ Box)",
				}),
				Example: `curl -X POST http://localhost:8080/api/v1/admin/packages/detergents -b cookies.txt -H "Content-Type: application/json" -d '{
  "id":"mawohonte",
  "name":"Ma Wo Ho Nte",
  "price":270,
  "items":"Madar / Kleesoft 400g (¼ Box) · Madar / Kleesoft Soap (¼ Box)"
}'`,
			},
			{
				Method:      http.MethodPatch,
				Path:        "/api/v1/admin/packages/detergents/:id",
				Description: "Update a Detergents department package (admin only).",
				Auth:        true,
				AdminOnly:   true,
				Request: map[string]any{
					"id":    "string (optional)",
					"name":  "string (optional)",
					"price": "integer (optional)",
					"items": "string (optional)",
				},
				ResponseSuccess: models.NewSuccessResponse("req-abc123", "Package updated successfully", map[string]any{
					"id":    "mawohonte",
					"name":  "Ma Wo Ho Nte",
					"price": 270,
					"items": "Madar / Kleesoft 400g (¼ Box)",
				}),
				Example: `curl -X PATCH http://localhost:8080/api/v1/admin/packages/detergents/mawohonte -b cookies.txt -H "Content-Type: application/json" -d '{
  "price":490,
  "items":"Madar / Kleesoft 400g (½ Box) · Madar / Kleesoft Soap (½ Box)"
}'`,
			},
			{
				Method:          http.MethodDelete,
				Path:            "/api/v1/admin/packages/detergents/:id",
				Description:     "Deactivate a Detergents department package (admin only).",
				Auth:            true,
				AdminOnly:       true,
				ResponseSuccess: models.NewSuccessResponse("req-abc123", "Detergent package deactivated successfully", nil),
				Example:         `curl -X DELETE http://localhost:8080/api/v1/admin/packages/detergents/mawohonte -b cookies.txt`,
			},
			{
				Method:      http.MethodGet,
				Path:        "/api/v1/admin/applications",
				Description: "List applications from all users (admin only), optionally filtered by status. Results are paginated and each application includes a lightweight customer object for dashboard display.",
				Auth:        true,
				AdminOnly:   true,
				Request: map[string]string{
					"status": "string (optional, filter by status)",
					"page":   "integer (optional, default 1)",
					"limit":  "integer (optional, default 20, max 100)",
				},
				Response: map[string]any{
					"applications": []map[string]any{
						{
							"id":      "550e8400-e29b-41d4-a716-446655440010",
							"user_id": "550e8400-e29b-41d4-a716-446655440000",
							"customer": map[string]any{
								"id":           "550e8400-e29b-41d4-a716-446655440000",
								"display_name": "Kwame",
								"phone_number": "+233240000000",
								"role":         "customer",
							},
							"package_type": "custom",
							"cart_items": []map[string]any{
								{
									"product_id": "550e8400-e29b-41d4-a716-446655440001",
									"name":       "Royal Aroma Rice 5kg",
									"image_url":  "https://res.cloudinary.com/demo/image/upload/v1/rice.jpg",
									"price":      120,
									"quantity":   2,
									"subtotal":   240,
								},
							},
							"total_amount":      650,
							"monthly_amount":    217,
							"status":            "pending",
							"staff_number":      "GS123456",
							"mandate_number":    "MND-001",
							"institution":       "Ghana Health Service",
							"ghana_card_number": "GHA-000-0000-0",
							"created_at":        "2026-04-28T12:00:00Z",
							"updated_at":        "2026-04-28T12:00:00Z",
						},
					},
					"meta": map[string]any{
						"total":       87,
						"page":        1,
						"limit":       20,
						"total_pages": 5,
						"has_next":    true,
						"has_prev":    false,
					},
				},
				Example: `curl 'http://localhost:8080/api/v1/admin/applications?status=pending&page=1&limit=20' \
  -b cookies.txt`,
			},
			{
				Method:      http.MethodPatch,
				Path:        "/api/v1/admin/applications/:id",
				Description: "Update the status of an application (admin only). Valid statuses: 'pending' (initial), 'reviewed' (admin has looked at it), 'approved' (accepted), 'declined' (rejected).",
				Auth:        true,
				AdminOnly:   true,
				Request: map[string]string{
					"status": "string ('pending', 'reviewed', 'approved', or 'declined')",
				},
				Example: `curl -X PATCH http://localhost:8080/api/v1/admin/applications/550e8400-e29b-41d4-a716-446655440000 \
  -H "Content-Type: application/json" \
  -b cookies.txt \
  -d '{"status":"approved"}'`,
			},
			{
				Method:      http.MethodGet,
				Path:        "/api/v1/admin/conversations",
				Description: "List all customer conversations in the shared admin inbox. Any authenticated admin can see the same thread list, with customer profile, last message, unread count, and pagination metadata.",
				Auth:        true,
				AdminOnly:   true,
				Request: map[string]string{
					"page":  "integer (optional, default 1)",
					"limit": "integer (optional, default 20, max 100)",
				},
				Response: map[string]any{
					"conversations": []map[string]any{
						{
							"id": "550e8400-e29b-41d4-a716-446655440020",
							"other_user": map[string]any{
								"id":           "550e8400-e29b-41d4-a716-446655440000",
								"display_name": "Kwame",
								"phone_number": "+233240000000",
								"role":         "customer",
							},
							"last_message": "Your application is approved.",
							"unread_count": 0,
							"created_at":   "2026-04-28T12:00:00Z",
						},
					},
					"meta": map[string]any{
						"total":       42,
						"page":        1,
						"limit":       20,
						"total_pages": 3,
						"has_next":    true,
						"has_prev":    false,
					},
				},
				Example: `curl 'http://localhost:8080/api/v1/admin/conversations?page=1&limit=20' \
  -b cookies.txt`,
			},
			{
				Method:      http.MethodPost,
				Path:        "/api/v1/admin/conversations/:id/messages",
				Description: "Admin sends a message to a customer in an existing conversation. Same semantics as customer messages but admin can initiate or reply.",
				Auth:        true,
				AdminOnly:   true,
				Request: map[string]string{
					"content": "string (message text, required)",
				},
				Example: `curl -X POST http://localhost:8080/api/v1/admin/conversations/550e8400-e29b-41d4-a716-446655440000/messages \
  -H "Content-Type: application/json" \
  -b cookies.txt \
  -d '{"content":"Your application has been approved"}'`,
			},
		},
		Notes: []string{
			"Versioning: All endpoints are mounted under /api/v1",
			"Authentication: Use httpOnly cookies (access_token, refresh_token) from login/verify-otp, or Authorization: Bearer <token> header for API clients",
			"Activation: Signup creates an inactive account and sends an OTP by SMS. Call POST /api/v1/auth/verify-otp before first login",
			"Token refresh: Call POST /api/v1/auth/refresh with refresh_token cookie to rotate tokens",
			"User roles: 'customer' for regular users, 'admin' for the bootstrap admin created from ADMIN_PHONE_NUMBER. Additional admins can be promoted later. Admin endpoints require role='admin'",
			"Minimum order: Custom packages require GHC 549 minimum. Fixed packages have predefined totals",
			"Pagination: List endpoints accept page and limit query params. Responses include data.meta with total, page, limit, total_pages, has_next, and has_prev",
			"Error responses: Field-level validation errors use field names as keys. Domain errors use keys like 'auth', 'server', 'cart'",
			"Timestamps: All dates are ISO 8601 UTC format (RFC 3339). Frontend handles timezone conversion",
			"Request IDs: Every response includes request_id in metadata. Use this for debugging and support tickets",
			"Rate limiting: Global per-IP limiter applies to /api/* routes (RATE_LIMIT_PER_MINUTE) and a stricter auth limiter applies to /api/v1/auth/* (AUTH_RATE_LIMIT_PER_MINUTE)",
			"CORS: Allowed origins are configured via CORS_ALLOWED_ORIGINS. Default development origin is http://localhost:5173",
			"Docs: Runtime docs are available at GET / and GET /api/v1/docs",
			"Offline mode: No offline sync yet. All requests require active network connection",
		},
	}

	return models.NewDocResponse("doc-root", doc)
}
