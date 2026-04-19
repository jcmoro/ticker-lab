# Ticker Lab

Financial data dashboard that ingests public economic data daily and displays it as a ticker-style dashboard.

**Current scope:** ECB exchange rates (29 currencies) via [Frankfurter API](https://frankfurter.dev).

## Stack

| Layer | Technology |
|-------|-----------|
| Backend | Node.js 24 + Fastify 5 + TypeScript |
| Frontend | Fastify SSR (Eta templates) |
| Database | PostgreSQL 16 + Drizzle ORM |
| Contract | OpenAPI 3.1 (source of truth) |
| Quality | Biome + Vitest (18 tests) |
| Infra | Docker + GitHub Actions + Fly.io |

## Quick Start

```bash
make setup       # Build containers, install dependencies
make db-migrate  # Create database tables
make job-ingest  # Fetch exchange rates from ECB
make dev         # Start development (http://localhost:3000)
```

## API

```bash
# Health check
curl http://localhost:3000/health

# Latest exchange rates
curl http://localhost:3000/api/v1/exchange-rates/latest

# Rates for a specific date
curl http://localhost:3000/api/v1/exchange-rates/2026-04-17

# Different base currency
curl http://localhost:3000/api/v1/exchange-rates/latest?base=USD
```

## Commands

```bash
make help          # Show all available targets
make ci            # Run lint + typecheck + test
make lint          # Biome linter
make format        # Biome formatter
make typecheck     # TypeScript checks
make test          # All tests
make db-migrate    # Run migrations
make job-ingest    # Manual data ingestion
make down          # Stop containers
make clean         # Full cleanup
```

## Project Structure

```
apps/api/src/
├── domain/           Entities, value objects, ports
├── application/      Use cases (IngestDailyRates, GetLatestRates, GetRatesByDate)
├── infrastructure/   Adapters (Fastify, Drizzle, Frankfurter, jobs)
├── views/            SSR templates (Eta)
└── main.ts           Composition root
packages/shared/      Shared types (generated from OpenAPI)
docker/               Dockerfiles
docs/                 Architecture, API, runbook, ADRs, roadmap
```

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
