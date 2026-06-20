package httpapi_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hayfordstanley/eightfivetwo/services/api/internal/adapter/media"
)

func postJSON(t *testing.T, url, body string) int {
	t.Helper()

	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, url, strings.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	defer func() { _ = res.Body.Close() }()

	return res.StatusCode
}

func getStatus(t *testing.T, url string) int {
	t.Helper()

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, url, nil)
	require.NoError(t, err)

	res, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	defer func() { _ = res.Body.Close() }()

	return res.StatusCode
}

func TestHealth(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t)

	for _, path := range []string{"/healthz", "/api/v1/healthz"} {
		assert.Equal(t, http.StatusOK, getStatus(t, srv.URL+path), path)
	}
}

func TestJoinWaitlist(t *testing.T) {
	t.Parallel()

	t.Run("creates subscriber", func(t *testing.T) {
		t.Parallel()

		srv := newTestServer(t)
		status := postJSON(t, srv.URL+"/api/v1/waitlist", `{"email":"ada@example.com","name":"Ada"}`)
		assert.Equal(t, http.StatusCreated, status)
	})

	t.Run("rejects duplicate email", func(t *testing.T) {
		t.Parallel()

		srv := newTestServer(t)
		url := srv.URL + "/api/v1/waitlist"
		require.Equal(t, http.StatusCreated, postJSON(t, url, `{"email":"ada@example.com","name":"Ada"}`))
		assert.Equal(t, http.StatusConflict, postJSON(t, url, `{"email":"ada@example.com","name":"Ada"}`))
	})

	t.Run("rejects invalid email", func(t *testing.T) {
		t.Parallel()

		srv := newTestServer(t)
		status := postJSON(t, srv.URL+"/api/v1/waitlist", `{"email":"nope","name":"Ada"}`)
		assert.Equal(t, http.StatusUnprocessableEntity, status)
	})

	t.Run("rejects malformed JSON", func(t *testing.T) {
		t.Parallel()

		srv := newTestServer(t)
		status := postJSON(t, srv.URL+"/api/v1/waitlist", `{not json`)
		assert.Equal(t, http.StatusBadRequest, status)
	})

	t.Run("rate-limits to stop email-bombing a third party", func(t *testing.T) {
		t.Parallel()

		srv := newTestServer(t)
		url := srv.URL + "/api/v1/waitlist"

		// Each signup persists a subscriber and sends a welcome email, so a flood
		// from one client must eventually be throttled with 429. The per-minute
		// budget is well under 30, so this loop is guaranteed to trip it.
		var limited bool

		for i := range 30 {
			body := fmt.Sprintf(`{"email":"victim%d@example.com","name":"X"}`, i)
			if postJSON(t, url, body) == http.StatusTooManyRequests {
				limited = true

				break
			}
		}

		assert.True(t, limited, "waitlist signups should be rate-limited")
	})
}

func TestListWaitlist_RequiresAdmin(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t)

	assert.Equal(t, http.StatusUnauthorized, getStatus(t, srv.URL+"/api/v1/admin/waitlist"),
		"waitlist listing must not be public")
}

func TestSignUpload_AdminOnlyAndNotConfigured(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t, "boss@e25.com")

	// Upload signing is an admin capability now.
	assert.Equal(t, http.StatusUnauthorized,
		postJSON(t, env.srv.URL+"/api/v1/admin/uploads/sign", `{}`))

	admin := env.signIn(t, "boss@e25.com")
	reply := doJSON(t, http.MethodPost, env.srv.URL+"/api/v1/admin/uploads/sign", `{}`, admin)
	assert.Equal(t, http.StatusServiceUnavailable, reply.status, "no Cloudinary creds in tests")
}

func TestSignUpload_UsesOptionalFolder(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t, "boss@e25.com")
	signer := media.NewCloudinarySigner("test-cloud", "test-key", "test-secret")
	signer.SetNow(func() time.Time { return time.Unix(1710000000, 0) })

	// Re-wire the test environment with a real signer.
	handlers := newHandlersWithSigner(env, signer)
	srv := newTestServerWithHandlers(t, handlers)
	admin := signInOnServer(t, srv, env.sender)

	// Default folder when omitted.
	defaultReply := doJSON(t, http.MethodPost, srv.URL+"/api/v1/admin/uploads/sign", `{}`, admin)
	require.Equal(t, http.StatusOK, defaultReply.status, defaultReply.body)
	assert.Contains(t, defaultReply.body, `"folder":"eightfivetwo"`)

	// Custom folder when provided.
	customReply := doJSON(t, http.MethodPost, srv.URL+"/api/v1/admin/uploads/sign",
		`{"folder":"spring-drop"}`, admin)
	require.Equal(t, http.StatusOK, customReply.status, customReply.body)

	var payload struct {
		Data struct {
			Folder    string `json:"folder"`
			Signature string `json:"signature"`
			Timestamp int64  `json:"timestamp"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal([]byte(customReply.body), &payload))
	assert.Equal(t, "spring-drop", payload.Data.Folder)
	assert.NotEmpty(t, payload.Data.Signature)
	assert.Equal(t, int64(1710000000), payload.Data.Timestamp)
}
