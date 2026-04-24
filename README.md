# Ticker Lab

Financial data dashboard that ingests public economic data daily and displays it as a ticker-style dashboard. Polyglot architecture experiment with Node.js and Go microservices.

**Current scope:** ECB exchange rates (30 currencies), crypto prices (top 20), macro economic indicators (14 series from FRED & ECB). Historical data, interactive charts, and currency converter.

## Stack

| Layer | Technology |
|-------|-----------|
| Backend (Node) | Node.js 24 + Fastify 5 + TypeScript |
| Backend (Go) | Go 1.25 + stdlib + pgx (converter + crypto + macro) |
| Frontend | Fastify SSR (Eta templates) + Chart.js |
| Database | PostgreSQL 16 + Drizzle ORM |
| Contract | OpenAPI 3.1 (source of truth) |
| Quality | Biome + Vitest (35 tests) + go test (21 tests) = **56 tests** |
| Infra | Docker (dev) + Render + Neon + GitHub Actions |

**Live:**
- Dashboard: https://tickerlab.onrender.com
- Go converter: https://tickerlab-go.onrender.com
- Go crypto: https://tickerlab-crypto.onrender.com
- Go macro: https://tickerlab-macro.onrender.com

## Quick Start

```bash
make setup        # Build containers, install dependencies
make db-migrate   # Create database tables
make job-ingest   # Fetch latest exchange rates from ECB
make job-backfill # Backfill historical rates (2024-01-01 to today)
make dev          # Start all services (Node :3000, Go converter :8080, Go crypto :8090, Go macro :8110)
```

## API

```bash
# ─── Exchange Rates (Node) ───────────────────────────────────
curl http://localhost:3000/api/v1/exchange-rates/latest
curl http://localhost:3000/api/v1/exchange-rates/2026-04-17
curl "http://localhost:3000/api/v1/exchange-rates/history?quote=USD&from=2025-01-01&to=2026-04-17"

# ─── Currency Converter (Node + Go) ─────────────────────────
curl "http://localhost:3000/api/v1/convert?from=GBP&to=JPY&amount=1000"
curl "http://localhost:8080/api/v1/go/convert?from=GBP&to=JPY&amount=1000"

# ─── Crypto (Go) ────────────────────────────────────────────
curl http://localhost:8090/api/v1/crypto/latest
curl "http://localhost:8090/api/v1/crypto/bitcoin/history?days=90"

# ─── Macro Indicators (Go) ──────────────────────────────────
curl http://localhost:8110/api/v1/macro/indicators
curl "http://localhost:8110/api/v1/macro/indicators?category=inflation"
curl "http://localhost:8110/api/v1/macro/fred/CPIAUCSL/history?days=365"
curl "http://localhost:8110/api/v1/macro/ecb/ICP/history?days=365"

# ─── System ──────────────────────────────────────────────────
curl http://localhost:3000/health
curl http://localhost:3000/ready
curl http://localhost:3000/metrics
```

**Interactive docs:** https://tickerlab.onrender.com/api/docs (ReDoc)

## Commands

```bash
make help            # Show all available targets

# Development
make setup           # Build containers, install dependencies
make dev             # Start all services (Node + Go converter + Go crypto + Go macro + Postgres)
make down            # Stop containers
make clean           # Remove containers, volumes, node_modules

# Quality
make ci              # Full CI pipeline (Node + Go)
make lint            # Biome linter (Node)
make format          # Biome formatter (Node)
make typecheck       # TypeScript checks
make test            # Node tests (35)
make go-vet          # go vet on all Go services
make go-test         # go test on all Go services (21 tests)
make go-ci           # Go quality gates (vet + test)

# Database
make db-migrate      # Run Drizzle migrations (exchange_rates)
make db-seed         # Seed development data

# Data
make job-ingest      # Fetch latest ECB rates
make job-backfill    # Backfill historical exchange rates
make job-crypto      # Fetch latest crypto prices from CoinGecko
make job-crypto-backfill # Backfill historical crypto prices (365 days, ~3.5 min)
make job-macro-ingest    # Ingest FRED + ECB macro indicators
make job-macro-backfill  # Backfill all macro indicators history

# Production
make deploy          # Trigger Render deploy
make prod-db         # Connect to Neon Postgres
make prod-ingest     # Run ECB ingestion against production
make prod-backfill   # Backfill exchange rates against production
make prod-crypto          # Fetch crypto prices against production
make prod-crypto-backfill # Backfill crypto history (365 days, ~3.5 min)
make prod-macro-ingest    # Ingest macro indicators against production
make prod-macro-backfill  # Backfill macro history against production
```

