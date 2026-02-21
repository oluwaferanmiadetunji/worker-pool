package handler_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"worker-pool/api"
	"worker-pool/internal/config"
	"worker-pool/internal/handler"
	sqlc "worker-pool/internal/db/sqlc/generated"
	"worker-pool/internal/services"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockStore struct {
	createWebhookFn func(ctx context.Context, arg sqlc.CreateWebhookParams) (sqlc.WebhookEvent, error)
}

func (m *mockStore) ClaimNextWebhook(ctx context.Context) (sqlc.WebhookEvent, error) {
	return sqlc.WebhookEvent{}, nil
}

func (m *mockStore) CreateWebhook(ctx context.Context, arg sqlc.CreateWebhookParams) (sqlc.WebhookEvent, error) {
	if m.createWebhookFn != nil {
		return m.createWebhookFn(ctx, arg)
	}
	return sqlc.WebhookEvent{}, nil
}

func (m *mockStore) MarkWebhookDone(ctx context.Context, id uuid.UUID) (sqlc.WebhookEvent, error) {
	return sqlc.WebhookEvent{}, nil
}

func (m *mockStore) MarkWebhookFailed(ctx context.Context, arg sqlc.MarkWebhookFailedParams) (sqlc.WebhookEvent, error) {
	return sqlc.WebhookEvent{}, nil
}

func newTestHandler(store *mockStore) *handler.Handler {
	svc := services.NewWebhookService(store)
	return handler.NewHandler(config.Config{Port: "3333"}, svc)
}

func TestWebhookPayment_Success(t *testing.T) {
	e := echo.New()
	h := newTestHandler(&mockStore{})
	reqBody := `{"event_id":"evt_1","type":"payment.completed","amount":"5000","currency":"NGN","occurred_at":"2026-01-10T12:00:00Z"}`

	req := httptest.NewRequest(http.MethodPost, "/webhooks/payments", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h.WebhookPayment(c, api.WebhookPaymentParams{})

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp api.WebhookAckResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.True(t, resp.Ok)
}

func TestWebhookPayment_InvalidJSON(t *testing.T) {
	e := echo.New()
	h := newTestHandler(&mockStore{})
	req := httptest.NewRequest(http.MethodPost, "/webhooks/payments", strings.NewReader("{"))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h.WebhookPayment(c, api.WebhookPaymentParams{})

	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var resp api.ErrorBadRequest
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, 400, resp.Code)
	assert.Equal(t, "Invalid request body", resp.Message)
}

func TestWebhookPayment_MissingRequiredFields(t *testing.T) {
	e := echo.New()
	h := newTestHandler(&mockStore{})
	reqBody := `{"event_id":"evt_1","type":"payment.completed","amount":"","currency":"NGN","occurred_at":"2026-01-10T12:00:00Z"}`
	req := httptest.NewRequest(http.MethodPost, "/webhooks/payments", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h.WebhookPayment(c, api.WebhookPaymentParams{})

	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var resp api.ErrorBadRequest
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, 400, resp.Code)
	assert.Equal(t, "Missing required fields", resp.Message)
}

func TestWebhookPayment_ServiceFailure(t *testing.T) {
	e := echo.New()
	h := newTestHandler(&mockStore{
		createWebhookFn: func(ctx context.Context, arg sqlc.CreateWebhookParams) (sqlc.WebhookEvent, error) {
			return sqlc.WebhookEvent{}, errors.New("db down")
		},
	})
	reqModel := api.WebhookPaymentJSONRequestBody{
		EventId:    "evt_2",
		Type:       "payment.failed",
		Amount:     "2300",
		Currency:   "USD",
		OccurredAt: time.Date(2026, 1, 10, 12, 0, 0, 0, time.UTC),
	}
	bodyBytes, err := json.Marshal(reqModel)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/webhooks/payments", strings.NewReader(string(bodyBytes)))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err = h.WebhookPayment(c, api.WebhookPaymentParams{})

	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	var resp api.ErrorInternal
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, 500, resp.Code)
	assert.Equal(t, "Failed to process webhook", resp.Message)
}
