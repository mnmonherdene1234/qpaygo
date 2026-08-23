package qpaygo

import (
	"context"
	"encoding/base64"
	"fmt"
	"math"
	"net/http"
	"time"
)

// tokenExpirySkew is how early a token is treated as expired. QPay's clock
// and the caller's clock may drift; treating a token as expired slightly
// before its nominal expiry avoids firing a request with a token that QPay
// already considers dead.
const tokenExpirySkew = 30 * time.Second

// AuthToken fetches a fresh access/refresh token pair via
// POST /v2/auth/token using Basic auth (username/password).
//
// AuthToken takes refreshMu (see CheckTokenAndRefresh) so that concurrent
// callers cannot fire duplicate logins, which QPay explicitly warns against
// ("do not fetch tokens repeatedly"), or clobber a freshly stored token with
// an older response.
func (q *QPayClient) AuthToken(ctx context.Context) error {
	q.refreshMu.Lock()
	defer q.refreshMu.Unlock()
	return q.authToken(ctx)
}

// authToken is AuthToken without the refreshMu wrapper; callers that already
// hold refreshMu (CheckTokenAndRefresh) use this directly to avoid deadlock.
func (q *QPayClient) authToken(ctx context.Context) error {
	authURL := q.url("/v2/auth/token")
	authHeader := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", q.username, q.password)))

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, authURL, nil)
	if err != nil {
		return err
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Basic "+authHeader)
	request.Header.Set("Accept", "application/json")

	response, err := q.client().Do(request)
	if err != nil {
		return err
	}

	return q.storeToken(response)
}

// RefreshToken exchanges the current refresh token for a new access/refresh
// token pair via POST /v2/auth/refresh. Per QPay's API, this endpoint
// authenticates using the refresh token itself as the Bearer credential
// (not the access token, and not a body parameter). If there is no usable
// refresh token, it falls back to a full AuthToken login.
//
// Like AuthToken, RefreshToken takes refreshMu. QPay rotates the refresh
// token on every refresh, so two concurrent refreshes with the same old
// token would conflict — serializing here is what keeps refresh safe for
// callers that invoke RefreshToken directly rather than through
// CheckTokenAndRefresh.
func (q *QPayClient) RefreshToken(ctx context.Context) error {
	q.refreshMu.Lock()
	defer q.refreshMu.Unlock()
	return q.refreshToken(ctx)
}

// refreshToken is RefreshToken without the refreshMu wrapper; callers that
// already hold refreshMu (CheckTokenAndRefresh) use this directly to avoid
// deadlock.
func (q *QPayClient) refreshToken(ctx context.Context) error {
	q.mu.Lock()
	tok := q.token
	q.mu.Unlock()

	if tok == nil || tok.RefreshToken == "" {
		return q.authToken(ctx)
	}

	refreshURL := q.url("/v2/auth/refresh")

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, refreshURL, nil)
	if err != nil {
		return err
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+tok.RefreshToken)
	request.Header.Set("Accept", "application/json")

	response, err := q.client().Do(request)
	if err != nil {
		return err
	}

	return q.storeToken(response)
}

func (q *QPayClient) storeToken(response *http.Response) error {
	var tokenResponse TokenResponse
	if err := decodeJSONResponse(response, &tokenResponse); err != nil {
		return err
	}

	// QPay's token payloads carry expires_in / refresh_expires_in as
	// ABSOLUTE Unix timestamps in seconds (documented type "date", example
	// 1646967792; the live access-token value equals the JWT exp claim).
	// qpayWireExpiryMillis converts them into absolute Unix milliseconds
	// for tokenPartExpired, and additionally tolerates TTL-style relative
	// values so a QPay format change degrades gracefully instead of causing
	// a re-login on every request.
	tokenResponse.ExpiresIn = qpayWireExpiryMillis(tokenResponse.ExpiresIn)
	tokenResponse.RefreshExpiresIn = qpayWireExpiryMillis(tokenResponse.RefreshExpiresIn)

	q.mu.Lock()
	q.token = &tokenResponse
	q.mu.Unlock()

	return nil
}

