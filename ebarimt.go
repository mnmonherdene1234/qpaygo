package qpaygo

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
)

// EbarimtReceiverType identifies who an eBarimt tax receipt is issued to.
type EbarimtReceiverType string

const (
	EbarimtReceiverCitizen EbarimtReceiverType = "CITIZEN"
	EbarimtReceiverCompany EbarimtReceiverType = "COMPANY" // doc has a typo elsewhere ("COMPAYNE"); this is the correct spelling
)

// CreateEbarimtRequest is the request body for CreateEbarimt.
type CreateEbarimtRequest struct {
	PaymentID           string              `json:"payment_id"`
	EbarimtReceiverType EbarimtReceiverType `json:"ebarimt_receiver_type"`
	EbarimtReceiver     string              `json:"ebarimt_receiver,omitempty"`
	CallbackURL         string              `json:"callback_url,omitempty"`
}

// EbarimtItem is a line item within an eBarimt receipt.
type EbarimtItem struct {
	ID                  string `json:"id"`
	BarimtID            string `json:"barimt_id"`
	MerchantProductCode string `json:"merchant_product_code"`
	TaxProductCode      string `json:"tax_product_code"`
	BarCode             string `json:"bar_code"`
	Name                string `json:"name"`
	UnitPrice           Number `json:"unit_price"`
	Quantity            Number `json:"quantity"`
	Amount              Number `json:"amount"`
	CityTaxAmount       Number `json:"city_tax_amount"`
	VATAmount           Number `json:"vat_amount"`
	Note                string `json:"note"`
	CreatedBy           string `json:"created_by"`
	CreatedDate         string `json:"created_date"`
	UpdatedBy           string `json:"updated_by"`
	UpdatedDate         string `json:"updated_date"`
	Status              bool   `json:"status"`
}

// EbarimtHistory is a status-change history entry for an eBarimt receipt.
type EbarimtHistory struct {
	ID                   string `json:"id"`
	BarimtID             string `json:"barimt_id"`
	EbarimtReceiverType  string `json:"ebarimt_receiver_type"`
	EbarimtReceiver      string `json:"ebarimt_receiver"`
	EbarimtRegisterNo    string `json:"ebarimt_register_no"`
	EbarimtBillID        string `json:"ebarimt_bill_id"`
	EbarimtDate          string `json:"ebarimt_date"`
	EbarimtMacAddress    string `json:"ebarimt_mac_address"`
	EbarimtInternalCode  string `json:"ebarimt_internal_code"`
	EbarimtBillType      string `json:"ebarimt_bill_type"`
	EbarimtQRData        string `json:"ebarimt_qr_data"`
	EbarimtLottery       string `json:"ebarimt_lottery"`
	EbarimtLotteryMsg    string `json:"ebarimt_lottery_msg"`
	EbarimtErrorCode     string `json:"ebarimt_error_code"`
	EbarimtErrorMsg      string `json:"ebarimt_error_msg"`
	EbarimtResponseCode  string `json:"ebarimt_response_code"`
	EbarimtResponseMsg   string `json:"ebarimt_response_msg"`
	Note                 string `json:"note"`
	BarimtStatus         string `json:"barimt_status"`
	BarimtStatusDate     string `json:"barimt_status_date"`
	EbarimtSentEmail     string `json:"ebarimt_sent_email"`
	EbarimtReceiverPhone string `json:"ebarimt_receiver_phone"`
	TaxType              string `json:"tax_type"`
	CreatedBy            string `json:"created_by"`
	CreatedDate          string `json:"created_date"`
	UpdatedBy            string `json:"updated_by"`
	UpdatedDate          string `json:"updated_date"`
	Status               bool   `json:"status"`
}

