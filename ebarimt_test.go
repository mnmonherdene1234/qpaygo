package qpaygo

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
)

const ebarimtResponseFixture = `{
	"id": "eb-1",
	"ebarimt_by": "merchant",
	"ebarimt_receiver_type": "CITIZEN",
	"ebarimt_receiver": "99001122",
	"g_payment_id": "pay-1",
	"paid_by": "P2P",
	"object_type": "INVOICE",
	"object_id": "inv-1",
	"amount": "100.00",
	"vat_amount": 9.09,
	"city_tax_amount": 1,
	"barimt_status": "REGISTERED",
	"barimt_items": [
		{"id":"item-1","barimt_id":"eb-1","name":"item","unit_price":"50.00","quantity":2,"amount":100,"city_tax_amount":1,"vat_amount":9.09,"status":true}
	],
	"barimt_transactions": [],
	"barimt_histories": [
		{"id":"h1","barimt_id":"eb-1","barimt_status":"REGISTERED","status":true}
	]
}`

func TestCreateEbarimt(t *testing.T) {
	mux := http.NewServeMux()
	var gotBody CreateEbarimtRequest
	mux.HandleFunc("/v2/ebarimt/create", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(ebarimtResponseFixture))
	})
	client, _ := newMockClient(t, mux)

	resp, err := client.CreateEbarimt(context.Background(), CreateEbarimtRequest{
		PaymentID:           "pay-1",
		EbarimtReceiverType: EbarimtReceiverCitizen,
	})
	if err != nil {
		t.Fatalf("CreateEbarimt: %v", err)
	}
	if gotBody.EbarimtReceiverType != EbarimtReceiverCitizen {
		t.Fatalf("got receiver type %q", gotBody.EbarimtReceiverType)
	}
	if resp.BarimtStatus != "REGISTERED" {
		t.Fatalf("got status %q", resp.BarimtStatus)
	}
	if resp.PaidBy != TransportP2P {
		t.Fatalf("got paid_by %q", resp.PaidBy)
	}
	if len(resp.BarimtItems) != 1 || resp.BarimtItems[0].Quantity.Float64() != 2 {
		t.Fatalf("got items %+v", resp.BarimtItems)
	}
	if len(resp.BarimtHistories) != 1 {
		t.Fatalf("got histories %+v", resp.BarimtHistories)
	}
}

func TestGetEbarimt(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v2/ebarimt/eb-1", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(ebarimtResponseFixture))
	})
	client, _ := newMockClient(t, mux)

	resp, err := client.GetEbarimt(context.Background(), "eb-1")
	if err != nil {
		t.Fatalf("GetEbarimt: %v", err)
	}
	if resp.ID != "eb-1" {
		t.Fatalf("got id %q", resp.ID)
	}
}

func TestCancelEbarimtNotSupported(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v2/ebarimt/eb-1", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusForbidden, map[string]string{
			"error": "EBARIMT_CANCEL_NOTSUPPERDED", "message": "qPay service eBarimt unregister function not supported",
		})
	})
	client, _ := newMockClient(t, mux)

	err := client.CancelEbarimt(context.Background(), "eb-1")
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Code != ErrEbarimtCancelNotSupported {
		t.Fatalf("got err %v", err)
	}
}

func TestCancelEbarimtSuccess(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v2/ebarimt/eb-2", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	})
	client, _ := newMockClient(t, mux)

	if err := client.CancelEbarimt(context.Background(), "eb-2"); err != nil {
		t.Fatalf("CancelEbarimt: %v", err)
	}
}
