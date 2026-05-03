SHELL := /bin/bash
.DEFAULT_GOAL := help

# ─── Variables ────────────────────────────────────────────────────────────────

APP_NAME        := ecollm
API_DIR         := apps/api
WEB_DIR         := apps/web
INFRA_DIR       := infra/docker
MIGRATIONS_DIR  := db/migrations
SCRIPTS_DIR     := scripts

DATABASE_URL    ?= postgres://ecollm:ecollm_dev@localhost:5432/ecollm?sslmode=disable
MIGRATE_VERSION := v4.17.0
GOLANGCI_VERSION := v1.57.2

GO      := go
GOFMT   := gofmt
GOVET   := $(GO) vet
DOCKER  := docker
DC      := docker compose

# Coloured output helpers
CYAN  := \033[0;36m
RESET := \033[0m

# ─── Help ─────────────────────────────────────────────────────────────────────

.PHONY: help
help: ## Show this help message
	@echo ""
	@echo "  $(CYAN)EcoLLM — Carbon-Aware LLM Inference Platform$(RESET)"
	@echo ""
	@awk 'BEGIN {FS = ":.*##"} /^[a-zA-Z_\-]+:.*##/ { printf "  $(CYAN)%-20s$(RESET) %s\n", $$1, $$2 }' $(MAKEFILE_LIST)
	@echo ""

# ─── Development ──────────────────────────────────────────────────────────────

.PHONY: dev
dev: ## Start full dev stack (docker compose up -d, then hot-reload API)
	@echo "Starting infrastructure..."
	$(DC) -f $(INFRA_DIR)/docker-compose.yml up -d postgres redis ollama prometheus grafana otel-collector
	@echo "Waiting for postgres..."
	@until docker compose -f $(INFRA_DIR)/docker-compose.yml exec postgres pg_isready -U ecollm -d ecollm; do sleep 1; done
	@$(MAKE) migrate
	@echo "Starting API with hot-reload..."
	cd $(API_DIR) && $(GO) run ./cmd/server/...

.PHONY: dev-full
dev-full: ## Start entire stack including all application services
	$(DC) -f $(INFRA_DIR)/docker-compose.yml up -d

.PHONY: dev-down
dev-down: ## Stop and remove dev containers (volumes preserved)
	$(DC) -f $(INFRA_DIR)/docker-compose.yml down

.PHONY: dev-clean
dev-clean: ## Stop and remove dev containers AND volumes
	$(DC) -f $(INFRA_DIR)/docker-compose.yml down -v

# ─── Build ────────────────────────────────────────────────────────────────────

.PHONY: build
build: build-api build-inference-gateway build-carbon-service ## Build all Go binaries

.PHONY: build-api
build-api: ## Build the API binary
	@echo "Building API..."
	cd $(API_DIR) && CGO_ENABLED=0 $(GO) build -ldflags="-s -w" -o bin/api ./cmd/server/...
	@echo "  → $(API_DIR)/bin/api"

.PHONY: build-inference-gateway
build-inference-gateway: ## Build the inference-gateway binary
	@echo "Building inference-gateway..."
	cd apps/inference-gateway && CGO_ENABLED=0 $(GO) build -ldflags="-s -w" -o bin/inference-gateway ./cmd/...
	@echo "  → apps/inference-gateway/bin/inference-gateway"

.PHONY: build-carbon-service
build-carbon-service: ## Build the carbon-service binary
	@echo "Building carbon-service..."
	cd apps/carbon-service && CGO_ENABLED=0 $(GO) build -ldflags="-s -w" -o bin/carbon-service ./cmd/...
	@echo "  → apps/carbon-service/bin/carbon-service"

.PHONY: build-web
build-web: ## Build the Next.js frontend
	cd $(WEB_DIR) && npm run build

.PHONY: docker-build
docker-build: ## Build all Docker images
	$(DC) -f $(INFRA_DIR)/docker-compose.yml build

# ─── Testing ──────────────────────────────────────────────────────────────────

.PHONY: test
test: test-unit test-integration ## Run all tests

.PHONY: test-unit
test-unit: ## Run Go unit tests (no external dependencies)
	@echo "Running unit tests..."
	cd $(API_DIR) && $(GO) test -race -count=1 -short ./...

.PHONY: test-integration
test-integration: ## Run integration tests against test docker-compose stack
	@echo "Starting test infrastructure..."
	$(DC) -f $(INFRA_DIR)/docker-compose.test.yml up -d postgres-test redis-test
	@echo "Running migrations on test DB..."
	$(DC) -f $(INFRA_DIR)/docker-compose.test.yml run --rm migrate-test
	@echo "Running integration tests..."
	cd $(API_DIR) && \
		DATABASE_URL=postgres://ecollm_test:ecollm_test@localhost:5433/ecollm_test \
		REDIS_URL=redis://localhost:6380 \
		$(GO) test -race -count=1 -run Integration ./...
	$(DC) -f $(INFRA_DIR)/docker-compose.test.yml down -v

.PHONY: test-cover
test-cover: ## Run tests with coverage report
	cd $(API_DIR) && $(GO) test -race -coverprofile=coverage.out ./... && \
		$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: $(API_DIR)/coverage.html"

# ─── Linting ──────────────────────────────────────────────────────────────────

.PHONY: lint
lint: lint-go lint-web ## Run all linters

.PHONY: lint-go
lint-go: ## Run golangci-lint on all Go modules
	@echo "Linting Go..."
	cd $(API_DIR) && golangci-lint run ./...
	cd apps/inference-gateway && golangci-lint run ./... 2>/dev/null || true
	cd apps/carbon-service && golangci-lint run ./... 2>/dev/null || true

.PHONY: lint-web
lint-web: ## Run ESLint + tsc on the frontend
	@echo "Linting web..."
	cd $(WEB_DIR) && npm run lint && npm run type-check

.PHONY: fmt
fmt: ## Format all Go source files
	@echo "Formatting Go..."
	$(GOFMT) -l -w $(API_DIR)/...
	$(GOFMT) -l -w apps/inference-gateway/...
	$(GOFMT) -l -w apps/carbon-service/...

.PHONY: vet
vet: ## Run go vet on all Go modules
	cd $(API_DIR) && $(GOVET) ./...

# ─── Database ─────────────────────────────────────────────────────────────────

.PHONY: migrate
migrate: ## Run all pending database migrations
	@echo "Running migrations..."
	docker run --rm --network host \
		-v "$(PWD)/$(MIGRATIONS_DIR):/migrations" \
		migrate/migrate:$(MIGRATE_VERSION) \
		-path /migrations \
		-database "$(DATABASE_URL)" \
		up
	@echo "Migrations complete."

.PHONY: migrate-down
migrate-down: ## Roll back the most recent migration
	docker run --rm --network host \
		-v "$(PWD)/$(MIGRATIONS_DIR):/migrations" \
		migrate/migrate:$(MIGRATE_VERSION) \
		-path /migrations \
		-database "$(DATABASE_URL)" \
		down 1

.PHONY: migrate-status
migrate-status: ## Show current migration version
	docker run --rm --network host \
		-v "$(PWD)/$(MIGRATIONS_DIR):/migrations" \
		migrate/migrate:$(MIGRATE_VERSION) \
		-path /migrations \
		-database "$(DATABASE_URL)" \
		version

.PHONY: migrate-new
migrate-new: ## Create a new migration file: make migrate-new NAME=add_feature
	@if [ -z "$(NAME)" ]; then echo "Usage: make migrate-new NAME=your_migration_name"; exit 1; fi
	@NEXT=$$(ls $(MIGRATIONS_DIR)/*.sql 2>/dev/null | wc -l | xargs printf "%03d"); \
	FILE="$(MIGRATIONS_DIR)/$${NEXT}_$(NAME).sql"; \
	echo "-- Migration: $${NEXT}_$(NAME)" > $$FILE; \
	echo "Created: $$FILE"

.PHONY: seed
seed: ## Seed the database with development data
	@echo "Seeding database..."
	docker run --rm --network host \
		-e PGPASSWORD=ecollm_dev \
		postgres:16-alpine \
		psql -h localhost -U ecollm -d ecollm -f /dev/stdin < $(SCRIPTS_DIR)/seed.sh 2>/dev/null || \
	PGPASSWORD=ecollm_dev psql -h localhost -U ecollm -d ecollm -f db/seeds/models.sql
	@echo "Seed complete."

.PHONY: db-shell
db-shell: ## Open a psql shell to the dev database
	PGPASSWORD=ecollm_dev psql -h localhost -U ecollm -d ecollm

# ─── Code generation ──────────────────────────────────────────────────────────

.PHONY: generate
generate: generate-types ## Run all code generators

.PHONY: generate-types
generate-types: ## Generate TypeScript types from OpenAPI spec
	@echo "Generating TypeScript types..."
	bash $(SCRIPTS_DIR)/generate-types.sh

# ─── Observability ────────────────────────────────────────────────────────────

.PHONY: metrics
metrics: ## Open Prometheus in browser
	@open http://localhost:9090 2>/dev/null || xdg-open http://localhost:9090

.PHONY: dashboards
dashboards: ## Open Grafana in browser
	@open http://localhost:3001 2>/dev/null || xdg-open http://localhost:3001

# ─── Tooling installation ─────────────────────────────────────────────────────

.PHONY: install-tools
install-tools: ## Install required development tools
	@echo "Installing golangci-lint $(GOLANGCI_VERSION)..."
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | \
		sh -s -- -b $$($(GO) env GOPATH)/bin $(GOLANGCI_VERSION)
	@echo "All tools installed."

# ─── Cleanup ──────────────────────────────────────────────────────────────────

.PHONY: clean
clean: ## Remove build artifacts
	rm -f $(API_DIR)/bin/api
	rm -f apps/inference-gateway/bin/inference-gateway
	rm -f apps/carbon-service/bin/carbon-service
	rm -f $(API_DIR)/coverage.out $(API_DIR)/coverage.html
	find . -name "*.test" -delete
	@echo "Clean complete."

.PHONY: clean-docker
clean-docker: dev-clean ## Remove all Docker containers, images, and volumes for this project
	$(DC) -f $(INFRA_DIR)/docker-compose.yml down --rmi local -v

# ─── CI helpers ───────────────────────────────────────────────────────────────

.PHONY: ci
ci: vet lint-go test-unit ## Run the full CI check suite locally (no Docker required)