// EbarimtResponse is returned by CreateEbarimt and GetEbarimt.
//
// QPay's official field table is riddled with typos that contradict its own
// JSON example body (the example uses barimt_status / merchant_register_no /
// ebarimt_receiver_type, which this struct follows).
type EbarimtResponse struct {
	ID                   string              `json:"id"`
	EbarimtBy            string              `json:"ebarimt_by"`
	GWalletID            string              `json:"g_wallet_id"`
	GWalletCustomerID    string              `json:"g_wallet_customer_id"`
	EbarimtReceiverType  EbarimtReceiverType `json:"ebarimt_receiver_type"`
	EbarimtReceiver      string              `json:"ebarimt_receiver"`
	EbarimtDistrictCode  string              `json:"ebarimt_district_code"`
	EbarimtBillType      string              `json:"ebarimt_bill_type"`
	GMerchantID          string              `json:"g_merchant_id"`
	MerchantBranchCode   string              `json:"merchant_branch_code"`
	MerchantTerminalCode string              `json:"merchant_terminal_code"`
	MerchantStaffCode    string              `json:"merchant_staff_code"`
	MerchantRegisterNo   string              `json:"merchant_register_no"`
	GPaymentID           string              `json:"g_payment_id"`
	PaidBy               TransportType       `json:"paid_by"`
	ObjectType           ObjectType          `json:"object_type"`
	ObjectID             string              `json:"object_id"`
	Amount               Number              `json:"amount"`
	VATAmount            Number              `json:"vat_amount"`
	CityTaxAmount        Number              `json:"city_tax_amount"`
	EbarimtQRData        string              `json:"ebarimt_qr_data"`
	EbarimtLottery       string              `json:"ebarimt_lottery"`
	Note                 string              `json:"note"`
	BarimtStatus         string              `json:"barimt_status"` // e.g. "REGISTERED"
	BarimtStatusDate     string              `json:"barimt_status_date"`
	EbarimtSentEmail     string              `json:"ebarimt_sent_email"`
	EbarimtReceiverPhone string              `json:"ebarimt_receiver_phone"`
	TaxType              string              `json:"tax_type"`
	CreatedBy            string              `json:"created_by"`
	CreatedDate          string              `json:"created_date"`
	UpdatedBy            string              `json:"updated_by"`
	UpdatedDate          string              `json:"updated_date"`
	Status               bool                `json:"status"`
	BarimtItems          []EbarimtItem       `json:"barimt_items"`
	// BarimtTransactions' shape is undocumented and always empty in every
	// example seen; modeled as raw JSON rather than guessed.
	BarimtTransactions []json.RawMessage `json:"barimt_transactions"`
	BarimtHistories    []EbarimtHistory  `json:"barimt_histories"`
}

// CreateEbarimt creates a tax receipt for a payment via
// POST /v2/ebarimt/create.
func (q *QPayClient) CreateEbarimt(ctx context.Context, req CreateEbarimtRequest) (*EbarimtResponse, error) {
	return requestJSON[EbarimtResponse](ctx, q, http.MethodPost, "/v2/ebarimt/create", req)
}

// GetEbarimt fetches eBarimt receipt details via GET /v2/ebarimt/:barimt_id.
func (q *QPayClient) GetEbarimt(ctx context.Context, barimtID string) (*EbarimtResponse, error) {
	return requestJSON[EbarimtResponse](ctx, q, http.MethodGet, "/v2/ebarimt/"+url.PathEscape(barimtID), nil)
}

// CancelEbarimt cancels an eBarimt receipt via DELETE /v2/ebarimt/:barimt_id.
//
// IMPORTANT — this endpoint is only thinly documented by QPay (it appears in
// the API overview only, with no dedicated request/response schema), and
// QPay defines an explicit error code EBARIMT_CANCEL_NOTSUPPERDED ("qPay
// service eBarimt unregister function not supported") suggesting
// cancellation may be rejected depending on merchant configuration. Treat
// that error as an expected, documented outcome rather than a bug.
func (q *QPayClient) CancelEbarimt(ctx context.Context, barimtID string) error {
	return requestEmpty(ctx, q, http.MethodDelete, "/v2/ebarimt/"+url.PathEscape(barimtID), nil)
}
