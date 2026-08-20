package qpaygo

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// newMockClient starts an httptest.Server whose mux always serves a fixed
// valid token at POST /v2/auth/token, plus whatever additional routes the
// caller registers, then returns a *QPayClient already pointed at it via
// WithHost and already authenticated.
func newMockClient(t *testing.T, mux *http.ServeMux) (*QPayClient, *httptest.Server) {
	t.Helper()

	mux.HandleFunc("/v2/auth/token", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(TokenResponse{
			TokenType:        "bearer",
			AccessToken:      "test-access",
			RefreshToken:     "test-refresh",
			ExpiresIn:        3600,
			RefreshExpiresIn: 7200,
		})
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	client, err := NewQPayClient(context.Background(), "u", "p", "INV", WithHost(srv.URL))
	if err != nil {
		t.Fatalf("NewQPayClient: %v", err)
	}
	return client, srv
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// mustParseTime parses s in QPay's "YYYY-MM-DD HH:mm:ss" layout, failing the
// test on error.
func mustParseTime(t *testing.T, s string) time.Time {
	t.Helper()
	tm, err := time.Parse("2006-01-02 15:04:05", s)
	if err != nil {
		t.Fatalf("mustParseTime(%q): %v", s, err)
	}
	return tm
}

// newTestServer starts an httptest.Server for mux and returns its base URL,
// registering cleanup. Use this (instead of newMockClient) when a test needs
// full control over the /v2/auth/token handler itself.
func newTestServer(t *testing.T, mux *http.ServeMux) string {
	t.Helper()
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv.URL
}
