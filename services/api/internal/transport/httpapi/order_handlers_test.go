package httpapi_test

import (
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hayfordstanley/eightfivetwo/services/api/internal/domain"
)

func createDesign(t *testing.T, env *testEnv, admin *http.Cookie) string {
	t.Helper()

	base := env.srv.URL + "/api/v1/admin"

	collection := doJSON(t, http.MethodPost, base+"/collections",
		`{"name":"Velvet","note":"first drop"}`, admin)
	require.Equal(t, http.StatusCreated, collection.status)

	var collectionResp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal([]byte(collection.body), &collectionResp))

	design := doJSON(t, http.MethodPost, base+"/designs", fmt.Sprintf(`{
		"collectionId": %q,
		"name": "Boardroom Blazer",
		"note": "tailored office wear",
		"photos": [{"publicId":"e25/blazer","order":0}],
		"sizeBands": [{"label":"8","pricePesewas":50000,"chart":{"bust":"86 cm"}}]
	}`, collectionResp.Data.ID), admin)
	require.Equal(t, http.StatusCreated, design.status)

	var designResp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal([]byte(design.body), &designResp))

	return designResp.Data.ID
}

func createCustomRequestOrder(t *testing.T, env *testEnv) string {
	t.Helper()

	ref := fmt.Sprintf("E25-CUSTOM-%d", time.Now().UnixNano())
	createdAt := time.Now().UTC()

	err := env.orders.Create(t.Context(), &domain.Order{
		Ref:            ref,
		CustomerID:     "user-1",
		DesignID:       "000000000000000000000002",
		DesignSnapshot: domain.DesignSnapshot{Name: "Blazer", PhotoPublicID: "e25/blazer", PricePesewas: 0},
		Type:           domain.OrderTypeDesignChange,
		Customisation:  domain.Customisation{SizeMode: "self", DesignChange: "longer sleeves"},
		Status:         domain.OrderStatusRequested,
		StatusHistory: []domain.StatusChange{{
			Status: domain.OrderStatusRequested,
			At:     createdAt,
			By:     "customer",
		}},
		CustomerPhone: "+233200000000",
		CreatedAt:     createdAt,
		UpdatedAt:     createdAt,
	})
	require.NoError(t, err)

	return ref
}

func TestCreateOrder(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t, "boss@e25.com")
	admin := env.signIn(t, "boss@e25.com")
	designID := createDesign(t, env, admin)

	base := env.srv.URL + "/api/v1"
	reply := doJSON(t, http.MethodPost, base+"/orders", fmt.Sprintf(`{
		"designId": %q,
		"bandLabel": "8",
		"delivery": "pickup",
		"customerPhone": "+233200000000",
		"email": "ama@example.com",
		"name": "Ama"
	}`, designID), nil)

	require.Equal(t, http.StatusCreated, reply.status, "body: %s", reply.body)
	assert.Contains(t, reply.body, "paymentUrl")
	assert.Contains(t, reply.body, `"status":"pending_payment"`)

	var session *http.Cookie

	for _, c := range reply.cookies {
		if c.Name == sessionCookieName {
			session = c
		}
	}

	// Anonymous checkout must NOT mint a session from a body-supplied email
	// (account takeover / order IDOR). Customers sign in via the magic link.
	assert.Nil(t, session, "checkout must not set a session cookie")
}

func TestCreateCustomRequest(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t, "boss@e25.com")
	admin := env.signIn(t, "boss@e25.com")
	designID := createDesign(t, env, admin)

	base := env.srv.URL + "/api/v1"
	reply := doJSON(t, http.MethodPost, base+"/orders/request", fmt.Sprintf(`{
		"designId": %q,
		"sizeMode": "self",
		"measurements": {"bust": "90 cm", "waist": "74 cm"},
		"delivery": "pickup",
		"customerPhone": "+233200000000",
		"email": "ama@example.com",
		"name": "Ama"
	}`, designID), nil)

	require.Equal(t, http.StatusCreated, reply.status, "body: %s", reply.body)
	assert.Contains(t, reply.body, `"status":"requested"`)
	assert.Contains(t, reply.body, `"type":"custom_size"`)

	var session *http.Cookie

	for _, c := range reply.cookies {
		if c.Name == sessionCookieName {
			session = c
		}
	}

	// Anonymous checkout — no session minted from a body-supplied email.
	assert.Nil(t, session, "custom request must not set a session cookie")
}

