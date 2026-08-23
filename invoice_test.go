package qpaygo

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
)

const invoiceResponseFixture = `{
	"invoice_id": "d50f49f2-0000-0000-0000-000000000000",
	"qr_text": "0002010102...6304C66D",
	"qr_image": "aGVsbG8=",
	"qPay_shortUrl": "https://s.qpay.mn/z1lKnIO5T",
	"urls": [
		{"name":"Khan bank","description":"Хаан банк","logo":"https://qpay.mn/q/logo/khanbank.png","link":"khanbank://q?qPay_QRcode=abc"}
	]
}`

func TestCreateSimpleInvoice(t *testing.T) {
	mux := http.NewServeMux()
	var gotBody CreateSimpleInvoiceRequest
	mux.HandleFunc("/v2/invoice", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(invoiceResponseFixture))
	})
	client, _ := newMockClient(t, mux)

	resp, err := client.CreateSimpleInvoice(context.Background(), CreateSimpleInvoiceRequest{
		SenderInvoiceNo:     "inv-1",
		InvoiceReceiverCode: "receiver-1",
		InvoiceDescription:  "desc",
		Amount:              2000,
		CallbackURL:         "https://example.com/callback",
	})
	if err != nil {
		t.Fatalf("CreateSimpleInvoice: %v", err)
	}

	if gotBody.InvoiceCode != "INV" {
		t.Fatalf("expected invoice_code to default to client's InvoiceCode, got %q", gotBody.InvoiceCode)
	}
	if gotBody.Amount != 2000 {
		t.Fatalf("got amount %v", gotBody.Amount)
	}

	if resp.InvoiceID != "d50f49f2-0000-0000-0000-000000000000" {
		t.Fatalf("got invoice id %q", resp.InvoiceID)
	}
	if len(resp.URLs) != 1 || resp.URLs[0].Name != "Khan bank" {
		t.Fatalf("got urls %+v", resp.URLs)
	}
}

func TestCreateInvoiceFullForm(t *testing.T) {
	mux := http.NewServeMux()
	var raw map[string]any
	mux.HandleFunc("/v2/invoice", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&raw)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(invoiceResponseFixture))
	})
	client, _ := newMockClient(t, mux)

	req := CreateInvoiceRequest{
		SenderInvoiceNo:     "inv-2",
		InvoiceReceiverCode: "receiver-2",
		InvoiceDescription:  "full form",
		CallbackURL:         "https://example.com/callback",
		Lines: []Line{
			{
				LineDescription: "item 1",
				LineQuantity:    1,
				LineUnitPrice:   1000,
				Discounts: []Discount{
					{Description: "promo", Amount: 100},
				},
				Surcharges: []Surcharge{
					{Description: "service", Amount: 50},
				},
				Taxes: []LineTax{
					{TaxCode: "VAT", Description: "vat", Amount: 90},
				},
			},
		},
		Transactions: []InvoiceTransaction{
			{
				Description: "split",
				Amount:      500,
				Accounts: []SettlementAccount{
					{AccountBankCode: "050000", AccountNumber: "123", AccountName: "test", AccountCurrency: "MNT", IsDefault: true},
				},
			},
		},
	}

	if _, err := client.CreateInvoice(context.Background(), req); err != nil {
		t.Fatalf("CreateInvoice: %v", err)
	}

	lines, ok := raw["lines"].([]any)
	if !ok || len(lines) != 1 {
		t.Fatalf("expected 1 line, got %+v", raw["lines"])
	}
	line := lines[0].(map[string]any)
	if _, ok := line["discounts"]; !ok {
		t.Fatalf("expected 'discounts' key (not 'disctounts'), got keys %+v", line)
	}

	txs, ok := raw["transactions"].([]any)
	if !ok || len(txs) != 1 {
		t.Fatalf("expected 1 transaction, got %+v", raw["transactions"])
	}
}

