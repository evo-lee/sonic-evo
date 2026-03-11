package middleware

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"time"

	"github.com/go-sonic/sonic/cache"
	"github.com/go-sonic/sonic/handler/web"
)

const (
	CSRFTokenHeader = "X-CSRF-Token"
	CSRFTokenCookie = "CSRF-TOKEN"
	CSRFCachePrefix = "csrf:token:"
	CSRFTokenLength = 32
	CSRFTokenExpiry = 24 * time.Hour
)

type CSRFMiddleware struct {
	Cache cache.Cache
}

func NewCSRFMiddleware(cache cache.Cache) *CSRFMiddleware {
	return &CSRFMiddleware{
		Cache: cache,
	}
}

// Handler returns a middleware that validates CSRF tokens for state-changing requests
func (c *CSRFMiddleware) Handler() web.HandlerFunc {
	return func(ctx web.Context) {
		method := ctx.Method()

		// Only validate CSRF for state-changing methods
		if method == http.MethodPost || method == http.MethodPut ||
			method == http.MethodDelete || method == http.MethodPatch {

			// Get token from header
			tokenFromHeader := ctx.Header(CSRFTokenHeader)
			if tokenFromHeader == "" {
				abortWithStatusJSON(ctx, http.StatusForbidden, "CSRF token missing")
				return
			}

			// Get token from cookie
			tokenFromCookie, err := ctx.Cookie(CSRFTokenCookie)
			if err != nil {
				abortWithStatusJSON(ctx, http.StatusForbidden, "CSRF token cookie missing")
				return
			}

			// Tokens must match
			if tokenFromHeader != tokenFromCookie {
				abortWithStatusJSON(ctx, http.StatusForbidden, "CSRF token mismatch")
				return
			}

			// Verify token exists in cache (not expired)
			cacheKey := CSRFCachePrefix + tokenFromCookie
			_, exists := c.Cache.Get(cacheKey)
			if !exists {
				abortWithStatusJSON(ctx, http.StatusForbidden, "CSRF token expired or invalid")
				return
			}

			// Token is valid, continue
			ctx.Next()
		} else {
			// For GET/HEAD/OPTIONS requests, generate and set a new token if one doesn't exist
			tokenFromCookie, err := ctx.Cookie(CSRFTokenCookie)
			if err != nil || tokenFromCookie == "" {
				// Generate new token
				token, err := generateCSRFToken()
				if err != nil {
					abortWithStatusJSON(ctx, http.StatusInternalServerError, "Failed to generate CSRF token")
					return
				}

				// Store in cache
				cacheKey := CSRFCachePrefix + token
				c.Cache.Set(cacheKey, true, CSRFTokenExpiry)

				// Set cookie
				ctx.SetCookie(CSRFTokenCookie, token, int(CSRFTokenExpiry.Seconds()), "/", "", false, true)

				// Also set in response header for SPA clients
				ctx.SetHeader(CSRFTokenHeader, token)
			}

			ctx.Next()
		}
	}
}

// generateCSRFToken generates a cryptographically secure random token
func generateCSRFToken() (string, error) {
	bytes := make([]byte, CSRFTokenLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}
