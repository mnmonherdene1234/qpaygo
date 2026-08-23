package qpaygo

import (
	"context"
	"encoding/base64"
	"errors"
	"math"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestAuthTokenSendsBasicAuthAndConvertsExpiry(t *testing.T) {
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

	// 600/1200 are below minQPayEpochSeconds, so qpayWireExpiryMillis treats
	// them as relative TTL seconds: the stored expiry must be ≈ now+600s
	// (absolute milliseconds), not 600*1000.
	tok := client.Token()
	lower := time.Now().Add(590 * time.Second).UnixMilli()
	upper := time.Now().Add(610 * time.Second).UnixMilli()
	if tok.ExpiresIn < lower || tok.ExpiresIn > upper {
		t.Fatalf("got ExpiresIn %d, want in [%d, %d] (now+600s)", tok.ExpiresIn, lower, upper)
	}
	lower = time.Now().Add(1190 * time.Second).UnixMilli()
	upper = time.Now().Add(1210 * time.Second).UnixMilli()
	if tok.RefreshExpiresIn < lower || tok.RefreshExpiresIn > upper {
		t.Fatalf("got RefreshExpiresIn %d, want in [%d, %d] (now+1200s)", tok.RefreshExpiresIn, lower, upper)
	}
}

func TestStoreTokenAbsoluteEpochSeconds(t *testing.T) {
	// QPay's documented and live format: absolute Unix seconds.
	const absSeconds = int64(1_787_496_817) // e.g. 2026-08-23T22:53:37+08
	mux := http.NewServeMux()
	mux.HandleFunc("/v2/auth/token", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, TokenResponse{
			TokenType:        "bearer",
			AccessToken:      "access-1",
			RefreshToken:     "refresh-1",
			ExpiresIn:        absSeconds,
			RefreshExpiresIn: absSeconds,
		})
	})
	srv := newTestServer(t, mux)

	client := NewQPayClientLazy("u", "p", "INV", WithHost(srv))
	if err := client.AuthToken(context.Background()); err != nil {
		t.Fatalf("AuthToken: %v", err)
	}
	tok := client.Token()
	if tok.ExpiresIn != absSeconds*1000 {
		t.Fatalf("got ExpiresIn %d, want %d (absolute seconds -> millis)", tok.ExpiresIn, absSeconds*1000)
	}
	if tok.RefreshExpiresIn != absSeconds*1000 {
		t.Fatalf("got RefreshExpiresIn %d, want %d", tok.RefreshExpiresIn, absSeconds*1000)
	}
}

func TestStoreTokenMissingExpiryForcesReauth(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v2/auth/token", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, TokenResponse{
			TokenType: "bearer", AccessToken: "access-1", RefreshToken: "refresh-1",
			// ExpiresIn omitted entirely.
		})
	})
	srv := newTestServer(t, mux)

	client := NewQPayClientLazy("u", "p", "INV", WithHost(srv))
	if err := client.AuthToken(context.Background()); err != nil {
		t.Fatalf("AuthToken: %v", err)
	}
	if !client.IsTokenExpired() {
		t.Fatal("token with missing expiry must be reported as expired (forces re-login)")
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

func TestTokenExpirySkewMargin(t *testing.T) {
	client := NewQPayClientLazy("u", "p", "INV")

	// 20s left: within the 30s skew margin → treated as expired.
	soon := time.Now().Add(20 * time.Second).UnixMilli()
	client.token = &TokenResponse{AccessToken: "a", ExpiresIn: soon}
	if !client.IsTokenExpired() {
		t.Fatal("token expiring in 20s (within skew margin) should be treated as expired")
	}

	// 60s left: outside the skew margin → still valid.
	later := time.Now().Add(60 * time.Second).UnixMilli()
	client.token = &TokenResponse{AccessToken: "a", ExpiresIn: later}
	if client.IsTokenExpired() {
		t.Fatal("token expiring in 60s should not be expired")
	}
}

func TestQPayWireExpiryMillis(t *testing.T) {
	const abs = int64(1_787_496_817) // absolute epoch seconds (2026)
	now := time.Now().UnixMilli()

	cases := []struct {
		name      string
		in        int64
		wantMin   int64 // inclusive
		wantMax   int64 // inclusive
		wantExact *int64
	}{
		{name: "zero is expired", in: 0, wantExact: int64Ptr(0)},
		{name: "negative is expired", in: -5, wantExact: int64Ptr(0)},
		{name: "absolute epoch seconds", in: abs, wantExact: int64Ptr(abs * 1000)},
		{name: "ttl seconds", in: 600, wantMin: now + 599_000, wantMax: now + 601_000},
		{name: "max int64 guarded", in: math.MaxInt64, wantExact: int64Ptr(math.MaxInt64)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := qpayWireExpiryMillis(tc.in)
			if tc.wantExact != nil {
				if got != *tc.wantExact {
					t.Fatalf("got %d, want %d", got, *tc.wantExact)
				}
				return
			}
			if got < tc.wantMin || got > tc.wantMax {
				t.Fatalf("got %d, want in [%d, %d]", got, tc.wantMin, tc.wantMax)
			}
		})
	}
}

