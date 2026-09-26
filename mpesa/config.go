package mpesa

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// Environment represents the target provider environment.
type Environment string

const (
	EnvSandbox    Environment = "sandbox"
	EnvProduction Environment = "production"
)

// DefaultHTTPTimeout is the default timeout for HTTP requests.
const DefaultHTTPTimeout = 30 * time.Second

// SandboxBaseURL is the Safaricom Daraja Sandbox gateway.
const SandboxBaseURL = "https://sandbox.safaricom.co.ke"

// ProductionBaseURL is the Safaricom Daraja Production gateway.
const ProductionBaseURL = "https://api.safaricom.co.ke"

// Config holds the configuration needed for Daraja communication.
type Config struct {
	Environment       Environment
	ConsumerKey       string
	ConsumerSecret    string
	BusinessShortCode string
	PassKey           string
	CallBackURL       string
	HTTPTimeout       time.Duration
	BaseURLOverride   string // Optional: used for mocking and test servers
}

// BaseURL returns the endpoint URL based on the environment.
func (c *Config) BaseURL() string {
	if c.BaseURLOverride != "" {
		return c.BaseURLOverride
	}
	if c.Environment == EnvProduction {
		return ProductionBaseURL
	}
	return SandboxBaseURL
}

// Validate checks whether mandatory configuration values are present.
func (c *Config) Validate() error {
	if strings.TrimSpace(c.ConsumerKey) == "" {
		return &Error{Kind: ErrorKindValidation, Message: "ConsumerKey is required"}
	}
	if strings.TrimSpace(c.ConsumerSecret) == "" {
		return &Error{Kind: ErrorKindValidation, Message: "ConsumerSecret is required"}
	}
	if strings.TrimSpace(c.BusinessShortCode) == "" {
		return &Error{Kind: ErrorKindValidation, Message: "BusinessShortCode is required"}
	}
	if strings.TrimSpace(c.PassKey) == "" {
		return &Error{Kind: ErrorKindValidation, Message: "PassKey is required"}
	}
	if strings.TrimSpace(c.CallBackURL) == "" {
		return &Error{Kind: ErrorKindValidation, Message: "CallBackURL is required"}
	}
	return nil
}

// LoadConfigFromEnv reads configuration from environment variables.
func LoadConfigFromEnv() (*Config, error) {
	env := strings.ToLower(strings.TrimSpace(os.Getenv("MPESA_ENVIRONMENT")))
	if env == "" {
		env = string(EnvSandbox)
	}

	shortCode := strings.TrimSpace(os.Getenv("MPESA_BUSINESS_SHORT_CODE"))
	if shortCode == "" && env == string(EnvSandbox) {
		shortCode = "174379"
	}

	cfg := &Config{
		Environment:       Environment(env),
		ConsumerKey:       strings.TrimSpace(os.Getenv("MPESA_CONSUMER_KEY")),
		ConsumerSecret:    strings.TrimSpace(os.Getenv("MPESA_CONSUMER_SECRET")),
		BusinessShortCode: shortCode,
		PassKey:           strings.TrimSpace(os.Getenv("MPESA_PASSKEY")),
		CallBackURL:       strings.TrimSpace(os.Getenv("MPESA_CALLBACK_URL")),
		HTTPTimeout:       DefaultHTTPTimeout,
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("configuration error: %w", err)
	}

	return cfg, nil
}
