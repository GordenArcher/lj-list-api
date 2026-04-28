package middleware

import (
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/GordenArcher/lj-list-api/internal/apperrors"
	"github.com/GordenArcher/lj-list-api/internal/config"
	"github.com/GordenArcher/lj-list-api/internal/utils"
	"github.com/gin-gonic/gin"
)

const (
	rateLimitWindow   = time.Minute
	rateLimiterTTL    = 10 * time.Minute
	globalLimiterName = "global"
	authLimiterName   = "auth"
)

type rateLimiterEntry struct {
	tokens     float64
	lastRefill time.Time
	lastSeen   time.Time
}

type ipRateLimiter struct {
	mu      sync.Mutex
	clients map[string]*rateLimiterEntry

	requests        int
	refillPerSecond float64
	capacity        float64

	name      string
	shouldRun func(*gin.Context) bool
}

// GlobalRateLimit applies a per-IP limiter to API endpoints.
// It is intentionally scoped to /api so docs/health endpoints are not throttled.
func GlobalRateLimit(cfg config.Config) gin.HandlerFunc {
	return newIPRateLimiter(
		globalLimiterName,
		cfg.RateLimitPerMinute,
		func(c *gin.Context) bool {
			return strings.HasPrefix(c.Request.URL.Path, "/api/")
		},
	)
}

// AuthRateLimit applies an additional stricter limiter for auth endpoints.
// This runs alongside the global limiter to slow brute-force/login abuse.
func AuthRateLimit(cfg config.Config) gin.HandlerFunc {
	return newIPRateLimiter(authLimiterName, cfg.AuthRateLimitPerMinute, func(_ *gin.Context) bool {
		return true
	})
}

// RateLimit is kept as backward-compatible alias for existing wiring.
func RateLimit(cfg config.Config) gin.HandlerFunc {
	return GlobalRateLimit(cfg)
}

func newIPRateLimiter(name string, requests int, shouldRun func(*gin.Context) bool) gin.HandlerFunc {
	if requests <= 0 {
		return func(c *gin.Context) { c.Next() }
	}

	// Keep average throughput equal to requests/minute, but cap immediate
	// burst so one client cannot dump the whole minute budget instantly.
	burst := requests / 4
	if burst < 1 {
		burst = 1
	}
	if requests >= 20 && burst < 5 {
		burst = 5
	}
	if burst > requests {
		burst = requests
	}

	limiter := &ipRateLimiter{
		clients:         make(map[string]*rateLimiterEntry),
		requests:        requests,
		refillPerSecond: float64(requests) / rateLimitWindow.Seconds(),
		capacity:        float64(burst),
		name:            name,
		shouldRun:       shouldRun,
	}

	return func(c *gin.Context) {
		if c.Request.Method == http.MethodOptions || !limiter.shouldRun(c) {
			c.Next()
			return
		}

		clientIP := c.ClientIP()
		if clientIP == "" {
			clientIP = "unknown"
		}

		now := time.Now().UTC()
		allowed, remaining, resetAt := limiter.allow(clientIP, now)

		c.Header("X-RateLimit-Limit", strconv.Itoa(limiter.requests))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(resetAt.Unix(), 10))
		c.Header("X-RateLimit-Scope", limiter.name)

		if !allowed {
			retryAfter := int(math.Ceil(time.Until(resetAt).Seconds()))
			if retryAfter < 1 {
				retryAfter = 1
			}
			c.Header("Retry-After", strconv.Itoa(retryAfter))

			utils.HandleError(c, apperrors.New(
				apperrors.KindRateLimited,
				"Too many requests. Please try again later.",
				map[string][]string{
					"rate_limit": {"request limit exceeded"},
				},
			), "")
			c.Abort()
			return
		}

		c.Next()
	}
}

func (l *ipRateLimiter) allow(clientIP string, now time.Time) (bool, int, time.Time) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.cleanup(now)

	entry, exists := l.clients[clientIP]
	if !exists {
		entry = &rateLimiterEntry{
			tokens:     l.capacity,
			lastRefill: now,
			lastSeen:   now,
		}
		l.clients[clientIP] = entry
	}
	entry.lastSeen = now

	l.refill(entry, now)

	allowed := false
	if entry.tokens >= 1 {
		entry.tokens -= 1
		allowed = true
	}

	remaining := int(math.Floor(entry.tokens))
	if remaining < 0 {
		remaining = 0
	}

	resetAt := l.nextTokenAt(entry, now)

	return allowed, remaining, resetAt
}

func (l *ipRateLimiter) refill(entry *rateLimiterEntry, now time.Time) {
	elapsed := now.Sub(entry.lastRefill).Seconds()
	if elapsed <= 0 {
		return
	}

	entry.tokens += elapsed * l.refillPerSecond
	if entry.tokens > l.capacity {
		entry.tokens = l.capacity
	}
	entry.lastRefill = now
}

func (l *ipRateLimiter) nextTokenAt(entry *rateLimiterEntry, now time.Time) time.Time {
	if entry.tokens >= 1 {
		return now
	}
	if l.refillPerSecond <= 0 {
		return now.Add(rateLimitWindow)
	}

	needed := 1 - entry.tokens
	if needed < 0 {
		needed = 0
	}

	delaySeconds := needed / l.refillPerSecond
	delay := time.Duration(math.Ceil(delaySeconds * float64(time.Second)))
	if delay < time.Second {
		delay = time.Second
	}

	return now.Add(delay)
}

func (l *ipRateLimiter) cleanup(now time.Time) {
	for clientIP, entry := range l.clients {
		if now.Sub(entry.lastSeen) > rateLimiterTTL {
			delete(l.clients, clientIP)
		}
	}
}
