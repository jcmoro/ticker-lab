# Ticker Lab

Financial data dashboard that ingests public economic data daily and displays it as a ticker-style dashboard. Polyglot architecture experiment with Node.js and Go microservices.

**Current scope:** ECB exchange rates (30 currencies), crypto prices (top 20), macro economic indicators (FRED + ECB + BdE), Spanish electricity (ESIOS, Tier 1 series), Spanish investment funds (CNMV monthly NAV). Historical data, interactive charts, and currency converter.

## Stack

| Layer | Technology |
|-------|-----------|
| Backend (Node) | Node.js 24 + Fastify 5 + TypeScript |
| Backend (Go) | Go 1.25 + stdlib + pgx (converter + crypto + macro + esios + cnmv) |
| Frontend | Fastify SSR (Eta templates) + Chart.js |
| Database | PostgreSQL 16 + Drizzle ORM |
| Contract | OpenAPI 3.1 (source of truth) |
| Quality | Biome + Vitest + go test (~150 tests across 6 modules) |
| Infra | Docker (dev) + Render + Neon + GitHub Actions |

**Live:**
- Dashboard: https://tickerlab.onrender.com
- Go converter: https://tickerlab-go.onrender.com
- Go crypto: https://tickerlab-crypto.onrender.com
- Go macro: https://macro-go.onrender.com
- Go ESIOS: https://tickerlab-esios.onrender.com
- Go CNMV: https://tickerlab-cnmv.onrender.com

## Quick Start

```bash
make setup        # Build containers, install dependencies
make db-migrate   # Create database tables
make job-ingest   # Fetch latest exchange rates from ECB
make job-backfill # Backfill historical rates (2024-01-01 to today)
make dev          # Start all services (Node :3000 + 5 Go services + Postgres)
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
curl "http://localhost:8110/api/v1/macro/bde/IRPH/history?days=365"

# ─── ESIOS Spanish Electricity (Go) ─────────────────────────
curl "http://localhost:8120/api/v1/electricity/indicators?page_size=20"
curl "http://localhost:8120/api/v1/electricity/indicators/1001/geos/8741/observations?start_date=2026-04-01&end_date=2026-05-01"

# ─── CNMV Spanish Funds (Go) ────────────────────────────────
curl "http://localhost:8130/api/v1/funds?gestora=MARCH&page_size=20"
curl "http://localhost:8130/api/v1/funds/ES0173534017"
curl "http://localhost:8130/api/v1/funds/ES0173534017/nav-observations?start_date=2025-01-01&end_date=2026-05-01"

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
make dev             # Start all services
make down            # Stop containers
make clean           # Remove containers, volumes, node_modules

# Quality
make ci              # Full CI pipeline (Node + Go)
make lint            # Biome linter (Node)
make format          # Biome formatter (Node)
make typecheck       # TypeScript checks
make test            # Node tests
make go-vet          # go vet on all Go services
make go-test         # go test on all Go services
make go-ci           # Go quality gates (vet + test)

# Database
make db-migrate      # Run Drizzle migrations
make db-seed         # Seed development data

# Data
make job-ingest          # Fetch latest ECB rates
make job-backfill        # Backfill historical exchange rates
make job-crypto          # Fetch latest crypto prices from CoinGecko
make job-crypto-backfill # Backfill historical crypto prices (365 days, ~3.5 min)
make job-macro-ingest    # Ingest FRED + ECB + BdE macro indicators
make job-macro-ingest-bde # Ingest only BdE Spanish rates
make job-macro-backfill  # Backfill all macro indicators history
make job-esios           # Ingest ESIOS Spanish electricity (requires ESIOS_API_KEY)
make job-esios-backfill  # Backfill ESIOS indicators (2020 → now, chunked monthly)
make job-cnmv            # Ingest CNMV Spanish funds (current + previous month)
make job-cnmv-backfill   # Backfill CNMV fund history (default fromYear=2020)

# Load testing
make load-test       # k6 load test (smoke + ramp-up) against local
make load-test-smoke # k6 smoke test (5 VUs, 30s)
make load-test-prod  # k6 load test against production (Render)

# Production
make deploy               # Trigger Render deploy
make prod-db              # Connect to Neon Postgres
make prod-ingest          # ECB ingestion against production
make prod-backfill        # Backfill exchange rates against production
make prod-crypto          # Fetch crypto prices against production
make prod-crypto-backfill # Backfill crypto history (365 days, ~3.5 min)
make prod-macro-ingest    # Ingest FRED + ECB + BdE against production
make prod-macro-backfill  # Backfill macro history against production
make prod-esios           # Ingest ESIOS against production (requires ESIOS_API_KEY)
make prod-esios-backfill  # Backfill ESIOS history against production
make prod-cnmv            # Ingest CNMV against production
make prod-cnmv-backfill   # Backfill CNMV history against production
```

