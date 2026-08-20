package qpaygo

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
)

// Address is a postal address, used in invoice sender/receiver data.
type Address struct {
	City      string `json:"city,omitempty"`
	District  string `json:"district,omitempty"`
	Street    string `json:"street,omitempty"`
	Building  string `json:"building,omitempty"`
	Address   string `json:"address,omitempty"`
	Zipcode   string `json:"zipcode,omitempty"`
	Longitude string `json:"longitude,omitempty"`
	Latitude  string `json:"latitude,omitempty"`
}

// BranchData describes the merchant branch issuing an invoice.
type BranchData struct {
	Register string   `json:"register,omitempty"`
	Name     string   `json:"name,omitempty"`
	Email    string   `json:"email,omitempty"`
	Phone    string   `json:"phone,omitempty"`
	Address  *Address `json:"address,omitempty"`
}

// StaffData describes the merchant staff member issuing an invoice.
type StaffData struct {
	Name  string `json:"name,omitempty"`
	Email string `json:"email,omitempty"`
	Phone string `json:"phone,omitempty"`
}

// ReceiverData describes the customer receiving an invoice.
type ReceiverData struct {
	Register string   `json:"register,omitempty"`
	Name     string   `json:"name,omitempty"`
	Email    string   `json:"email,omitempty"`
	Phone    string   `json:"phone,omitempty"`
	Address  *Address `json:"address,omitempty"`
}

// Discount is a line-item discount.
type Discount struct {
	DiscountCode string `json:"discount_code,omitempty"`
	Description  string `json:"description"`
	Amount       Number `json:"amount"`
	Note         string `json:"note,omitempty"`
}

// Surcharge is a line-item surcharge.
type Surcharge struct {
	SurchargeCode string `json:"surcharge_code,omitempty"`
	Description   string `json:"description"`
	Amount        Number `json:"amount"`
	Note          string `json:"note,omitempty"`
}

// LineTax is a line-item tax (VAT or city tax).
type LineTax struct {
	TaxCode     string `json:"tax_code,omitempty"` // "VAT" | "CITY_TAX"
	Description string `json:"description"`
	Amount      Number `json:"amount"`
	Note        string `json:"note,omitempty"`
}

// Line is a single invoice line item.
type Line struct {
	TaxProductCode  string      `json:"tax_product_code,omitempty"`
	LineDescription string      `json:"line_description"`
	LineQuantity    Number      `json:"line_quantity"`
	LineUnitPrice   Number      `json:"line_unit_price"`
	Note            string      `json:"note,omitempty"`
	Discounts       []Discount  `json:"discounts,omitempty"`
	Surcharges      []Surcharge `json:"surcharges,omitempty"`
	Taxes           []LineTax   `json:"taxes,omitempty"`
}

// SettlementAccount is a bank account entry within a split-settlement
// invoice transaction.
type SettlementAccount struct {
	AccountBankCode string `json:"account_bank_code"`
	AccountNumber   string `json:"account_number"`
	AccountName     string `json:"account_name"`
	AccountCurrency string `json:"account_currency"`
	IsDefault       bool   `json:"is_default"`
}

// InvoiceTransaction splits part of an invoice's settlement to specific bank
// accounts. Requires operator/aggregator rights on the merchant account.
type InvoiceTransaction struct {
	Description string              `json:"description"`
	Amount      Number              `json:"amount"`
	Accounts    []SettlementAccount `json:"accounts,omitempty"`
}

// CreateInvoiceRequest is the full-form request body for POST /v2/invoice,
// supporting line items, discounts/surcharges/taxes, split-settlement
// transactions, and subscription/tax configuration.
type CreateInvoiceRequest struct {
	InvoiceCode          string               `json:"invoice_code"`
	SenderInvoiceNo      string               `json:"sender_invoice_no"`
	SenderBranchCode     string               `json:"sender_branch_code,omitempty"`
	SenderBranchData     *BranchData          `json:"sender_branch_data,omitempty"`
	SenderStaffCode      string               `json:"sender_staff_code,omitempty"`
	SenderStaffData      *StaffData           `json:"sender_staff_data,omitempty"`
	SenderTerminalCode   string               `json:"sender_terminal_code,omitempty"`
	SenderTerminalData   string               `json:"sender_terminal_data,omitempty"`
	InvoiceReceiverCode  string               `json:"invoice_receiver_code"`
	InvoiceReceiverData  *ReceiverData        `json:"invoice_receiver_data,omitempty"`
	InvoiceDescription   string               `json:"invoice_description"`
	EnableExpiry         bool                 `json:"enable_expiry,omitempty"`
	ExpiryDate           string               `json:"expiry_date,omitempty"`
	CalculateVAT         bool                 `json:"calculate_vat,omitempty"`
	TaxType              string               `json:"tax_type,omitempty"` // "1" VAT product | "2" no VAT | "3" VAT-exempt
	TaxCustomerCode      string               `json:"tax_customer_code,omitempty"`
	LineTaxCode          string               `json:"line_tax_code,omitempty"`
	AllowPartial         bool                 `json:"allow_partial"` // doc table says "array", every real example is a bool
	MinimumAmount        *Number              `json:"minimum_amount,omitempty"`
	AllowExceed          bool                 `json:"allow_exceed"`
	MaximumAmount        *Number              `json:"maximum_amount,omitempty"`
	Amount               Number               `json:"amount,omitempty"` // omit when Lines carries the amount instead
	CallbackURL          string               `json:"callback_url"`
	AllowSubscribe       bool                 `json:"allow_subscribe,omitempty"`
	SubscriptionInterval string               `json:"subscription_interval,omitempty"`
	SubscriptionWebhook  string               `json:"subscription_webhook,omitempty"`
	Note                 string               `json:"note,omitempty"`
	Lines                []Line               `json:"lines,omitempty"`
	Transactions         []InvoiceTransaction `json:"transactions,omitempty"`
}

