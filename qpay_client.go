package qpaygo

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
)

// QPay үйлчилгээний үндсэн API хаяг
const (
	DefaultHost = "https://merchant.qpay.mn"
)

// QpayClient нь QPay төлбөрийн API-тай харилцахад ашиглагдана
type QpayClient struct {
	Username      string
	Password      string
	InvoiceCode   string
	Client        *http.Client
	TokenResponse *QPayTokenResponse
	Host          string
}

// NewQpayClient нь шинэ QpayClient үүсгэж буцаана
func NewQpayClient(username, password, invoiceCode string) *QpayClient {

	qpayClient := &QpayClient{
		Username:    username,
		Password:    password,
		InvoiceCode: invoiceCode,
		Client:      &http.Client{},
		Host:        DefaultHost,
	}

	qpayClient.AuthToken()

	return qpayClient
}

// AuthToken нь QPay API-аас шинэ хандалтын токен авна
func (q *QpayClient) AuthToken() error {
	authURL := q.Host + "/v2/auth/token"

	// Үндсэн аутентификацийн толгой үүсгэнэ
	authHeader := base64.StdEncoding.EncodeToString(fmt.Appendf(nil, "%s:%s", q.Username, q.Password))
	request, err := http.NewRequest("POST", authURL, nil)
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

	if err := json.NewDecoder(response.Body).Decode(q.TokenResponse); err != nil {
		return err
	}

	return nil
}