## Services

| Service | Port (dev) | Language | Data source |
|---------|-----------|----------|-------------|
| `api` | 3000 | Node.js | Frankfurter (ECB) |
| `converter-go` | 8080 | Go | Shared DB |
| `crypto-go` | 8090 | Go | CoinGecko |
| `macro-go` | 8110 | Go | FRED + ECB + BdE |
| `esios-go` | 8120 | Go | REE ESIOS |
| `cnmv-go` | 8130 | Go | CNMV monthly files |
| `db` | 5432 | Postgres | — |

## Project Structure

```
apps/
├── api/src/                  Node.js — exchange rates + SSR dashboard + converter
│   ├── domain/               Entities, value objects, ports
│   ├── application/          Use cases (Ingest, GetLatest, GetHistory, Convert...)
│   ├── infrastructure/       Fastify, Drizzle, Frankfurter, jobs
│   ├── views/                SSR templates
│   └── main.ts               Composition root
├── converter-go/             Go — currency converter microservice
├── crypto-go/                Go — crypto prices microservice (CoinGecko)
├── macro-go/                 Go — macro indicators microservice (FRED + ECB + BdE)
├── esios-go/                 Go — Spanish electricity microservice (REE ESIOS)
├── cnmv-go/                  Go — Spanish funds microservice (CNMV monthly NAV)
└── internal/httpx/           Shared Go package (CORS, ProblemDetails, pagination)
packages/shared/              Shared types (generated from OpenAPI)
docker/                       Dockerfiles per service
docs/                         Architecture, API, runbook, ADRs, integration specs, roadmap
tests/load/                   k6 load test scenarios (smoke + ramp-up + prod)
```

## Pages

| URL | Service | Description |
|-----|---------|-------------|
| `/` | Node | Dashboard — 30 currencies with flags, names, rates |
| `/rates/:quote` | Node | Detail — Chart.js chart with 30d/90d/180d/365d selector |
| `/crypto` | Node → Go | Top 20 crypto prices with 24h change |
| `/crypto/:id` | Node → Go | Crypto detail — Chart.js chart with period selector |
| `/macro` | Node → Go | Macro indicators grouped by category (FRED + ECB + BdE) |
| `/macro/:source/:id` | Node → Go | Macro detail — Chart.js chart with 3M/6M/1Y/5Y/ALL selector |
| `/electricity` | Node → Go | ESIOS indicators with category filter |
| `/electricity/:indicator/:geo` | Node → Go | Electricity detail — hourly time series |
| `/funds` | Node → Go | CNMV Spanish funds with filters (tipo, gestora, free-text) |
| `/funds/:isin` | Node → Go | Fund detail — NAV chart with date range selector |
| `/converter` | Node | Currency converter — Node/Go/Both toggle with response times |
| `/api/docs` | Node | Interactive API documentation (ReDoc) |

## Documentation

- [Docs index](docs/README.md)
- [Architecture](docs/architecture.md)
- [API navigation](docs/api.md)
- [API design standards](docs/api-design-standards.md)
- [Plan status](docs/plan-status.md)
- [Tech debt analysis](docs/tech-debt-analysis.md)
- [Runbook](docs/runbook.md)
- [Changelog](docs/changelog.md)
- [Future Providers](docs/future-providers.md)
- [Future Features](docs/future-features.md)
- Integration specs:
  - [Macro Indicators](docs/macro-indicators-integration.md)
  - [BdE](docs/bde-integration.md)
  - [ESIOS](docs/esios-integration.md)
  - [CNMV](docs/cnmv-integration.md)
  - [RateHawk](docs/ratehawk-integration.md)
  - [Comparison providers research](docs/comparison-providers-research.md)
- ADRs:
  - [ADR-001: Tech Stack](docs/decisions/001-tech-stack.md)
  - [ADR-002: Frontend SSR](docs/decisions/002-frontend-ssr.md)
  - [ADR-003: Hosting Strategy](docs/decisions/003-hosting-strategy.md)

## Other directories
- **`skills/`** — Reusable agent skills developed (changelog, feature-spec).

## License

MIT
