package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type httpSnapshot struct {
	status      int
	body        string
	contentType string
	requestID   string
	locale      string
	allowOrigin string
}

type frameworkSnapshot struct {
	http         map[string]httpSnapshot
	login        httpSnapshot
	loginRaw     map[string]any
	profile      httpSnapshot
	profileRaw   map[string]any
	accessToken  string
	refreshToken string
	expiresIn    any
}

func TestFrameworkParity(t *testing.T) {
	if os.Getenv("SONIC_ENABLE_FRAMEWORK_PARITY") != "1" {
		t.Skip("set SONIC_ENABLE_FRAMEWORK_PARITY=1 to run framework parity integration test")
	}
	bin := buildTestBinary(t)
	baseline := runFrameworkSnapshot(t, bin, "gin")
	candidate := runFrameworkSnapshot(t, bin, "hertz")

	for path, want := range baseline.http {
		got := candidate.http[path]
		if got.status != want.status {
			t.Fatalf("status mismatch for %s: gin=%d hertz=%d", path, want.status, got.status)
		}
		if got.contentType != want.contentType {
			t.Fatalf("content-type mismatch for %s: gin=%q hertz=%q", path, want.contentType, got.contentType)
		}
		if got.body != want.body {
			t.Fatalf("body mismatch for %s", path)
		}
		if want.requestID != "" && got.requestID == "" {
			t.Fatalf("missing request id for %s in hertz response", path)
		}
		if got.locale != want.locale {
			t.Fatalf("locale mismatch for %s: gin=%q hertz=%q", path, want.locale, got.locale)
		}
		if got.allowOrigin != want.allowOrigin {
			t.Fatalf("cors origin mismatch for %s: gin=%q hertz=%q", path, want.allowOrigin, got.allowOrigin)
		}
	}

	assertDynamicLoginParity(t, baseline, candidate)
	assertDynamicProfileParity(t, baseline, candidate)

	if baseline.profileRaw["username"] != candidate.profileRaw["username"] {
		t.Fatalf("profile username mismatch: gin=%v hertz=%v", baseline.profileRaw["username"], candidate.profileRaw["username"])
	}
	if baseline.profileRaw["email"] != candidate.profileRaw["email"] {
		t.Fatalf("profile email mismatch: gin=%v hertz=%v", baseline.profileRaw["email"], candidate.profileRaw["email"])
	}
}

func assertDynamicLoginParity(t *testing.T, want, got frameworkSnapshot) {
	t.Helper()
	if got.login.status != want.login.status {
		t.Fatalf("status mismatch for POST /api/admin/login: gin=%d hertz=%d", want.login.status, got.login.status)
	}
	if got.login.contentType != want.login.contentType {
		t.Fatalf("content-type mismatch for POST /api/admin/login: gin=%q hertz=%q", want.login.contentType, got.login.contentType)
	}
	if got.login.locale != want.login.locale {
		t.Fatalf("locale mismatch for POST /api/admin/login: gin=%q hertz=%q", want.login.locale, got.login.locale)
	}
	if got.login.allowOrigin != want.login.allowOrigin {
		t.Fatalf("cors origin mismatch for POST /api/admin/login: gin=%q hertz=%q", want.login.allowOrigin, got.login.allowOrigin)
	}
	if got.loginRaw["status"] != want.loginRaw["status"] {
		t.Fatalf("login payload status mismatch: gin=%v hertz=%v", want.loginRaw["status"], got.loginRaw["status"])
	}
	if got.loginRaw["message"] != want.loginRaw["message"] {
		t.Fatalf("login payload message mismatch: gin=%v hertz=%v", want.loginRaw["message"], got.loginRaw["message"])
	}
	if got.expiresIn != want.expiresIn {
		t.Fatalf("login expires_in mismatch: gin=%v hertz=%v", want.expiresIn, got.expiresIn)
	}
	if want.accessToken == "" || got.accessToken == "" {
		t.Fatalf("missing access token in login response")
	}
	if want.refreshToken == "" || got.refreshToken == "" {
		t.Fatalf("missing refresh token in login response")
	}
}

