package qpaygo

import (
	"context"
	"fmt"
	"io"
	"net/http"
)

// ExtractPaymentID extracts the QPay payment id from a callback request's
// query string. QPay's callback documentation names the parameter
// qpay_payment_id, but some example payloads elsewhere in QPay's own docs
// use payment_id instead — this checks both, preferring qpay_payment_id.
func ExtractPaymentID(r *http.Request) (string, bool) {
	q := r.URL.Query()
	if id := q.Get("qpay_payment_id"); id != "" {
		return id, true
	}
	if id := q.Get("payment_id"); id != "" {
		return id, true
	}
	return "", false
}

// WriteCallbackAck writes the exact response QPay requires for a callback
// request: HTTP 200 with body exactly "SUCCESS". QPay's documentation
// explicitly states any other response format is prohibited.
func WriteCallbackAck(w http.ResponseWriter) {
	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, "SUCCESS")
}

// VerifyCallback extracts the payment id from a callback request and
// independently confirms its status via GetPayment, using the client's own
// bearer token. QPay's v2 API has no signature/HMAC verification of
// callback requests, so the callback hit alone must never be trusted as
// proof of payment — VerifyCallback is the recommended way to close that
// gap.
//
// Call this ONLY from your callback HTTP handler, in direct response to an
// incoming request. Do not call it (or GetPayment/CheckPayment) from a cron
// job or any periodic reconciliation loop — QPay's documentation explicitly
// forbids polling payment/check or payment/get outside of a callback.
func (q *QPayClient) VerifyCallback(ctx context.Context, r *http.Request) (*GetPaymentResponse, error) {
	id, ok := ExtractPaymentID(r)
	if !ok {
		return nil, fmt.Errorf("qpaygo: callback request missing qpay_payment_id/payment_id")
	}
	return q.GetPayment(ctx, id)
}
