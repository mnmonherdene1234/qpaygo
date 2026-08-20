package qpaygo

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// ErrorCode is one of QPay's machine-readable error identifiers, as
// documented in the "error_message" sheet of QPay's API doc, plus at least
// one code (ErrPaymentSettled) that is observed in real responses but
// missing from that table — QPay's own documentation of error codes is known
// to be incomplete. Some values below carry QPay's own documented spelling
// even where it looks like a typo (e.g. ErrEbarimtCancelNotSupported,
// ErrNoCredentials), since these are the literal values QPay sends on the
// wire.
type ErrorCode string

const (
	ErrAccountBankDuplicated              ErrorCode = "ACCOUNT_BANK_DUPLICATED"
	ErrAccountSelectionInvalid            ErrorCode = "ACCOUNT_SELECTION_INVALID"
	ErrAuthenticationFailed               ErrorCode = "AUTHENTICATION_FAILED"
	ErrBankAccountNotFound                ErrorCode = "BANK_ACCOUNT_NOTFOUND"
	ErrBankMCCAlreadyAdded                ErrorCode = "BANK_MCC_ALREADY_ADDED"
	ErrBankMCCNotFound                    ErrorCode = "BANK_MCC_NOT_FOUND"
	ErrCardTerminalNotFound               ErrorCode = "CARD_TERMINAL_NOTFOUND"
	ErrClientNotFound                     ErrorCode = "CLIENT_NOTFOUND"
	ErrClientUsernameDuplicated           ErrorCode = "CLIENT_USERNAME_DUPLICATED"
	ErrCustomerDuplicate                  ErrorCode = "CUSTOMER_DUPLICATE"
	ErrCustomerNotFound                   ErrorCode = "CUSTOMER_NOTFOUND"
	ErrCustomerRegisterInvalid            ErrorCode = "CUSTOMER_REGISTER_INVALID"
	ErrEbarimtCancelNotSupported          ErrorCode = "EBARIMT_CANCEL_NOTSUPPERDED" // sic — real wire value
	ErrEbarimtNotRegistered               ErrorCode = "EBARIMT_NOT_REGISTERED"
	ErrEbarimtQRCodeInvalid               ErrorCode = "EBARIMT_QR_CODE_INVALID"
	ErrInformNotFound                     ErrorCode = "INFORM_NOTFOUND"
	ErrInputCodeRegistered                ErrorCode = "INPUT_CODE_REGISTERED"
	ErrInputNotFound                      ErrorCode = "INPUT_NOTFOUND"
	ErrInvalidAmount                      ErrorCode = "INVALID_AMOUNT"
	ErrInvalidObjectType                  ErrorCode = "INVALID_OBJECT_TYPE"
	ErrInvoiceAlreadyCanceled             ErrorCode = "INVOICE_ALREADY_CANCELED"
	ErrInvoiceCodeInvalid                 ErrorCode = "INVOICE_CODE_INVALID"
	ErrInvoiceCodeRegistered              ErrorCode = "INVOICE_CODE_REGISTERED"
	ErrInvoiceLineRequired                ErrorCode = "INVOICE_LINE_REQUIRED"
	ErrInvoiceNotFound                    ErrorCode = "INVOICE_NOTFOUND"
	ErrInvoicePaid                        ErrorCode = "INVOICE_PAID"
	ErrInvoiceReceiverDataAddressRequired ErrorCode = "INVOICE_RECEIVER_DATA_ADDRESS_REQUIRED"
	ErrInvoiceReceiverDataEmailRequired   ErrorCode = "INVOICE_RECEIVER_DATA_EMAIL_REQUIRED"
	ErrInvoiceReceiverDataPhoneRequired   ErrorCode = "INVOICE_RECEIVER_DATA_PHONE_REQUIRED"
	ErrInvoiceReceiverDataRequired        ErrorCode = "INVOICE_RECEIVER_DATA_REQUIRED"
	ErrMaxAmount                          ErrorCode = "MAX_AMOUNT_ERR"
	ErrMCCNotFound                        ErrorCode = "MCC_NOTFOUND"
	ErrMerchantAlreadyRegistered          ErrorCode = "MERCHANT_ALREADY_REGISTERED"
	ErrMerchantInactive                   ErrorCode = "MERCHANT_INACTIVE"
	ErrMerchantNotFound                   ErrorCode = "MERCHANT_NOTFOUND"
	ErrMinAmount                          ErrorCode = "MIN_AMOUNT_ERR"
	ErrNoCredentials                      ErrorCode = "NO_CREDENDIALS" // sic
	ErrObjectDataError                    ErrorCode = "OBJECT_DATA_ERROR"
	ErrP2PTerminalNotFound                ErrorCode = "P2P_TERMINAL_NOTFOUND"
	ErrPaymentAlreadyCanceled             ErrorCode = "PAYMENT_ALREADY_CANCELED"
	ErrPaymentNotPaid                     ErrorCode = "PAYMENT_NOT_PAID"
	ErrPaymentNotFound                    ErrorCode = "PAYMENT_NOTFOUND"
	ErrPaymentSettled                     ErrorCode = "PAYMENT_SETTLED" // not in QPay's own error table, but observed on cancel/refund of non-card payments
	ErrPermissionDenied                   ErrorCode = "PERMISSION_DENIED"
	ErrQRAccountInactive                  ErrorCode = "QRACCOUNT_INACTIVE"
	ErrQRAccountNotFound                  ErrorCode = "QRACCOUNT_NOTFOUND"
	ErrQRCodeNotFound                     ErrorCode = "QRCODE_NOTFOUND"
	ErrQRCodeUsed                         ErrorCode = "QRCODE_USED"
	ErrSenderBranchDataRequired           ErrorCode = "SENDER_BRANCH_DATA_REQUIRED"
	ErrTaxLineRequired                    ErrorCode = "TAX_LINE_REQUIRED"
	ErrTaxProductCodeRequired             ErrorCode = "TAX_PRODUCT_CODE_REQUIRED"
	ErrTransactionNotApproved             ErrorCode = "TRANSACTION_NOT_APPROVED"
	ErrTransactionRequired                ErrorCode = "TRANSACTION_REQUIRED"
)

