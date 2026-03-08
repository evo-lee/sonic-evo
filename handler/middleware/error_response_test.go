package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/go-sonic/sonic/util/xerr"
)

func TestErrorCodeFromStatus(t *testing.T) {
	cases := []struct {
		status int
		code   string
	}{
		{status: http.StatusBadRequest, code: "bad_request"},
		{status: http.StatusUnauthorized, code: "unauthorized"},
		{status: http.StatusForbidden, code: "forbidden"},
		{status: http.StatusNotFound, code: "not_found"},
		{status: http.StatusInternalServerError, code: "internal_error"},
	}

	for _, tc := range cases {
		if got := ErrorCodeFromStatus(tc.status); got != tc.code {
			t.Fatalf("status=%d expected code=%s, got=%s", tc.status, tc.code, got)
		}
	}
}

func TestErrorCodeFromError(t *testing.T) {
	cases := []struct {
		err  error
		code string
	}{
		{err: xerr.BadParam.New("bad"), code: "bad_request"},
		{err: xerr.NoRecord.New("nf"), code: "not_found"},
		{err: xerr.Forbidden.New("forbidden"), code: "forbidden"},
		{err: xerr.DB.New("db"), code: "db_error"},
		{err: xerr.Email.New("email"), code: "email_error"},
		{err: xerr.WithStatus(nil, http.StatusUnauthorized), code: "unauthorized"},
	}

	for _, tc := range cases {
		if got := ErrorCodeFromError(tc.err); got != tc.code {
			t.Fatalf("expected code=%s, got=%s", tc.code, got)
		}
	}
}

func TestAbortWithErrorJSONIncludesRequestIDAndCode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(RequestIDHeader, "req-test")
	ctx.Request = req

	NewRequestIDMiddleware().RequestID()(ctx)
	AbortWithErrorJSON(ctx, http.StatusBadRequest, "bad_request", "bad request")

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status=%d, got=%d", http.StatusBadRequest, w.Code)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response failed: %v", err)
	}

	if payload["code"] != "bad_request" {
		t.Fatalf("expected code=bad_request, got=%v", payload["code"])
	}
	if payload["request_id"] != "req-test" {
		t.Fatalf("expected request_id=req-test, got=%v", payload["request_id"])
	}
	if payload["message"] != "bad request" {
		t.Fatalf("expected message=bad request, got=%v", payload["message"])
	}
}
