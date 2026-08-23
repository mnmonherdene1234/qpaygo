package qpaygo

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	// DefaultHost is QPay's production merchant API host.
	DefaultHost = "https://merchant.qpay.mn"
	// SandboxHost is QPay's sandbox merchant API host.
	SandboxHost = "https://merchant-sandbox.qpay.mn"

	DefaultTimeout = 15 * time.Second
)

// QPayClient is a client for QPay's v2 merchant API.
type QPayClient struct {
	username    string
	password    string
	invoiceCode string
	host        string
	httpClient  *http.Client

	mu    sync.Mutex
	token *TokenResponse

	// refreshMu serializes concurrent token refresh/re-auth attempts so
	// that multiple goroutines racing CheckTokenAndRefresh coalesce into a
	// single network call instead of each firing their own
	// AuthToken/RefreshToken request (wasteful, and unsafe if QPay rotates
	// refresh tokens per use).
	refreshMu sync.Mutex
}

// Option configures a QPayClient constructed via NewQPayClient or
// NewQPayClientLazy.
type Option func(*QPayClient)

// WithHTTPClient overrides the *http.Client used for all requests.
func WithHTTPClient(c *http.Client) Option {
	return func(q *QPayClient) { q.httpClient = c }
}

// WithHost overrides the API host (e.g. SandboxHost).
func WithHost(host string) Option {
	return func(q *QPayClient) { q.host = host }
}

// WithTimeout sets the timeout of the client's *http.Client. If a client was
// supplied via WithHTTPClient, WithTimeout copies it rather than mutating the
// caller's *http.Client in place, so callers who share that client elsewhere
// aren't surprised by a side effect.
func WithTimeout(d time.Duration) Option {
	return func(q *QPayClient) {
		if q.httpClient == nil {
			q.httpClient = &http.Client{}
		} else {
			cp := *q.httpClient
			q.httpClient = &cp
		}
		q.httpClient.Timeout = d
	}
}

// NewQPayClientLazy creates a client without fetching a token. Call
// AuthToken explicitly, or let CheckTokenAndRefresh fetch one lazily on
// first use via Request/the typed endpoint methods.
func NewQPayClientLazy(username, password, invoiceCode string, opts ...Option) *QPayClient {
	q := &QPayClient{
		username:    username,
		password:    password,
		invoiceCode: invoiceCode,
		host:        DefaultHost,
		httpClient:  &http.Client{Timeout: DefaultTimeout},
	}
	for _, opt := range opts {
		opt(q)
	}
	return q
}

// NewQPayClient creates a client and immediately fetches an access token.
func NewQPayClient(ctx context.Context, username, password, invoiceCode string, opts ...Option) (*QPayClient, error) {
	q := NewQPayClientLazy(username, password, invoiceCode, opts...)
	if err := q.AuthToken(ctx); err != nil {
		return nil, err
	}
	return q, nil
}

// InvoiceCode returns the merchant invoice code the client was created with.
func (q *QPayClient) InvoiceCode() string { return q.invoiceCode }

// Host returns the API host the client is configured to use.
func (q *QPayClient) Host() string { return q.host }

// Token returns a copy of the current token response, or nil if no token has
// been fetched yet. It is a copy so callers cannot mutate client-internal
// state.
func (q *QPayClient) Token() *TokenResponse {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.token == nil {
		return nil
	}
	cp := *q.token
	return &cp
}

func (q *QPayClient) url(path string) string { return strings.TrimRight(q.host, "/") + path }

func (q *QPayClient) client() *http.Client {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.httpClient == nil {
		q.httpClient = &http.Client{Timeout: DefaultTimeout}
	}
	return q.httpClient
}

// Request is the generic authenticated-request escape hatch for endpoints
// not (yet) covered by a typed method. It ensures a valid access token is
// present (refreshing/re-authenticating as needed) before sending the
// request.
//
// The caller owns the returned *http.Response and MUST close its body
// (response.Body.Close()) when done — unlike the typed methods, Request does
// not decode or close the response for you.
func (q *QPayClient) Request(ctx context.Context, method, path string, body any) (*http.Response, error) {
	if err := q.CheckTokenAndRefresh(ctx); err != nil {
		return nil, err
	}
	return q.doRequest(ctx, method, path, body)
}

func (q *QPayClient) doRequest(ctx context.Context, method, path string, body any) (*http.Response, error) {
	req, err := q.newJSONRequest(ctx, method, q.url(path), body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+q.Token().AccessToken)
	return q.client().Do(req)
}

// requestJSON is the shared implementation behind the typed endpoint methods
// that decode a JSON response body (CreateInvoice, GetPayment, etc.): send an
// authenticated request and decode its JSON body into a new T.
func requestJSON[T any](ctx context.Context, q *QPayClient, method, path string, body any) (*T, error) {
	response, err := q.Request(ctx, method, path, body)
	if err != nil {
		return nil, err
	}
	var out T
	if err := decodeJSONResponse(response, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// requestEmpty is the shared implementation behind the typed endpoint methods
// that expect an empty ("{}") success body (CancelInvoice, RefundPayment,
// etc.): send an authenticated request and discard its body on success.
func requestEmpty(ctx context.Context, q *QPayClient, method, path string, body any) error {
	response, err := q.Request(ctx, method, path, body)
	if err != nil {
		return err
	}
	return decodeEmptyResponse(response)
}

func (q *QPayClient) newJSONRequest(ctx context.Context, method, url string, body any) (*http.Request, error) {
	var jsonBody []byte
	var err error
	var reader io.Reader

	if body != nil {
		jsonBody, err = json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return req, nil
}
