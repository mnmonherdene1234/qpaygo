package qpaygo

import (
	"fmt"
	"strconv"
	"strings"
)

// Number represents a decimal amount or quantity. QPay's v2 API encodes the
// same logical field inconsistently across (and sometimes within) endpoints:
// as a bare JSON number (e.g. 100.5) or as a quoted numeric string (e.g.
// "100.50"). UnmarshalJSON accepts either; MarshalJSON always emits a bare
// JSON number.
type Number float64

// Float64 returns n as a float64.
func (n Number) Float64() float64 { return float64(n) }

func (n *Number) UnmarshalJSON(data []byte) error {
	s := strings.TrimSpace(string(data))
	if s == "null" || s == "" {
		*n = 0
		return nil
	}
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		s = s[1 : len(s)-1]
		if s == "" {
			*n = 0
			return nil
		}
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return fmt.Errorf("qpaygo: invalid number %q: %w", s, err)
	}
	*n = Number(f)
	return nil
}

func (n Number) MarshalJSON() ([]byte, error) {
	return []byte(strconv.FormatFloat(float64(n), 'f', -1, 64)), nil
}

// TokenResponse is the OAuth2/Keycloak-style token payload returned by
// POST /v2/auth/token and POST /v2/auth/refresh.
type TokenResponse struct {
	TokenType        string `json:"token_type"`
	RefreshExpiresIn int64  `json:"refresh_expires_in"`
	RefreshToken     string `json:"refresh_token"`
	AccessToken      string `json:"access_token"`
	ExpiresIn        int64  `json:"expires_in"`
	Scope            string `json:"scope"`
	NotBeforePolicy  string `json:"not-before-policy"`
	SessionState     string `json:"session_state"`
}

// TransportType identifies whether a payment moved via a P2P bank transfer or
// a card network. QPay names the JSON field carrying this value differently
// per endpoint: transaction_type (GetPayment), payment_type (CheckPayment),
// paid_by (ListPayments, eBarimt) — the values themselves are consistent.
type TransportType string

const (
	TransportP2P  TransportType = "P2P"
	TransportCard TransportType = "CARD"
)

// PaymentStatus is QPay's payment lifecycle state.
type PaymentStatus string

const (
	PaymentStatusNew      PaymentStatus = "NEW"
	PaymentStatusFailed   PaymentStatus = "FAILED"
	PaymentStatusPaid     PaymentStatus = "PAID"
	PaymentStatusPartial  PaymentStatus = "PARTIAL"
	PaymentStatusRefunded PaymentStatus = "REFUNDED"
)

// ObjectType identifies what a payment lookup is scoped to.
type ObjectType string

const (
	ObjectTypeInvoice  ObjectType = "INVOICE"
	ObjectTypeQR       ObjectType = "QR"
	ObjectTypeItem     ObjectType = "ITEM"     // valid for CheckPayment only
	ObjectTypeMerchant ObjectType = "MERCHANT" // valid for ListPayments only
)

// Offset paginates payment/check and payment/list requests.
type Offset struct {
	PageNumber int `json:"page_number"`
	PageLimit  int `json:"page_limit"`
}
