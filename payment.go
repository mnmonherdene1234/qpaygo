package qpaygo

import (
	"context"
	"net/http"
	"net/url"
	"time"
)

// CardTransaction is a card-network transaction leg of a payment.
type CardTransaction struct {
	CardMerchantCode     string `json:"card_merchant_code"`
	CardTerminalCode     string `json:"card_terminal_code"`
	CardNumber           string `json:"card_number"`
	CardType             string `json:"card_type"`
	IsCrossBorder        bool   `json:"is_cross_border"`
	TransactionAmount    Number `json:"transaction_amount"`
	TransactionCurrency  string `json:"transaction_currency"`
	TransactionDate      string `json:"transaction_date"`
	TransactionStatus    string `json:"transaction_status"`
	SettlementStatus     string `json:"settlement_status"`
	SettlementStatusDate string `json:"settlement_status_date"`
}

// P2PTransaction is a bank-transfer transaction leg of a payment.
type P2PTransaction struct {
	TransactionBankCode string `json:"transaction_bank_code"`
	AccountBankCode     string `json:"account_bank_code"`
	AccountBankName     string `json:"account_bank_name"`
	AccountNumber       string `json:"account_number"`
	Status              string `json:"status"`
	Amount              Number `json:"amount"`
	Currency            string `json:"currency"`
	SettlementStatus    string `json:"settlement_status"`
}

// GetPaymentResponse is returned by GetPayment.
type GetPaymentResponse struct {
	PaymentID           string            `json:"payment_id"`
	PaymentStatus       PaymentStatus     `json:"payment_status"`
	PaymentAmount       Number            `json:"payment_amount"`
	PaymentFee          Number            `json:"payment_fee"`
	PaymentCurrency     string            `json:"payment_currency"`
	PaymentDate         string            `json:"payment_date"`
	PaymentWallet       string            `json:"payment_wallet"`
	TransportType       TransportType     `json:"transaction_type"`
	ObjectType          ObjectType        `json:"object_type"`
	ObjectID            string            `json:"object_id"`
	NextPaymentDate     string            `json:"next_payment_date"`
	NextPaymentDatetime string            `json:"next_payment_datetime"`
	CardTransactions    []CardTransaction `json:"card_transactions"`
	P2PTransactions     []P2PTransaction  `json:"p2p_transactions"`
}

// GetPayment fetches payment details for paymentID via
// GET /v2/payment/:payment_id.
//
// IMPORTANT — QPay explicitly forbids calling this (or CheckPayment) on a
// cron/scheduled basis. Only call it synchronously in direct response to a
// callback hit on your callback_url handler. See ExtractPaymentID/
// VerifyCallback for the intended flow.
//
// IMPORTANT — a callback hit is NOT proof of payment by itself: QPay's v2
// API has no signature/HMAC verification, so callback requests could in
// principle be spoofed. Always independently confirm via GetPayment/
// CheckPayment using your own bearer token before treating a payment as
// settled.
func (q *QPayClient) GetPayment(ctx context.Context, paymentID string) (*GetPaymentResponse, error) {
	return requestJSON[GetPaymentResponse](ctx, q, http.MethodGet, "/v2/payment/"+url.PathEscape(paymentID), nil)
}

// CheckPaymentRequest is the request body for CheckPayment.
type CheckPaymentRequest struct {
	ObjectType ObjectType `json:"object_type"` // INVOICE | QR | ITEM
	ObjectID   string     `json:"object_id"`
	Offset     Offset     `json:"offset"`
}

// PaymentCheckRow is one payment row within a CheckPaymentResponse.
type PaymentCheckRow struct {
	PaymentID           string            `json:"payment_id"`
	PaymentStatus       PaymentStatus     `json:"payment_status"`
	PaymentAmount       Number            `json:"payment_amount"`
	TrxFee              Number            `json:"trx_fee"`
	PaymentCurrency     string            `json:"payment_currency"`
	PaymentWallet       string            `json:"payment_wallet"`
	TransportType       TransportType     `json:"payment_type"`
	NextPaymentDate     string            `json:"next_payment_date"`
	NextPaymentDatetime string            `json:"next_payment_datetime"`
	CardTransactions    []CardTransaction `json:"card_transactions"`
	P2PTransactions     []P2PTransaction  `json:"p2p_transactions"`
}

