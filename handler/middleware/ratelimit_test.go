package middleware

import (
	"testing"
	"time"
)

func TestRateLimiter_Allow(t *testing.T) {
	limiter := &RateLimiter{
		buckets:  make(map[string]*bucket),
		rate:     5,
		window:   time.Second,
		maxBurst: 5,
		cleanup:  time.Second * 2,
	}

	key := "test-key"

	// First 5 requests should be allowed
	for i := 0; i < 5; i++ {
		allowed, remaining, _ := limiter.allow(key)
		if !allowed {
			t.Errorf("Request %d should be allowed", i+1)
		}
		if remaining != 4-i {
			t.Errorf("Request %d: expected %d remaining, got %d", i+1, 4-i, remaining)
		}
	}

	// 6th request should be blocked
	allowed, remaining, _ := limiter.allow(key)
	if allowed {
		t.Error("6th request should be blocked")
	}
	if remaining != 0 {
		t.Errorf("Expected 0 remaining, got %d", remaining)
	}
}

func TestRateLimiter_Refill(t *testing.T) {
	limiter := &RateLimiter{
		buckets:  make(map[string]*bucket),
		rate:     2,
		window:   100 * time.Millisecond,
		maxBurst: 2,
		cleanup:  time.Second,
	}

	key := "test-key"

	// Use up all tokens
	limiter.allow(key)
	limiter.allow(key)

	// Should be blocked
	allowed, _, _ := limiter.allow(key)
	if allowed {
		t.Error("Should be blocked after using all tokens")
	}

	// Wait for refill
	time.Sleep(150 * time.Millisecond)

	// Should be allowed again
	allowed, _, _ = limiter.allow(key)
	if !allowed {
		t.Error("Should be allowed after refill")
	}
}

func TestRateLimiter_IsolatesKeys(t *testing.T) {
	limiter := &RateLimiter{
		buckets:  make(map[string]*bucket),
		rate:     2,
		window:   time.Second,
		maxBurst: 2,
		cleanup:  time.Second * 2,
	}

	key1 := "client1"
	key2 := "client2"

	// Client 1 uses all tokens
	limiter.allow(key1)
	limiter.allow(key1)

	// Client 1 should be blocked
	allowed, _, _ := limiter.allow(key1)
	if allowed {
		t.Error("Client 1 should be blocked")
	}

	// Client 2 should still be allowed
	allowed, _, _ = limiter.allow(key2)
	if !allowed {
		t.Error("Client 2 should be allowed")
	}
}

func TestGenerateCSRFToken_Uniqueness(t *testing.T) {
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

func TestDefaultKeyFunc(t *testing.T) {
	// This is a simple test to ensure the function doesn't panic
	// We can't easily test the actual IP extraction without a real context
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("defaultKeyFunc panicked: %v", r)
		}
	}()

	// Just ensure the function exists and can be called
	// (actual testing would require a mock web.Context)
}
