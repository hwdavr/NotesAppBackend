-include build/.env.local
export

.PHONY: up down logs build-local run-local migrate test test-integration integration-test-files fmt-check vet check harness-init integration-db-up integration-db-down help

COMPOSE ?= docker compose
GO_CACHE_DIR ?= /tmp/notes-app-backend-go-cache

help:
	@echo "Available targets:"
	@echo "  up          - Build and start the entire stack (Docker)"
	@echo "  down        - Stop the stack (Docker)"
	@echo "  logs        - View app logs (Docker)"
	@echo "  migrate     - Run all forward database migrations"
	@echo "  build-local - Build binary (requires local Go)"
	@echo "  run-local   - Run locally (requires local Go)"
	@echo "  test        - Run deterministic Go tests"
	@echo "  test-integration - Run integration-tagged tests against disposable Postgres"
	@echo "  check       - Run the complete backend harness gate"
	@echo "  harness-init - Create harness entry-point symlinks and run checks"

up:
	@echo "Starting the stack (Building inside Docker)..."
	$(COMPOSE) -f build/docker-compose.yml up --build -d

down:
	@echo "Stopping the stack..."
	$(COMPOSE) -f build/docker-compose.yml down

logs:
	@echo "Following app logs..."
	$(COMPOSE) -f build/docker-compose.yml logs -f app

migrate:
	@echo "Running migrations..."
	@test -n "$(DATABASE_URL)" || (echo "DATABASE_URL is required" >&2; exit 1)
	@for migration in migrations/*.up.sql; do \
		echo "Applying $$migration"; \
		psql "$(DATABASE_URL)" -v ON_ERROR_STOP=1 -f "$$migration"; \
	done

build-local:
	@echo "Building binary locally (requires Go on host)..."
	go build -o app ./cmd/server

run-local:
	@echo "Running locally (requires Go on host)..."
	go run ./cmd/server

fmt-check:
	@test -z "$$(gofmt -l $$(find . -type f -name '*.go' -not -path './vendor/*' -not -path './.git/*' -not -path './.kilo/*' -print))"

vet:
	GOCACHE=$(GO_CACHE_DIR) go vet ./...

test:
	GOCACHE=$(GO_CACHE_DIR) go test ./...

integration-test-files:
	@test -n "$$(rg -l '^//go:build integration' --glob '*_test.go' --glob '!**/.kilo/**' .)" || (echo "No integration-tagged Go tests found" >&2; exit 1)

integration-db-up:
	$(COMPOSE) -f build/docker-compose.test.yml up -d --wait db-test

integration-db-down:
	$(COMPOSE) -f build/docker-compose.test.yml down

test-integration: integration-test-files integration-db-up
	DATABASE_URL=postgres://postgres:postgres@localhost:55432/notes_app_test?sslmode=disable GOCACHE=$(GO_CACHE_DIR) go test -tags=integration ./...

check:
	bash harness/scripts/check-full-source-rules.sh

harness-init:
	bash .harness/harness/scripts/init-harness.sh
