package handler

import (
	"worker-pool/api"
	"worker-pool/internal/config"
	"worker-pool/internal/services"

	"github.com/labstack/echo/v4"
)

type Handler struct {
	config         config.Config
	webhookService *services.WebhookService
}

func NewHandler(cfg config.Config, webhookService *services.WebhookService) *Handler {
	return &Handler{
		config:         cfg,
		webhookService: webhookService,
	}
}

func (h *Handler) WebhookPayment(ctx echo.Context, params api.WebhookPaymentParams) error {
	var req api.WebhookPaymentJSONRequestBody

	if err := ctx.Bind(&req); err != nil {
		return ctx.JSON(400, api.ErrorBadRequest{
			Code:    400,
			Message: "Invalid request body",
		})
	}

	if req.Amount == "" || req.Currency == "" || req.EventId == "" || req.Type == "" {
		return ctx.JSON(400, api.ErrorBadRequest{
			Code:    400,
			Message: "Missing required fields",
		})
	}

	if err := h.webhookService.ProcessPaymentWebhook(req); err != nil {
		return ctx.JSON(500, api.ErrorInternal{
			Code:    500,
			Message: "Failed to process webhook",
		})
	}

	return ctx.JSON(200, api.WebhookAckResponse{Ok: true})
}
