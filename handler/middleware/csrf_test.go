package middleware

import (
	"testing"
	"time"

	"github.com/go-sonic/sonic/cache"
)

func TestGenerateCSRFToken(t *testing.T) {
	token1, err := generateCSRFToken()
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	if token1 == "" {
		t.Error("Generated token should not be empty")
	}

	// Generate another token to ensure uniqueness
	token2, err := generateCSRFToken()
	if err != nil {
		t.Fatalf("Failed to generate second token: %v", err)
	}

	if token1 == token2 {
		t.Error("Generated tokens should be unique")
	}

	// Check token length (32 bytes base64 encoded should be ~44 chars)
	if len(token1) < 40 {
		t.Errorf("Token seems too short: %d chars", len(token1))
	}
}

func TestCSRFMiddleware_Creation(t *testing.T) {
	c := cache.NewCache()
	csrfMiddleware := NewCSRFMiddleware(c)

	if csrfMiddleware == nil {
		t.Error("NewCSRFMiddleware should not return nil")
	}

	if csrfMiddleware.Cache == nil {
		t.Error("CSRFMiddleware should have a cache")
	}
}

func TestCSRFTokenCaching(t *testing.T) {
	c := cache.NewCache()
	token, _ := generateCSRFToken()
	cacheKey := CSRFCachePrefix + token

	// Store token in cache
	c.Set(cacheKey, true, CSRFTokenExpiry)

	// Verify it exists
	_, exists := c.Get(cacheKey)
	if !exists {
		t.Error("Token should exist in cache")
	}

	// Wait for expiry
	c.Set(cacheKey, true, 10*time.Millisecond)
	time.Sleep(20 * time.Millisecond)

	// Should be expired
	_, exists = c.Get(cacheKey)
	if exists {
		t.Error("Token should have expired")
	}
}

func TestCSRFConstants(t *testing.T) {
	if CSRFTokenHeader == "" {
		t.Error("CSRFTokenHeader should not be empty")
	}

	if CSRFTokenCookie == "" {
		t.Error("CSRFTokenCookie should not be empty")
	}

	if CSRFCachePrefix == "" {
		t.Error("CSRFCachePrefix should not be empty")
	}

	if CSRFTokenLength <= 0 {
		t.Error("CSRFTokenLength should be positive")
	}

	if CSRFTokenExpiry <= 0 {
		t.Error("CSRFTokenExpiry should be positive")
	}
}
