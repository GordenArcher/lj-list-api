package routes

import (
	"github.com/GordenArcher/lj-list-api/internal/config"
	"github.com/GordenArcher/lj-list-api/internal/handlers"
	"github.com/GordenArcher/lj-list-api/internal/middleware"
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
	packageRepo := repositories.NewPackageRepository(pool)
	categoryRepo := repositories.NewCategoryRepository(pool)
	productImageRepo := repositories.NewProductImageRepository(pool)
	applicationRepo := repositories.NewApplicationRepository(pool)
	conversationRepo := repositories.NewConversationRepository(pool)
	messageRepo := repositories.NewMessageRepository(pool)
	dashboardRepo := repositories.NewDashboardRepository(pool)

	// Services
	smsService := services.NewSMSService(cfg, userRepo)
	authService := services.NewAuthService(userRepo, smsService, cfg)
	userService := services.NewUserService(userRepo, cfg)
	productService := services.NewProductService(productRepo, productImageRepo, cfg)
	packageService := services.NewPackageService(productRepo, productImageRepo, packageRepo, cfg)
	categoryService := services.NewCategoryService(categoryRepo)
	applicationService := services.NewApplicationService(applicationRepo, productRepo, packageRepo, userRepo, cfg)
	conversationService := services.NewConversationService(conversationRepo, userRepo)
	messageService := services.NewMessageService(messageRepo, conversationRepo)
	dashboardService := services.NewDashboardService(dashboardRepo, cfg)

	// Handlers
	authHandler := handlers.NewAuthHandler(authService, cfg)
	userHandler := handlers.NewUserHandler(userService)
	productHandler := handlers.NewProductHandler(productService)
	packageHandler := handlers.NewPackageHandler(packageService)
	categoryHandler := handlers.NewCategoryHandler(categoryService)
	dashboardHandler := handlers.NewDashboardHandler(dashboardService)
	applicationHandler := handlers.NewApplicationHandler(applicationService, smsService)
	conversationHandler := handlers.NewConversationHandler(conversationService, smsService)
	messageHandler := handlers.NewMessageHandler(messageService, smsService)
	adminHandler := handlers.NewAdminHandler(applicationService)

	// Versioned API root. Every endpoint in this service lives under /api/v1.
	// If we ever add v2, we mount a second group and keep v1 stable.
	v1 := router.Group(apiVersionPrefix)

	// Public auth routes with stricter limiter.
	// Refresh is intentionally public because the refresh-token cookie is the credential.
	authPublic := v1.Group("/auth")
	authPublic.Use(middleware.AuthRateLimit(cfg))
	authPublic.POST("/signup", authHandler.Signup)
	authPublic.POST("/verify-otp", authHandler.VerifyOTP)
	authPublic.POST("/resend-otp", authHandler.ResendOTP)
	authPublic.POST("/login", authHandler.Login)
	authPublic.POST("/refresh", authHandler.Refresh)

	adminAuth := v1.Group("/admin/auth")
	adminAuth.Use(middleware.AuthRateLimit(cfg))
	adminAuth.POST("/login", authHandler.AdminLogin)

	// Public non-auth routes.
	v1.GET("/products", productHandler.List)
	v1.GET("/products/:id", productHandler.Get)
	v1.GET("/products/categories", productHandler.Categories)
	v1.GET("/packages", packageHandler.List)
	v1.GET("/packages/fixed", packageHandler.Fixed)
	v1.GET("/packages/provisions", packageHandler.Provisions)
	v1.GET("/packages/detergents", packageHandler.Detergents)

	// Protected auth route (logout) keeps stricter auth limiter too.
	authPrivate := v1.Group("/auth")
	authPrivate.Use(middleware.AuthRequired(cfg), middleware.AuthRateLimit(cfg))
	authPrivate.POST("/logout", authHandler.Logout)

	// Customer routes, require valid JWT cookie or Authorization header.
	customer := v1.Group("")
	customer.Use(middleware.AuthRequired(cfg))
	{
		customer.GET("/profile", userHandler.GetProfile)
		customer.PATCH("/profile", userHandler.UpdateProfile)

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
		admin.GET("/categories", categoryHandler.List)
		admin.GET("/categories/:id", categoryHandler.Get)
		admin.POST("/categories", categoryHandler.Create)
		admin.PATCH("/categories/:id", categoryHandler.Update)
		admin.DELETE("/categories/:id", categoryHandler.Delete)
		admin.GET("/products", productHandler.AdminList)
		admin.GET("/products/:id", productHandler.AdminGet)
		admin.POST("/products", productHandler.Create)
		admin.PATCH("/products/:id", productHandler.Update)
		admin.DELETE("/products/:id", productHandler.Delete)
		admin.GET("/products/:id/images", productHandler.ListImages)
		admin.POST("/products/:id/images", productHandler.AddImages)
		admin.DELETE("/products/:id/images/:imageId", productHandler.DeleteImage)
		admin.GET("/packages", packageHandler.AdminList)
		admin.GET("/packages/fixed", packageHandler.FixedAdmin)
		admin.GET("/packages/fixed/:id", packageHandler.FixedAdminByID)
		admin.POST("/packages/fixed", packageHandler.CreateFixed)
		admin.PATCH("/packages/fixed/:id", packageHandler.UpdateFixed)
		admin.DELETE("/packages/fixed/:id", packageHandler.DeleteFixed)
		admin.GET("/packages/provisions", packageHandler.ProvisionsAdmin)
		admin.GET("/packages/provisions/:id", packageHandler.ProvisionsAdminByID)
		admin.POST("/packages/provisions", packageHandler.CreateProvisions)
		admin.PATCH("/packages/provisions/:id", packageHandler.UpdateProvisions)
		admin.DELETE("/packages/provisions/:id", packageHandler.DeleteProvisions)
		admin.GET("/packages/detergents", packageHandler.DetergentsAdmin)
		admin.GET("/packages/detergents/:id", packageHandler.DetergentsAdminByID)
		admin.POST("/packages/detergents", packageHandler.CreateDetergents)
		admin.PATCH("/packages/detergents/:id", packageHandler.UpdateDetergents)
		admin.DELETE("/packages/detergents/:id", packageHandler.DeleteDetergents)
		admin.GET("/users", userHandler.ListUsers)
		admin.PATCH("/users/:id", userHandler.UpdateUser)
		admin.GET("/applications", adminHandler.ListApplications)
		admin.PATCH("/applications/:id", adminHandler.UpdateApplication)
		admin.GET("/dashboard", dashboardHandler.Stats)
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
