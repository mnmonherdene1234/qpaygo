package qpaygo

import (
	"net/http"
	"testing"
)

func TestNewQpayClient(t *testing.T) {

	client := NewQpayClient(Username, Password, InvoiceCode)

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
	client := &QpayClient{
		Username:    Username,
		Password:    Password,
		InvoiceCode: InvoiceCode,
		Client:      &http.Client{},
	}

	err := client.AuthToken()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if client.TokenResponse == nil {
		t.Fatalf("expected token response, got nil")
	}
}