func TestCreateCustomRequest_InvalidInput(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t, "boss@e25.com")
	admin := env.signIn(t, "boss@e25.com")
	designID := createDesign(t, env, admin)

	base := env.srv.URL + "/api/v1"
	reply := doJSON(t, http.MethodPost, base+"/orders/request", fmt.Sprintf(`{
		"designId": %q,
		"sizeMode": "self",
		"delivery": "pickup",
		"customerPhone": "",
		"email": "ama@example.com",
		"name": "Ama"
	}`, designID), nil)

	assert.Equal(t, http.StatusUnprocessableEntity, reply.status)
}

func TestCreateOrder_InvalidInput(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t, "boss@e25.com")
	admin := env.signIn(t, "boss@e25.com")
	designID := createDesign(t, env, admin)

	base := env.srv.URL + "/api/v1"
	reply := doJSON(t, http.MethodPost, base+"/orders", fmt.Sprintf(`{
		"designId": %q,
		"bandLabel": "8",
		"delivery": "pickup",
		"customerPhone": "",
		"email": "ama@example.com",
		"name": "Ama"
	}`, designID), nil)

	assert.Equal(t, http.StatusUnprocessableEntity, reply.status)
}

func TestGetOrder_CustomerOwnsOrder(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t, "boss@e25.com")
	admin := env.signIn(t, "boss@e25.com")
	designID := createDesign(t, env, admin)

	base := env.srv.URL + "/api/v1"
	create := doJSON(t, http.MethodPost, base+"/orders", fmt.Sprintf(`{
		"designId": %q,
		"bandLabel": "8",
		"delivery": "pickup",
		"customerPhone": "+233200000000",
		"email": "ama@example.com",
		"name": "Ama"
	}`, designID), nil)
	require.Equal(t, http.StatusCreated, create.status)

	var createResp struct {
		Data struct {
			Order struct {
				Ref string `json:"ref"`
			} `json:"order"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal([]byte(create.body), &createResp))

	// Checkout is anonymous; the customer signs in via the magic link to view
	// their order. Signing in as the same email owns the order created above.
	session := env.signIn(t, "ama@example.com")

	get := doJSON(t, http.MethodGet, base+"/orders/"+createResp.Data.Order.Ref, "", session)
	assert.Equal(t, http.StatusOK, get.status)
	assert.Contains(t, get.body, createResp.Data.Order.Ref)
}

func TestGetOrder_AdminCanViewAny(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t, "boss@e25.com")
	admin := env.signIn(t, "boss@e25.com")
	designID := createDesign(t, env, admin)

	base := env.srv.URL + "/api/v1"
	create := doJSON(t, http.MethodPost, base+"/orders", fmt.Sprintf(`{
		"designId": %q,
		"bandLabel": "8",
		"delivery": "pickup",
		"customerPhone": "+233200000000",
		"email": "ama@example.com",
		"name": "Ama"
	}`, designID), nil)
	require.Equal(t, http.StatusCreated, create.status)

	var createResp struct {
		Data struct {
			Order struct {
				Ref string `json:"ref"`
			} `json:"order"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal([]byte(create.body), &createResp))

	get := doJSON(t, http.MethodGet, base+"/orders/"+createResp.Data.Order.Ref, "", admin)
	assert.Equal(t, http.StatusOK, get.status)
}

func TestListCustomerOrders(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t, "boss@e25.com")
	admin := env.signIn(t, "boss@e25.com")
	designID := createDesign(t, env, admin)

	base := env.srv.URL + "/api/v1"
	create := doJSON(t, http.MethodPost, base+"/orders", fmt.Sprintf(`{
		"designId": %q,
		"bandLabel": "8",
		"delivery": "pickup",
		"customerPhone": "+233200000000",
		"email": "ama@example.com",
		"name": "Ama"
	}`, designID), nil)
	require.Equal(t, http.StatusCreated, create.status)

	// Anonymous checkout; the customer signs in via the magic link to list orders.
	session := env.signIn(t, "ama@example.com")

	list := doJSON(t, http.MethodGet, base+"/orders", "", session)
	assert.Equal(t, http.StatusOK, list.status)
	assert.Contains(t, list.body, `"ref":`)
}

