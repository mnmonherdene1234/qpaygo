package qpaygo

import (
	"log"
	"testing"
)

func TestNewQpayClient(t *testing.T) {

	client, err := NewQPayClient(Username, Password, InvoiceCode)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if client.Username != Username {
		t.Errorf("expected Username %s, got %s", Username, client.Username)
	}
	if client.Password != Password {
		t.Errorf("expected Password %s, got %s", Password, client.Password)
	}
	if client.InvoiceCode != InvoiceCode {
		t.Errorf("expected InvoiceCode %s, got %s", InvoiceCode, client.InvoiceCode)
	}
	if client.Host != DefaultHost {
		t.Errorf("expected host %s, got %s", DefaultHost, client.Host)
	}
}

func TestAuthToken(t *testing.T) {
	client, err := NewQPayClient(Username, Password, InvoiceCode)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	err = client.AuthToken()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if client.TokenResponse == nil {
		t.Fatalf("expected token response, got nil")
	}

	log.Println(client.TokenResponse.AccessToken)
}

func TestCreateAmountInvoice(t *testing.T) {
	client, err := NewQPayClient(Username, Password, InvoiceCode)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	err = client.AuthToken()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	response, err := client.CreateAmountInvoice(
		"123456",
		"123456",
		"Тестийн нэхэмжлэлийн утга",
		200,
		"https://example.com/callback",
	)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if response == nil {
		t.Fatalf("expected response, got nil")
	}

	log.Println(response.InvoiceID)
}

func TestIsTokenExpired(t *testing.T) {
	client, err := NewQPayClient(Username, Password, InvoiceCode)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if client.IsTokenExpired() {
		t.Fatalf("expected token to be valid, but it is expired")
	}
}

func TestCheckTokenAndRefresh(t *testing.T) {
	client, err := NewQPayClient(Username, Password, InvoiceCode)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	err = client.CheckTokenAndRefresh()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if client.TokenResponse == nil {
		t.Fatalf("expected token response, got nil")
	}
}

func TestGetInvoice(t *testing.T) {
	client, err := NewQPayClient(Username, Password, InvoiceCode)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	err = client.AuthToken()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	response, err := client.GetInvoice("8a712739-ac1e-4739-8bd3-3826a5746107")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if response == nil {
		t.Fatalf("expected response, got nil")
	}

	log.Println(response.TotalAmount)
}
