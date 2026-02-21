DB_URL=postgresql://postgres:password@localhost:5432/worker-pool?sslmode=disable

new_migration:
	migrate create -ext sql -dir internal/db/sqlc/migrations $(name)

sqlc:
	sqlc generate

# Install dependencies: go mod tidy & go mod vendor
install:
	go mod tidy && go mod vendor

# Run the server
run:
	air

# Run the load simulator (sends random bursts of payment webhooks to the API; uses API_URL from .env)
loadsim:
	go run ./cmd/loadsim

# Run the worker pool (processes webhooks from DB; optional: WORKER_POOL_SIZE, WORKER_POLL_INTERVAL, WORKER_PROCESS_DELAY)
workerpool:
	go run ./cmd/worker-pool

# Start services concurrently using Air
start:
	docker compose up --build -d
	

stop:
	docker compose down --remove-orphans -v

lint:
	golangci-lint fmt

test:
	go test -mod=mod ./... -v


# Generate mocks using mockery
mocks:
	@echo "Generating mocks..."
	@go run github.com/vektra/mockery/v2@latest --config .mockery.yaml
	
# OpenAPI code generation
openapi-generate:
	@echo "Generating API code from OpenAPI spec..."
	@go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest \
		-config api/oapi-codegen.yaml \
		api/openapi.yaml

# Validate OpenAPI spec
openapi-validate:
	@echo "Validating OpenAPI spec..."
	@go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest \
		-generate types \
		-package api \
		api/openapi.yaml > /dev/null && \
		echo "✓ OpenAPI spec is valid" || \
		(echo "✗ OpenAPI spec has errors" && exit 1)

# Generate and validate OpenAPI
openapi: openapi-validate openapi-generate

.PHONY: install start new_migration stop sqlc run lint test mocks openapi openapi-generate openapi-validate workerpool