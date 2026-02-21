package services_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
	"worker-pool/api"
	sqlc "worker-pool/internal/db/sqlc/generated"
	"worker-pool/internal/services"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockStore struct {
	createWebhookFn     func(ctx context.Context, arg sqlc.CreateWebhookParams) (sqlc.WebhookEvent, error)
	createWebhookCalls  int
	lastCreateWebhookArg sqlc.CreateWebhookParams
}

func (m *mockStore) ClaimNextWebhook(ctx context.Context) (sqlc.WebhookEvent, error) {
	return sqlc.WebhookEvent{}, nil
}

func (m *mockStore) CreateWebhook(ctx context.Context, arg sqlc.CreateWebhookParams) (sqlc.WebhookEvent, error) {
	m.createWebhookCalls++
	m.lastCreateWebhookArg = arg
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

func TestProcessPaymentWebhook_Success(t *testing.T) {
	store := &mockStore{}
	svc := services.NewWebhookService(store)
	req := api.WebhookPaymentJSONRequestBody{
		EventId:    "evt_123",
		Type:       "payment.completed",
		Amount:     "5000",
		Currency:   "NGN",
		OccurredAt: time.Date(2026, 1, 10, 12, 0, 0, 0, time.UTC),
	}

	err := svc.ProcessPaymentWebhook(req)

	require.NoError(t, err)
	assert.Equal(t, 1, store.createWebhookCalls)
	assert.Equal(t, req.EventId, store.lastCreateWebhookArg.EventID)
	require.NotNil(t, store.lastCreateWebhookArg.Type)
	assert.Equal(t, req.Type, *store.lastCreateWebhookArg.Type)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(store.lastCreateWebhookArg.Payload, &payload))
	assert.Equal(t, req.EventId, payload["event_id"])
	assert.Equal(t, req.Type, payload["type"])
	assert.Equal(t, req.Amount, payload["amount"])
	assert.Equal(t, req.Currency, payload["currency"])
}

func TestProcessPaymentWebhook_StoreError(t *testing.T) {
	store := &mockStore{
		createWebhookFn: func(ctx context.Context, arg sqlc.CreateWebhookParams) (sqlc.WebhookEvent, error) {
			return sqlc.WebhookEvent{}, errors.New("db write failed")
		},
	}
	svc := services.NewWebhookService(store)
	req := api.WebhookPaymentJSONRequestBody{
		EventId:    "evt_123",
		Type:       "payment.failed",
		Amount:     "900",
		Currency:   "USD",
		OccurredAt: time.Now().UTC(),
	}

	err := svc.ProcessPaymentWebhook(req)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "create webhook")
	assert.Equal(t, 1, store.createWebhookCalls)
}