func int64Ptr(v int64) *int64 { return &v }

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

func TestCheckTokenAndRefreshKeepsBothErrorsWhenBothFail(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v2/auth/refresh", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "AUTHENTICATION_FAILED", "message": "refresh revoked"})
	})
	mux.HandleFunc("/v2/auth/token", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "INTERNAL_ERROR", "message": "login down"})
	})
	srv := newTestServer(t, mux)

	client := NewQPayClientLazy("u", "p", "INV", WithHost(srv))
	client.token = &TokenResponse{AccessToken: "a", RefreshToken: "r"}
	client.token.ExpiresIn = time.Now().Add(-time.Hour).UnixMilli()
	client.token.RefreshExpiresIn = time.Now().Add(time.Hour).UnixMilli()

	err := client.CheckTokenAndRefresh(context.Background())
	if err == nil {
		t.Fatal("expected error when both refresh and re-login fail")
	}
	for _, want := range []string{"refresh", "re-login", "refresh revoked", "login down"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error %q must contain %q", err, want)
		}
	}
	// The wrapped login APIError must remain reachable via errors.As.
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected joined error to unwrap to *APIError, got %T", err)
	}
}

func TestAuthTokenUsesQPayURLWithTrailingSlashHost(t *testing.T) {
	mux := http.NewServeMux()
	var gotPath string
	mux.HandleFunc("/v2/auth/token", func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		writeJSON(w, http.StatusOK, TokenResponse{
			TokenType: "bearer", AccessToken: "a", RefreshToken: "r",
			ExpiresIn: 1_700_000_000, RefreshExpiresIn: 1_700_000_000,
		})
	})
	srv := newTestServer(t, mux)

	// Trailing slash host must not produce a double-slash path.
	client := NewQPayClientLazy("u", "p", "INV", WithHost(srv+"/"))
	if err := client.AuthToken(context.Background()); err != nil {
		t.Fatalf("AuthToken: %v", err)
	}
	if gotPath != "/v2/auth/token" {
		t.Fatalf("got path %q, want /v2/auth/token (no double slash)", gotPath)
	}
}

func TestRefreshTokenUsesQPayURLWithTrailingSlashHost(t *testing.T) {
	mux := http.NewServeMux()
	var gotPath string
	mux.HandleFunc("/v2/auth/refresh", func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		writeJSON(w, http.StatusOK, TokenResponse{
			TokenType: "bearer", AccessToken: "new", RefreshToken: "r2",
			ExpiresIn: 1_700_000_000, RefreshExpiresIn: 1_700_000_000,
		})
	})
	srv := newTestServer(t, mux)

	client := NewQPayClientLazy("u", "p", "INV", WithHost(srv+"/"))
	client.token = &TokenResponse{
		AccessToken: "old", RefreshToken: "r1",
		ExpiresIn: 1_700_000_000, RefreshExpiresIn: 1_700_000_000,
	}
	if err := client.RefreshToken(context.Background()); err != nil {
		t.Fatalf("RefreshToken: %v", err)
	}
	if gotPath != "/v2/auth/refresh" {
		t.Fatalf("got path %q, want /v2/auth/refresh (no double slash)", gotPath)
	}
}

func TestAuthTokenSerializedByRefreshMu(t *testing.T) {
	// Two goroutines calling AuthToken directly must not both fire logins
	// into the server at once: the first holds refreshMu for the whole call.
	var inFlight, maxInFlight int32
	mux := http.NewServeMux()
	mux.HandleFunc("/v2/auth/token", func(w http.ResponseWriter, r *http.Request) {
		cur := atomic.AddInt32(&inFlight, 1)
		for {
			old := atomic.LoadInt32(&maxInFlight)
			if cur <= old || atomic.CompareAndSwapInt32(&maxInFlight, old, cur) {
				break
			}
		}
		time.Sleep(30 * time.Millisecond)
		atomic.AddInt32(&inFlight, -1)
		writeJSON(w, http.StatusOK, TokenResponse{
			TokenType: "bearer", AccessToken: "a", RefreshToken: "r",
			ExpiresIn: 1_700_000_000, RefreshExpiresIn: 1_700_000_000,
		})
	})
	srv := newTestServer(t, mux)

	client := NewQPayClientLazy("u", "p", "INV", WithHost(srv))
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = client.AuthToken(context.Background())
		}()
	}
	wg.Wait()
	if maxInFlight > 1 {
		t.Fatalf("max concurrent /v2/auth/token calls = %d, want 1 (refreshMu must serialize)", maxInFlight)
	}
}
