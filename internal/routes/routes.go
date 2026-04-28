package routes

import (
	"net/http"

	"github.com/GordenArcher/lj-list-api/internal/config"
	"github.com/GordenArcher/lj-list-api/internal/handlers"
	"github.com/GordenArcher/lj-list-api/internal/middleware"
	"github.com/GordenArcher/lj-list-api/internal/models"
	"github.com/GordenArcher/lj-list-api/internal/repositories"
	"github.com/GordenArcher/lj-list-api/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	// apiBasePrefix is kept separate from the version segment so we can
	// support multiple versions side-by-side later (for example /api/v2)
	// without rewriting the entire route tree.
	apiBasePrefix = "/api"

	// apiVersion is the active major version. Breaking API changes should bump
	// this value and register a parallel route set rather than mutating v1.
	apiVersion = "v1"

	// apiVersionPrefix is the concrete router group mounted in Gin.
	apiVersionPrefix = apiBasePrefix + "/" + apiVersion
)

// Register wires every dependency and attaches routes to the Gin router.
// Dependencies flow: pool → repository → service → handler → route.
// Nothing is pulled from a global. Every handler receives exactly what it
// needs through its constructor. This function is the only place in the
// codebase that knows how everything connects — if a dependency graph
// changes, this is the single file that changes with it.
func Register(router *gin.Engine, pool *pgxpool.Pool, cfg config.Config) {
	// Repositories
	userRepo := repositories.NewUserRepository(pool)
	productRepo := repositories.NewProductRepository(pool)
	applicationRepo := repositories.NewApplicationRepository(pool)
	conversationRepo := repositories.NewConversationRepository(pool)
	messageRepo := repositories.NewMessageRepository(pool)

	// Services
	authService := services.NewAuthService(userRepo, cfg)
	productService := services.NewProductService(productRepo)
	applicationService := services.NewApplicationService(applicationRepo, productRepo, cfg)
	conversationService := services.NewConversationService(conversationRepo, userRepo)
	messageService := services.NewMessageService(messageRepo, conversationRepo)

	// Handlers
	authHandler := handlers.NewAuthHandler(authService, cfg)
	productHandler := handlers.NewProductHandler(productService)
	applicationHandler := handlers.NewApplicationHandler(applicationService)
	conversationHandler := handlers.NewConversationHandler(conversationService, userRepo, cfg)
	messageHandler := handlers.NewMessageHandler(messageService)
	adminHandler := handlers.NewAdminHandler(applicationService)

	// Versioned API root. Every endpoint in this service lives under /api/v1.
	// If we ever add v2, we mount a second group and keep v1 stable.
	v1 := router.Group(apiVersionPrefix)

	// Public auth routes with stricter limiter.
	// Refresh is intentionally public because the refresh-token cookie is the credential.
	authPublic := v1.Group("/auth")
	authPublic.Use(middleware.AuthRateLimit(cfg))
	authPublic.POST("/signup", authHandler.Signup)
	authPublic.POST("/login", authHandler.Login)
	authPublic.POST("/refresh", authHandler.Refresh)

	// Public non-auth routes.
	v1.GET("/products", productHandler.List)
	v1.GET("/products/categories", productHandler.Categories)

	// Protected auth route (logout) keeps stricter auth limiter too.
	authPrivate := v1.Group("/auth")
	authPrivate.Use(middleware.AuthRequired(cfg), middleware.AuthRateLimit(cfg))
	authPrivate.POST("/logout", authHandler.Logout)

	// Customer routes, require valid JWT cookie or Authorization header.
	customer := v1.Group("")
	customer.Use(middleware.AuthRequired(cfg))
	{
		customer.POST("/applications", applicationHandler.Create)
		customer.GET("/applications", applicationHandler.List)
		customer.GET("/applications/:id", applicationHandler.Get)

		customer.GET("/conversations", conversationHandler.List)
		customer.POST("/conversations", conversationHandler.Create)
		customer.GET("/conversations/:id/messages", messageHandler.List)
		customer.POST("/conversations/:id/messages", messageHandler.Send)
	}

	// Admin routes, require valid JWT cookie + admin role.
	admin := v1.Group("/admin")
	admin.Use(middleware.AuthRequired(cfg), middleware.AdminRequired)
	{
		admin.GET("/applications", adminHandler.ListApplications)
		admin.PATCH("/applications/:id", adminHandler.UpdateApplication)
		admin.GET("/conversations", conversationHandler.List)
		admin.POST("/conversations/:id/messages", messageHandler.Send)
	}

	// Self-documenting root endpoint. Returns every route, method, auth
	// requirement, and a working curl example. Frontend devs can open this
	// in a browser and integrate without reading source code.
	router.GET("/", func(c *gin.Context) {
		doc := buildAPIDocumentation()
		c.JSON(200, doc)
	})
	v1.GET("/docs", func(c *gin.Context) {
		doc := buildAPIDocumentation()
		c.JSON(200, doc)
	})
}

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
				Description: "Create a new customer account or admin account. If the email matches ADMIN_EMAIL env var, role is 'admin', otherwise 'customer'. The password is hashed with bcrypt before storage.",
				Auth:        false,
				Request: map[string]string{
					"email":        "string (valid email required)",
					"password":     "string (minimum 8 characters)",
					"display_name": "string (2-100 characters)",
				},
				ResponseSuccess: models.NewSuccessResponse("req-abc123", "Account created successfully", map[string]any{
					"user": map[string]string{
						"id":           "550e8400-e29b-41d4-a716-446655440000",
						"email":        "kwame@email.com",
						"display_name": "Kwame",
						"phone":        "",
						"role":         "customer",
					},
				}),
				Example: `curl -X POST http://localhost:8080/api/v1/auth/signup \
  -H "Content-Type: application/json" \
  -d '{
    "email":"kwame@email.com",
    "password":"password123",
    "display_name":"Kwame"
  }'`,
			},
			{
				Method:      http.MethodPost,
				Path:        "/api/v1/auth/login",
				Description: "Authenticate with email and password. Returns an access_token cookie (15 min expiry) and refresh_token cookie (7 days expiry). Both are httpOnly for XSS protection.",
				Auth:        false,
				Request: map[string]string{
					"email":    "string (valid email required)",
					"password": "string",
				},
				ResponseSuccess: models.NewSuccessResponse("req-abc123", "Login successful", map[string]any{
					"user": map[string]string{
						"id":           "550e8400-e29b-41d4-a716-446655440000",
						"email":        "kwame@email.com",
						"display_name": "Kwame",
						"phone":        "",
						"role":         "customer",
					},
				}),
				Example: `curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -c cookies.txt \
  -d '{
    "email":"kwame@email.com",
    "password":"password123"
  }'`,
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

			// Product Endpoints
			{
				Method:      http.MethodGet,
				Path:        "/api/v1/products",
				Description: "List active products with optional category filtering. This endpoint is paginated using page and limit query params.",
				Auth:        false,
				Request: map[string]string{
					"category": "string (optional, e.g., 'Rice & Grains')",
					"page":     "integer (optional, default 1)",
					"limit":    "integer (optional, default 20, max 100)",
				},
				Response: map[string]any{
					"products": []map[string]any{
						{
							"id":        "550e8400-e29b-41d4-a716-446655440001",
							"name":      "Royal Aroma Rice 5kg",
							"category":  "Rice & Grains",
							"price":     120,
							"image_url": "https://res.cloudinary.com/demo/image/upload/v1/rice.jpg",
							"unit":      "bag",
							"active":    true,
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
				Example: `curl 'http://localhost:8080/api/v1/products?category=Rice%20%26%20Grains&page=1&limit=20'`,
			},
			{
				Method:      http.MethodGet,
				Path:        "/api/v1/products/categories",
				Description: "List all distinct product categories. Use this to populate category filters on the frontend.",
				Auth:        false,
				Example:     `curl http://localhost:8080/api/v1/products/categories`,
			},

			// Application Endpoints (Customer)
			{
				Method:      http.MethodPost,
				Path:        "/api/v1/applications",
				Description: "Submit a new grocery application. package_type is 'fixed' (predefined sets) or 'custom' (build your own cart). Cart total must exceed GHC 549 (MIN_ORDER config). Staff number, mandate number, institution, and Ghana card are required for government verification.",
				Auth:        true,
				Request: map[string]any{
					"package_type":      "string ('fixed' or 'custom')",
					"package_name":      "string (required for fixed packages, e.g., 'Family Bundle')",
					"cart_items":        "[]object (required for custom packages)",
					"staff_number":      "string (government ID)",
					"mandate_number":    "string (mandate ID)",
					"institution":       "string (employer name)",
					"ghana_card_number": "string",
				},
				Example: `curl -X POST http://localhost:8080/api/v1/applications \
  -H "Content-Type: application/json" \
  -b cookies.txt \
  -d '{
    "package_type":"custom",
    "cart_items":[
      {"product_id":"uuid","quantity":2},
      {"product_id":"uuid","quantity":5}
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
				Description: "Start a new conversation with the store admin. If a conversation already exists with this user, returns the existing conversation instead of creating a duplicate.",
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
				Description: "List conversations for the authenticated user (or admin) with other-user profile, last message, and unread count. Results are paginated.",
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
								"email":        "admin@ljlist.com",
								"phone":        nil,
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
				Path:        "/api/v1/admin/applications",
				Description: "List applications from all users (admin only), optionally filtered by status. Results are paginated.",
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
				Description: "List conversations for the authenticated admin. Returns other-user profile, last message, unread count, and pagination metadata.",
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
								"email":        "kwame@email.com",
								"phone":        "0240000000",
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
			"Authentication: Use httpOnly cookies (access_token, refresh_token) from login/signup, or Authorization: Bearer <token> header for API clients",
			"Token refresh: Call POST /api/v1/auth/refresh with refresh_token cookie to rotate tokens",
			"User roles: 'customer' for regular users, 'admin' for account matching ADMIN_EMAIL env var. Admin endpoints require role='admin'",
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
