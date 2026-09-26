package mpesa

import (
	"net/http"
)

// Client handles communication with Safaricom Daraja API.
type Client struct {
	Config       *Config
	HTTPClient   *http.Client
	TokenManager *TokenManager
	STK          *STKService
}

// NewClient creates an initialized Daraja M-Pesa client.
func NewClient(cfg *Config) *Client {
	timeout := cfg.HTTPTimeout
	if timeout <= 0 {
		timeout = DefaultHTTPTimeout
	}

	httpClient := &http.Client{
		Timeout: timeout,
	}

	tm := NewTokenManager(cfg.ConsumerKey, cfg.ConsumerSecret, cfg.BaseURL(), httpClient)

	client := &Client{
		Config:       cfg,
		HTTPClient:   httpClient,
		TokenManager: tm,
	}

	client.STK = NewSTKService(client)
	return client
}
