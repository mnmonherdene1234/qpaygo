package qpaygo

import (
	"context"
	"encoding/base64"
	"net/http"
	"testing"
	"time"
)

func TestAuthTokenSendsBasicAuthAndAppliesSecondsToMillis(t *testing.T) {
	mux := http.NewServeMux()
	var gotAuth string
	mux.HandleFunc("/v2/auth/token", func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		writeJSON(w, http.StatusOK, TokenResponse{
			TokenType:        "bearer",
			AccessToken:      "access-1",
			RefreshToken:     "refresh-1",
			ExpiresIn:        600,
			RefreshExpiresIn: 1200,
		})
	})
	srv := newTestServer(t, mux)

	client := NewQPayClientLazy("myuser", "mypass", "INV", WithHost(srv))
	if err := client.AuthToken(context.Background()); err != nil {
		t.Fatalf("AuthToken: %v", err)
	}

	wantAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("myuser:mypass"))
	if gotAuth != wantAuth {
		t.Fatalf("got Authorization %q, want %q", gotAuth, wantAuth)
	}

	tok := client.Token()
	if tok.ExpiresIn != 600*1000 {
		t.Fatalf("got ExpiresIn %d, want %d (secondsToMillis must be applied)", tok.ExpiresIn, 600*1000)
	}
	if tok.RefreshExpiresIn != 1200*1000 {
		t.Fatalf("got RefreshExpiresIn %d, want %d", tok.RefreshExpiresIn, 1200*1000)
	}
}

func TestRefreshTokenUsesRefreshTokenAsBearer(t *testing.T) {
	mux := http.NewServeMux()
	var gotAuth string
	mux.HandleFunc("/v2/auth/refresh", func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		writeJSON(w, http.StatusOK, TokenResponse{
			AccessToken:      "new-access",
			RefreshToken:     "new-refresh",
			ExpiresIn:        600,
			RefreshExpiresIn: 1200,
		})
	})
	client, _ := newMockClient(t, mux)

	if err := client.RefreshToken(context.Background()); err != nil {
		t.Fatalf("RefreshToken: %v", err)
	}

	if gotAuth != "Bearer test-refresh" {
		t.Fatalf("got Authorization %q, want Bearer test-refresh", gotAuth)
	}
	if client.Token().AccessToken != "new-access" {
		t.Fatalf("got access token %q", client.Token().AccessToken)
	}
}

func TestIsTokenExpiredPureCases(t *testing.T) {
	client := NewQPayClientLazy("u", "p", "INV")

	if !client.IsTokenExpired() {
		t.Fatal("nil token should be expired")
	}

	client.token = &TokenResponse{AccessToken: "a", ExpiresIn: 0}
	if !client.IsTokenExpired() {
		t.Fatal("zero ExpiresIn should be expired")
	}

	past := time.Now().Add(-time.Hour).UnixMilli()
	client.token = &TokenResponse{AccessToken: "a", ExpiresIn: past}
	if !client.IsTokenExpired() {
		t.Fatal("past expiry should be expired")
	}

	future := time.Now().Add(time.Hour).UnixMilli()
	client.token = &TokenResponse{AccessToken: "a", ExpiresIn: future}
	if client.IsTokenExpired() {
		t.Fatal("future expiry should not be expired")
	}
}

func TestIsRefreshTokenExpiredPureCases(t *testing.T) {
	client := NewQPayClientLazy("u", "p", "INV")

	if !client.IsRefreshTokenExpired() {
		t.Fatal("nil token should be expired")
	}

	future := time.Now().Add(time.Hour).UnixMilli()
	client.token = &TokenResponse{RefreshToken: "r", RefreshExpiresIn: future}
	if client.IsRefreshTokenExpired() {
		t.Fatal("future refresh expiry should not be expired")
	}

	past := time.Now().Add(-time.Hour).UnixMilli()
	client.token = &TokenResponse{RefreshToken: "r", RefreshExpiresIn: past}
	if !client.IsRefreshTokenExpired() {
		t.Fatal("past refresh expiry should be expired")
	}
}

