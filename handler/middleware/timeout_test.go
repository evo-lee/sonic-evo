package middleware

import (
	"context"
	"testing"
	"time"
)

func TestWithDBTimeout(t *testing.T) {
	ctx := context.Background()

	// Test with default timeout
	timeoutCtx, cancel := WithDBTimeout(ctx, 0)
	defer cancel()

	deadline, ok := timeoutCtx.Deadline()
	if !ok {
		t.Error("Expected deadline to be set")
	}

	// Should be approximately 5 seconds from now
	expectedDeadline := time.Now().Add(5 * time.Second)
	diff := deadline.Sub(expectedDeadline)
	if diff < -100*time.Millisecond || diff > 100*time.Millisecond {
		t.Errorf("Deadline not as expected, diff: %v", diff)
	}
}

func TestWithDBTimeout_CustomTimeout(t *testing.T) {
	ctx := context.Background()

	// Test with custom timeout
	customTimeout := 2 * time.Second
	timeoutCtx, cancel := WithDBTimeout(ctx, customTimeout)
	defer cancel()

	deadline, ok := timeoutCtx.Deadline()
	if !ok {
		t.Error("Expected deadline to be set")
	}

	// Should be approximately 2 seconds from now
	expectedDeadline := time.Now().Add(customTimeout)
	diff := deadline.Sub(expectedDeadline)
	if diff < -100*time.Millisecond || diff > 100*time.Millisecond {
		t.Errorf("Deadline not as expected, diff: %v", diff)
	}
}

func TestWithAPITimeout(t *testing.T) {
	ctx := context.Background()

	// Test with default timeout
	timeoutCtx, cancel := WithAPITimeout(ctx, 0)
	defer cancel()

	deadline, ok := timeoutCtx.Deadline()
	if !ok {
		t.Error("Expected deadline to be set")
	}

	// Should be approximately 10 seconds from now
	expectedDeadline := time.Now().Add(10 * time.Second)
	diff := deadline.Sub(expectedDeadline)
	if diff < -100*time.Millisecond || diff > 100*time.Millisecond {
		t.Errorf("Deadline not as expected, diff: %v", diff)
	}
}

func TestWithAPITimeout_CustomTimeout(t *testing.T) {
	ctx := context.Background()

	// Test with custom timeout
	customTimeout := 3 * time.Second
	timeoutCtx, cancel := WithAPITimeout(ctx, customTimeout)
	defer cancel()

	deadline, ok := timeoutCtx.Deadline()
	if !ok {
		t.Error("Expected deadline to be set")
	}

	// Should be approximately 3 seconds from now
	expectedDeadline := time.Now().Add(customTimeout)
	diff := deadline.Sub(expectedDeadline)
	if diff < -100*time.Millisecond || diff > 100*time.Millisecond {
		t.Errorf("Deadline not as expected, diff: %v", diff)
	}
}

func TestTimeoutMiddleware_Creation(t *testing.T) {
	// Test with default timeout
	middleware := NewTimeoutMiddleware(TimeoutConfig{})
	if middleware == nil {
		t.Error("NewTimeoutMiddleware should not return nil")
	}
	if middleware.timeout != 30*time.Second {
		t.Errorf("Expected default timeout of 30s, got %v", middleware.timeout)
	}

	// Test with custom timeout
	customTimeout := 15 * time.Second
	middleware = NewTimeoutMiddleware(TimeoutConfig{Timeout: customTimeout})
	if middleware.timeout != customTimeout {
		t.Errorf("Expected timeout of %v, got %v", customTimeout, middleware.timeout)
	}
}

func TestContextCancellation(t *testing.T) {
	ctx := context.Background()
	timeoutCtx, cancel := WithDBTimeout(ctx, 100*time.Millisecond)

	// Wait for timeout
	time.Sleep(150 * time.Millisecond)

	// Context should be done
	select {
	case <-timeoutCtx.Done():
		if timeoutCtx.Err() != context.DeadlineExceeded {
			t.Errorf("Expected DeadlineExceeded error, got %v", timeoutCtx.Err())
		}
	default:
		t.Error("Context should be done after timeout")
	}

	cancel()
}

func TestContextCancellation_ManualCancel(t *testing.T) {
	ctx := context.Background()
	timeoutCtx, cancel := WithDBTimeout(ctx, 5*time.Second)

	// Cancel immediately
	cancel()

	// Context should be done
	select {
	case <-timeoutCtx.Done():
		if timeoutCtx.Err() != context.Canceled {
			t.Errorf("Expected Canceled error, got %v", timeoutCtx.Err())
		}
	default:
		t.Error("Context should be done after cancel")
	}
}