// CheckPaymentResponse is returned by CheckPayment.
type CheckPaymentResponse struct {
	Count      int               `json:"count"`
	PaidAmount Number            `json:"paid_amount"`
	Rows       []PaymentCheckRow `json:"rows"`
}

// CheckPayment checks payment status for an invoice/QR/item via
// POST /v2/payment/check. Carries the same no-cron-polling and
// always-reverify-callback warnings as GetPayment.
func (q *QPayClient) CheckPayment(ctx context.Context, req CheckPaymentRequest) (*CheckPaymentResponse, error) {
	return requestJSON[CheckPaymentResponse](ctx, q, http.MethodPost, "/v2/payment/check", req)
}

// CancelPaymentRequest is the request body for CancelPayment and
// RefundPayment.
type CancelPaymentRequest struct {
	Note string `json:"note"`
	// CallbackURL is seen in one real example curl but not in the documented
	// field table; included defensively, omit if unused.
	CallbackURL string `json:"callback_url,omitempty"`
}

// RefundPaymentRequest is the request body for RefundPayment (identical
// wire shape to CancelPaymentRequest).
type RefundPaymentRequest = CancelPaymentRequest

// CancelPayment cancels a settled payment via DELETE /v2/payment/cancel/:id.
//
// IMPORTANT — this only works for CARD transactions. Calling it for a P2P /
// bank-transfer payment fails with error code PAYMENT_SETTLED (an error code
// QPay's own error table omits, but which is confirmed in their API
// examples).
func (q *QPayClient) CancelPayment(ctx context.Context, paymentID string, req CancelPaymentRequest) error {
	return requestEmpty(ctx, q, http.MethodDelete, "/v2/payment/cancel/"+url.PathEscape(paymentID), req)
}

// RefundPayment refunds a settled payment via DELETE /v2/payment/refund/:id.
// Carries the identical card-only restriction as CancelPayment.
func (q *QPayClient) RefundPayment(ctx context.Context, paymentID string, req RefundPaymentRequest) error {
	return requestEmpty(ctx, q, http.MethodDelete, "/v2/payment/refund/"+url.PathEscape(paymentID), req)
}

// ListPaymentsRequest is the request body for ListPayments.
type ListPaymentsRequest struct {
	ObjectType ObjectType `json:"object_type"` // MERCHANT | INVOICE | QR
	ObjectID   string     `json:"object_id"`
	StartDate  string     `json:"start_date"` // "YYYY-MM-DD HH:mm:ss"
	EndDate    string     `json:"end_date"`
	Offset     Offset     `json:"offset"`
}

// PaymentListRow is one payment row within a ListPaymentsResponse.
type PaymentListRow struct {
	PaymentID          string        `json:"payment_id"`
	PaymentDate        string        `json:"payment_date"`
	PaymentStatus      PaymentStatus `json:"payment_status"`
	PaymentFee         Number        `json:"payment_fee"`
	PaymentAmount      Number        `json:"payment_amount"`
	PaymentCurrency    string        `json:"payment_currency"`
	PaymentWallet      string        `json:"payment_wallet"`
	PaymentName        string        `json:"payment_name"`
	PaymentDescription string        `json:"payment_description"`
	QRCode             string        `json:"qr_code"`
	TransportType      TransportType `json:"paid_by"`
	ObjectType         ObjectType    `json:"object_type"`
	ObjectID           string        `json:"object_id"`
}

// ListPaymentsResponse is returned by ListPayments.
type ListPaymentsResponse struct {
	Count int              `json:"count"`
	Rows  []PaymentListRow `json:"rows"`
}

// ListPayments lists payments for a merchant/invoice/QR within a date range
// via POST /v2/payment/list.
func (q *QPayClient) ListPayments(ctx context.Context, req ListPaymentsRequest) (*ListPaymentsResponse, error) {
	return requestJSON[ListPaymentsResponse](ctx, q, http.MethodPost, "/v2/payment/list", req)
}

// FormatQPayTime formats t in the "YYYY-MM-DD HH:mm:ss" layout QPay expects
// for ListPaymentsRequest's StartDate/EndDate fields.
func FormatQPayTime(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}
