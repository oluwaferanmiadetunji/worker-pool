package services

import (
	"context"
	"encoding/json"
	"fmt"
	"worker-pool/api"
	"worker-pool/internal/db"
	sqlc "worker-pool/internal/db/sqlc/generated"
	"github.com/rs/zerolog/log"
)

type WebhookService struct {
	store db.Store
}

func NewWebhookService(store db.Store) *WebhookService {
	return &WebhookService{
		store: store,
	}
}

func (s *WebhookService) ProcessPaymentWebhook(req api.WebhookPaymentJSONRequestBody) error {
	log.Info().Str("request", fmt.Sprintf("%+v", req)).
		Msg("Processing payment webhook data")

	payload, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal webhook payload: %w", err)
	}

	_, err = s.store.CreateWebhook(context.Background(), sqlc.CreateWebhookParams{
		EventID: req.EventId,
		Type:    &req.Type,
		Payload: payload,
	})

	if err != nil {
		return fmt.Errorf("create webhook: %w", err)
	}

	return nil
}