func TestHandlePaymentWebhook_BooksOrder(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t, "boss@e25.com")
	admin := env.signIn(t, "boss@e25.com")
	designID := createDesign(t, env, admin)

	base := env.srv.URL + "/api/v1"
	create := doJSON(t, http.MethodPost, base+"/orders", fmt.Sprintf(`{
		"designId": %q,
		"bandLabel": "8",
		"delivery": "pickup",
		"customerPhone": "+233200000000",
		"email": "ama@example.com",
		"name": "Ama"
	}`, designID), nil)
	require.Equal(t, http.StatusCreated, create.status)

	var createResp struct {
		Data struct {
			Order struct {
				Ref string `json:"ref"`
			} `json:"order"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal([]byte(create.body), &createResp))

	order, err := env.orders.GetByRef(t.Context(), createResp.Data.Order.Ref)
	require.NoError(t, err)
	require.Len(t, order.Payments, 1)

	providerRef := order.Payments[0].ProviderRef
	payload := []byte(`{"event":"charge.success","data":{"reference":"` + providerRef + `","amount":50000}}`)
	mac := hmac.New(sha512.New, []byte("secret"))
	_, _ = mac.Write(payload)
	signature := hex.EncodeToString(mac.Sum(nil))

	req, err := http.NewRequestWithContext(
		t.Context(), http.MethodPost, base+"/payments/webhook", strings.NewReader(string(payload)),
	)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Paystack-Signature", signature)

	res, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	defer func() { _ = res.Body.Close() }()

	assert.Equal(t, http.StatusOK, res.StatusCode)

	updated, err := env.orders.GetByRef(t.Context(), createResp.Data.Order.Ref)
	require.NoError(t, err)
	assert.Equal(t, domain.OrderStatusBooked, updated.Status)
	assert.Equal(t, "ama@example.com", env.sender.lastStatusUpdateTo)
	assert.Equal(t, "order confirmed", env.sender.lastStatus)
}

func TestHandlePaymentWebhook_InvalidSignature(t *testing.T) {
	t.Parallel()

	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost,
		newTestServer(t).URL+"/api/v1/payments/webhook", strings.NewReader(`{}`))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Paystack-Signature", "bad")

	res, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	defer func() { _ = res.Body.Close() }()

	assert.Equal(t, http.StatusUnauthorized, res.StatusCode)
}

func TestAdminListOrders(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t, "boss@e25.com")
	admin := env.signIn(t, "boss@e25.com")
	designID := createDesign(t, env, admin)

	base := env.srv.URL + "/api/v1"
	create := doJSON(t, http.MethodPost, base+"/orders", fmt.Sprintf(`{
		"designId": %q,
		"bandLabel": "8",
		"delivery": "pickup",
		"customerPhone": "+233200000000",
		"email": "ama@example.com",
		"name": "Ama"
	}`, designID), nil)
	require.Equal(t, http.StatusCreated, create.status)

	list := doJSON(t, http.MethodGet, base+"/admin/orders", "", admin)
	assert.Equal(t, http.StatusOK, list.status)
	assert.Contains(t, list.body, `"type":"standard"`)
}

// --- admin order management tests ---------------------------------------------

func TestAdminGetOrder(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t, "boss@e25.com")
	admin := env.signIn(t, "boss@e25.com")
	designID := createDesign(t, env, admin)

	base := env.srv.URL + "/api/v1"
	create := doJSON(t, http.MethodPost, base+"/orders", fmt.Sprintf(`{
		"designId": %q,
		"bandLabel": "8",
		"delivery": "pickup",
		"customerPhone": "+233200000000",
		"email": "ama@example.com",
		"name": "Ama"
	}`, designID), nil)
	require.Equal(t, http.StatusCreated, create.status)

	var createResp struct {
		Data struct {
			Order struct {
				Ref string `json:"ref"`
			} `json:"order"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal([]byte(create.body), &createResp))

	get := doJSON(t, http.MethodGet, base+"/admin/orders/"+createResp.Data.Order.Ref, "", admin)
	assert.Equal(t, http.StatusOK, get.status)
	assert.Contains(t, get.body, createResp.Data.Order.Ref)
	assert.Contains(t, get.body, `"type":"standard"`)
}

