-include build/.env.local
export

.PHONY: up down logs build-local run-local migrate test test-integration check-migration-runner integration-test-files fmt-check vet check harness-init integration-db-up integration-db-down help

COMPOSE ?= docker compose
PSQL ?= psql
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
	@$(PSQL) "$(DATABASE_URL)" -v ON_ERROR_STOP=1 -c 'CREATE TABLE IF NOT EXISTS schema_migrations (version TEXT PRIMARY KEY, applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW())'
	@for migration in migrations/*.up.sql; do \
		version=$$(basename "$$migration" .up.sql); \
		applied=$$($(PSQL) "$(DATABASE_URL)" -Atqc "SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE version = '$$version')"); \
		if [ "$$applied" = "t" ]; then \
			echo "Skipping $$migration (already applied)"; \
			continue; \
		fi; \
		case "$$version" in \
			0001_init) check_sql="SELECT to_regclass('public.items') IS NOT NULL" ;; \
			0002_add_is_favorite) check_sql="SELECT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema = 'public' AND table_name = 'items' AND column_name = 'is_favorite')" ;; \
			0003_add_note_shares) check_sql="SELECT to_regclass('public.note_shares') IS NOT NULL" ;; \
			0004_add_note_block_comments) check_sql="SELECT to_regclass('public.note_block_comments') IS NOT NULL" ;; \
			0005_add_note_block_comment_metadata) check_sql="SELECT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema = 'public' AND table_name = 'note_block_comments' AND column_name = 'parent_comment_id') AND EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema = 'public' AND table_name = 'note_block_comments' AND column_name = 'mentions')" ;; \
		esac; \
		existing=$$($(PSQL) "$(DATABASE_URL)" -Atqc "$$check_sql"); \
		if [ "$$existing" = "t" ]; then \
			echo "Recording existing $$migration"; \
		else \
			echo "Applying $$migration"; \
			$(PSQL) "$(DATABASE_URL)" -v ON_ERROR_STOP=1 -f "$$migration"; \
		fi; \
		$(PSQL) "$(DATABASE_URL)" -v ON_ERROR_STOP=1 -c "INSERT INTO schema_migrations (version) VALUES ('$$version') ON CONFLICT (version) DO NOTHING"; \
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
	$(COMPOSE) -f build/docker-compose.test.yml up -d --wait --force-recreate db-test

integration-db-down:
	$(COMPOSE) -f build/docker-compose.test.yml down

check-migration-runner:
	@$(MAKE) integration-db-up
	@trap '$(MAKE) integration-db-down' EXIT; \
		MIGRATION_DATABASE_URL=postgres://postgres:postgres@localhost:5432/notes_app_test?sslmode=disable \
		PSQL_COMMAND="$(CURDIR)/harness/scripts/tests/psql-test-client.sh" \
		bash harness/scripts/tests/check-migration-runner.sh

test-integration: integration-test-files integration-db-up
	@trap '$(MAKE) integration-db-down' EXIT; \
		MIGRATION_DATABASE_URL=postgres://postgres:postgres@localhost:5432/notes_app_test?sslmode=disable \
		PSQL_COMMAND="$(CURDIR)/harness/scripts/tests/psql-test-client.sh" \
		bash harness/scripts/tests/check-migration-runner.sh; \
		DATABASE_URL=postgres://postgres:postgres@localhost:55432/notes_app_test?sslmode=disable GOCACHE=$(GO_CACHE_DIR) go test -tags=integration ./...

check:
	bash harness/scripts/check-full-source-rules.sh

harness-init:
	bash .harness/harness/scripts/init-harness.sh