func TestGetInvoiceDecodesMixedAmountTypes(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v2/invoice/inv-123", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"invoice_id": "inv-123",
			"invoice_status": "OPEN",
			"minimum_amount": 10.5,
			"maximum_amount": "500.00",
			"total_amount": "100",
			"gross_amount": 100,
			"lines": [
				{"line_description":"x","line_quantity":"2","line_unit_price":"50.00","discounts":[],"surcharges":[],"taxes":[]}
			],
			"transactions": [],
			"inputs": []
		}`))
	})
	client, _ := newMockClient(t, mux)

	resp, err := client.GetInvoice(context.Background(), "inv-123")
	if err != nil {
		t.Fatalf("GetInvoice: %v", err)
	}

	if resp.MinimumAmount.Float64() != 10.5 {
		t.Fatalf("got MinimumAmount %v", resp.MinimumAmount)
	}
	if resp.MaximumAmount.Float64() != 500 {
		t.Fatalf("got MaximumAmount %v (should parse quoted string)", resp.MaximumAmount)
	}
	if len(resp.Lines) != 1 || resp.Lines[0].LineQuantity.Float64() != 2 {
		t.Fatalf("got lines %+v", resp.Lines)
	}
}

func TestCancelInvoiceSuccess(t *testing.T) {
	mux := http.NewServeMux()
	var gotMethod string
	mux.HandleFunc("/v2/invoice/inv-to-cancel", func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	})
	client, _ := newMockClient(t, mux)

	if err := client.CancelInvoice(context.Background(), "inv-to-cancel"); err != nil {
		t.Fatalf("CancelInvoice: %v", err)
	}
	if gotMethod != http.MethodDelete {
		t.Fatalf("got method %q, want DELETE", gotMethod)
	}
}

func TestCancelInvoiceAlreadyCanceled(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v2/invoice/inv-already", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusConflict, map[string]string{
			"error": "INVOICE_ALREADY_CANCELED", "message": "already canceled",
		})
	})
	client, _ := newMockClient(t, mux)

	err := client.CancelInvoice(context.Background(), "inv-already")
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Code != ErrInvoiceAlreadyCanceled {
		t.Fatalf("got err %v", err)
	}
}

func TestInvoiceResponseDecodesQPayDeeplinkAlias(t *testing.T) {
	// QPay's official field table names the deeplink array "qPay_deeplink"
	// while the live API sends "urls". Both must populate URLs.
	mux := http.NewServeMux()
	mux.HandleFunc("/v2/invoice", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"invoice_id": "inv-dl",
			"qr_text": "0002010102...",
			"qr_image": "aGVsbG8=",
			"qPay_shortUrl": "https://s.qpay.mn/z1lKnIO5T",
			"qPay_deeplink": [
				{"name":"Khan bank","description":"Хаан банк","logo":"https://qpay.mn/q/logo/khanbank.png","link":"khanbank://q?qPay_QRcode=abc"}
			]
		}`))
	})
	client, _ := newMockClient(t, mux)

	resp, err := client.CreateSimpleInvoice(context.Background(), CreateSimpleInvoiceRequest{
		SenderInvoiceNo:     "inv-dl",
		InvoiceReceiverCode: "r",
		InvoiceDescription:  "d",
		Amount:              100,
		CallbackURL:         "https://example.com/cb",
	})
	if err != nil {
		t.Fatalf("CreateSimpleInvoice: %v", err)
	}
	if len(resp.URLs) != 1 || resp.URLs[0].Name != "Khan bank" {
		t.Fatalf("qPay_deeplink must populate URLs, got %+v", resp.URLs)
	}
}