// APIError represents a non-2xx response from QPay's API. When QPay's
// response body is JSON of the documented {"error": "...", "message": "..."}
// shape, Code and Message are populated from it; otherwise Code is empty and
// Body carries the raw response text.
type APIError struct {
	StatusCode int
	Code       ErrorCode
	Message    string
	Body       string
}

func (e *APIError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("qpaygo: %s (http %d): %s", e.Code, e.StatusCode, e.Message)
	}
	return fmt.Sprintf("qpaygo: http %d: %s", e.StatusCode, e.Body)
}

func parseAPIError(response *http.Response) *APIError {
	body, readErr := io.ReadAll(response.Body)
	apiErr := &APIError{
		StatusCode: response.StatusCode,
		Body:       strings.TrimSpace(string(body)),
	}
	if readErr != nil {
		// The body may be truncated/incomplete; say so rather than
		// silently treating a partial read as the whole (possibly
		// non-JSON-parseable) error body.
		apiErr.Body = fmt.Sprintf("%s (response body read failed: %v)", apiErr.Body, readErr)
		return apiErr
	}
	if apiErr.Body == "" {
		apiErr.Body = response.Status
	}

	var wire struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	if json.Unmarshal(body, &wire) == nil && wire.Error != "" {
		apiErr.Code = ErrorCode(wire.Error)
		apiErr.Message = wire.Message
	}
	return apiErr
}

func isSuccess(statusCode int) bool {
	return statusCode >= http.StatusOK && statusCode < http.StatusMultipleChoices
}

// decodeJSONResponse decodes a successful JSON response body into dst, or
// returns an *APIError describing the failure.
func decodeJSONResponse(response *http.Response, dst any) error {
	defer response.Body.Close()

	if !isSuccess(response.StatusCode) {
		return parseAPIError(response)
	}

	return json.NewDecoder(response.Body).Decode(dst)
}

// decodeEmptyResponse consumes a successful response whose body carries no
// useful payload (QPay returns "{}" for most cancel/refund endpoints), or
// returns an *APIError describing the failure.
func decodeEmptyResponse(response *http.Response) error {
	defer response.Body.Close()

	if !isSuccess(response.StatusCode) {
		return parseAPIError(response)
	}

	_, _ = io.Copy(io.Discard, response.Body)
	return nil
}
