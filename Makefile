.PHONY: help setup dev down clean build lint format typecheck test test-unit test-functional ci go-vet go-test go-ci db-migrate db-seed openapi-generate job-ingest job-crypto-backfill job-macro-ingest job-macro-backfill load-test load-test-smoke docker-build deploy fly-setup fly-logs fly-status fly-console fly-db fly-ingest fly-rollback

.DEFAULT_GOAL := help

help: ## Show available targets
	@echo ""
	@echo "  \033[1mDevelopment\033[0m"
	@echo "  \033[36msetup\033[0m              Build containers and install dependencies"
	@echo "  \033[36mdev\033[0m                Start development environment"
	@echo "  \033[36mdown\033[0m               Stop all containers"
	@echo "  \033[36mclean\033[0m              Remove containers, volumes, and node_modules"
	@echo ""
	@echo "  \033[1mQuality\033[0m"
	@echo "  \033[36mlint\033[0m               Run Biome linter (Node)"
	@echo "  \033[36mformat\033[0m             Run Biome formatter (Node)"
	@echo "  \033[36mtypecheck\033[0m          TypeScript type checking"
	@echo "  \033[36mgo-vet\033[0m             Run go vet on all Go services"
	@echo "  \033[36mgo-test\033[0m            Run go test on all Go services"
	@echo "  \033[36mgo-ci\033[0m              Go quality gates (vet + test)"
	@echo "  \033[36mci\033[0m                 Full CI pipeline (Node + Go)"
	@echo ""
	@echo "  \033[1mTesting\033[0m"
	@echo "  \033[36mtest\033[0m               Run all tests"
	@echo "  \033[36mtest-unit\033[0m          Run unit tests only"
	@echo "  \033[36mtest-functional\033[0m    Run functional tests only"
	@echo ""
	@echo "  \033[1mDatabase\033[0m"
	@echo "  \033[36mdb-migrate\033[0m         Run database migrations"
	@echo "  \033[36mdb-seed\033[0m            Seed development data"
	@echo ""
	@echo "  \033[1mOpenAPI & Jobs\033[0m"
	@echo "  \033[36mopenapi-generate\033[0m   Generate TypeScript types from OpenAPI spec"
	@echo "  \033[36mjob-ingest\033[0m         Fetch latest ECB exchange rates (local)"
	@echo "  \033[36mjob-backfill\033[0m       Backfill historical exchange rates (local)"
	@echo "  \033[36mjob-crypto\033[0m         Fetch latest crypto prices (local)"
	@echo "  \033[36mjob-crypto-backfill\033[0m Backfill historical crypto prices (local)"
	@echo "  \033[36mjob-macro-ingest\033[0m   Ingest FRED + ECB macro indicators (local)"
	@echo "  \033[36mjob-macro-backfill\033[0m Backfill all macro indicators history (local)"
	@echo ""
	@echo "  \033[1mLoad Testing\033[0m"
	@echo "  \033[36mload-test\033[0m          Run k6 load test (full: smoke + ramp-up)"
	@echo "  \033[36mload-test-smoke\033[0m    Quick smoke test (5 VUs, 30s)"
	@echo ""
	@echo "  \033[1mBuild & Deploy\033[0m"
	@echo "  \033[36mbuild\033[0m              Build for production"
	@echo "  \033[36mdocker-build\033[0m       Build production Docker image"
	@echo "  \033[36mdeploy\033[0m             Trigger Render deploy"
	@echo ""
	@echo "  \033[1mProduction (Neon + Render)\033[0m"
	@echo "  \033[36mprod-db\033[0m            Connect to Neon Postgres"
	@echo "  \033[36mprod-ingest\033[0m        Run daily ingestion against production DB"
	@echo "  \033[36mprod-backfill\033[0m      Backfill historical rates against production DB"
	@echo "  \033[36mprod-crypto\033[0m        Fetch crypto prices against production DB"
	@echo "  \033[36mprod-crypto-backfill\033[0m Backfill crypto history against production DB"
	@echo "  \033[36mprod-macro-ingest\033[0m  Ingest macro indicators against production DB"
	@echo "  \033[36mprod-macro-backfill\033[0m Backfill macro history against production DB"
	@echo ""

# ─── Development ─────────────────────────────────────────────

setup:
	cp -n .env.example .env 2>/dev/null || true
	docker compose build
	docker compose run --rm api pnpm install

dev:
	docker compose up

down:
	docker compose down

clean:
	docker compose down -v
	rm -rf node_modules apps/*/node_modules packages/*/node_modules

# ─── Quality ─────────────────────────────────────────────────

lint:
	docker compose run --rm api pnpm lint

format:
	docker compose run --rm api pnpm format

typecheck:
	docker compose run --rm api pnpm typecheck

go-vet:
	docker compose run --rm converter-go go vet ./...
	docker compose run --rm crypto-go go vet ./...
	docker compose run --rm macro-go go vet ./...

go-test:
	docker compose run --rm converter-go go test ./...
	docker compose run --rm crypto-go go test ./...
	docker compose run --rm macro-go go test ./...