func TestInvoiceResponsePrefersURLsOverDeeplink(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v2/invoice", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"invoice_id": "inv-both",
			"urls": [{"name":"from urls","description":"","logo":"","link":""}],
			"qPay_deeplink": [{"name":"from deeplink","description":"","logo":"","link":""}]
		}`))
	})
	client, _ := newMockClient(t, mux)

	resp, err := client.CreateSimpleInvoice(context.Background(), CreateSimpleInvoiceRequest{
		SenderInvoiceNo:     "inv-both",
		InvoiceReceiverCode: "r",
		InvoiceDescription:  "d",
		Amount:              100,
		CallbackURL:         "https://example.com/cb",
	})
	if err != nil {
		t.Fatalf("CreateSimpleInvoice: %v", err)
	}
	if len(resp.URLs) != 1 || resp.URLs[0].Name != "from urls" {
		t.Fatalf("urls must win over qPay_deeplink, got %+v", resp.URLs)
	}
}

func TestGetInvoiceToleratesObjectSenderData(t *testing.T) {
	// The create request models sender_branch_data as an object; if the GET
	// response ever echoes it as an object instead of a string, the decode
	// must still succeed.
	mux := http.NewServeMux()
	mux.HandleFunc("/v2/invoice/inv-obj", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"invoice_id": "inv-obj",
			"invoice_status": "OPEN",
			"sender_branch_data": {"register":"UZ96021105","name":"Branch 1","email":"b@example.com","phone":"88614450"},
			"sender_staff_data": {"name":"Staff","email":"s@example.com","phone":"99001122"},
			"sender_terminal_data": {"name": null},
			"lines": []
		}`))
	})
	client, _ := newMockClient(t, mux)

	resp, err := client.GetInvoice(context.Background(), "inv-obj")
	if err != nil {
		t.Fatalf("GetInvoice: %v", err)
	}
	if resp.SenderBranchData == "" || resp.SenderStaffData == "" || resp.SenderTerminalData == "" {
		t.Fatalf("object sender data must be preserved, got branch=%q staff=%q terminal=%q",
			resp.SenderBranchData, resp.SenderStaffData, resp.SenderTerminalData)
	}
	if !strings.Contains(string(resp.SenderBranchData), "Branch 1") {
		t.Fatalf("sender_branch_data must carry the object JSON, got %q", resp.SenderBranchData)
	}
}

func TestGetInvoiceStringSenderDataStillWorks(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v2/invoice/inv-str", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"invoice_id": "inv-str",
			"invoice_status": "CANCELED",
			"sender_branch_data": "",
			"sender_staff_data": "",
			"sender_terminal_data": "",
			"lines": []
		}`))
	})
	client, _ := newMockClient(t, mux)

	resp, err := client.GetInvoice(context.Background(), "inv-str")
	if err != nil {
		t.Fatalf("GetInvoice: %v", err)
	}
	if resp.InvoiceStatus != "CANCELED" {
		t.Fatalf("got status %q, want CANCELED", resp.InvoiceStatus)
	}
	if resp.SenderBranchData != "" {
		t.Fatalf("empty string sender data must stay empty, got %q", resp.SenderBranchData)
	}
}

func TestCreateInvoiceSendsBooleanSwitchesExplicitly(t *testing.T) {
	mux := http.NewServeMux()
	var raw map[string]any
	mux.HandleFunc("/v2/invoice", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&raw)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(invoiceResponseFixture))
	})
	client, _ := newMockClient(t, mux)

	if _, err := client.CreateInvoice(context.Background(), CreateInvoiceRequest{
		SenderInvoiceNo:     "inv-bools",
		InvoiceReceiverCode: "r",
		InvoiceDescription:  "d",
		CallbackURL:         "https://example.com/cb",
		Amount:              100,
	}); err != nil {
		t.Fatalf("CreateInvoice: %v", err)
	}
	for _, key := range []string{"enable_expiry", "calculate_vat", "allow_partial", "allow_exceed", "allow_subscribe"} {
		if _, ok := raw[key]; !ok {
			t.Fatalf("key %q must be sent explicitly (false), got body %v", key, raw)
		}
	}
}
