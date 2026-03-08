package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRequestID_GenerateWhenHeaderMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx.Request = req

	m := NewRequestIDMiddleware()
	m.RequestID()(ctx)

	got := w.Header().Get(RequestIDHeader)
	if got == "" {
		t.Fatal("expected generated request id in response header")
	}
	if GetRequestID(ctx) != got {
		t.Fatalf("expected context request id %q, got %q", got, GetRequestID(ctx))
	}
}

func TestRequestID_UseIncomingHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(RequestIDHeader, "req-123")
	ctx.Request = req

	m := NewRequestIDMiddleware()
	m.RequestID()(ctx)

	if got := w.Header().Get(RequestIDHeader); got != "req-123" {
		t.Fatalf("expected response request id req-123, got %q", got)
	}
	if got := GetRequestID(ctx); got != "req-123" {
		t.Fatalf("expected context request id req-123, got %q", got)
	}
}
