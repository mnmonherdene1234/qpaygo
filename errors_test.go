package qpaygo

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestParseAPIErrorNestedObjectShape(t *testing.T) {
	// Live QPay validation failures return nested objects instead of the
	// documented {"error":"CODE","message":"..."} string shape.
	rec := httptest.NewRecorder()
	rec.Code = http.StatusBadRequest
	rec.Body.WriteString(`{"error":{"invoice_code":{"type":"INVALID","message":"Invalid!"}},"message":{"invoice_code":{"type":"INVALID","message":"Invalid!"}}}`)

	apiErr := parseAPIError(rec.Result())
	if apiErr.Code != "" {
		t.Fatalf("expected empty code for object-shaped error, got %q", apiErr.Code)
	}
	for _, want := range []string{"invoice_code", "INVALID", "Invalid!"} {
		if !strings.Contains(apiErr.Message, want) {
			t.Fatalf("message %q must contain %q", apiErr.Message, want)
		}
	}
	if !strings.Contains(apiErr.Error(), "INVALID") {
		t.Fatalf("Error() %q must surface the flattened message", apiErr.Error())
	}
}

func TestParseAPIErrorNestedMinNumberShape(t *testing.T) {
	// The exact live shape for offset validation failures.
	rec := httptest.NewRecorder()
	rec.Code = http.StatusBadRequest
	rec.Body.WriteString(`{"error":{"page_limit":{"type":"MIN_NUMBER","message":"Number min value [1]!"}},"message":{"page_limit":{"type":"MIN_NUMBER","message":"Number min value [1]!"}}}`)

	apiErr := parseAPIError(rec.Result())
	if !strings.Contains(apiErr.Message, "MIN_NUMBER") || !strings.Contains(apiErr.Message, "page_limit") {
		t.Fatalf("got message %q, want it to mention page_limit and MIN_NUMBER", apiErr.Message)
	}
}

func TestParseAPIErrorNoCredentialsProductionSpelling(t *testing.T) {
	rec := httptest.NewRecorder()
	rec.Code = http.StatusUnauthorized
	rec.Body.WriteString(`{"error":"NO_CREDENTIALS","message":"Хандах эрхгүй байна. Нэвтрэнэ үү."}`)

	apiErr := parseAPIError(rec.Result())
	if apiErr.Code != ErrNoCredentialsProduction {
		t.Fatalf("got code %q, want %q", apiErr.Code, ErrNoCredentialsProduction)
	}
}

func TestParseAPIErrorSandboxSpellingStillRecognized(t *testing.T) {
	rec := httptest.NewRecorder()
	rec.Code = http.StatusUnauthorized
	rec.Body.WriteString(`{"error":"NO_CREDENDIALS","message":"NO_CREDENDIALS"}`)

	apiErr := parseAPIError(rec.Result())
	if apiErr.Code != ErrNoCredentials {
		t.Fatalf("got code %q, want %q", apiErr.Code, ErrNoCredentials)
	}
}

func TestDecodeJSONResponseNoContent(t *testing.T) {
	rec := httptest.NewRecorder()
	rec.Code = http.StatusNoContent

	var dst struct{ Any string }
	err := decodeJSONResponse(rec.Result(), &dst)
	if err == nil {
		t.Fatal("expected a clear error for 204, got nil")
	}
	if !strings.Contains(err.Error(), "204") {
		t.Fatalf("error %q must mention 204", err)
	}
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		t.Fatalf("204 must not be reported as an APIError, got %+v", apiErr)
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
