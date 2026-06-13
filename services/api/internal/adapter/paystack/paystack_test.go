package paystack_test

import (
	"context"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hayfordstanley/eightfivetwo/services/api/internal/adapter/paystack"
	"github.com/hayfordstanley/eightfivetwo/services/api/internal/domain"
)

const testSecret = "sk_test_1234567890"

func TestNewClient_NoSecret_ReturnsNotConfigured(t *testing.T) {
	t.Parallel()

	client := paystack.NewClient("")
	_, _, err := client.InitTransaction(context.Background(), 50000, "a@b.com", "ref", "")
	require.ErrorIs(t, err, domain.ErrNotConfigured)

	_, err = client.VerifyWebhook([]byte("{}"), "sig")
	require.ErrorIs(t, err, domain.ErrNotConfigured)

	_, err = client.VerifyTransaction(context.Background(), "ref")
	require.ErrorIs(t, err, domain.ErrNotConfigured)
}

func TestInitTransaction(t *testing.T) {
	t.Parallel()

	var requestBody map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/transaction/initialize", r.URL.Path)
		assert.Equal(t, "Bearer "+testSecret, r.Header.Get("Authorization"))

		err := json.NewDecoder(r.Body).Decode(&requestBody)
		if err != nil {
			t.Errorf("decode request: %v", err)

			return
		}

		res := map[string]any{
			"status":  true,
			"message": "Initialized",
			"data": map[string]any{
				"authorization_url": "https://checkout.paystack.com/abc",
				"reference":         requestBody["reference"],
				"access_code":       "access_123",
			},
		}

		_ = json.NewEncoder(w).Encode(res)
	}))
	defer server.Close()

	client := paystack.NewClientWithBaseURL(testSecret, server.URL)
	url, ref, err := client.InitTransaction(
		context.Background(), 50000, "a@b.com", "E25-XYZ", "https://shop.test/callback",
	)

	require.NoError(t, err)
	assert.Equal(t, "https://checkout.paystack.com/abc", url)
	assert.Equal(t, "E25-XYZ", ref)
	assert.EqualValues(t, 50000, requestBody["amount"])
	assert.Equal(t, "https://shop.test/callback", requestBody["callback_url"])
}

func TestVerifyWebhook(t *testing.T) {
	t.Parallel()

	payload := []byte(`{"event":"charge.success","data":{"reference":"E25-XYZ","status":"success","amount":50000}}`)
	mac := hmac.New(sha512.New, []byte(testSecret))
	_, _ = mac.Write(payload)
	signature := hex.EncodeToString(mac.Sum(nil))

	client := paystack.NewClient(testSecret)
	event, err := client.VerifyWebhook(payload, signature)

	require.NoError(t, err)
	assert.Equal(t, "charge.success", event.Type)
	assert.Equal(t, "E25-XYZ", event.ProviderRef)
	assert.Equal(t, "success", event.Status)
	assert.Equal(t, int64(50000), event.AmountPesewas)
}

func TestVerifyWebhook_NonChargeEventNeverSuccess(t *testing.T) {
	t.Parallel()

	// A signed transfer.success with data.status=success must not normalize to
	// a success event: only charge.success may ever book an order.
	payload := []byte(`{"event":"transfer.success","data":{"reference":"E25-XYZ","status":"success","amount":50000}}`)
	mac := hmac.New(sha512.New, []byte(testSecret))
	_, _ = mac.Write(payload)
	signature := hex.EncodeToString(mac.Sum(nil))

	client := paystack.NewClient(testSecret)
	event, err := client.VerifyWebhook(payload, signature)

	require.NoError(t, err)
	assert.Equal(t, "transfer.success", event.Type)
	assert.Empty(t, event.Status)
}

func TestVerifyWebhook_BadSignature(t *testing.T) {
	t.Parallel()

	client := paystack.NewClient(testSecret)
	_, err := client.VerifyWebhook([]byte(`{}`), "bad")

	assert.ErrorIs(t, err, domain.ErrWebhookInvalid)
}

func TestVerifyTransaction(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/transaction/verify/E25-XYZ", r.URL.Path)

		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":  true,
			"message": "Verification successful",
			"data":    map[string]any{"status": "success"},
		})
	}))
	defer server.Close()

	client := paystack.NewClientWithBaseURL(testSecret, server.URL)
	status, err := client.VerifyTransaction(context.Background(), "E25-XYZ")

	require.NoError(t, err)
	assert.Equal(t, "success", status)
}