// minQPayEpochSeconds is the smallest absolute epoch-seconds timestamp
// qpayWireExpiryMillis will accept as an absolute expiry. Any value at or
// above this bound is interpreted as "seconds since the Unix epoch";
// anything smaller is interpreted as a relative TTL in seconds.
//
// QPay's documented and live format is absolute (current values are
// ~1.8e9). The smallest absolute timestamp QPay could realistically send
// (their API postdates 2010, ~1.2e9) is far above 1e9, while no real token
// TTL comes anywhere near 1e9 seconds (~32 years), so the two formats
// cannot be confused.
const minQPayEpochSeconds = int64(1_000_000_000)

// qpayWireExpiryMillis converts a raw expires_in / refresh_expires_in value
// from QPay into an absolute expiry instant in Unix milliseconds:
//
//   - value >= minQPayEpochSeconds: absolute Unix seconds → milliseconds
//     (QPay's documented and live format);
//   - 0 < value < minQPayEpochSeconds: relative TTL seconds → now + value
//     (tolerated for robustness; QPay currently never sends this);
//   - value <= 0 (missing/invalid): 0, which tokenPartExpired reports as
//     expired, forcing a re-login instead of trusting a dead token.
func qpayWireExpiryMillis(seconds int64) int64 {
	switch {
	case seconds <= 0:
		return 0
	case seconds >= minQPayEpochSeconds:
		if seconds > math.MaxInt64/1000 {
			// Absurd value; never treat as expired rather than overflowing
			// into a past timestamp that would force re-login every request.
			return math.MaxInt64
		}
		return seconds * 1000
	default:
		return time.Now().Add(time.Duration(seconds) * time.Second).UnixMilli()
	}
}

// tokenPartExpired reports whether a token value/expiry pair (either the
// access token's or the refresh token's) should be treated as expired: an
// empty value or a zero expiry both count as "never had a usable token yet".
// Expiry is checked with a small clock-skew margin (tokenExpirySkew).
func tokenPartExpired(value string, expiresAtMillis int64) bool {
	if value == "" || expiresAtMillis == 0 {
		return true
	}
	return time.Now().After(time.UnixMilli(expiresAtMillis).Add(-tokenExpirySkew))
}

// IsTokenExpired нь token-г шалгах функц ба хугацаа дууссан бол true буцаана
func (q *QPayClient) IsTokenExpired() bool {
	q.mu.Lock()
	tok := q.token
	q.mu.Unlock()

	if tok == nil {
		return true
	}
	return tokenPartExpired(tok.AccessToken, tok.ExpiresIn)
}

// IsRefreshTokenExpired mirrors IsTokenExpired's exact pattern, applied to
// the refresh token's own expiry fields instead of the access token's.
func (q *QPayClient) IsRefreshTokenExpired() bool {
	q.mu.Lock()
	tok := q.token
	q.mu.Unlock()

	if tok == nil {
		return true
	}
	return tokenPartExpired(tok.RefreshToken, tok.RefreshExpiresIn)
}

// CheckTokenAndRefresh ensures a usable access token is present, refreshing
// or re-authenticating as needed. It never calls AuthToken when a cheaper
// refresh is possible, but falls back to a full login if refreshing fails
// (e.g. a revoked refresh token) or if the refresh token itself has expired.
//
// Concurrent callers are serialized on refreshMu: only one goroutine
// actually hits the network, and every other goroutine blocked on refreshMu
// re-checks IsTokenExpired once it acquires the lock and, finding the token
// already refreshed, returns immediately instead of firing its own redundant
// (and, because QPay rotates refresh tokens per use, potentially
// conflicting) refresh/login request.
//
// If both the refresh AND the fallback login fail, the returned error joins
// both failures so neither diagnostic detail is lost.
func (q *QPayClient) CheckTokenAndRefresh(ctx context.Context) error {
	if !q.IsTokenExpired() {
		return nil
	}

	q.refreshMu.Lock()
	defer q.refreshMu.Unlock()

	if !q.IsTokenExpired() {
		return nil
	}

	if q.IsRefreshTokenExpired() {
		return q.authToken(ctx)
	}

	if err := q.refreshToken(ctx); err != nil {
		if authErr := q.authToken(ctx); authErr != nil {
			return fmt.Errorf("qpaygo: token refresh failed: %v; re-login failed: %w", err, authErr)
		}
		return nil
	}

	return nil
}
