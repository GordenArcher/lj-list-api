package config

import (
	"strings"

	"github.com/GordenArcher/godenv"
)

// Config holds every setting the application needs. It's a concrete struct,
// not a global, packages receive it via dependency injection. This means
// tests can inject a custom Config without touching environment variables.
type Config struct {
	Port        string
	DatabaseURL string
	JWTSecret   string
	// RateLimitPerMinute is the global per-IP request cap applied by
	// middleware. Set to 0 to disable rate limiting.
	RateLimitPerMinute int
	// AuthRateLimitPerMinute is an additional stricter limiter for auth routes
	// (signup/login/refresh/logout). Set to 0 to disable auth-route limiter.
	AuthRateLimitPerMinute int

	// Cloudinary is our default storage backend. Product images and chat
	// attachments all go through these credentials. If we switch providers
	// later, only the storage package changes.
	CloudinaryCloudName string
	CloudinaryAPIKey    string
	CloudinaryAPISecret string

	// Hubtel is our SMS provider for Ghana. Used to notify the admin about
	// new applications and new chat messages.
	HubtelAPIKey   string
	HubtelSenderID string

	// AllowedOrigins is the CORS allowlist. Default includes Vite dev server.
	AllowedOrigins []string

	// Cookie settings for auth tokens.
	CookieSecure   bool
	CookieDomain   string
	CookieSameSite string

	// AdminEmail is the address that gets auto-promoted to admin on signup.
	// Set this before deploying so the business owner can create an account
	// and immediately access admin endpoints.
	AdminEmail string

	// MinOrder is the minimum cart total in Ghana cedis. Applications below
	// this threshold are rejected. Lives in config rather than a constant
	// so different deployments can adjust it without recompiling.
	MinOrder int
}

// Load reads configuration from environment variables using godenv.Get.
// Sensible defaults are provided where possible. Required values (like
// DATABASE_URL) will be empty strings if not set, the packages that use
// them are responsible for validating.
func Load() Config {
	return Config{
		Port:        godenv.Get("PORT", "8080"),
		DatabaseURL: godenv.Get("DATABASE_URL", ""),
		JWTSecret:   godenv.Get("JWT_SECRET", ""),
		RateLimitPerMinute: parseNonNegativeInt(
			godenv.Get("RATE_LIMIT_PER_MINUTE", "120"),
			120,
		),
		AuthRateLimitPerMinute: parseNonNegativeInt(
			godenv.Get("AUTH_RATE_LIMIT_PER_MINUTE", "30"),
			30,
		),

		CloudinaryCloudName: godenv.Get("CLOUDINARY_CLOUD_NAME", ""),
		CloudinaryAPIKey:    godenv.Get("CLOUDINARY_API_KEY", ""),
		CloudinaryAPISecret: godenv.Get("CLOUDINARY_API_SECRET", ""),

		HubtelAPIKey:   godenv.Get("HUBTEL_API_KEY", ""),
		HubtelSenderID: godenv.Get("HUBTEL_SENDER_ID", "LJList"),
		AllowedOrigins: parseCSV(
			godenv.Get("CORS_ALLOWED_ORIGINS", "http://localhost:5173"),
			[]string{"http://localhost:5173"},
		),
		CookieSecure:   parseBool(godenv.Get("COOKIE_SECURE", "false"), false),
		CookieDomain:   godenv.Get("COOKIE_DOMAIN", ""),
		CookieSameSite: normalizeSameSite(godenv.Get("COOKIE_SAME_SITE", "Lax"), "Lax"),

		AdminEmail: godenv.Get("ADMIN_EMAIL", ""),
		MinOrder:   parsePositiveInt(godenv.Get("MIN_ORDER", "549"), 549),
	}
}

// parsePositiveInt converts a string to a positive integer with a fallback.
// Malformed values, empty strings, or non-positive numbers all resolve to
// fallback so bad env values never crash bootstrap.
func parsePositiveInt(s string, fallback int) int {
	if s == "" {
		return fallback
	}

	var n int
	for _, c := range s {
		if c < '0' || c > '9' {
			return fallback
		}
		n = n*10 + int(c-'0')
	}

	if n <= 0 {
		return fallback
	}

	return n
}

// parseNonNegativeInt converts a string to a non-negative integer with a
// fallback. Malformed values and empty strings resolve to fallback.
func parseNonNegativeInt(s string, fallback int) int {
	if s == "" {
		return fallback
	}

	var n int
	for _, c := range s {
		if c < '0' || c > '9' {
			return fallback
		}
		n = n*10 + int(c-'0')
	}

	return n
}

func parseBool(s string, fallback bool) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "1", "true", "yes", "y", "on":
		return true
	case "0", "false", "no", "n", "off":
		return false
	default:
		return fallback
	}
}

func parseCSV(s string, fallback []string) []string {
	raw := strings.Split(s, ",")
	values := make([]string, 0, len(raw))
	for _, part := range raw {
		v := strings.TrimSpace(part)
		if v != "" {
			values = append(values, v)
		}
	}
	if len(values) == 0 {
		return fallback
	}
	return values
}

func normalizeSameSite(s string, fallback string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "strict":
		return "Strict"
	case "lax":
		return "Lax"
	case "none":
		return "None"
	default:
		return fallback
	}
}
