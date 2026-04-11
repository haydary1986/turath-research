# ══════════════════════════════════════════════════
# Turath Research — Makefile
# ══════════════════════════════════════════════════

.PHONY: help dev up down build logs migrate test lint

# ── Help ──────────────────────────────────────────
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

# ── Development ───────────────────────────────────
dev: ## Start all services in development mode
	docker compose up --build

up: ## Start all services in background
	docker compose up -d --build

down: ## Stop all services
	docker compose down

down-clean: ## Stop all services and remove volumes (WARNING: deletes data)
	docker compose down -v

logs: ## Show logs for all services
	docker compose logs -f

logs-backend: ## Show backend logs
	docker compose logs -f backend

logs-frontend: ## Show frontend logs
	docker compose logs -f frontend

logs-postgres: ## Show PostgreSQL logs
	docker compose logs -f postgres

# ── Build ─────────────────────────────────────────
build: ## Build all Docker images
	docker compose build

build-backend: ## Build backend image only
	docker compose build backend

build-frontend: ## Build frontend image only
	docker compose build frontend

# ── Database ──────────────────────────────────────
migrate: ## Run database migrations
	docker compose exec backend /server migrate up

migrate-down: ## Rollback last migration
	docker compose exec backend /server migrate down 1

psql: ## Open PostgreSQL shell
	docker compose exec postgres psql -U turath -d turath

# ── Testing ───────────────────────────────────────
test-backend: ## Run backend tests
	cd backend && go test ./... -v -cover

test-frontend: ## Run frontend tests
	cd frontend && npm test

test: test-backend test-frontend ## Run all tests

# ── Linting ───────────────────────────────────────
lint-backend: ## Lint backend code
	cd backend && golangci-lint run

lint-frontend: ## Lint frontend code
	cd frontend && npm run lint

lint: lint-backend lint-frontend ## Lint all code

# ── Status ────────────────────────────────────────
status: ## Show status of all services
	docker compose ps

health: ## Check health of all services
	@echo "=== PostgreSQL ===" && docker compose exec postgres pg_isready -U turath
	@echo "=== Redis ===" && docker compose exec redis redis-cli ping
	@echo "=== Backend ===" && curl -s http://localhost:8080/api/v1/health | head -c 200
	@echo ""