// CreateSimpleInvoiceRequest is the minimal-field request body for
// POST /v2/invoice ("simple" form).
type CreateSimpleInvoiceRequest struct {
	InvoiceCode         string `json:"invoice_code"`
	SenderInvoiceNo     string `json:"sender_invoice_no"`
	InvoiceReceiverCode string `json:"invoice_receiver_code"`
	InvoiceDescription  string `json:"invoice_description"`
	SenderBranchCode    string `json:"sender_branch_code,omitempty"`
	Amount              Number `json:"amount"`
	CallbackURL         string `json:"callback_url"`
}

// BankLink is a deep-link option for opening a specific banking app to pay
// an invoice's QR code.
type BankLink struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Logo        string `json:"logo"`
	Link        string `json:"link"`
}

// InvoiceResponse is returned by both CreateInvoice and CreateSimpleInvoice.
type InvoiceResponse struct {
	InvoiceID    string     `json:"invoice_id"`
	QRText       string     `json:"qr_text"`
	QRImage      string     `json:"qr_image"`
	QPayShortURL string     `json:"qPay_shortUrl"`
	URLs         []BankLink `json:"urls"`
}

// GetInvoiceResponse is returned by GetInvoice.
type GetInvoiceResponse struct {
	InvoiceID          string `json:"invoice_id"`
	InvoiceStatus      string `json:"invoice_status"` // "OPEN" | "CLOSED"
	SenderInvoiceNo    string `json:"sender_invoice_no"`
	SenderBranchCode   string `json:"sender_branch_code"`
	SenderBranchData   string `json:"sender_branch_data"`
	SenderStaffCode    string `json:"sender_staff_code"`
	SenderStaffData    string `json:"sender_staff_data"`
	SenderTerminalCode string `json:"sender_terminal_code"`
	SenderTerminalData string `json:"sender_terminal_data"`
	InvoiceDescription string `json:"invoice_description"`
	InvoiceDueDate     string `json:"invoice_due_date"`
	EnableExpiry       bool   `json:"enable_expiry"`
	ExpiryDate         string `json:"expiry_date"`
	AllowPartial       bool   `json:"allow_partial"`
	MinimumAmount      Number `json:"minimum_amount"`
	AllowExceed        bool   `json:"allow_exceed"`
	MaximumAmount      Number `json:"maximum_amount"`
	TotalAmount        Number `json:"total_amount"`
	GrossAmount        Number `json:"gross_amount"`
	TaxAmount          Number `json:"tax_amount"`
	SurchargeAmount    Number `json:"surcharge_amount"`
	DiscountAmount     Number `json:"discount_amount"`
	CallbackURL        string `json:"callback_url"`
	Note               string `json:"note"`
	Lines              []Line `json:"lines"`
	// Transactions/Inputs shapes are not documented anywhere in QPay's
	// response schema (only in the create *request*, for Transactions);
	// modeled as raw JSON so callers get valid structured data without
	// qpaygo guessing at an unverified shape.
	Transactions []json.RawMessage `json:"transactions"`
	Inputs       []json.RawMessage `json:"inputs"`
}

// CreateInvoice creates a full-form invoice via POST /v2/invoice, supporting
// line items, discounts/surcharges/taxes, and split-settlement transactions.
func (q *QPayClient) CreateInvoice(ctx context.Context, req CreateInvoiceRequest) (*InvoiceResponse, error) {
	if req.InvoiceCode == "" {
		req.InvoiceCode = q.invoiceCode
	}
	return requestJSON[InvoiceResponse](ctx, q, http.MethodPost, "/v2/invoice", req)
}

// CreateSimpleInvoice creates a minimal-field invoice via POST /v2/invoice.
func (q *QPayClient) CreateSimpleInvoice(ctx context.Context, req CreateSimpleInvoiceRequest) (*InvoiceResponse, error) {
	if req.InvoiceCode == "" {
		req.InvoiceCode = q.invoiceCode
	}
	return requestJSON[InvoiceResponse](ctx, q, http.MethodPost, "/v2/invoice", req)
}

// GetInvoice fetches invoice details via GET /v2/invoice/:invoice_id.
func (q *QPayClient) GetInvoice(ctx context.Context, invoiceID string) (*GetInvoiceResponse, error) {
	return requestJSON[GetInvoiceResponse](ctx, q, http.MethodGet, "/v2/invoice/"+url.PathEscape(invoiceID), nil)
}

// CancelInvoice cancels an invoice via DELETE /v2/invoice/:invoice_id. QPay
// returns an empty {} body on success.
func (q *QPayClient) CancelInvoice(ctx context.Context, invoiceID string) error {
	return requestEmpty(ctx, q, http.MethodDelete, "/v2/invoice/"+url.PathEscape(invoiceID), nil)
}
