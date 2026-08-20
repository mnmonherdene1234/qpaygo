package qpaygo

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParseAPIErrorDocumentedShape(t *testing.T) {
	rec := httptest.NewRecorder()
	rec.Code = http.StatusUnprocessableEntity
	rec.Body.WriteString(`{"error":"INVOICE_NOTFOUND","message":"Invoice not found"}`)

	resp := rec.Result()
	apiErr := parseAPIError(resp)

	if apiErr.Code != ErrInvoiceNotFound {
		t.Fatalf("got code %q, want %q", apiErr.Code, ErrInvoiceNotFound)
	}
	if apiErr.Message != "Invoice not found" {
		t.Fatalf("got message %q", apiErr.Message)
	}
	if apiErr.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("got status %d", apiErr.StatusCode)
	}
}

func TestParseAPIErrorPaymentSettled(t *testing.T) {
	rec := httptest.NewRecorder()
	rec.Code = http.StatusConflict
	rec.Body.WriteString(`{"error":"PAYMENT_SETTLED","message":"PAYMENT_SETTLED"}`)

	apiErr := parseAPIError(rec.Result())
	if apiErr.Code != ErrPaymentSettled {
		t.Fatalf("got code %q, want %q", apiErr.Code, ErrPaymentSettled)
	}
}

func TestParseAPIErrorNonJSONBody(t *testing.T) {
	rec := httptest.NewRecorder()
	rec.Code = http.StatusInternalServerError
	rec.Body.WriteString("internal server error")

	apiErr := parseAPIError(rec.Result())
	if apiErr.Code != "" {
		t.Fatalf("expected empty code, got %q", apiErr.Code)
	}
	if apiErr.Body != "internal server error" {
		t.Fatalf("got body %q", apiErr.Body)
	}
}

func TestDecodeJSONResponseErrorsAs(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/fail", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error":   "INVOICE_NOTFOUND",
			"message": "not found",
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/fail")
	if err != nil {
		t.Fatalf("http.Get: %v", err)
	}

	var dst struct{}
	err = decodeJSONResponse(resp, &dst)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T: %v", err, err)
	}
	if apiErr.Code != ErrInvoiceNotFound {
		t.Fatalf("got code %q", apiErr.Code)
	}
}

func TestDecodeEmptyResponseSuccess(t *testing.T) {
	rec := httptest.NewRecorder()
	rec.Code = http.StatusOK
	rec.Body.WriteString("{}")

	if err := decodeEmptyResponse(rec.Result()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