go-ci: go-vet go-test

ci: lint typecheck test go-ci

# ─── Testing ─────────────────────────────────────────────────

test:
	docker compose run --rm api pnpm test

test-unit:
	docker compose run --rm api pnpm test:unit

test-functional:
	docker compose run --rm api pnpm test:functional

# ─── Database ────────────────────────────────────────────────

db-migrate:
	docker compose run --rm api pnpm --filter @ticker-lab/api db:migrate

db-seed:
	docker compose run --rm api pnpm --filter @ticker-lab/api db:seed

# ─── OpenAPI & Jobs ──────────────────────────────────────────

openapi-generate:
	docker compose run --rm api pnpm --filter @ticker-lab/api openapi:generate

job-ingest:
	docker compose run --rm api pnpm --filter @ticker-lab/api job:ingest

job-backfill: ## Backfill historical rates (default: 2024-01-01 to today)
	docker compose run --rm api pnpm --filter @ticker-lab/api job:backfill

job-crypto: ## Fetch latest crypto prices from CoinGecko
	docker compose run --rm crypto-go ./crypto-go ingest

job-crypto-backfill: ## Backfill historical crypto prices (default: 365 days)
	docker compose run --rm crypto-go ./crypto-go backfill 365

job-macro-ingest: ## Ingest FRED + ECB macro indicators
	docker compose run --rm macro-go ./macro-go ingest
	docker compose run --rm macro-go ./macro-go ingest-ecb

job-macro-backfill: ## Backfill all macro indicators history
	docker compose run --rm macro-go ./macro-go backfill

# ─── Load Testing ───────────────────────────────────────────

load-test: ## Run k6 load test against local services (generates HTML report)
	docker run --rm --add-host=host.docker.internal:host-gateway \
		-v $(PWD)/tests/load:/scripts \
		-v $(PWD)/tests/load:/results \
		grafana/k6 run /scripts/main.js
	@echo ""
	@echo "Report: tests/load/report.html"

load-test-smoke: ## Quick smoke test (5 VUs, 30s)
	docker run --rm --add-host=host.docker.internal:host-gateway \
		-v $(PWD)/tests/load:/scripts \
		-v $(PWD)/tests/load:/results \
		-e K6_SCENARIOS='{"smoke":{"executor":"constant-vus","vus":5,"duration":"30s"}}' \
		grafana/k6 run /scripts/main.js
	@echo ""
	@echo "Report: tests/load/report.html"

load-test-prod: ## Run k6 load test against production (Render)
	docker run --rm \
		-v $(PWD)/tests/load:/scripts \
		-v $(PWD)/tests/load:/results \
		-e API_BASE=https://tickerlab.onrender.com \
		-e CRYPTO_BASE=https://tickerlab-crypto.onrender.com \
		-e MACRO_BASE=https://macro-go.onrender.com \
		grafana/k6 run /scripts/main.js
	@echo ""
	@echo "Report: tests/load/report.html"

# ─── Build & Deploy ──────────────────────────────────────────

build:
	docker compose run --rm api pnpm --filter @ticker-lab/api build

docker-build:
	docker build -f docker/api/Dockerfile --target prod -t ticker-lab-api .

deploy: ## Trigger Render deploy (requires RENDER_DEPLOY_HOOK env var)
	@curl -s -X POST "$$RENDER_DEPLOY_HOOK" && echo "Deploy triggered"

# ─── Production (Neon + Render) ──────────────────────────────

-include .env.prod

prod-db: ## Connect to Neon Postgres
	@psql "$(DATABASE_URL)"

prod-ingest: ## Run daily ingestion against production DB
	docker compose run --rm -e DATABASE_URL="$(DATABASE_URL)" api pnpm --filter @ticker-lab/api job:ingest

prod-backfill: ## Backfill historical rates against production DB
	docker compose run --rm -e DATABASE_URL="$(DATABASE_URL)" api pnpm --filter @ticker-lab/api job:backfill

prod-crypto: ## Fetch crypto prices against production DB
	cd apps/crypto-go && DATABASE_URL="$(DATABASE_URL)" /usr/local/go/bin/go run . ingest

prod-crypto-backfill: ## Backfill crypto history against production DB (365 days)
	cd apps/crypto-go && DATABASE_URL="$(DATABASE_URL)" /usr/local/go/bin/go run . backfill 365

prod-macro-ingest: ## Ingest macro indicators against production DB
	cd apps/macro-go && DATABASE_URL="$(DATABASE_URL)" FRED_API_KEY="$(FRED_API_KEY)" /usr/local/go/bin/go run . ingest
	cd apps/macro-go && DATABASE_URL="$(DATABASE_URL)" /usr/local/go/bin/go run . ingest-ecb

prod-macro-backfill: ## Backfill macro history against production DB
	cd apps/macro-go && DATABASE_URL="$(DATABASE_URL)" FRED_API_KEY="$(FRED_API_KEY)" /usr/local/go/bin/go run . backfill
