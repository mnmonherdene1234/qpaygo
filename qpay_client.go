package qpaygo

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	DefaultHost    = "https://merchant.qpay.mn"
	DefaultTimeout = 15 * time.Second
)

type QPayClient struct {
	Username      string         // Нэвтрэх нэр
	Password      string         // Нууц үг
	InvoiceCode   string         // Нэхэмчлэх код
	Client        *http.Client   // HTTP Client
	TokenResponse *TokenResponse // Token хариу
	Host          string         // Хост хаяг
}

// NewQPayClient нь шинэ QPayClient үүсгэх функц
func NewQPayClient(username, password, invoiceCode string) (*QPayClient, error) {
	client := newClient(username, password, invoiceCode)

	if err := client.AuthToken(); err != nil {
		return nil, err
	}

	return client, nil
}

// NewQPayClientLazy creates a client without fetching a token.
func NewQPayClientLazy(username, password, invoiceCode string) (*QPayClient, error) {
	client := newClient(username, password, invoiceCode)
	return client, nil
}

func newClient(username, password, invoiceCode string) *QPayClient {
	return &QPayClient{
		Username:    username,
		Password:    password,
		InvoiceCode: invoiceCode,
		Client:      &http.Client{Timeout: DefaultTimeout},
		Host:        DefaultHost,
	}
}

func (q *QPayClient) httpClient() *http.Client {
	if q.Client == nil {
		q.Client = &http.Client{Timeout: DefaultTimeout}
	}

	return q.Client
}

// AuthToken нь token авах функц
func (q *QPayClient) AuthToken() error {
	return q.AuthTokenWithContext(context.Background())
}

// AuthTokenWithContext fetches a token using the provided context.
func (q *QPayClient) AuthTokenWithContext(ctx context.Context) error {
	authURL := fmt.Sprintf("%s/v2/auth/token", q.Host)
	authHeader := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", q.Username, q.Password)))

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, authURL, nil)
	if err != nil {
		return err
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Basic "+authHeader)
	request.Header.Set("Accept", "application/json")

	response, err := q.httpClient().Do(request)
	if err != nil {
		return err
	}

	var tokenResponse TokenResponse
	if err := decodeJSONResponse(response, &tokenResponse); err != nil {
		return err
	}

	tokenResponse.ExpiresIn = secondsToMillis(tokenResponse.ExpiresIn)
	tokenResponse.RefreshExpiresIn = secondsToMillis(tokenResponse.RefreshExpiresIn)
	q.TokenResponse = &tokenResponse

	return nil
}

// IsTokenExpired нь token-г шалгах функц ба хугацаа дууссан бол true буцаана
func (q *QPayClient) IsTokenExpired() bool {
	if q.TokenResponse == nil || q.TokenResponse.AccessToken == "" || q.TokenResponse.ExpiresIn == 0 {
		return true
	}

	expirationTime := time.UnixMilli(q.TokenResponse.ExpiresIn)
	return time.Now().After(expirationTime)
}

// CheckTokenAndRefresh нь token-г шалгах ба хугацаа дууссан бол дахин шинэ token авах функц
func (q *QPayClient) CheckTokenAndRefresh() error {
	return q.CheckTokenAndRefreshWithContext(context.Background())
}

// CheckTokenAndRefreshWithContext refreshes the token if needed using the provided context.
func (q *QPayClient) CheckTokenAndRefreshWithContext(ctx context.Context) error {
	if q.IsTokenExpired() {
		return q.AuthTokenWithContext(ctx)
	}

	return nil
}

// Request нь HTTP хүсэлт илгээх функц
func (q *QPayClient) Request(method, path string, body any) (*http.Response, error) {
	return q.RequestWithContext(context.Background(), method, path, body)
}

// RequestWithContext sends an authenticated request using the provided context.
func (q *QPayClient) RequestWithContext(ctx context.Context, method, path string, body any) (*http.Response, error) {
	if err := q.CheckTokenAndRefreshWithContext(ctx); err != nil {
		return nil, err
	}

	return q.doRequestWithContext(ctx, method, path, body)
}

func (q *QPayClient) doRequestWithContext(ctx context.Context, method, path string, body any) (*http.Response, error) {
	request, err := q.newJSONRequest(ctx, method, q.Host+path, body)
	if err != nil {
		return nil, err
	}

	request.Header.Set("Authorization", "Bearer "+q.TokenResponse.AccessToken)

	return q.httpClient().Do(request)
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

	request, err := http.NewRequestWithContext(ctx, method, url, reader)
	if err != nil {
		return nil, err
	}

	request.Header.Set("Accept", "application/json")
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}

	return request, nil
}

func decodeJSONResponse(response *http.Response, dst any) error {
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		bodyBytes, _ := io.ReadAll(response.Body)
		message := strings.TrimSpace(string(bodyBytes))
		if message == "" {
			message = response.Status
		}
		return fmt.Errorf("qpay: request failed: status %d: %s", response.StatusCode, message)
	}

	return json.NewDecoder(response.Body).Decode(dst)
}

func secondsToMillis(seconds int64) int64 {
	return seconds * 1000
}

// CreateAmountInvoice нь шинэ нэхэмжлэх үүсгэх функц
func (q *QPayClient) CreateAmountInvoice(
	senderInvoiceNo, invoiceReceiverCode, description string,
	amount uint, callbackURL string,
) (*CreateAmountInvoiceResponse, error) {
	response, err := q.Request(http.MethodPost, "/v2/invoice", CreateAmountInvoiceRequest{
		InvoiceCode:         q.InvoiceCode,
		SenderInvoiceNo:     senderInvoiceNo,
		InvoiceReceiverCode: invoiceReceiverCode,
		InvoiceDescription:  description,
		Amount:              amount,
		CallbackURL:         callbackURL,
	})

	if err != nil {
		return nil, err
	}

	var invoiceResponse CreateAmountInvoiceResponse

	if err := decodeJSONResponse(response, &invoiceResponse); err != nil {
		return nil, err
	}

	return &invoiceResponse, nil
}

// GetInvoice нь invoiceID-г ашиглан нэхэмчлэх авах функц
func (q *QPayClient) GetInvoice(invoiceID string) (*GetInvoiceResponse, error) {
	response, err := q.Request(http.MethodGet, "/v2/invoice/"+invoiceID, nil)

	if err != nil {
		return nil, err
	}

	var getInvoiceResponse GetInvoiceResponse

	if err := decodeJSONResponse(response, &getInvoiceResponse); err != nil {
		return nil, err
	}

	return &getInvoiceResponse, nil
}
