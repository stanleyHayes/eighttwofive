package httpapi_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createSlotPayload(startOffset, endOffset time.Duration) string {
	start := time.Now().UTC().Add(startOffset).Format(time.RFC3339)
	end := time.Now().UTC().Add(endOffset).Format(time.RFC3339)

	return fmt.Sprintf(`{"start": %q, "end": %q}`, start, end)
}

func extractSlotID(t *testing.T, reply jsonReply) string {
	t.Helper()

	var resp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal([]byte(reply.body), &resp))

	return resp.Data.ID
}

func extractVisitID(t *testing.T, reply jsonReply) string {
	t.Helper()

	var resp struct {
		Data struct {
			Visit struct {
				ID string `json:"id"`
			} `json:"visit"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal([]byte(reply.body), &resp))

	return resp.Data.Visit.ID
}

func bookingBody() string {
	return `{
		"email": "ama@example.com",
		"name": "Ama",
		"phone": "+233200000000"
	}`
}

func TestListOpenSlots(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t)

	res := doJSON(t, http.MethodGet, srv.URL+"/api/v1/slots", "", nil)
	assert.Equal(t, http.StatusOK, res.status)
}

func TestBookSlot(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t, "boss@e25.com")
	admin := env.signIn(t, "boss@e25.com")

	base := env.srv.URL + "/api/v1"
	slot := doJSON(t, http.MethodPost, base+"/admin/slots", createSlotPayload(24*time.Hour, 25*time.Hour), admin)
	require.Equal(t, http.StatusCreated, slot.status)

	book := doJSON(t, http.MethodPost, base+"/slots/"+extractSlotID(t, slot)+"/book", bookingBody(), nil)
	require.Equal(t, http.StatusCreated, book.status, "body: %s", book.body)
	assert.Contains(t, book.body, "paymentUrl")
	assert.Contains(t, book.body, `"status":"booked"`)
	assert.Contains(t, book.body, `"type":"visit"`)

	var session *http.Cookie

	for _, cookie := range book.cookies {
		if cookie.Name == sessionCookieName {
			session = cookie
		}
	}

	require.NotNil(t, session, "booking must set a session cookie")
	assert.True(t, session.HttpOnly)
}

func TestBookSlot_SlotUnavailable(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t, "boss@e25.com")
	admin := env.signIn(t, "boss@e25.com")

	base := env.srv.URL + "/api/v1"
	slot := doJSON(t, http.MethodPost, base+"/admin/slots", createSlotPayload(24*time.Hour, 25*time.Hour), admin)
	require.Equal(t, http.StatusCreated, slot.status)

	slotID := extractSlotID(t, slot)

	first := doJSON(t, http.MethodPost, base+"/slots/"+slotID+"/book", bookingBody(), nil)
	require.Equal(t, http.StatusCreated, first.status)

	second := doJSON(t, http.MethodPost, base+"/slots/"+slotID+"/book", `{
		"email": "kofi@example.com",
		"name": "Kofi",
		"phone": "+233200000001"
	}`, nil)
	assert.Equal(t, http.StatusConflict, second.status)
}

func TestAdminCreateSlot(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t, "boss@e25.com")
	admin := env.signIn(t, "boss@e25.com")

	res := doJSON(
		t, http.MethodPost,
		env.srv.URL+"/api/v1/admin/slots",
		createSlotPayload(24*time.Hour, 25*time.Hour),
		admin,
	)

	require.Equal(t, http.StatusCreated, res.status, "body: %s", res.body)
	assert.Contains(t, res.body, `"status":"open"`)
}

func TestAdminCloseAndReopenSlot(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t, "boss@e25.com")
	admin := env.signIn(t, "boss@e25.com")

	base := env.srv.URL + "/api/v1/admin/slots"
	create := doJSON(t, http.MethodPost, base, createSlotPayload(24*time.Hour, 25*time.Hour), admin)
	require.Equal(t, http.StatusCreated, create.status)

	slotID := extractSlotID(t, create)

	closeRes := doJSON(t, http.MethodPost, base+"/"+slotID+"/close", "", admin)
	assert.Equal(t, http.StatusOK, closeRes.status)

	reopenRes := doJSON(t, http.MethodPost, base+"/"+slotID+"/reopen", "", admin)
	assert.Equal(t, http.StatusOK, reopenRes.status)
}

func TestAdminCancelVisit(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t, "boss@e25.com")
	admin := env.signIn(t, "boss@e25.com")

	base := env.srv.URL + "/api/v1"
	slot := doJSON(t, http.MethodPost, base+"/admin/slots", createSlotPayload(24*time.Hour, 25*time.Hour), admin)
	require.Equal(t, http.StatusCreated, slot.status)

	slotID := extractSlotID(t, slot)

	book := doJSON(t, http.MethodPost, base+"/slots/"+slotID+"/book", bookingBody(), nil)
	require.Equal(t, http.StatusCreated, book.status)

	visitID := extractVisitID(t, book)

	cancel := doJSON(t, http.MethodPost, base+"/admin/visits/"+visitID+"/cancel", "", admin)
	assert.Equal(t, http.StatusOK, cancel.status)
	assert.Contains(t, cancel.body, `"status":"cancelled"`)
}

func TestAdminRescheduleVisit(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t, "boss@e25.com")
	admin := env.signIn(t, "boss@e25.com")

	base := env.srv.URL + "/api/v1"
	oldSlot := doJSON(t, http.MethodPost, base+"/admin/slots", createSlotPayload(24*time.Hour, 25*time.Hour), admin)
	require.Equal(t, http.StatusCreated, oldSlot.status)

	newSlot := doJSON(t, http.MethodPost, base+"/admin/slots", createSlotPayload(48*time.Hour, 49*time.Hour), admin)
	require.Equal(t, http.StatusCreated, newSlot.status)

	oldSlotID := extractSlotID(t, oldSlot)
	newSlotID := extractSlotID(t, newSlot)

	book := doJSON(t, http.MethodPost, base+"/slots/"+oldSlotID+"/book", bookingBody(), nil)
	require.Equal(t, http.StatusCreated, book.status)

	visitID := extractVisitID(t, book)

	reschedule := doJSON(
		t, http.MethodPost,
		base+"/admin/visits/"+visitID+"/reschedule",
		fmt.Sprintf(`{"newSlotId": %q}`, newSlotID),
		admin,
	)
	assert.Equal(t, http.StatusOK, reschedule.status)
	assert.Contains(t, reschedule.body, newSlotID)
}
