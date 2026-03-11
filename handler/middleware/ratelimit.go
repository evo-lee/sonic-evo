package middleware

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/go-sonic/sonic/handler/web"
)

// RateLimiter implements a token bucket rate limiter
type RateLimiter struct {
	mu       sync.RWMutex
	buckets  map[string]*bucket
	rate     int           // tokens per window
	window   time.Duration // time window
	maxBurst int           // maximum burst size
	cleanup  time.Duration // cleanup interval for old buckets
}

type bucket struct {
	tokens     int
	lastRefill time.Time
	mu         sync.Mutex
}

type RateLimitMiddleware struct {
	limiter *RateLimiter
}

// RateLimitConfig configures rate limiting behavior
type RateLimitConfig struct {
	RequestsPerWindow int           // number of requests allowed per window
	Window            time.Duration // time window duration
	MaxBurst          int           // maximum burst size (0 = same as RequestsPerWindow)
	KeyFunc           func(web.Context) string
}

// NewRateLimitMiddleware creates a new rate limit middleware with the given configuration
func NewRateLimitMiddleware(config RateLimitConfig) *RateLimitMiddleware {
	if config.MaxBurst == 0 {
		config.MaxBurst = config.RequestsPerWindow
	}
	if config.KeyFunc == nil {
		config.KeyFunc = defaultKeyFunc
	}

	limiter := &RateLimiter{
		buckets:  make(map[string]*bucket),
		rate:     config.RequestsPerWindow,
		window:   config.Window,
		maxBurst: config.MaxBurst,
		cleanup:  config.Window * 2,
	}

	// Start cleanup goroutine
	go limiter.cleanupLoop()

	return &RateLimitMiddleware{
		limiter: limiter,
	}
}

// Handler returns the rate limiting middleware handler
func (r *RateLimitMiddleware) Handler(keyFunc func(web.Context) string) web.HandlerFunc {
	if keyFunc == nil {
		keyFunc = defaultKeyFunc
	}

	return func(ctx web.Context) {
		key := keyFunc(ctx)

		allowed, remaining, resetTime := r.limiter.allow(key)

		// Set rate limit headers
		ctx.SetHeader("X-RateLimit-Limit", formatInt(r.limiter.rate))
		ctx.SetHeader("X-RateLimit-Remaining", formatInt(remaining))
		ctx.SetHeader("X-RateLimit-Reset", formatInt64(resetTime.Unix()))

		if !allowed {
			ctx.SetHeader("Retry-After", formatInt(int(time.Until(resetTime).Seconds())))
			abortWithStatusJSON(ctx, http.StatusTooManyRequests, "Rate limit exceeded. Please try again later.")
			return
		}

		ctx.Next()
	}
}

// allow checks if a request is allowed for the given key
func (rl *RateLimiter) allow(key string) (allowed bool, remaining int, resetTime time.Time) {
	rl.mu.RLock()
	b, exists := rl.buckets[key]
	rl.mu.RUnlock()

	if !exists {
		rl.mu.Lock()
		// Double-check after acquiring write lock
		b, exists = rl.buckets[key]
		if !exists {
			b = &bucket{
				tokens:     rl.maxBurst,
				lastRefill: time.Now(),
			}
			rl.buckets[key] = b
		}
		rl.mu.Unlock()
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(b.lastRefill)

	// Refill tokens based on elapsed time
	if elapsed >= rl.window {
		periods := int(elapsed / rl.window)
		tokensToAdd := periods * rl.rate
		b.tokens = min(b.tokens+tokensToAdd, rl.maxBurst)
		b.lastRefill = b.lastRefill.Add(time.Duration(periods) * rl.window)
	}

	// Calculate reset time (when next token will be available)
	resetTime = b.lastRefill.Add(rl.window)

	if b.tokens > 0 {
		b.tokens--
		return true, b.tokens, resetTime
	}

	return false, 0, resetTime
}

// cleanupLoop periodically removes old buckets to prevent memory leaks
func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(rl.cleanup)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for key, b := range rl.buckets {
			b.mu.Lock()
			if now.Sub(b.lastRefill) > rl.cleanup {
				delete(rl.buckets, key)
			}
			b.mu.Unlock()
		}
		rl.mu.Unlock()
	}
}

// defaultKeyFunc uses client IP as the rate limit key
func defaultKeyFunc(ctx web.Context) string {
	return ctx.ClientIP()
}

// AuthKeyFunc uses client IP + user ID for authenticated requests
func AuthKeyFunc(ctx web.Context) string {
	ip := ctx.ClientIP()
	// Try to get user from context
	if user, ok := ctx.Get("user"); ok {
		return ip + ":" + string(rune(user.(int)))
	}
	return ip
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func formatInt(n int) string {
	return strconv.Itoa(n)
}

func formatInt64(n int64) string {
	return strconv.FormatInt(n, 10)
}
