package pesabridge

import (
	"context"
	"time"

	"github.com/dev-bramwel/pesabridge/mpesa"
)

// Re-export common configuration and types for ergonomics.
type Environment = mpesa.Environment

const (
	EnvSandbox    = mpesa.EnvSandbox
	EnvProduction = mpesa.EnvProduction
)

// Config represents high-level configuration.
type Config struct {
	Environment       Environment
	ConsumerKey       string
	ConsumerSecret    string
	BusinessShortCode string
	PassKey           string
	CallBackURL       string
	HTTPTimeout       time.Duration
}

// PaymentRequest represents an application's intent to initiate a payment.
type PaymentRequest struct {
	Amount           int64  // Amount in KES (whole shillings)
	PhoneNumber      string // Safaricom phone number (e.g., "0712345678", "254712345678")
	AccountReference string // Account / Bill reference (e.g. "INV-1001")
	Description      string // Purpose / Description of the transaction
}

// PaymentResponse represents the result of initiating a payment request.
type PaymentResponse struct {
	MerchantRequestID   string
	CheckoutRequestID   string
	ResponseCode        string
	ResponseDescription string
	CustomerMessage     string
}

// Client is the primary interface for PesaBridge consumers.
type Client struct {
	config      *Config
	mpesaClient *mpesa.Client
}

// NewClient initializes a new PesaBridge client with the provided configuration.
func NewClient(cfg *Config) (*Client, error) {
	if cfg == nil {
		return nil, &Error{Kind: ErrorKindValidation, Message: "config cannot be nil"}
	}

	mpesaCfg := &mpesa.Config{
		Environment:       cfg.Environment,
		ConsumerKey:       cfg.ConsumerKey,
		ConsumerSecret:    cfg.ConsumerSecret,
		BusinessShortCode: cfg.BusinessShortCode,
		PassKey:           cfg.PassKey,
		CallBackURL:       cfg.CallBackURL,
		HTTPTimeout:       cfg.HTTPTimeout,
	}

	if err := mpesaCfg.Validate(); err != nil {
		return nil, &Error{Kind: ErrorKindValidation, Message: err.Error(), Err: err}
	}

	mpesaCli := mpesa.NewClient(mpesaCfg)
	return &Client{
		config:      cfg,
		mpesaClient: mpesaCli,
	}, nil
}

// LoadConfigFromEnv reads configuration from environment variables.
func LoadConfigFromEnv() (*Config, error) {
	mCfg, err := mpesa.LoadConfigFromEnv()
	if err != nil {
		return nil, err
	}
	return &Config{
		Environment:       mCfg.Environment,
		ConsumerKey:       mCfg.ConsumerKey,
		ConsumerSecret:    mCfg.ConsumerSecret,
		BusinessShortCode: mCfg.BusinessShortCode,
		PassKey:           mCfg.PassKey,
		CallBackURL:       mCfg.CallBackURL,
		HTTPTimeout:       mCfg.HTTPTimeout,
	}, nil
}

// InitiateSTK triggers an STK Push prompt on the customer's phone.
func (c *Client) InitiateSTK(ctx context.Context, req PaymentRequest) (*PaymentResponse, error) {
	resp, err := c.mpesaClient.STK.Initiate(ctx, mpesa.STKPushRequest{
		Amount:           req.Amount,
		PhoneNumber:      req.PhoneNumber,
		AccountReference: req.AccountReference,
		Description:      req.Description,
	})
	if err != nil {
		return nil, err
	}

	return (*PaymentResponse)(resp), nil
}