func TestAdminUpdateQuote(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t, "boss@e25.com")
	admin := env.signIn(t, "boss@e25.com")

	// Create a customer so the custom order has a valid customer ID.
	createResp := doJSON(t, http.MethodPost, env.srv.URL+"/api/v1/auth/request-link",
		`{"email":"ama@example.com","name":"Ama"}`, nil)
	require.Equal(t, http.StatusAccepted, createResp.status)

	ref := createCustomRequestOrder(t, env)

	base := env.srv.URL + "/api/v1"
	update := doJSON(t, http.MethodPut, base+"/admin/orders/"+ref+"/quote", `{
		"pricePesewas": 75000,
		"timeline": "2 weeks",
		"notes": "Premium fabric upgrade"
	}`, admin)
	require.Equal(t, http.StatusOK, update.status, "body: %s", update.body)

	order, err := env.orders.GetByRef(t.Context(), ref)
	require.NoError(t, err)
	assert.Equal(t, int64(75000), order.Quote.PricePesewas)
	assert.Equal(t, domain.OrderStatusQuoted, order.Status)
}

func TestAdminSendPaymentLink(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t, "boss@e25.com")
	admin := env.signIn(t, "boss@e25.com")

	doJSON(t, http.MethodPost, env.srv.URL+"/api/v1/auth/request-link",
		`{"email":"ama@example.com","name":"Ama"}`, nil)

	ref := createCustomRequestOrder(t, env)

	base := env.srv.URL + "/api/v1"
	quote := doJSON(t, http.MethodPut, base+"/admin/orders/"+ref+"/quote", `{
		"pricePesewas": 60000,
		"timeline": "2 weeks",
		"notes": ""
	}`, admin)
	require.Equal(t, http.StatusOK, quote.status)

	link := doJSON(t, http.MethodPost, base+"/admin/orders/"+ref+"/payment-link", "", admin)
	require.Equal(t, http.StatusOK, link.status, "body: %s", link.body)
	assert.Contains(t, link.body, "paymentUrl")

	order, err := env.orders.GetByRef(t.Context(), ref)
	require.NoError(t, err)
	assert.Equal(t, domain.OrderStatusPaymentLinkSent, order.Status)
}

func TestAdminMarkPaid(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t, "boss@e25.com")
	admin := env.signIn(t, "boss@e25.com")

	doJSON(t, http.MethodPost, env.srv.URL+"/api/v1/auth/request-link",
		`{"email":"ama@example.com","name":"Ama"}`, nil)

	ref := createCustomRequestOrder(t, env)

	base := env.srv.URL + "/api/v1"
	quote := doJSON(t, http.MethodPut, base+"/admin/orders/"+ref+"/quote", `{
		"pricePesewas": 60000,
		"timeline": "2 weeks",
		"notes": ""
	}`, admin)
	require.Equal(t, http.StatusOK, quote.status)

	paid := doJSON(t, http.MethodPost, base+"/admin/orders/"+ref+"/mark-paid", `{"note":"Bank transfer"}`, admin)
	require.Equal(t, http.StatusOK, paid.status, "body: %s", paid.body)

	order, err := env.orders.GetByRef(t.Context(), ref)
	require.NoError(t, err)
	assert.Equal(t, domain.OrderStatusBooked, order.Status)
	assert.True(t, order.IsPaid())
}

