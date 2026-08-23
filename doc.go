// Package qpaygo is a client for QPay's (qpay.mn) v2 merchant API: invoice
// creation and lifecycle, payment lifecycle, and eBarimt tax receipts.
//
// # Two operational rules QPay enforces
//
// QPay's documentation states these rules explicitly; violating them can
// result in blocked API access:
//
//  1. Never poll GetPayment/CheckPayment on a cron/scheduled basis. Only call
//     them synchronously in direct response to a callback hit on your
//     callback_url handler. See ExtractPaymentID, WriteCallbackAck, and
//     VerifyCallback for the intended request flow.
//
//  2. A callback request is not proof of payment by itself: QPay's v2 API has
//     no signature/HMAC verification of callback requests. Always
//     independently confirm payment status via GetPayment or CheckPayment
//     using your own bearer token before treating a payment as settled.
//     VerifyCallback does this in one call.
//
// # Raw requests
//
// Request sends an authenticated request to any endpoint path and returns
// the raw *http.Response. The caller owns the response and must close its
// body (response.Body.Close()).
package qpaygo
