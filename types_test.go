package qpaygo

import (
	"encoding/json"
	"math"
	"testing"
)

func TestNumberUnmarshalJSON(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		want    Number
		wantErr bool
	}{
		{name: "bare number", input: `100.5`, want: 100.5},
		{name: "quoted string", input: `"100.50"`, want: 100.5},
		{name: "integer", input: `2000`, want: 2000},
		{name: "null", input: `null`, want: 0},
		{name: "empty string", input: `""`, want: 0},
		{name: "invalid string", input: `"not-a-number"`, wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var n Number
			err := json.Unmarshal([]byte(tc.input), &n)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil (n=%v)", n)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if n != tc.want {
				t.Fatalf("got %v, want %v", n, tc.want)
			}
		})
	}
}

func TestNumberMarshalJSON(t *testing.T) {
	n := Number(100.5)
	data, err := json.Marshal(n)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != "100.5" {
		t.Fatalf("got %s, want 100.5", data)
	}
}

func TestNumberRoundTrip(t *testing.T) {
	type wrapper struct {
		Amount Number `json:"amount"`
	}

	var w wrapper
	if err := json.Unmarshal([]byte(`{"amount":"250.00"}`), &w); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if w.Amount.Float64() != 250 {
		t.Fatalf("got %v, want 250", w.Amount.Float64())
	}

	data, err := json.Marshal(w)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(data) != `{"amount":250}` {
		t.Fatalf("got %s, want {\"amount\":250}", data)
	}
}

func TestNumberRejectsNonFiniteOnMarshal(t *testing.T) {
	for _, n := range []Number{Number(math.NaN()), Number(math.Inf(1)), Number(math.Inf(-1))} {
		if _, err := json.Marshal(n); err == nil {
			t.Fatalf("expected error marshaling non-finite Number %v, got nil", n)
		}
	}
}

func TestNumberRejectsNonFiniteOnUnmarshal(t *testing.T) {
	for _, in := range []string{`"NaN"`, `"+Inf"`, `"-Inf"`, `"Inf"`, `NaN`, `+Inf`} {
		var n Number
		err := json.Unmarshal([]byte(in), &n)
		if err == nil {
			t.Fatalf("expected error unmarshaling %q, got %v", in, n)
		}
	}
}

func TestNumberPoisonedValueCannotEnterStruct(t *testing.T) {
	type wrapper struct {
		Amount Number `json:"amount"`
	}
	var w wrapper
	if err := json.Unmarshal([]byte(`{"amount":"NaN"}`), &w); err == nil {
		t.Fatal("expected unmarshal of poisoned NaN to fail")
	}
}
