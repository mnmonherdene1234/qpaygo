package qpaygo

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

const (
	DefaultHost = "https://merchant.qpay.mn"
)

type QPayClient struct {
	Username      string
	Password      string
	InvoiceCode   string
	Client        *http.Client
	TokenResponse *TokenResponse
	Host          string
}

func NewQPayClient(username, password, invoiceCode string) (*QPayClient, error) {
	client := &QPayClient{
		Username:    username,
		Password:    password,
		InvoiceCode: invoiceCode,
		Client:      &http.Client{},
		Host:        DefaultHost,
	}

	if err := client.AuthToken(); err != nil {
		return nil, err
	}

	return client, nil
}

func (q *QPayClient) AuthToken() error {
	authURL := fmt.Sprintf("%s/v2/auth/token", q.Host)
	authHeader := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", q.Username, q.Password)))

	request, err := http.NewRequest(http.MethodPost, authURL, nil)
	if err != nil {
		return err
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Basic "+authHeader)

	response, err := q.Client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return errors.New("failed to authenticate")
	}

	if err := json.NewDecoder(response.Body).Decode(&q.TokenResponse); err != nil {
		return err
	}

	return nil
}

func (q *QPayClient) IsTokenExpired() bool {
	if q.TokenResponse == nil || q.TokenResponse.AccessToken == "" || q.TokenResponse.ExpiresIn == 0 {
		return true
	}

	expirationTime := time.Unix(q.TokenResponse.ExpiresIn, 0)
	return time.Now().After(expirationTime)
}

func (q *QPayClient) CheckTokenAndRefresh() error {
	if q.IsTokenExpired() {
		if err := q.AuthToken(); err != nil {
			return err
		}
	}

	return nil
}

func (q *QPayClient) CreateAmountInvoice(
	senderInvoiceNo, invoiceReceiverCode, description string,
	amount float64, callbackURL string,
) error {
	if err := q.CheckTokenAndRefresh(); err != nil {
		return err
	}

	return nil
}
