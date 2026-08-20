package qpaygo

import (
	"context"
	"net/http"
	"testing"
)

func TestNewQPayClientLazyDoesNotCallNetwork(t *testing.T) {
	client := NewQPayClientLazy("u", "p", "INV")
	if client.Token() != nil {
		t.Fatal("expected no token before AuthToken is called")
	}
	if client.InvoiceCode() != "INV" {
		t.Fatalf("got invoice code %q", client.InvoiceCode())
	}
	if client.Host() != DefaultHost {
		t.Fatalf("got host %q, want %q", client.Host(), DefaultHost)
	}
}

func TestNewQPayClientFetchesToken(t *testing.T) {
	mux := http.NewServeMux()
	client, _ := newMockClient(t, mux)

	tok := client.Token()
	if tok == nil || tok.AccessToken != "test-access" {
		t.Fatalf("got token %+v", tok)
	}
}

func TestWithHostOption(t *testing.T) {
	client := NewQPayClientLazy("u", "p", "INV", WithHost(SandboxHost))
	if client.Host() != SandboxHost {
		t.Fatalf("got host %q, want %q", client.Host(), SandboxHost)
	}
}

func TestRequestSetsBearerHeader(t *testing.T) {
	mux := http.NewServeMux()
	var gotAuth string
	mux.HandleFunc("/v2/ping", func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		writeJSON(w, http.StatusOK, map[string]string{"ok": "true"})
	})
	client, _ := newMockClient(t, mux)

	resp, err := client.Request(context.Background(), http.MethodGet, "/v2/ping", nil)
	if err != nil {
		t.Fatalf("Request: %v", err)
	}
	resp.Body.Close()

	if gotAuth != "Bearer test-access" {
		t.Fatalf("got Authorization header %q", gotAuth)
	}
}
