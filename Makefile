.PHONY: help build run test clean docker-build docker-up docker-down docker-logs migrate

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the application
	go build -o bin/app ./cmd/app

run: ## Run the application locally
	go run ./cmd/app

test: ## Run tests
	go test -v -race -coverprofile=coverage.out ./...

test-coverage: test ## Run tests and show coverage
	go tool cover -html=coverage.out

clean: ## Clean build artifacts
	rm -rf bin/
	rm -f coverage.out

lint: ## Run linter
	golangci-lint run ./...

docker-build: ## Build Docker image
	docker compose build

docker-up: ## Start services
	docker compose up -d

docker-down: ## Stop services
	docker compose down

docker-logs: ## Show logs
	docker compose logs -f app

docker-restart: docker-down docker-build docker-up ## Rebuild and restart

migrate: ## Run database migrations
	@echo "Applying migrations..."
	psql "$$DATABASE_URL" -f migrations/001_init.sql

deps: ## Download dependencies
	go mod download
	go mod tidy

vendor: ## Vendor dependencies
	go mod vendor

.DEFAULT_GOAL := help
