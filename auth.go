package qpaygo

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"time"
)

// AuthToken fetches a fresh access/refresh token pair via
// POST /v2/auth/token using Basic auth (username/password).
func (q *QPayClient) AuthToken(ctx context.Context) error {
	authURL := fmt.Sprintf("%s/v2/auth/token", q.host)
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
func (q *QPayClient) RefreshToken(ctx context.Context) error {
	q.mu.Lock()
	tok := q.token
	q.mu.Unlock()

	if tok == nil || tok.RefreshToken == "" {
		return q.AuthToken(ctx)
	}

	refreshURL := fmt.Sprintf("%s/v2/auth/refresh", q.host)

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

	tokenResponse.ExpiresIn = secondsToMillis(tokenResponse.ExpiresIn)
	tokenResponse.RefreshExpiresIn = secondsToMillis(tokenResponse.RefreshExpiresIn)

	q.mu.Lock()
	q.token = &tokenResponse
	q.mu.Unlock()

	return nil
}

// tokenPartExpired reports whether a token value/expiry pair (either the
// access token's or the refresh token's) should be treated as expired: an
// empty value or a zero expiry both count as "never had a usable token yet".
func tokenPartExpired(value string, expiresAtMillis int64) bool {
	if value == "" || expiresAtMillis == 0 {
		return true
	}
	return time.Now().After(time.UnixMilli(expiresAtMillis))
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
// (and, if QPay rotates refresh tokens per use, potentially conflicting)
// refresh/login request.
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
		return q.AuthToken(ctx)
	}

	if err := q.RefreshToken(ctx); err != nil {
		return q.AuthToken(ctx)
	}

	return nil
}

func secondsToMillis(seconds int64) int64 {
	return seconds * 1000
}
