SHELL := bash
.DEFAULT_GOAL := help

GO ?= go
COMPOSE := docker compose --env-file .env -f deploy/docker-compose.yml
TEMPL := $(GO) tool templ

ifneq (,$(wildcard ./.env))
include .env
export
endif

.PHONY: help env templ fmt vet test test-integration check build run \
	migrate-up migrate-down migrate-status dev-up dev-down dev-reset dev-logs clean

help: ## Show the available targets.
	@awk 'BEGIN {FS = ":.*## "} /^[a-zA-Z_-]+:.*## / {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

env: ## Create .env from .env.example when it does not exist.
	@test -f .env || (cp .env.example .env && echo "created .env")

templ: ## Generate Go code from Templ templates.
	$(TEMPL) generate

fmt: ## Format Go and Templ sources.
	$(GO) fmt ./...
	$(TEMPL) fmt .

vet: ## Run go vet.
	$(GO) vet ./...

test: ## Run the unit tests.
	$(GO) test ./...

test-integration: ## Run tests that need PostgreSQL and Garage (uses .env).
	MOADA_TEST_DATABASE_URL="$(MOADA_DATABASE_URL)" \
	MOADA_TEST_S3_ENDPOINT="$(MOADA_S3_ENDPOINT)" \
	MOADA_TEST_S3_ACCESS_KEY="$(MOADA_S3_ACCESS_KEY)" \
	MOADA_TEST_S3_SECRET_KEY="$(MOADA_S3_SECRET_KEY)" \
	MOADA_TEST_S3_BUCKET="$(MOADA_S3_BUCKET)" \
	MOADA_TEST_S3_REGION="$(MOADA_S3_REGION)" \
	MOADA_TEST_S3_USE_SSL="$(MOADA_S3_USE_SSL)" \
	$(GO) test ./... -count=1

check: fmt vet test ## Format, vet and test the project.

build: templ ## Build the application binaries.
	mkdir -p bin
	$(GO) build -trimpath -o bin/moada ./cmd/moada
	$(GO) build -trimpath -o bin/moada-migrate ./cmd/migrate

run: templ ## Run the web application.
	$(GO) run ./cmd/moada

migrate-up: ## Apply all pending database migrations.
	$(GO) run ./cmd/migrate up

migrate-down: ## Roll back one database migration.
	$(GO) run ./cmd/migrate down

migrate-status: ## Show database migration status.
	$(GO) run ./cmd/migrate status

dev-up: env ## Start PostgreSQL and Garage.
	$(COMPOSE) up -d --wait

dev-down: ## Stop PostgreSQL and Garage, keeping their data.
	$(COMPOSE) down

dev-reset: ## Stop the services and delete all local data. Destructive.
	$(COMPOSE) down -v

dev-logs: ## Follow the infrastructure logs.
	$(COMPOSE) logs -f

clean: ## Remove build output.
	rm -rf bin coverage.out
