//go:build integration

// Package qpaygo integration tests hit QPay's real API. They are gated
// behind the "integration" build tag and the QPAY_USERNAME/QPAY_PASSWORD/
// QPAY_INVOICE_CODE environment variables so `go test ./...` never needs
// credentials or network access.
//
// Run against sandbox (default host if QPAY_HOST is unset):
//
//	QPAY_USERNAME=... QPAY_PASSWORD=... QPAY_INVOICE_CODE=... go test -tags=integration ./...
//
// Run against production, explicitly:
//
//	QPAY_HOST=https://merchant.qpay.mn QPAY_USERNAME=... QPAY_PASSWORD=... QPAY_INVOICE_CODE=... go test -tags=integration ./...
//
// Per QPay's own documentation, do not run these on a cron/schedule — they
// are for manual, occasional verification only.
package qpaygo

import (
	"context"
	"os"
	"testing"
)

func integrationClient(t *testing.T) *QPayClient {
	t.Helper()

	username := os.Getenv("QPAY_USERNAME")
	password := os.Getenv("QPAY_PASSWORD")
	invoiceCode := os.Getenv("QPAY_INVOICE_CODE")
	if username == "" || password == "" || invoiceCode == "" {
		t.Skip("QPAY_USERNAME/QPAY_PASSWORD/QPAY_INVOICE_CODE not set, skipping integration test")
	}

	host := os.Getenv("QPAY_HOST")
	if host == "" {
		host = SandboxHost
	}

	client, err := NewQPayClient(context.Background(), username, password, invoiceCode, WithHost(host))
	if err != nil {
		t.Fatalf("NewQPayClient: %v", err)
	}
	return client
}

func TestIntegration_AuthToken(t *testing.T) {
	client := integrationClient(t)

	tok := client.Token()
	if tok == nil || tok.AccessToken == "" {
		t.Fatalf("expected a populated access token, got %+v", tok)
	}
}

func TestIntegration_CreateAndCancelInvoice(t *testing.T) {
	client := integrationClient(t)
	ctx := context.Background()

	resp, err := client.CreateSimpleInvoice(ctx, CreateSimpleInvoiceRequest{
		SenderInvoiceNo:     "qpaygo-integration-test",
		InvoiceReceiverCode: "terminal",
		InvoiceDescription:  "qpaygo integration test invoice",
		Amount:              100,
		CallbackURL:         "https://example.com/qpaygo-integration-callback",
	})
	if err != nil {
		t.Fatalf("CreateSimpleInvoice: %v", err)
	}
	if resp.InvoiceID == "" {
		t.Fatal("expected a non-empty invoice id")
	}

	t.Cleanup(func() {
		if err := client.CancelInvoice(context.Background(), resp.InvoiceID); err != nil {
			t.Logf("cleanup: CancelInvoice(%s): %v", resp.InvoiceID, err)
		}
	})

	got, err := client.GetInvoice(ctx, resp.InvoiceID)
	if err != nil {
		t.Fatalf("GetInvoice: %v", err)
	}
	if got.InvoiceID != resp.InvoiceID {
		t.Fatalf("got invoice id %q, want %q", got.InvoiceID, resp.InvoiceID)
	}
}
