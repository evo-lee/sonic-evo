package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/go-sonic/sonic/handler/web"
)

// TimeoutMiddleware enforces request-level timeouts
type TimeoutMiddleware struct {
	timeout time.Duration
}

// TimeoutConfig configures timeout behavior
type TimeoutConfig struct {
	Timeout time.Duration // Request timeout duration
}

// NewTimeoutMiddleware creates a new timeout middleware
func NewTimeoutMiddleware(config TimeoutConfig) *TimeoutMiddleware {
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second // Default 30 seconds
	}

	return &TimeoutMiddleware{
		timeout: config.Timeout,
	}
}

// Handler returns the timeout middleware handler
func (t *TimeoutMiddleware) Handler() web.HandlerFunc {
	return func(ctx web.Context) {
		// Create a context with timeout
		timeoutCtx, cancel := context.WithTimeout(ctx.RequestContext(), t.timeout)
		defer cancel()

		// Create a channel to signal completion
		done := make(chan struct{})

		// Run the handler in a goroutine
		go func() {
			ctx.Next()
			close(done)
		}()

		// Wait for either completion or timeout
		select {
		case <-done:
			// Request completed successfully
			return
		case <-timeoutCtx.Done():
			// Timeout occurred
			if timeoutCtx.Err() == context.DeadlineExceeded {
				abortWithStatusJSON(ctx, http.StatusGatewayTimeout, "Request timeout")
			}
			return
		}
	}
}

// WithDBTimeout returns a context with database operation timeout
func WithDBTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout == 0 {
		timeout = 5 * time.Second // Default 5 seconds for DB operations
	}
	return context.WithTimeout(ctx, timeout)
}

// WithAPITimeout returns a context with API call timeout
func WithAPITimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout == 0 {
		timeout = 10 * time.Second // Default 10 seconds for API calls
	}
	return context.WithTimeout(ctx, timeout)
}
