package qpaygo

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestExtractPaymentID(t *testing.T) {
	cases := []struct {
		name   string
		url    string
		wantID string
		wantOK bool
	}{
		{name: "qpay_payment_id", url: "/callback?qpay_payment_id=pay-1", wantID: "pay-1", wantOK: true},
		{name: "payment_id fallback", url: "/callback?payment_id=pay-2", wantID: "pay-2", wantOK: true},
		{name: "prefers qpay_payment_id", url: "/callback?qpay_payment_id=pay-1&payment_id=pay-2", wantID: "pay-1", wantOK: true},
		{name: "neither present", url: "/callback", wantID: "", wantOK: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.url, nil)
			id, ok := ExtractPaymentID(req)
			if id != tc.wantID || ok != tc.wantOK {
				t.Fatalf("got (%q, %v), want (%q, %v)", id, ok, tc.wantID, tc.wantOK)
			}
		})
	}
}

func TestWriteCallbackAck(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteCallbackAck(rec)

	if rec.Code != http.StatusOK {
		t.Fatalf("got status %d, want 200", rec.Code)
	}
	if rec.Body.String() != "SUCCESS" {
		t.Fatalf("got body %q, want %q", rec.Body.String(), "SUCCESS")
	}
}

func TestVerifyCallback(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v2/payment/pay-1", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"payment_id":"pay-1","payment_status":"PAID"}`))
	})
	client, _ := newMockClient(t, mux)

	req := httptest.NewRequest(http.MethodGet, "/callback?qpay_payment_id=pay-1", nil)
	resp, err := client.VerifyCallback(context.Background(), req)
	if err != nil {
		t.Fatalf("VerifyCallback: %v", err)
	}
	if resp.PaymentStatus != PaymentStatusPaid {
		t.Fatalf("got status %q", resp.PaymentStatus)
	}
}

func TestVerifyCallbackMissingID(t *testing.T) {
	client := NewQPayClientLazy("u", "p", "INV")
	req := httptest.NewRequest(http.MethodGet, "/callback", nil)

	if _, err := client.VerifyCallback(context.Background(), req); err == nil {
		t.Fatal("expected error for missing payment id")
	}
}
