package app

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

// newTestServer wires an App against a temp store and returns its handler.
func newTestServer(t *testing.T, storePath string) (*App, http.Handler) {
	t.Helper()
	a, err := New(storePath)
	if err != nil {
		t.Fatalf("new app: %v", err)
	}
	if err := a.Auth.SeedAdminIfAbsent(); err != nil {
		t.Fatalf("seed admin: %v", err)
	}
	return a, a.Routes()
}

// call performs one request and returns the status, the decoded envelope and
// the raw body.
func call(t *testing.T, h http.Handler, method, path, token string, body any) (int, map[string]any, string) {
	t.Helper()
	var reader *bytes.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		reader = bytes.NewReader(raw)
	} else {
		reader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	raw := rec.Body.String()
	env := map[string]any{}
	if raw != "" {
		if err := json.Unmarshal([]byte(raw), &env); err != nil {
			t.Fatalf("%s %s: body is not a single JSON envelope (%v): %s", method, path, err, raw)
		}
	}
	return rec.Code, env, raw
}

// tokenFor logs a user in and returns the session token.
func tokenFor(t *testing.T, h http.Handler, email, password string) string {
	t.Helper()
	status, env, raw := call(t, h, http.MethodPost, "/api/v1/auth/login", "", map[string]any{
		"email": email, "password": password,
	})
	if status != http.StatusOK {
		t.Fatalf("login %s: status %d body %s", email, status, raw)
	}
	data, _ := env["data"].(map[string]any)
	token, _ := data["token"].(string)
	if token == "" {
		t.Fatalf("login %s returned no token: %s", email, raw)
	}
	return token
}

// TestUnauthenticatedPlansWritesOneEnvelope guards a handler that used to
// ignore the requireUser failure and write a second response body.
func TestUnauthenticatedPlansWritesOneEnvelope(t *testing.T) {
	_, h := newTestServer(t, filepath.Join(t.TempDir(), "store.json"))

	// call() already asserts the body is a single JSON object.
	status, env, raw := call(t, h, http.MethodGet, "/api/v1/plans", "", nil)
	if status != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", status, raw)
	}
	if code, _ := env["code"].(float64); code != http.StatusUnauthorized {
		t.Fatalf("expected envelope code 401, got %v: %s", env["code"], raw)
	}
}

