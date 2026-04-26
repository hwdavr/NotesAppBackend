-include .env.local
export

.PHONY: build run migrate help

# Get the root directory relative to this Makefile
ROOT_DIR := $(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))/..

help:
	@echo "Available targets:"
	@echo "  build        - Build the application binary at the root"
	@echo "  run          - Run the application locally using .env.local"
	@echo "  migrate      - Run database migrations using DATABASE_URL"
	@echo "  docker-build - Build the Docker image"
	@echo "  docker-run   - Run the Docker container using .env.local"
	@echo "  up           - Start the entire stack (App + DB) using Docker Compose"
	@echo "  down         - Stop the entire stack using Docker Compose"
	@echo "  logs         - Follow the app container logs using Docker Compose"

build:
	@echo "Building application..."
	cd $(ROOT_DIR) && go build -o app ./cmd/server

run:
	@echo "Running application locally..."
	cd $(ROOT_DIR) && go run ./cmd/server

migrate:
	@echo "Running migrations..."
	psql $(DATABASE_URL) -f $(ROOT_DIR)/migrations/0001_init.up.sql

docker-build:
	@echo "Building Docker image..."
	docker build -t notes-backend -f Dockerfile $(ROOT_DIR)

docker-run:
	@echo "Running Docker container..."
	@echo "Note: If using local DB, ensure DATABASE_URL uses host.docker.internal or your host IP."
	@echo "Note: For Android AVD, access the server via http://10.0.2.2:8080"
	docker run --rm -it -p 8080:8080 --env-file .env.local notes-backend

up:
	@echo "Starting the stack..."
	docker-compose up --build -d

down:
	@echo "Stopping the stack..."
	docker-compose down

logs:
	@echo "Following app logs..."
	docker-compose logs -f app
