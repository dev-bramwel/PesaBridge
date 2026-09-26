package mpesa

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSanitizePhoneNumber(t *testing.T) {
	tests := []struct {
		input    string
		expected string
		hasErr   bool
	}{
		{"0712345678", "254712345678", false},
		{"0112345678", "254112345678", false},
		{"+254712345678", "254712345678", false},
		{"254712345678", "254712345678", false},
		{"12345", "", true},
		{"0712345", "", true},
		{"254700000000000", "", true},
	}

	for _, tt := range tests {
		res, err := SanitizePhoneNumber(tt.input)
		if tt.hasErr && err == nil {
			t.Errorf("SanitizePhoneNumber(%q) expected error, got nil", tt.input)
		}
		if !tt.hasErr && (err != nil || res != tt.expected) {
			t.Errorf("SanitizePhoneNumber(%q) = %q, %v; expected %q, nil", tt.input, res, err, tt.expected)
		}
	}
}

func TestGeneratePassword(t *testing.T) {
	shortCode := "174379"
	passKey := "dummy_passkey"
	timestamp := "20160216165627"

	pwd := GeneratePassword(shortCode, passKey, timestamp)
	if pwd == "" {
		t.Fatal("expected non-empty base64 password")
	}
}

func TestSTKPushInitiation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth/v1/generate":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(authResponse{
				AccessToken: "test-token-123",
				ExpiresIn:   "3599",
			})
		case "/mpesa/stkpush/v1/processrequest":
			auth := r.Header.Get("Authorization")
			if auth != "Bearer test-token-123" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(STKPushRawResponse{
				MerchantRequestID:   "29115-34620561-1",
				CheckoutRequestID:   "ws_CO_191220191020363925",
				ResponseCode:        "0",
				ResponseDescription: "Success. Request accepted for processing",
				CustomerMessage:     "Success. Request accepted for processing",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	cfg := &Config{
		Environment:       EnvSandbox,
		ConsumerKey:       "dummy_key",
		ConsumerSecret:    "dummy_secret",
		BusinessShortCode: "174379",
		PassKey:           "dummy_passkey",
		CallBackURL:       "https://test.ngrok-free.app/callback",
		HTTPTimeout:       5 * time.Second,
		BaseURLOverride:   server.URL,
	}

	cli := NewClient(cfg)
	res, err := cli.STK.Initiate(context.Background(), STKPushRequest{
		Amount:           10,
		PhoneNumber:      "0712345678",
		AccountReference: "TEST-INV",
		Description:      "Test Payment",
	})

	if err != nil {
		t.Fatalf("unexpected error initiating STK: %v", err)
	}

	if res.ResponseCode != "0" {
		t.Errorf("expected ResponseCode 0, got %s", res.ResponseCode)
	}
	if res.CheckoutRequestID != "ws_CO_191220191020363925" {
		t.Errorf("expected CheckoutRequestID ws_CO_191220191020363925, got %s", res.CheckoutRequestID)
	}
}