// TestAdminConsoleOverHTTP exercises the operator console end to end: plan
// changes, disable/enable, and org deletion, plus the 403 guard for normal
// users.
func TestAdminConsoleOverHTTP(t *testing.T) {
	_, h := newTestServer(t, filepath.Join(t.TempDir(), "store.json"))

	// A normal user registers and creates an org.
	status, _, raw := call(t, h, http.MethodPost, "/api/v1/auth/register", "", map[string]any{
		"email": "bob@test.dev", "password": "secret123", "nickname": "Bob",
	})
	if status != http.StatusOK {
		t.Fatalf("register: %d %s", status, raw)
	}
	bobToken := tokenFor(t, h, "bob@test.dev", "secret123")

	status, env, raw := call(t, h, http.MethodPost, "/api/v1/orgs", bobToken, map[string]any{"name": "Bob Co"})
	if status != http.StatusOK {
		t.Fatalf("create org: %d %s", status, raw)
	}
	orgData, _ := env["data"].(map[string]any)
	orgID, _ := orgData["id"].(string)
	if orgID == "" {
		t.Fatalf("no org id: %s", raw)
	}

	// Non-admins are refused by the console API.
	if status, _, raw := call(t, h, http.MethodGet, "/api/v1/admin/orgs", bobToken, nil); status != http.StatusForbidden {
		t.Fatalf("expected 403 for non-admin, got %d: %s", status, raw)
	}

	adminToken := tokenFor(t, h, "admin@ragflow.dev", "admin123")

	// The console lists Bob and his org.
	status, env, raw = call(t, h, http.MethodGet, "/api/v1/admin/users", adminToken, nil)
	if status != http.StatusOK {
		t.Fatalf("admin users: %d %s", status, raw)
	}
	users, _ := env["data"].([]any)
	var bobID string
	for _, rawUser := range users {
		u, _ := rawUser.(map[string]any)
		if u["email"] == "bob@test.dev" {
			bobID, _ = u["id"].(string)
		}
	}
	if bobID == "" {
		t.Fatalf("bob missing from admin user list: %s", raw)
	}

	// Plan change takes effect.
	if status, _, raw := call(t, h, http.MethodPatch, "/api/v1/admin/orgs/"+orgID+"/plan", adminToken,
		map[string]any{"plan_code": "pro"}); status != http.StatusOK {
		t.Fatalf("set plan: %d %s", status, raw)
	}
	status, env, raw = call(t, h, http.MethodGet, "/api/v1/orgs/"+orgID, bobToken, nil)
	if status != http.StatusOK {
		t.Fatalf("get org: %d %s", status, raw)
	}
	orgData, _ = env["data"].(map[string]any)
	if orgData["plan_code"] != "pro" {
		t.Fatalf("plan not applied: %s", raw)
	}

	// An unknown plan is rejected as a bad request.
	if status, _, raw := call(t, h, http.MethodPatch, "/api/v1/admin/orgs/"+orgID+"/plan", adminToken,
		map[string]any{"plan_code": "bogus"}); status != http.StatusBadRequest {
		t.Fatalf("expected 400 for unknown plan, got %d: %s", status, raw)
	}

	// Disabling blocks login; re-enabling restores it.
	if status, _, raw := call(t, h, http.MethodPatch, "/api/v1/admin/users/"+bobID+"/status", adminToken,
		map[string]any{"status": "0"}); status != http.StatusOK {
		t.Fatalf("disable user: %d %s", status, raw)
	}
	if status, _, _ := call(t, h, http.MethodPost, "/api/v1/auth/login", "", map[string]any{
		"email": "bob@test.dev", "password": "secret123",
	}); status != http.StatusUnauthorized {
		t.Fatalf("disabled user could still log in: %d", status)
	}
	if status, _, raw := call(t, h, http.MethodPatch, "/api/v1/admin/users/"+bobID+"/status", adminToken,
		map[string]any{"status": "1"}); status != http.StatusOK {
		t.Fatalf("enable user: %d %s", status, raw)
	}
	tokenFor(t, h, "bob@test.dev", "secret123")

	// Deleting the org empties the console list.
	if status, _, raw := call(t, h, http.MethodDelete, "/api/v1/admin/orgs/"+orgID, adminToken, nil); status != http.StatusOK {
		t.Fatalf("delete org: %d %s", status, raw)
	}
	status, env, raw = call(t, h, http.MethodGet, "/api/v1/admin/orgs", adminToken, nil)
	if status != http.StatusOK {
		t.Fatalf("admin orgs: %d %s", status, raw)
	}
	if orgs, _ := env["data"].([]any); len(orgs) != 0 {
		t.Fatalf("expected no orgs after delete: %s", raw)
	}
}

// TestDataSurvivesRestart proves the persistence layer keeps accounts and
// sessions' credentials across a server restart.
func TestDataSurvivesRestart(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "store.json")

	_, h := newTestServer(t, storePath)
	if status, _, raw := call(t, h, http.MethodPost, "/api/v1/auth/register", "", map[string]any{
		"email": "durable@test.dev", "password": "secret123", "nickname": "D",
	}); status != http.StatusOK {
		t.Fatalf("register: %d %s", status, raw)
	}
	firstToken := tokenFor(t, h, "durable@test.dev", "secret123")
	if status, _, raw := call(t, h, http.MethodPost, "/api/v1/orgs", firstToken, map[string]any{"name": "Durable Co"}); status != http.StatusOK {
		t.Fatalf("create org: %d %s", status, raw)
	}

	// Second App on the same file = restart.
	_, h2 := newTestServer(t, storePath)
	if status, _, raw := call(t, h2, http.MethodPost, "/api/v1/auth/login", "", map[string]any{
		"email": "durable@test.dev", "password": "secret123",
	}); status != http.StatusOK {
		t.Fatalf("login after restart: %d %s", status, raw)
	}
	token := tokenFor(t, h2, "durable@test.dev", "secret123")
	status, env, raw := call(t, h2, http.MethodGet, "/api/v1/orgs", token, nil)
	if status != http.StatusOK {
		t.Fatalf("list orgs after restart: %d %s", status, raw)
	}
	orgs, _ := env["data"].([]any)
	if len(orgs) != 1 {
		t.Fatalf("expected 1 org after restart, got %d: %s", len(orgs), raw)
	}
}
