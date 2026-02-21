package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"worker-pool/api"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const (
	minBurstSize = 500
	maxBurstSize = 1000
	minInterval  = 0
	maxInterval  = 10 * time.Second
)

var (
	eventTypes = []string{"payment.completed", "payment.pending", "payment.failed", "payment.refunded"}
	currencies = []string{"NGN", "USD", "GBP", "EUR"}
	baseURL    = "http://localhost:3333"
)

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	log.Info().
		Str("base_url", baseURL).
		Int("min_burst", minBurstSize).
		Int("max_burst", maxBurstSize).
		Str("interval", fmt.Sprintf("%v–%v", minInterval, maxInterval)).
		Msg("Load simulator started; send SIGINT/SIGTERM to stop")

	for {
		interval := minInterval + time.Duration(rand.IntN(int(maxInterval-minInterval)))
		if interval > 0 {
			log.Debug().Dur("sleep", interval).Msg("Waiting until next burst")
			select {
			case <-ctx.Done():
				log.Info().Msg("Load simulator stopped")
				return
			case <-time.After(interval):
			}
		}

		n := minBurstSize + rand.IntN(maxBurstSize-minBurstSize+1)
		sendBurst(ctx, baseURL, n)
	}
}

func sendBurst(ctx context.Context, baseURL string, count int) {
	url := baseURL + "/webhooks/payments"
	var wg sync.WaitGroup
	wg.Add(count)

	start := time.Now()
	for range count {
		go func() {
			defer wg.Done()
			if err := sendOne(ctx, url); err != nil {
				log.Debug().Err(err).Msg("Webhook request failed")
			}
		}()
	}
	wg.Wait()
	elapsed := time.Since(start)

	log.Info().
		Int("requests", count).
		Dur("elapsed", elapsed).
		Msg("Burst completed")
}

func sendOne(ctx context.Context, url string) error {
	payload := randomPaymentRequest()
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Webhook-Signature", randomSignature())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	log.Info().
		Int("status", resp.StatusCode).
		Str("response", string(respBody)).
		Msg("Webhook response")

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status %d", resp.StatusCode)
	}
	return nil
}

func randomPaymentRequest() api.WebhookPaymentRequest {
	amount := fmt.Sprintf("%d", 100+rand.IntN(1_000_000))
	return api.WebhookPaymentRequest{
		EventId:    "evt_" + randomHex(12),
		Type:       eventTypes[rand.IntN(len(eventTypes))],
		Amount:     amount,
		Currency:   currencies[rand.IntN(len(currencies))],
		OccurredAt: time.Now().UTC().Add(-time.Duration(rand.IntN(3600)) * time.Second),
	}
}

func randomSignature() string {
	return randomHex(32)
}

func randomHex(n int) string {
	const hex = "0123456789abcdef"
	b := make([]byte, n)
	for i := range b {
		b[i] = hex[rand.IntN(len(hex))]
	}
	return string(b)
}