func assertDynamicProfileParity(t *testing.T, want, got frameworkSnapshot) {
	t.Helper()
	if got.profile.status != want.profile.status {
		t.Fatalf("status mismatch for GET /api/admin/users/profiles: gin=%d hertz=%d", want.profile.status, got.profile.status)
	}
	if got.profile.contentType != want.profile.contentType {
		t.Fatalf("content-type mismatch for GET /api/admin/users/profiles: gin=%q hertz=%q", want.profile.contentType, got.profile.contentType)
	}
	if got.profile.locale != want.profile.locale {
		t.Fatalf("locale mismatch for GET /api/admin/users/profiles: gin=%q hertz=%q", want.profile.locale, got.profile.locale)
	}
	if got.profile.allowOrigin != want.profile.allowOrigin {
		t.Fatalf("cors origin mismatch for GET /api/admin/users/profiles: gin=%q hertz=%q", want.profile.allowOrigin, got.profile.allowOrigin)
	}
	if want.profile.requestID != "" && got.profile.requestID == "" {
		t.Fatalf("missing request id for GET /api/admin/users/profiles in hertz response")
	}
}

func assertHTTPSnapshotParity(t *testing.T, name string, want, got httpSnapshot) {
	t.Helper()
	if got.status != want.status {
		t.Fatalf("status mismatch for %s: gin=%d hertz=%d", name, want.status, got.status)
	}
	if got.contentType != want.contentType {
		t.Fatalf("content-type mismatch for %s: gin=%q hertz=%q", name, want.contentType, got.contentType)
	}
	if got.body != want.body {
		t.Fatalf("body mismatch for %s", name)
	}
	if want.requestID != "" && got.requestID == "" {
		t.Fatalf("missing request id for %s in hertz response", name)
	}
	if got.locale != want.locale {
		t.Fatalf("locale mismatch for %s: gin=%q hertz=%q", name, want.locale, got.locale)
	}
	if got.allowOrigin != want.allowOrigin {
		t.Fatalf("cors origin mismatch for %s: gin=%q hertz=%q", name, want.allowOrigin, got.allowOrigin)
	}
}

func buildTestBinary(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "sonic-test-bin")
	cmd := exec.Command("go", "build", "-o", bin, ".")
	cmd.Env = append(os.Environ(),
		"GOTOOLCHAIN=local",
		"GOPROXY=https://proxy.golang.org,direct",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build binary: %v\n%s", err, out)
	}
	return bin
}

