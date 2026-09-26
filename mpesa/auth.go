package mpesa

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"
)

type authResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   string `json:"expires_in"`
}

// TokenManager handles fetching and caching the Daraja OAuth access token.
type TokenManager struct {
	consumerKey    string
	consumerSecret string
	baseURL        string
	httpClient     *http.Client

	mu        sync.RWMutex
	token     string
	expiresAt time.Time
}

// NewTokenManager creates a new TokenManager instance.
func NewTokenManager(consumerKey, consumerSecret, baseURL string, client *http.Client) *TokenManager {
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	return &TokenManager{
		consumerKey:    consumerKey,
		consumerSecret: consumerSecret,
		baseURL:        baseURL,
		httpClient:     client,
	}
}

// GetAccessToken returns a valid cached access token or fetches a new one.
func (tm *TokenManager) GetAccessToken(ctx context.Context) (string, error) {
	tm.mu.RLock()
	if tm.token != "" && time.Now().Before(tm.expiresAt) {
		token := tm.token
		tm.mu.RUnlock()
		return token, nil
	}
	tm.mu.RUnlock()

	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Double check under write lock
	if tm.token != "" && time.Now().Before(tm.expiresAt) {
		return tm.token, nil
	}

	token, expiresAt, err := tm.fetchToken(ctx)
	if err != nil {
		return "", err
	}

	tm.token = token
	tm.expiresAt = expiresAt
	return token, nil
}

func (tm *TokenManager) fetchToken(ctx context.Context) (string, time.Time, error) {
	endpoint := fmt.Sprintf("%s/oauth/v1/generate?grant_type=client_credentials", tm.baseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", time.Time{}, &Error{
			Kind:    ErrorKindNetwork,
			Message: "failed to construct auth request",
			Err:     err,
		}
	}

	authHeader := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", tm.consumerKey, tm.consumerSecret)))
	req.Header.Set("Authorization", "Basic "+authHeader)
	req.Header.Set("Accept", "application/json")

	resp, err := tm.httpClient.Do(req)
	if err != nil {
		return "", time.Time{}, &Error{
			Kind:    ErrorKindNetwork,
			Message: "failed to communicate with OAuth endpoint",
			Err:     err,
		}
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", time.Time{}, &Error{
			Kind:    ErrorKindNetwork,
			Message: "failed to read OAuth response body",
			Err:     err,
		}
	}

	if resp.StatusCode != http.StatusOK {
		return "", time.Time{}, &Error{
			Kind:       ErrorKindAuthentication,
			Message:    fmt.Sprintf("OAuth authentication failed with status %d: %s", resp.StatusCode, string(bodyBytes)),
			StatusCode: resp.StatusCode,
		}
	}

	var authResp authResponse
	if err := json.Unmarshal(bodyBytes, &authResp); err != nil {
		return "", time.Time{}, &Error{
			Kind:    ErrorKindAuthentication,
			Message: "failed to decode OAuth response JSON",
			Err:     err,
		}
	}

	if authResp.AccessToken == "" {
		return "", time.Time{}, &Error{
			Kind:    ErrorKindAuthentication,
			Message: "empty access_token received from provider",
		}
	}

	expirySeconds, err := strconv.Atoi(authResp.ExpiresIn)
	if err != nil || expirySeconds <= 0 {
		expirySeconds = 3599
	}

	// Buffer expiration by 60 seconds to avoid race condition on token expiry
	buffer := 60 * time.Second
	expiresAt := time.Now().Add(time.Duration(expirySeconds) * time.Second).Add(-buffer)

	return authResp.AccessToken, expiresAt, nil
}
