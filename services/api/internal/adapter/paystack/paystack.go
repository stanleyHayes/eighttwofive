// Package paystack implements the domain.PaymentProvider port using the
// Paystack REST API.
package paystack

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/hayfordstanley/eightfivetwo/services/api/internal/domain"
)

const (
	baseURL         = "https://api.paystack.co"
	headerSignature = "x-paystack-signature"
	providerName    = "paystack"
	httpTimeout     = 15 * time.Second
	statusSuccess   = "success"
)

var (
	errInitializeFailed = errors.New("paystack initialize failed")
	errVerifyFailed     = errors.New("paystack verify failed")
	errPaystackHTTP     = errors.New("paystack returned an error")
)

// Client calls the Paystack API and verifies webhooks.
type Client struct {
	secretKey string
	http      *http.Client
	baseURL   string
}

// NewClient builds a Paystack client. If secretKey is empty every call returns
// domain.ErrNotConfigured.
func NewClient(secretKey string) *Client {
	return &Client{
		secretKey: secretKey,
		http:      &http.Client{Timeout: httpTimeout},
		baseURL:   baseURL,
	}
}

// NewClientWithBaseURL is used by tests to point at an httptest server.
func NewClientWithBaseURL(secretKey, baseURL string) *Client {
	return &Client{
		secretKey: secretKey,
		http:      &http.Client{Timeout: httpTimeout},
		baseURL:   baseURL,
	}
}

// InitTransaction starts a checkout on Paystack.
func (c *Client) InitTransaction(
	ctx context.Context,
	amountPesewas int64,
	email, reference, callbackURL string,
) (string, string, error) {
	if c.secretKey == "" {
		return "", "", domain.ErrNotConfigured
	}

	payload := map[string]any{
		"amount":    amountPesewas,
		"email":     email,
		"reference": reference,
	}
	if callbackURL != "" {
		payload["callback_url"] = callbackURL
	}

	var result struct {
		Status  bool   `json:"status"`
		Message string `json:"message"`
		Data    struct {
			//nolint:tagliatelle // Paystack's own JSON field names.
			AuthorizationURL string `json:"authorization_url"`
			Reference        string `json:"reference"`
			//nolint:tagliatelle // Paystack's own JSON field names.
			AccessCode string `json:"access_code"`
		} `json:"data"`
	}

	err := c.post(ctx, "/transaction/initialize", payload, &result)
	if err != nil {
		return "", "", fmt.Errorf("paystack initialize: %w", err)
	}

	if !result.Status {
		return "", "", fmt.Errorf("%w: %s", errInitializeFailed, result.Message)
	}

	return result.Data.AuthorizationURL, result.Data.Reference, nil
}

// VerifyWebhook validates the HMAC-SHA512 signature and parses the event.
func (c *Client) VerifyWebhook(payload []byte, signature string) (*domain.WebhookEvent, error) {
	if c.secretKey == "" {
		return nil, domain.ErrNotConfigured
	}

	mac := hmac.New(sha512.New, []byte(c.secretKey))
	_, _ = mac.Write(payload)
	expected := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(expected), []byte(signature)) {
		return nil, fmt.Errorf("%w: signature mismatch", domain.ErrWebhookInvalid)
	}

	return parseWebhookEvent(payload)
}

// VerifyTransaction fetches the current transaction status from Paystack.
func (c *Client) VerifyTransaction(ctx context.Context, providerRef string) (string, error) {
	if c.secretKey == "" {
		return "", domain.ErrNotConfigured
	}

	var result struct {
		Status  bool   `json:"status"`
		Message string `json:"message"`
		Data    struct {
			Status string `json:"status"`
		} `json:"data"`
	}

	err := c.get(ctx, "/transaction/verify/"+providerRef, &result)
	if err != nil {
		return "", fmt.Errorf("paystack verify: %w", err)
	}

	if !result.Status {
		return "", fmt.Errorf("%w: %s", errVerifyFailed, result.Message)
	}

	return result.Data.Status, nil
}

func (c *Client) post(ctx context.Context, path string, body any, dst any) error {
	var buf bytes.Buffer

	err := json.NewEncoder(&buf).Encode(body)
	if err != nil {
		return fmt.Errorf("encode request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, &buf)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.secretKey)
	req.Header.Set("Content-Type", "application/json")

	res, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}

	defer func() { _ = res.Body.Close() }()

	return decodeResponse(res, dst)
}

func (c *Client) get(ctx context.Context, path string, dst any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.secretKey)

	res, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}

	defer func() { _ = res.Body.Close() }()

	return decodeResponse(res, dst)
}

func decodeResponse(res *http.Response, dst any) error {
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if res.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("%w: status %d: %s", errPaystackHTTP, res.StatusCode, string(body))
	}

	if dst == nil {
		return nil
	}

	err = json.Unmarshal(body, dst)
	if err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	return nil
}

func parseWebhookEvent(payload []byte) (*domain.WebhookEvent, error) {
	var event struct {
		Event string `json:"event"`
		Data  struct {
			Reference string `json:"reference"`
			Status    string `json:"status"`
			Amount    int64  `json:"amount"`
		} `json:"data"`
	}

	err := json.Unmarshal(payload, &event)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", domain.ErrWebhookInvalid, err)
	}

	if event.Event == "" && event.Data.Reference == "" {
		return nil, fmt.Errorf("%w: missing event data", domain.ErrWebhookInvalid)
	}

	// Only charge.success may carry a success status: other signed events
	// (transfers, refunds, payment requests) must never book an order.
	status := ""
	if event.Event == domain.WebhookEventChargeSuccess {
		status = event.Data.Status
		if status == "" {
			status = statusSuccess
		}
	}

	return &domain.WebhookEvent{
		Type:          event.Event,
		ProviderRef:   event.Data.Reference,
		Status:        status,
		AmountPesewas: event.Data.Amount,
	}, nil
}

// FormatAmount formats integer pesewas as a string for provider payloads.
func FormatAmount(pesewas int64) string {
	return strconv.FormatInt(pesewas, 10)
}

var _ domain.PaymentProvider = (*Client)(nil)

// ProviderName returns the adapter identifier used for audit events.
func ProviderName() string { return providerName }
