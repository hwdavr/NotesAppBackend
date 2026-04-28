-include build/.env.local
export

.PHONY: up down logs build-local run-local migrate help

help:
	@echo "Available targets:"
	@echo "  up          - Build and start the entire stack (Docker)"
	@echo "  down        - Stop the stack (Docker)"
	@echo "  logs        - View app logs (Docker)"
	@echo "  migrate     - Run database migrations"
	@echo "  build-local - Build binary (requires local Go)"
	@echo "  run-local   - Run locally (requires local Go)"

up:
	@echo "Starting the stack (Building inside Docker)..."
	docker-compose -f build/docker-compose.yml up --build -d

down:
	@echo "Stopping the stack..."
	docker-compose -f build/docker-compose.yml down

logs:
	@echo "Following app logs..."
	docker-compose -f build/docker-compose.yml logs -f app

migrate:
	@echo "Running migrations..."
	psql $(DATABASE_URL) -f ./migrations/0001_init.up.sql

build-local:
	@echo "Building binary locally (requires Go on host)..."
	go build -o app ./cmd/server

run-local:
	@echo "Running locally (requires Go on host)..."
	go run ./cmd/server