func TestCheckTokenAndRefreshValidTokenNoHTTPCalls(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("unexpected HTTP call to %s", r.URL.Path)
	})
	client, _ := newMockClient(t, mux)

	future := time.Now().Add(time.Hour).UnixMilli()
	client.token.ExpiresIn = future

	if err := client.CheckTokenAndRefresh(context.Background()); err != nil {
		t.Fatalf("CheckTokenAndRefresh: %v", err)
	}
}

func TestCheckTokenAndRefreshExpiredAccessValidRefreshCallsRefresh(t *testing.T) {
	mux := http.NewServeMux()
	var tokenCalls, refreshCalls int
	mux.HandleFunc("/v2/auth/token", func(w http.ResponseWriter, r *http.Request) {
		tokenCalls++
		writeJSON(w, http.StatusOK, TokenResponse{
			AccessToken: "initial", ExpiresIn: 600, RefreshToken: "r1", RefreshExpiresIn: 1200,
		})
	})
	mux.HandleFunc("/v2/auth/refresh", func(w http.ResponseWriter, r *http.Request) {
		refreshCalls++
		writeJSON(w, http.StatusOK, TokenResponse{
			AccessToken: "refreshed", ExpiresIn: 600, RefreshToken: "r2", RefreshExpiresIn: 1200,
		})
	})
	srv := newTestServer(t, mux)

	client, err := NewQPayClient(context.Background(), "u", "p", "INV", WithHost(srv))
	if err != nil {
		t.Fatalf("NewQPayClient: %v", err)
	}
	if tokenCalls != 1 {
		t.Fatalf("got %d initial token calls, want 1", tokenCalls)
	}

	client.token.ExpiresIn = time.Now().Add(-time.Hour).UnixMilli()
	client.token.RefreshExpiresIn = time.Now().Add(time.Hour).UnixMilli()

	if err := client.CheckTokenAndRefresh(context.Background()); err != nil {
		t.Fatalf("CheckTokenAndRefresh: %v", err)
	}
	if refreshCalls != 1 {
		t.Fatalf("got %d refresh calls, want 1", refreshCalls)
	}
	if tokenCalls != 1 {
		t.Fatal("did not expect a second full re-auth call")
	}
	if client.Token().AccessToken != "refreshed" {
		t.Fatalf("got access token %q", client.Token().AccessToken)
	}
}

func TestCheckTokenAndRefreshBothExpiredCallsFullLogin(t *testing.T) {
	mux := http.NewServeMux()
	client, _ := newMockClient(t, mux)

	client.token.ExpiresIn = time.Now().Add(-time.Hour).UnixMilli()
	client.token.RefreshExpiresIn = time.Now().Add(-time.Hour).UnixMilli()

	if err := client.CheckTokenAndRefresh(context.Background()); err != nil {
		t.Fatalf("CheckTokenAndRefresh: %v", err)
	}
	// newMockClient's /v2/auth/token handler always returns "test-access".
	if client.Token().AccessToken != "test-access" {
		t.Fatalf("got access token %q", client.Token().AccessToken)
	}
}

func TestCheckTokenAndRefreshFallsBackWhenRefreshFails(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v2/auth/refresh", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "AUTHENTICATION_FAILED", "message": "expired"})
	})
	client, _ := newMockClient(t, mux)

	client.token.ExpiresIn = time.Now().Add(-time.Hour).UnixMilli()
	client.token.RefreshExpiresIn = time.Now().Add(time.Hour).UnixMilli()

	if err := client.CheckTokenAndRefresh(context.Background()); err != nil {
		t.Fatalf("CheckTokenAndRefresh: %v", err)
	}
	if client.Token().AccessToken != "test-access" {
		t.Fatalf("expected fallback to full login, got access token %q", client.Token().AccessToken)
	}
}
