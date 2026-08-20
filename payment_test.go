package qpaygo

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
)

func TestGetPaymentP2P(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v2/payment/pay-1", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"payment_id": "pay-1",
			"payment_status": "PAID",
			"payment_fee": "1.00",
			"payment_amount": "100.00",
			"payment_currency": "MNT",
			"payment_date": "2022-03-11T05:57:47.336Z",
			"object_type": "INVOICE",
			"object_id": "inv-1",
			"transaction_type": "P2P",
			"p2p_transactions": [
				{"transaction_bank_code":"050000","account_bank_code":"050000","account_bank_name":"Khan","account_number":"123","status":"PAID","amount":100,"currency":"MNT","settlement_status":"DONE"}
			]
		}`))
	})
	client, _ := newMockClient(t, mux)

	resp, err := client.GetPayment(context.Background(), "pay-1")
	if err != nil {
		t.Fatalf("GetPayment: %v", err)
	}
	if resp.TransportType != TransportP2P {
		t.Fatalf("got transport type %q", resp.TransportType)
	}
	if resp.PaymentFee.Float64() != 1.0 {
		t.Fatalf("got fee %v", resp.PaymentFee)
	}
	if len(resp.P2PTransactions) != 1 {
		t.Fatalf("got p2p transactions %+v", resp.P2PTransactions)
	}
}

func TestGetPaymentCard(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v2/payment/pay-2", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"payment_id": "pay-2",
			"payment_status": "PAID",
			"payment_fee": 2.5,
			"payment_amount": 200,
			"transaction_type": "CARD",
			"card_transactions": [
				{"card_merchant_code":"m1","card_terminal_code":"t1","card_number":"411111******1111","card_type":"VISA","is_cross_border":false,"transaction_amount":"200.00","transaction_currency":"MNT","transaction_date":"2022-03-11T05:57:47.336Z","transaction_status":"APPROVED","settlement_status":"DONE","settlement_status_date":"2022-03-11T05:57:47.336Z"}
			]
		}`))
	})
	client, _ := newMockClient(t, mux)

	resp, err := client.GetPayment(context.Background(), "pay-2")
	if err != nil {
		t.Fatalf("GetPayment: %v", err)
	}
	if resp.TransportType != TransportCard {
		t.Fatalf("got transport type %q", resp.TransportType)
	}
	if len(resp.CardTransactions) != 1 || resp.CardTransactions[0].CardType != "VISA" {
		t.Fatalf("got card transactions %+v", resp.CardTransactions)
	}
}

func TestCheckPaymentNoneYet(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v2/payment/check", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"count":0,"paid_amount":0,"rows":[]}`))
	})
	client, _ := newMockClient(t, mux)

	resp, err := client.CheckPayment(context.Background(), CheckPaymentRequest{
		ObjectType: ObjectTypeInvoice,
		ObjectID:   "inv-1",
		Offset:     Offset{PageNumber: 1, PageLimit: 100},
	})
	if err != nil {
		t.Fatalf("CheckPayment: %v", err)
	}
	if resp.Count != 0 || len(resp.Rows) != 0 {
		t.Fatalf("got resp %+v", resp)
	}
}

func TestCheckPaymentPaidUsesPaymentTypeAndTrxFee(t *testing.T) {
	mux := http.NewServeMux()
	var gotBody CheckPaymentRequest
	mux.HandleFunc("/v2/payment/check", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"count": 1,
			"paid_amount": 100,
			"rows": [
				{"payment_id":"pay-1","payment_status":"PAID","payment_amount":100,"trx_fee":"1.00","payment_currency":"MNT","payment_type":"CARD"}
			]
		}`))
	})
	client, _ := newMockClient(t, mux)

	resp, err := client.CheckPayment(context.Background(), CheckPaymentRequest{
		ObjectType: ObjectTypeQR,
		ObjectID:   "qr-1",
	})
	if err != nil {
		t.Fatalf("CheckPayment: %v", err)
	}
	if gotBody.ObjectType != ObjectTypeQR {
		t.Fatalf("got object type %q", gotBody.ObjectType)
	}
	if len(resp.Rows) != 1 {
		t.Fatalf("got rows %+v", resp.Rows)
	}
	row := resp.Rows[0]
	if row.TransportType != TransportCard {
		t.Fatalf("got transport type %q (from payment_type key)", row.TransportType)
	}
	if row.TrxFee.Float64() != 1.0 {
		t.Fatalf("got trx_fee %v", row.TrxFee)
	}
}

func TestCancelPaymentCardOnlyRestriction(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v2/payment/cancel/pay-p2p", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusConflict, map[string]string{
			"error": "PAYMENT_SETTLED", "message": "PAYMENT_SETTLED",
		})
	})
	client, _ := newMockClient(t, mux)

	err := client.CancelPayment(context.Background(), "pay-p2p", CancelPaymentRequest{Note: "test"})
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Code != ErrPaymentSettled {
		t.Fatalf("got err %v", err)
	}
}

func TestCancelPaymentCardSuccess(t *testing.T) {
	mux := http.NewServeMux()
	var gotNote string
	mux.HandleFunc("/v2/payment/cancel/pay-card", func(w http.ResponseWriter, r *http.Request) {
		var body CancelPaymentRequest
		_ = json.NewDecoder(r.Body).Decode(&body)
		gotNote = body.Note
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	})
	client, _ := newMockClient(t, mux)

	if err := client.CancelPayment(context.Background(), "pay-card", CancelPaymentRequest{Note: "customer request"}); err != nil {
		t.Fatalf("CancelPayment: %v", err)
	}
	if gotNote != "customer request" {
		t.Fatalf("got note %q", gotNote)
	}
}

func TestRefundPaymentSuccess(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v2/payment/refund/pay-card", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	})
	client, _ := newMockClient(t, mux)

	if err := client.RefundPayment(context.Background(), "pay-card", RefundPaymentRequest{Note: "refund"}); err != nil {
		t.Fatalf("RefundPayment: %v", err)
	}
}

func TestListPaymentsUsesPaidBy(t *testing.T) {
	mux := http.NewServeMux()
	var gotBody ListPaymentsRequest
	mux.HandleFunc("/v2/payment/list", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"count": 1,
			"rows": [
				{"payment_id":"p1","payment_date":"2022-03-11 05:57:47","payment_status":"PAID","payment_fee":1,"payment_amount":100,"payment_currency":"MNT","paid_by":"P2P","object_type":"INVOICE","object_id":"inv-1"}
			]
		}`))
	})
	client, _ := newMockClient(t, mux)

	resp, err := client.ListPayments(context.Background(), ListPaymentsRequest{
		ObjectType: ObjectTypeMerchant,
		ObjectID:   "merchant-1",
		StartDate:  FormatQPayTime(mustParseTime(t, "2022-03-01 00:00:00")),
		EndDate:    FormatQPayTime(mustParseTime(t, "2022-03-31 23:59:59")),
	})
	if err != nil {
		t.Fatalf("ListPayments: %v", err)
	}
	if gotBody.StartDate != "2022-03-01 00:00:00" {
		t.Fatalf("got start date %q", gotBody.StartDate)
	}
	if len(resp.Rows) != 1 || resp.Rows[0].TransportType != TransportP2P {
		t.Fatalf("got rows %+v", resp.Rows)
	}
}