func runFrameworkSnapshot(t *testing.T, bin, framework string) frameworkSnapshot {
	t.Helper()
	port := freePort(t)
	cfg := writeTempConfig(t, framework, port)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, "-config", cfg)
	cmd.Dir = repoRoot(t)
	var log bytes.Buffer
	cmd.Stdout = &log
	cmd.Stderr = &log
	if err := cmd.Start(); err != nil {
		t.Fatalf("start %s: %v", framework, err)
	}
	defer func() {
		cancel()
		_ = cmd.Wait()
	}()
	waitForReady(t, port, &log)

	client := &http.Client{Timeout: 5 * time.Second}
	base := fmt.Sprintf("http://127.0.0.1:%d", port)
	paths := map[string]*http.Request{
		"GET /ping":                        mustRequest(t, http.MethodGet, base+"/ping", nil),
		"GET /":                            mustRequest(t, http.MethodGet, base+"/", nil),
		"GET /api/admin/is_installed":      mustRequest(t, http.MethodGet, base+"/api/admin/is_installed", nil),
		"GET /api/content/options/comment": mustRequest(t, http.MethodGet, base+"/api/content/options/comment", nil),
		"GET /admin_random/":               mustRequest(t, http.MethodGet, base+"/admin_random/", nil),
		"GET /css/app.a231e5ba.css":        mustRequest(t, http.MethodGet, base+"/css/app.a231e5ba.css", nil),
		"OPTIONS /api/admin/login":         mustCORSRequest(t, base+"/api/admin/login"),
	}
	out := make(map[string]httpSnapshot, len(paths))
	for name, req := range paths {
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("%s %s request failed: %v\n%s", framework, name, err, log.String())
		}
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		out[name] = httpSnapshot{
			status:      resp.StatusCode,
			body:        string(body),
			contentType: resp.Header.Get("Content-Type"),
			requestID:   resp.Header.Get("X-Request-ID"),
			locale:      resp.Header.Get("Content-Language"),
			allowOrigin: resp.Header.Get("Access-Control-Allow-Origin"),
		}
	}

	username := envOrDefault("SONIC_FRAMEWORK_PARITY_USERNAME", "litang")
	password := envOrDefault("SONIC_FRAMEWORK_PARITY_PASSWORD", "Ll3313222")
	loginBody := fmt.Sprintf(`{"username":%q,"password":%q}`, username, password)
	loginReq := mustRequest(t, http.MethodPost, base+"/api/admin/login", strings.NewReader(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginResp, err := client.Do(loginReq)
	if err != nil {
		t.Fatalf("%s POST /api/admin/login request failed: %v\n%s", framework, err, log.String())
	}
	loginRespBody, _ := io.ReadAll(loginResp.Body)
	_ = loginResp.Body.Close()
	login := httpSnapshot{
		status:      loginResp.StatusCode,
		body:        string(loginRespBody),
		contentType: loginResp.Header.Get("Content-Type"),
		requestID:   loginResp.Header.Get("X-Request-ID"),
		locale:      loginResp.Header.Get("Content-Language"),
		allowOrigin: loginResp.Header.Get("Access-Control-Allow-Origin"),
	}
	loginRaw, accessToken, refreshToken, expiresIn := extractLoginData(t, framework, loginRespBody, &log)
	profileReq := mustRequest(t, http.MethodGet, base+"/api/admin/users/profiles", nil)
	profileReq.Header.Set("Admin-Authorization", accessToken)
	profileResp, err := client.Do(profileReq)
	if err != nil {
		t.Fatalf("%s GET /api/admin/users/profiles request failed: %v\n%s", framework, err, log.String())
	}
	profileRespBody, _ := io.ReadAll(profileResp.Body)
	_ = profileResp.Body.Close()
	profile := httpSnapshot{
		status:      profileResp.StatusCode,
		body:        string(profileRespBody),
		contentType: profileResp.Header.Get("Content-Type"),
		requestID:   profileResp.Header.Get("X-Request-ID"),
		locale:      profileResp.Header.Get("Content-Language"),
		allowOrigin: profileResp.Header.Get("Access-Control-Allow-Origin"),
	}
	return frameworkSnapshot{
		http:         out,
		login:        login,
		loginRaw:     loginRaw,
		profile:      profile,
		profileRaw:   extractProfileData(t, framework, profileRespBody, &log),
		accessToken:  accessToken,
		refreshToken: refreshToken,
		expiresIn:    expiresIn,
	}
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func extractLoginData(t *testing.T, framework string, body []byte, log *bytes.Buffer) (map[string]any, string, string, any) {
	t.Helper()
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("%s login response is not valid JSON: %v\nbody=%s\n%s", framework, err, body, log.String())
	}
	data, _ := payload["data"].(map[string]any)
	accessToken, _ := data["access_token"].(string)
	refreshToken, _ := data["refresh_token"].(string)
	expiresIn := data["expired_in"]
	status, _ := payload["status"].(float64)
	if int(status) != http.StatusOK || accessToken == "" || refreshToken == "" {
		t.Fatalf("%s login did not return valid auth tokens\nbody=%s\n%s", framework, body, log.String())
	}
	return payload, accessToken, refreshToken, expiresIn
}

func extractProfileData(t *testing.T, framework string, body []byte, log *bytes.Buffer) map[string]any {
	t.Helper()
	var payload struct {
		Status int            `json:"status"`
		Data   map[string]any `json:"data"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("%s profile response is not valid JSON: %v\nbody=%s\n%s", framework, err, body, log.String())
	}
	if payload.Status != http.StatusOK || payload.Data == nil {
		t.Fatalf("%s profile response did not return user data\nbody=%s\n%s", framework, body, log.String())
	}
	return payload.Data
}

func mustRequest(t *testing.T, method, url string, body io.Reader) *http.Request {
	t.Helper()
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		t.Fatal(err)
	}
	return req
}

func mustCORSRequest(t *testing.T, url string) *http.Request {
	t.Helper()
	req := mustRequest(t, http.MethodOptions, url, nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", http.MethodPost)
	return req
}

func waitForReady(t *testing.T, port int, log *bytes.Buffer) {
	t.Helper()
	client := &http.Client{Timeout: 1 * time.Second}
	url := fmt.Sprintf("http://127.0.0.1:%d/ping", port)
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err == nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatalf("server did not become ready on port %d\n%s", port, log.String())
}

func freePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		if strings.Contains(err.Error(), "operation not permitted") {
			t.Skip("sandbox does not allow binding a local port for integration tests")
		}
		t.Fatal(err)
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port
}

func writeTempConfig(t *testing.T, framework string, port int) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(repoRoot(t), "conf", "config.dev.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	content = strings.Replace(content, "framework: gin", "framework: "+framework, 1)
	content = strings.Replace(content, "port: 8080", fmt.Sprintf("port: %d", port), 1)
	cfg := filepath.Join(t.TempDir(), framework+".yaml")
	if err := os.WriteFile(cfg, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return cfg
}

func repoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return wd
}