## Services

| Service | Port (dev) | Language | Data source |
|---------|-----------|----------|-------------|
| `api` | 3000 | Node.js | Frankfurter (ECB) |
| `converter-go` | 8080 | Go | Shared DB |
| `crypto-go` | 8090 | Go | CoinGecko |
| `macro-go` | 8110 | Go | FRED + ECB |
| `db` | 5432 | Postgres | — |

## Project Structure

```
apps/
├── api/src/                  Node.js — exchange rates + dashboard + converter
│   ├── domain/               Entities, value objects, ports
│   ├── application/          Use cases (Ingest, GetLatest, GetHistory, Convert...)
│   ├── infrastructure/       Fastify, Drizzle, Frankfurter, jobs
│   ├── views/                SSR templates (dashboard, rate detail, converter)
│   └── main.ts               Composition root
├── converter-go/             Go — currency converter microservice
│   ├── main.go               HTTP server + pgx + cross-rate logic
│   └── main_test.go          9 tests
├── crypto-go/                Go — crypto prices microservice
│   ├── main.go               HTTP server + routing + ingestion CLI
│   ├── coingecko.go          CoinGecko API client
│   ├── repository.go         Postgres repository (auto-migrate)
│   ├── handlers.go           HTTP handlers
│   ├── models.go             Types + top 20 coins config
│   └── main_test.go          6 tests
├── macro-go/                 Go — macro indicators microservice
│   ├── main.go               HTTP server + CLI (ingest, ingest-ecb, backfill)
│   ├── fred.go               FRED API client (API key auth)
│   ├── ecb.go                ECB Data Portal client (CSV format)
│   ├── repository.go         Postgres repository (auto-migrate)
│   ├── handlers.go           HTTP handlers
│   ├── models.go             Types + 14 series config (FRED + ECB)
│   └── main_test.go          6 tests
packages/shared/              Shared types (generated from OpenAPI)
docker/                       Dockerfiles (api, converter-go, crypto-go, macro-go)
docs/                         Architecture, API, runbook, ADRs, roadmap
```

## Pages

| URL | Service | Description |
|-----|---------|-------------|
| `/` | Node | Dashboard — 30 currencies with flags, names, rates |
| `/rates/:quote` | Node | Detail — Chart.js chart with 30d/90d/180d/365d selector |
| `/crypto` | Node → Go | Top 20 crypto prices with 24h change |
| `/crypto/:id` | Node → Go | Crypto detail — Chart.js chart with period selector |
| `/macro` | Node → Go | Macro indicators grouped by category (FRED + ECB) |
| `/macro/:source/:id` | Node → Go | Macro detail — Chart.js chart with 3M/6M/1Y/5Y/ALL selector |
| `/converter` | Node | Currency converter — Node/Go/Both toggle with response times |
| `/api/docs` | Node | Interactive API documentation (ReDoc) |

## Documentation

- [Docs index](docs/README.md)
- [Architecture](docs/architecture.md)
- [API](docs/api.md)
- [Runbook](docs/runbook.md)
- [Changelog](docs/changelog.md)
- [Future Providers](docs/future-providers.md)
- [Future Features](docs/future-features.md)
- [Macro Indicators Integration](docs/macro-indicators-integration.md)
- [RateHawk Integration](docs/ratehawk-integration.md)
- [ADR-001: Tech Stack](docs/decisions/001-tech-stack.md)
- [ADR-002: Frontend SSR](docs/decisions/002-frontend-ssr.md)

## License

MIT
