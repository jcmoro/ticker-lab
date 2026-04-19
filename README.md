# Ticker Lab

Financial data dashboard that ingests public economic data daily and displays it as a ticker-style dashboard.

**Current scope:** ECB exchange rates (30 currencies) via [Frankfurter API](https://frankfurter.dev). Historical data from 2024, interactive charts, and currency converter.

## Stack

| Layer | Technology |
|-------|-----------|
| Backend (Node) | Node.js 24 + Fastify 5 + TypeScript |
| Backend (Go) | Go 1.25 + stdlib + pgx |
| Frontend | Fastify SSR (Eta templates) + Chart.js |
| Database | PostgreSQL 16 + Drizzle ORM |
| Contract | OpenAPI 3.1 (source of truth) |
| Quality | Biome + Vitest (32 tests) + go test (9 tests) |
| Infra | Docker (dev) + Render + Neon + GitHub Actions |

**Live:**
- Dashboard: https://tickerlab.onrender.com
- Go converter: https://tickerlab-go.onrender.com

## Quick Start

```bash
make setup        # Build containers, install dependencies
make db-migrate   # Create database tables
make job-ingest   # Fetch latest exchange rates from ECB
make job-backfill # Backfill historical rates (2024-01-01 to today)
make dev          # Start development (http://localhost:3000)
```

## API

```bash
# Health / readiness
curl http://localhost:3000/health
curl http://localhost:3000/ready

# Latest exchange rates
curl http://localhost:3000/api/v1/exchange-rates/latest

# Rates for a specific date
curl http://localhost:3000/api/v1/exchange-rates/2026-04-17

# Historical time series
curl "http://localhost:3000/api/v1/exchange-rates/history?quote=USD&from=2025-01-01&to=2026-04-17"

# Currency converter — Node.js engine
curl "http://localhost:3000/api/v1/convert?from=GBP&to=JPY&amount=1000"

# Currency converter — Go engine
curl "http://localhost:8080/api/v1/go/convert?from=GBP&to=JPY&amount=1000"

# Metrics
curl http://localhost:3000/metrics
```

**Interactive docs:** https://tickerlab.onrender.com/api/docs (ReDoc)

## Commands

```bash
make help            # Show all available targets

# Development
make setup           # Build containers, install dependencies
make dev             # Start development environment
make down            # Stop containers
make clean           # Remove containers, volumes, node_modules

# Quality
make ci              # Run lint + typecheck + test
make lint            # Biome linter
make format          # Biome formatter
make typecheck       # TypeScript checks
make test            # Node tests (32)
# Go tests: cd apps/converter-go && go test ./...

# Database
make db-migrate      # Run migrations
make db-seed         # Seed development data

# Data
make job-ingest      # Fetch latest ECB rates
make job-backfill    # Backfill historical rates

# Production
make deploy          # Trigger Render deploy
make prod-db         # Connect to Neon Postgres
make prod-ingest     # Run ingestion against production
make prod-backfill   # Backfill against production
```

## Project Structure

```
apps/api/src/
├── domain/           Entities, value objects, ports (ExchangeRate, errors)
├── application/      Use cases (IngestDailyRates, GetLatestRates, GetRatesByDate,
│                                GetRateHistory, ConvertCurrency)
├── infrastructure/
│   ├── http/         Fastify server, routes, error handler, metrics
│   ├── persistence/  Drizzle schema, repository, migrations
│   ├── providers/    FrankfurterClient (ECB data)
│   └── jobs/         Ingestion + backfill scripts
├── views/            SSR templates (dashboard, rate detail, converter)
└── main.ts           Composition root
apps/converter-go/    Go microservice (currency converter)
├── main.go           HTTP server + pgx Postgres
├── main_test.go      9 tests
└── Dockerfile        Multi-stage (~15MB image)
packages/shared/      Shared types (generated from OpenAPI)
docker/               Dockerfiles (api + converter-go)
docs/                 Architecture, API, runbook, ADRs, roadmap
```

## Pages

| URL | Description |
|-----|-------------|
| `/` | Dashboard — 30 currencies with flags, names, rates |
| `/rates/:quote` | Detail — Chart.js chart with 30d/90d/180d/365d selector |
| `/converter` | Currency converter — Node/Go/Both toggle with response times |
| `/api/docs` | Interactive API documentation (ReDoc) |

## Documentation

- [Docs index](docs/README.md)
- [Architecture](docs/architecture.md)
- [API](docs/api.md)
- [Runbook](docs/runbook.md)
- [Changelog](docs/changelog.md)
- [Future Providers](docs/future-providers.md)
- [Future Features](docs/future-features.md)
- [ADR-001: Tech Stack](docs/decisions/001-tech-stack.md)
- [ADR-002: Frontend SSR](docs/decisions/002-frontend-ssr.md)

## License

MIT
