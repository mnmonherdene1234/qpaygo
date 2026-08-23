package qpaygo

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
)

// ErrorCode is one of QPay's machine-readable error identifiers, as
// documented in the "error_message" sheet of QPay's API doc, plus at least
// one code (ErrPaymentSettled) that is observed in real responses but
// missing from that table — QPay's own documentation of error codes is known
// to be incomplete. Some values below carry QPay's own documented spelling
// even where it looks like a typo (e.g. ErrEbarimtCancelNotSupported,
// ErrNoCredentials), since these are the literal values QPay sends on the
// wire. Other spellings (ErrNoCredentialsProduction, ErrSystemBusy, ...)
// were captured from live production/sandbox responses.
type ErrorCode string

const (
	ErrAccountBankDuplicated              ErrorCode = "ACCOUNT_BANK_DUPLICATED"
	ErrAccountSelectionInvalid            ErrorCode = "ACCOUNT_SELECTION_INVALID"
	ErrAuthenticationFailed               ErrorCode = "AUTHENTICATION_FAILED"
	ErrBankAccountNotFound                ErrorCode = "BANK_ACCOUNT_NOTFOUND"
	ErrBankMCCAlreadyAdded                ErrorCode = "BANK_MCC_ALREADY_ADDED"
	ErrBankMCCNotFound                    ErrorCode = "BANK_MCC_NOT_FOUND"
	ErrBarimtNotFound                     ErrorCode = "BARIMT_NOT_FOUND" // observed live: GET /v2/ebarimt on an unknown id (422)
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
	ErrNoCredentials                      ErrorCode = "NO_CREDENDIALS" // sic — QPay's documented (sandbox) spelling
	ErrNoCredentialsProduction            ErrorCode = "NO_CREDENTIALS" // observed live: production returns this for a missing/invalid bearer token (401)
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
	ErrSystemBusy                         ErrorCode = "SYSTEM_BUSY" // observed live: sandbox/production return this (500) for e.g. cancel-ebarimt on an unknown id
	ErrTaxLineRequired                    ErrorCode = "TAX_LINE_REQUIRED"
	ErrTaxProductCodeRequired             ErrorCode = "TAX_PRODUCT_CODE_REQUIRED"
	ErrTransactionNotApproved             ErrorCode = "TRANSACTION_NOT_APPROVED"
	ErrTransactionRequired                ErrorCode = "TRANSACTION_REQUIRED"
	ErrTypeError                          ErrorCode = "TypeError" // observed live: sandbox ebarimt create returns this (500) — a QPay server-side defect, not client-actionable
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
	if e.Message != "" {
		return fmt.Sprintf("qpaygo: http %d: %s", e.StatusCode, e.Message)
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

	// QPay's documented error shape is {"error": "<CODE>", "message": "..."},
	// but live responses show two deviations:
	//   1. the values may be nested OBJECTS (field-validation failures, e.g.
	//      {"error":{"page_limit":{"type":"MIN_NUMBER","message":"..."}}}),
	//      in which case there is no single code — the object is flattened
	//      into Message;
	//   2. the spelling of codes may differ per environment
	//      (NO_CREDENDIALS vs NO_CREDENTIALS), which is why Code is stored
	//      as the raw wire string rather than being forced into a constant.
	var wire struct {
		Error   json.RawMessage `json:"error"`
		Message json.RawMessage `json:"message"`
	}
	if json.Unmarshal(body, &wire) == nil {
		if code, ok := jsonRawText(wire.Error); ok && code != "" {
			apiErr.Code = ErrorCode(code)
		} else if len(wire.Error) > 0 {
			apiErr.Message = flattenErrorObject(wire.Error)
		}
		if msg, ok := jsonRawText(wire.Message); ok && msg != "" {
			apiErr.Message = msg
		} else if len(wire.Message) > 0 && apiErr.Message == "" {
			apiErr.Message = flattenErrorObject(wire.Message)
		}
	}
	return apiErr
}

// jsonRawText returns the string value of a JSON string literal, or ("", false)
// when raw is not a string literal (e.g. an object, number, null, or empty).
func jsonRawText(raw json.RawMessage) (string, bool) {
	if len(raw) == 0 {
		return "", false
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return "", false
	}
	return s, true
}

// flattenErrorObject converts an object-shaped error value into a readable
// single-line string, e.g.
//
//	{"page_limit":{"type":"MIN_NUMBER","message":"Number min value [1]!"}}
//	→ "page_limit: MIN_NUMBER: Number min value [1]!"
//
// Nested values are handled recursively; keys are sorted for determinism.
// An object of the shape {"type": "...", "message": "..."} is rendered as
// "type: message".
func flattenErrorObject(raw json.RawMessage) string {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil {
		// Not an object; fall back to the compact raw form.
		var compact any
		if err := json.Unmarshal(raw, &compact); err != nil {
			return string(raw)
		}
		if b, err := json.Marshal(compact); err == nil {
			return string(b)
		}
		return string(raw)
	}

	if len(obj) == 2 {
		if typ, ok := jsonRawText(obj["type"]); ok {
			if msg, ok := jsonRawText(obj["message"]); ok {
				if typ != "" && msg != "" {
					return typ + ": " + msg
				}
				if msg != "" {
					return msg
				}
			}
			if typ != "" {
				return typ
			}
		}
	}

	keys := make([]string, 0, len(obj))
	for k := range obj {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(obj))
	for _, k := range keys {
		v := obj[k]
		if s, ok := jsonRawText(v); ok {
			parts = append(parts, k+": "+s)
			continue
		}
		parts = append(parts, k+": "+flattenErrorObject(v))
	}
	return strings.Join(parts, "; ")
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

	if response.StatusCode == http.StatusNoContent {
		// QPay's endpoints that return JSON always return a body today, but
		// guard against a bare 204 so callers get a clear error instead of a
		// confusing "EOF" from the decoder.
		return fmt.Errorf("qpaygo: unexpected 204 No Content response (endpoint returned no JSON body)")
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
