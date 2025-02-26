package qpaygo

import (
	"bytes"
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
	Username      string         // Нэвтрэх нэр
	Password      string         // Нууц үг
	InvoiceCode   string         // Нэхэмчлэх код
	Client        *http.Client   // HTTP Client
	TokenResponse *TokenResponse // Token хариу
	Host          string         // Хост хаяг
}

// NewQPayClient нь шинэ QPayClient үүсгэх функц
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

// AuthToken нь token авах функц
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

// IsTokenExpired нь token-г шалгах функц ба хугацаа дууссан бол true буцаана
func (q *QPayClient) IsTokenExpired() bool {
	if q.TokenResponse == nil || q.TokenResponse.AccessToken == "" || q.TokenResponse.ExpiresIn == 0 {
		return true
	}

	expirationTime := time.Unix(q.TokenResponse.ExpiresIn, 0)
	return time.Now().After(expirationTime)
}

// CheckTokenAndRefresh нь token-г шалгах ба хугацаа дууссан бол дахин шинэ token авах функц
func (q *QPayClient) CheckTokenAndRefresh() error {
	if q.IsTokenExpired() {
		if err := q.AuthToken(); err != nil {
			return err
		}
	}

	return nil
}

// Request нь HTTP хүсэлт илгээх функц
func (q *QPayClient) Request(method, path string, body any) (*http.Response, error) {
	var err error

	if err = q.CheckTokenAndRefresh(); err != nil {
		return nil, err
	}

	var jsonBody []byte

	if body != nil {
		jsonBody, err = json.Marshal(body)

		if err != nil {
			return nil, err
		}
	}

	url := q.Host + path
	var request *http.Request

	switch method {
	case http.MethodGet:
		request, err = http.NewRequest(http.MethodGet, url, nil)
	case http.MethodPost:
		request, err = http.NewRequest(http.MethodPost, url, bytes.NewBuffer(jsonBody))
	case http.MethodPut:
		request, err = http.NewRequest(http.MethodPut, url, bytes.NewBuffer(jsonBody))
	case http.MethodPatch:
		request, err = http.NewRequest(http.MethodPatch, url, bytes.NewBuffer(jsonBody))
	case http.MethodDelete:
		request, err = http.NewRequest(http.MethodDelete, url, bytes.NewBuffer(jsonBody))
	default:
		return nil, errors.New("unsupported method")
	}

	if err != nil {
		return nil, err
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+q.TokenResponse.AccessToken)

	return q.Client.Do(request)
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

	if err := json.NewDecoder(response.Body).Decode(&invoiceResponse); err != nil {
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

	if err := json.NewDecoder(response.Body).Decode(&getInvoiceResponse); err != nil {
		return nil, err
	}

	return &getInvoiceResponse, nil
}
