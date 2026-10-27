# AfriOracle — run, lint, test, Docker and demo
#
# Usage: make <target>   (run `make help` for the list)
# Local run defaults to the mock AI path: no META_API_KEY needed.

.DEFAULT_GOAL := help

SHELL        := /bin/bash
BACKEND_DIR  := backend
FRONTEND_DIR := frontend
BASE         ?= http://localhost:8000
BACKEND_PORT ?= 8000
GO           ?= go
NPM          ?= npm

.PHONY: help install deps run dev backend frontend build env demo check ci \
        lint vet fmt fmt-check typecheck test test-backend test-frontend \
        docker-build docker-up docker-down docker-logs docker-ps clean

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}'

## --- Install / environment -------------------------------------------------

install: deps ## Install all project dependencies (alias of deps)
deps: ## Install Go module + frontend npm dependencies
	cd $(BACKEND_DIR) && $(GO) mod download
	cd $(FRONTEND_DIR) && $(NPM) install

env: ## Create .env from .env.example if it does not exist
	@if [ -f .env ]; then echo ".env already exists"; else cp .env.example .env && echo ".env created from .env.example (mock mode: no META_API_KEY set)"; fi

## --- Run -------------------------------------------------------------------

run: backend ## Run the backend locally (mock AI, :8000) — see `make dev` for the frontend
backend: ## Run backend: go run (:8000, mock AI when META_API_KEY unset)
	cd $(BACKEND_DIR) && PORT=$(BACKEND_PORT) $(GO) run .

frontend: ## Run frontend dev server (:3000, talks to :8000)
	cd $(FRONTEND_DIR) && $(NPM) run dev

dev: ## Print how to run backend + frontend in two terminals
	@echo "Terminal 1:  make backend"
	@echo "Terminal 2:  make frontend"
	@echo "Backend:  http://localhost:$(BACKEND_PORT)/api/state"
	@echo "Frontend: http://localhost:3000"

demo: ## Run the demo contract against a running backend (make backend first)
	BASE=$(BASE) bash scripts/demo.sh

## --- Quality: lint / format / typecheck / test -----------------------------

lint: vet fmt-check typecheck ## Backend vet + fmt-check, frontend TypeScript check
	@echo "lint OK"

vet: ## go vet on the backend
	cd $(BACKEND_DIR) && $(GO) vet ./...

fmt: ## gofmt -w on the backend
	cd $(BACKEND_DIR) && gofmt -w .

fmt-check: ## Fail if any backend Go file is not gofmt-clean
	@cd $(BACKEND_DIR) && files="$$(gofmt -l .)"; \
	if [ -n "$$files" ]; then echo "not gofmt-clean: $$files" >&2; exit 1; fi

typecheck: ## TypeScript check of the frontend (no emit; requires `make deps` first)
	@if [ -x $(FRONTEND_DIR)/node_modules/.bin/tsc ]; then \
		cd $(FRONTEND_DIR) && ./node_modules/.bin/tsc --noEmit; \
	else \
		echo "frontend deps missing — run: make deps" >&2; exit 1; \
	fi

test: test-backend test-frontend ## Run backend tests + frontend build (frontend has no test suite)
test-backend: ## go test (race) on the backend
	cd $(BACKEND_DIR) && $(GO) test -race ./...

test-frontend: ## Build the frontend (only automated frontend check available)
	cd $(FRONTEND_DIR) && $(NPM) run build

build: ## Build backend binary + frontend production build
	cd $(BACKEND_DIR) && $(GO) build -o afrioracle .
	cd $(FRONTEND_DIR) && $(NPM) run build

check: lint test build ## lint + test + build (what local CI runs)

## --- Docker / CI -----------------------------------------------------------

ci: deps check docker-smoke ## Full local CI: deps, lint, test, build, compose smoke + demo

docker-build: ## Build compose images
	docker compose build

docker-up: ## Start compose stack detached (backend :8000, frontend :3000)
	docker compose up -d --build

docker-down: ## Stop compose stack and remove volumes
	docker compose down -v

docker-logs: ## Follow compose logs
	docker compose logs -f

docker-ps: ## Show compose services
	docker compose ps

docker-smoke: ## Build up, wait for /api/state, check frontend, run demo, tear down
	docker compose up -d --build
	@trap 'docker compose down -v' EXIT; \
	for i in $$(seq 1 30); do curl -sf http://localhost:8000/api/state >/dev/null && break || sleep 2; done; \
	curl -sf http://localhost:8000/api/state; \
	for i in $$(seq 1 30); do curl -sf http://localhost:3000 >/dev/null && break || sleep 2; done; \
	curl -sf http://localhost:3000 >/dev/null; \
	echo; BASE=http://localhost:8000 bash scripts/demo.sh; \
	echo "docker smoke OK"; \
	docker compose down -v; \
	trap - EXIT

## --- Cleanup ----------------------------------------------------------------

clean: ## Remove build artifacts (binary, .next, node_modules kept)
	rm -f $(BACKEND_DIR)/afrioracle
	rm -rf $(FRONTEND_DIR)/.next
