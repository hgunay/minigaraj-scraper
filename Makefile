# ============================================================================
# MiniGaraj Scraper - Makefile
# ============================================================================

.PHONY: help build run dev test test-cover lint tidy \
        docker-up docker-down docker-logs docker-prod \
        start-hotwheels start-matchbox start-minigt \
        list-jobs list-models list-brands list-seeds

APP_NAME   = minigaraj-scraper
BINARY     = ./bin/scraper
MAIN       = ./cmd/scraper

# ── Help ─────────────────────────────────────────────────────────────────────
help:
	@echo ""
	@echo "  MiniGaraj Scraper - Available Commands"
	@echo "  ─────────────────────────────────────"
	@echo ""
	@echo "  Development:"
	@echo "    make build          Build the binary"
	@echo "    make run            Run locally (needs DB)"
	@echo "    make dev            Hot-reload with Air"
	@echo "    make test           Run tests"
	@echo "    make test-cover     Run tests with coverage"
	@echo "    make lint           Run go vet"
	@echo "    make tidy           go mod tidy"
	@echo ""
	@echo "  Docker:"
	@echo "    make docker-up      Start all services (dev)"
	@echo "    make docker-down    Stop all services"
	@echo "    make docker-logs    Tail scraper logs"
	@echo "    make docker-prod    Start in production mode"
	@echo ""
	@echo "  API Shortcuts:"
	@echo "    make start-hotwheels  Start Hot Wheels crawl"
	@echo "    make start-matchbox   Start Matchbox crawl"
	@echo "    make start-minigt     Start Mini GT crawl"
	@echo "    make list-jobs        List recent jobs"
	@echo "    make list-models      List pending models"
	@echo "    make list-brands      List available brands"
	@echo "    make list-seeds       List seed URLs"
	@echo ""

# ── Build ────────────────────────────────────────────────────────────────────
build:
	@mkdir -p bin
	go build -ldflags="-w -s" -o $(BINARY) $(MAIN)
	@echo "Built: $(BINARY)"

# ── Run ──────────────────────────────────────────────────────────────────────
run: build
	$(BINARY)

# ── Dev (hot reload) ──────────────────────────────────────────────────────────
dev:
	@which air > /dev/null 2>&1 || go install github.com/air-verse/air@latest
	air -c .air.toml

# ── Test ─────────────────────────────────────────────────────────────────────
test:
	go test ./... -v -count=1 -timeout 60s

test-cover:
	go test ./... -coverprofile=coverage.out -timeout 60s
	go tool cover -func=coverage.out
	@rm -f coverage.out

lint:
	go vet ./...

# ── Tidy ─────────────────────────────────────────────────────────────────────
tidy:
	go mod tidy

# ── Docker (dev) ─────────────────────────────────────────────────────────────
docker-up:
	docker compose up --build -d
	@echo "Scraper running on http://localhost:8300"
	@echo "  Health:  http://localhost:8300/health"
	@echo "  Metrics: http://localhost:8300/metrics"

docker-down:
	docker compose down

docker-logs:
	docker compose logs -f scraper

docker-rebuild:
	docker compose up --build --force-recreate -d

# ── Docker (production) ──────────────────────────────────────────────────────
docker-prod:
	docker compose -f docker-compose.yml -f docker-compose.prod.yml up --build -d

docker-prod-down:
	docker compose -f docker-compose.yml -f docker-compose.prod.yml down

# ── DB ───────────────────────────────────────────────────────────────────────
db-shell:
	docker compose exec scraper-db psql -U scraper -d minigaraj_scraper

# ── API Shortcuts ─────────────────────────────────────────────────────────────
API_URL   = http://localhost:8300
API_KEY   =
AUTH_FLAG = $(if $(API_KEY),-H "X-API-Key: $(API_KEY)",)

start-hotwheels:
	@curl -s -X POST $(API_URL)/api/v1/jobs $(AUTH_FLAG) \
		-H "Content-Type: application/json" \
		-d '{"brand":"Hot Wheels"}' | jq .

start-matchbox:
	@curl -s -X POST $(API_URL)/api/v1/jobs $(AUTH_FLAG) \
		-H "Content-Type: application/json" \
		-d '{"brand":"Matchbox"}' | jq .

start-minigt:
	@curl -s -X POST $(API_URL)/api/v1/jobs $(AUTH_FLAG) \
		-H "Content-Type: application/json" \
		-d '{"brand":"Mini GT"}' | jq .

list-jobs:
	@curl -s $(API_URL)/api/v1/jobs $(AUTH_FLAG) | jq .

list-models:
	@curl -s "$(API_URL)/api/v1/models?status=pending&limit=10" $(AUTH_FLAG) | jq .

list-brands:
	@curl -s $(API_URL)/api/v1/brands $(AUTH_FLAG) | jq .

list-seeds:
	@curl -s $(API_URL)/api/v1/seeds $(AUTH_FLAG) | jq .

health:
	@curl -s $(API_URL)/health | jq .
