# Worker Pool

This project ingests payment webhook events through an HTTP API, stores them in PostgreSQL, and processes them asynchronously using a configurable worker pool.

It is designed to demonstrate a common production pattern:
- accept webhook requests quickly,
- persist them durably,
- process in the background with retries/status tracking.

## How It Works

1. The API server receives `POST /webhooks/payments`.
2. The webhook payload is validated and written to the `webhook_events` table with `status='received'`.
3. Worker processes poll for the next available webhook, claim it, process it, then mark it as:
   - `done` on success, or
   - `failed` with `last_error` on failure.
4. DB migrations are run automatically when the server or worker starts.

## Tech Stack

- Go
- PostgreSQL (primary queue/state store)
- Echo (HTTP server)
- sqlc + golang-migrate

## Project Structure

- `cmd/server` - HTTP API server
- `cmd/worker-pool` - background workers
- `cmd/loadsim` - load simulator that sends random webhook bursts
- `internal/services` - webhook persistence logic
- `internal/db/sqlc/migrations` - database migrations
- `api/openapi.yaml` - API contract

## Prerequisites

- Go `1.25+`
- PostgreSQL running locally (or reachable by connection string)

## Environment Variables

The app loads variables from `.env` in non-production mode.

Required:
- `PORT` (example: `3333`)
- `DB_URL` (example: `postgresql://postgres:password@localhost:5432/worker-pool?sslmode=disable`)

Optional:
- `WORKER_POOL_SIZE` (default: `5`)
- `WORKER_POLL_INTERVAL` (default: `2s`)
- `WORKER_PROCESS_DELAY` (default: `100ms`)

## Setup

1. Install dependencies:
   ```bash
   make install
   ```
2. Ensure PostgreSQL is running and the database in `DB_URL` exists.

## Run the Project

Open separate terminals for the server and worker pool.

1. Start API server:
   ```bash
   go run ./cmd/server
   ```
   or:
   ```bash
   make run
   ```

2. Start worker pool:
   ```bash
   go run ./cmd/worker-pool
   ```
   or:
   ```bash
   make workerpool
   ```

3. (Optional) Start load simulator:
   ```bash
   make loadsim
   ```

## Test the API Manually

```bash
curl -X POST http://localhost:3333/webhooks/payments \
  -H "Content-Type: application/json" \
  -H "X-Webhook-Signature: test-signature" \
  -d '{
    "event_id": "evt_12345",
    "type": "payment.completed",
    "amount": "5000",
    "currency": "NGN",
    "occurred_at": "2026-01-10T12:00:00Z"
  }'
```

Expected response:

```json
{"ok": true}
```

## Useful Commands

- `make test` - run tests
- `make lint` - format/lint command
- `make sqlc` - regenerate sqlc queries
- `make openapi` - validate and regenerate OpenAPI code

## Notes

- Migrations are applied automatically on startup by the app.
