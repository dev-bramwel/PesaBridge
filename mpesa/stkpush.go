package mpesa

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// STKPushRawRequest is the JSON payload required by Safaricom Daraja STK Push API.
type STKPushRawRequest struct {
	BusinessShortCode string `json:"BusinessShortCode"`
	Password          string `json:"Password"`
	Timestamp         string `json:"Timestamp"`
	TransactionType   string `json:"TransactionType"`
	Amount            int64  `json:"Amount"`
	PartyA            string `json:"PartyA"`
	PartyB            string `json:"PartyB"`
	PhoneNumber       string `json:"PhoneNumber"`
	CallBackURL       string `json:"CallBackURL"`
	AccountReference  string `json:"AccountReference"`
	TransactionDesc   string `json:"TransactionDesc"`
}

// STKPushRawResponse is the direct JSON response from Daraja STK Push API.
type STKPushRawResponse struct {
	MerchantRequestID   string `json:"MerchantRequestID"`
	CheckoutRequestID   string `json:"CheckoutRequestID"`
	ResponseCode        string `json:"ResponseCode"`
	ResponseDescription string `json:"ResponseDescription"`
	CustomerMessage     string `json:"CustomerMessage"`

	// In case of error
	RequestId    string `json:"requestId,omitempty"`
	ErrorCode    string `json:"errorCode,omitempty"`
	ErrorMessage string `json:"errorMessage,omitempty"`
}

// STKPushRequest models input params for initiating an STK Push.
type STKPushRequest struct {
	Amount           int64
	PhoneNumber      string
	AccountReference string
	Description      string
}

// STKPushResponse models the response returned from an STK Push initiation.
type STKPushResponse struct {
	MerchantRequestID   string
	CheckoutRequestID   string
	ResponseCode        string
	ResponseDescription string
	CustomerMessage     string
}

// GeneratePassword creates the Base64(ShortCode + PassKey + Timestamp) required by Daraja.
func GeneratePassword(shortCode, passKey, timestamp string) string {
	raw := fmt.Sprintf("%s%s%s", shortCode, passKey, timestamp)
	return base64.StdEncoding.EncodeToString([]byte(raw))
}

// FormatTimestamp returns the current time in YYYYMMDDHHmmss format.
func FormatTimestamp(t time.Time) string {
	return t.Format("20060102150405")
}

// SanitizePhoneNumber normalizes phone number to 2547XXXXXXXX or 2541XXXXXXXX format.
func SanitizePhoneNumber(phone string) (string, error) {
	cleaned := strings.TrimSpace(phone)
	cleaned = strings.TrimPrefix(cleaned, "+")

	if strings.HasPrefix(cleaned, "0") && len(cleaned) == 10 {
		cleaned = "254" + cleaned[1:]
	}

	if len(cleaned) != 12 || !strings.HasPrefix(cleaned, "254") {
		return "", fmt.Errorf("invalid Kenyan phone number %q, must be in format 2547XXXXXXXX or 2541XXXXXXXX", phone)
	}

	return cleaned, nil
}

// STKService handles STK push requests against Daraja.
type STKService struct {
	client *Client
}

// NewSTKService creates a new STKService.
func NewSTKService(client *Client) *STKService {
	return &STKService{client: client}
}

// Initiate executes the STK push process request on Daraja.
func (s *STKService) Initiate(ctx context.Context, req STKPushRequest) (*STKPushResponse, error) {
	if req.Amount <= 0 {
		return nil, &Error{
			Kind:    ErrorKindValidation,
			Message: "amount must be greater than zero",
		}
	}

	phone, err := SanitizePhoneNumber(req.PhoneNumber)
	if err != nil {
		return nil, &Error{
			Kind:    ErrorKindValidation,
			Message: err.Error(),
			Err:     err,
		}
	}

	accRef := strings.TrimSpace(req.AccountReference)
	if accRef == "" {
		return nil, &Error{
			Kind:    ErrorKindValidation,
			Message: "account reference cannot be empty",
		}
	}

	desc := strings.TrimSpace(req.Description)
	if desc == "" {
		desc = "Payment"
	}

	token, err := s.client.TokenManager.GetAccessToken(ctx)
	if err != nil {
		return nil, err
	}

	timestamp := FormatTimestamp(time.Now())
	password := GeneratePassword(s.client.Config.BusinessShortCode, s.client.Config.PassKey, timestamp)

	rawReq := STKPushRawRequest{
		BusinessShortCode: s.client.Config.BusinessShortCode,
		Password:          password,
		Timestamp:         timestamp,
		TransactionType:   "CustomerPayBillOnline",
		Amount:            req.Amount,
		PartyA:            phone,
		PartyB:            s.client.Config.BusinessShortCode,
		PhoneNumber:       phone,
		CallBackURL:       s.client.Config.CallBackURL,
		AccountReference:  accRef,
		TransactionDesc:   desc,
	}

	reqBytes, err := json.Marshal(rawReq)
	if err != nil {
		return nil, &Error{
			Kind:    ErrorKindValidation,
			Message: "failed to serialize STK push request",
			Err:     err,
		}
	}

	endpoint := fmt.Sprintf("%s/mpesa/stkpush/v1/processrequest", s.client.Config.BaseURL())
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBuffer(reqBytes))
	if err != nil {
		return nil, &Error{
			Kind:    ErrorKindNetwork,
			Message: "failed to construct STK HTTP request",
			Err:     err,
		}
	}

	httpReq.Header.Set("Authorization", "Bearer "+token)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	httpResp, err := s.client.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, &Error{
			Kind:    ErrorKindNetwork,
			Message: "failed to execute STK HTTP request",
			Err:     err,
		}
	}
	defer httpResp.Body.Close()

	bodyBytes, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, &Error{
			Kind:    ErrorKindNetwork,
			Message: "failed to read STK response body",
			Err:     err,
		}
	}

	var rawResp STKPushRawResponse
	if err := json.Unmarshal(bodyBytes, &rawResp); err != nil {
		return nil, &Error{
			Kind:       ErrorKindProvider,
			Message:    fmt.Sprintf("failed to parse STK response (status %d): %s", httpResp.StatusCode, string(bodyBytes)),
			StatusCode: httpResp.StatusCode,
			Err:        err,
		}
	}

	if httpResp.StatusCode != http.StatusOK || (rawResp.ResponseCode != "0" && rawResp.ResponseCode != "") {
		errMsg := rawResp.ResponseDescription
		if errMsg == "" {
			errMsg = rawResp.ErrorMessage
		}
		if errMsg == "" {
			errMsg = string(bodyBytes)
		}
		return nil, &Error{
			Kind:       ErrorKindProvider,
			Message:    fmt.Sprintf("STK push failed: %s (ResponseCode: %s)", errMsg, rawResp.ResponseCode),
			StatusCode: httpResp.StatusCode,
		}
	}

	return &STKPushResponse{
		MerchantRequestID:   rawResp.MerchantRequestID,
		CheckoutRequestID:   rawResp.CheckoutRequestID,
		ResponseCode:        rawResp.ResponseCode,
		ResponseDescription: rawResp.ResponseDescription,
		CustomerMessage:     rawResp.CustomerMessage,
	}, nil
}