func TestAdminUpdateOrderStatus(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t, "boss@e25.com")
	admin := env.signIn(t, "boss@e25.com")
	designID := createDesign(t, env, admin)

	base := env.srv.URL + "/api/v1"
	create := doJSON(t, http.MethodPost, base+"/orders", fmt.Sprintf(`{
		"designId": %q,
		"bandLabel": "8",
		"delivery": "pickup",
		"customerPhone": "+233200000000",
		"email": "ama@example.com",
		"name": "Ama"
	}`, designID), nil)
	require.Equal(t, http.StatusCreated, create.status)

	var createResp struct {
		Data struct {
			Order struct {
				Ref string `json:"ref"`
			} `json:"order"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal([]byte(create.body), &createResp))

	// Pay the order through the webhook so it can enter production.
	order, err := env.orders.GetByRef(t.Context(), createResp.Data.Order.Ref)
	require.NoError(t, err)
	require.Len(t, order.Payments, 1)

	providerRef := order.Payments[0].ProviderRef
	payload := []byte(`{"event":"charge.success","data":{"reference":"` + providerRef + `","amount":50000}}`)
	mac := hmac.New(sha512.New, []byte("secret"))
	_, _ = mac.Write(payload)
	signature := hex.EncodeToString(mac.Sum(nil))

	req, err := http.NewRequestWithContext(
		t.Context(), http.MethodPost, base+"/payments/webhook", strings.NewReader(string(payload)),
	)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Paystack-Signature", signature)

	res, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, res.StatusCode)
	_ = res.Body.Close()

	// Transition to in production.
	status := doJSON(t, http.MethodPost, base+"/admin/orders/"+createResp.Data.Order.Ref+"/status",
		`{"status":"in_production"}`, admin)
	require.Equal(t, http.StatusOK, status.status, "body: %s", status.body)

	updated, err := env.orders.GetByRef(t.Context(), createResp.Data.Order.Ref)
	require.NoError(t, err)
	assert.Equal(t, domain.OrderStatusInProduction, updated.Status)
}

func TestAdminUpdateOrderStatus_BlocksUnpaidProduction(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t, "boss@e25.com")
	admin := env.signIn(t, "boss@e25.com")
	designID := createDesign(t, env, admin)

	base := env.srv.URL + "/api/v1"
	create := doJSON(t, http.MethodPost, base+"/orders", fmt.Sprintf(`{
		"designId": %q,
		"bandLabel": "8",
		"delivery": "pickup",
		"customerPhone": "+233200000000",
		"email": "ama@example.com",
		"name": "Ama"
	}`, designID), nil)
	require.Equal(t, http.StatusCreated, create.status)

	var createResp struct {
		Data struct {
			Order struct {
				Ref string `json:"ref"`
			} `json:"order"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal([]byte(create.body), &createResp))

	status := doJSON(t, http.MethodPost, base+"/admin/orders/"+createResp.Data.Order.Ref+"/status",
		`{"status":"in_production"}`, admin)
	assert.Equal(t, http.StatusUnprocessableEntity, status.status)
}

func TestAdminUpdateOrderStatus_RejectsUnknownStatus(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t, "boss@e25.com")
	admin := env.signIn(t, "boss@e25.com")

	doJSON(t, http.MethodPost, env.srv.URL+"/api/v1/auth/request-link",
		`{"email":"ama@example.com","name":"Ama"}`, nil)

	ref := createCustomRequestOrder(t, env)

	base := env.srv.URL + "/api/v1"
	status := doJSON(t, http.MethodPost, base+"/admin/orders/"+ref+"/status",
		`{"status":"paid_lol"}`, admin)
	assert.Equal(t, http.StatusUnprocessableEntity, status.status, "body: %s", status.body)

	order, err := env.orders.GetByRef(t.Context(), ref)
	require.NoError(t, err)
	assert.Equal(t, domain.OrderStatusRequested, order.Status, "arbitrary strings must never be persisted")
}

func TestAdminUpdateOrderStatus_RejectsBooked(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t, "boss@e25.com")
	admin := env.signIn(t, "boss@e25.com")

	doJSON(t, http.MethodPost, env.srv.URL+"/api/v1/auth/request-link",
		`{"email":"ama@example.com","name":"Ama"}`, nil)

	ref := createCustomRequestOrder(t, env)

	base := env.srv.URL + "/api/v1"
	status := doJSON(t, http.MethodPost, base+"/admin/orders/"+ref+"/status",
		`{"status":"booked"}`, admin)
	assert.Equal(t, http.StatusUnprocessableEntity, status.status, "body: %s", status.body)

	order, err := env.orders.GetByRef(t.Context(), ref)
	require.NoError(t, err)
	assert.Equal(t, domain.OrderStatusRequested, order.Status)
	assert.False(t, order.IsPaid(), "the status endpoint must never make an order paid or booked")
}

func TestHandlePaymentWebhook_UnknownReferenceReturns200(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t, "boss@e25.com")
	base := env.srv.URL + "/api/v1"

	// A correctly signed event whose reference matches no order must be
	// acknowledged with 200 so Paystack does not retry for days.
	payload := []byte(`{"event":"charge.success","data":{"reference":"E25-UNKNOWN","amount":50000}}`)
	mac := hmac.New(sha512.New, []byte("secret"))
	_, _ = mac.Write(payload)
	signature := hex.EncodeToString(mac.Sum(nil))

	req, err := http.NewRequestWithContext(
		t.Context(), http.MethodPost, base+"/payments/webhook", strings.NewReader(string(payload)),
	)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Paystack-Signature", signature)

	res, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	defer func() { _ = res.Body.Close() }()

	assert.Equal(t, http.StatusOK, res.StatusCode)
}
