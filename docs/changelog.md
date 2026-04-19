# Changelog

Reverse-chronological log of significant changes to Ticker Lab.

---

## 2026-04-19 — Phase 2: Exchange Rates MVP

**Summary:** First functional data pipeline — ECB exchange rates ingested daily from Frankfurter API, served via REST and displayed in SSR dashboard.

**Endpoints added:**
- `GET /api/v1/exchange-rates/latest` — latest exchange rates (query: `?base=EUR`)
- `GET /api/v1/exchange-rates/:date` — rates for a specific date (query: `?base=EUR`)

**Schema changes:**
- Created `exchange_rates` table (base_currency, quote_currency, rate, date) with unique constraint on (base, quote, date) and index on (base, date)

**Other:**
- Domain layer: `ExchangeRate` entity with validation, 4 domain errors, 2 ports (Provider, Repository)
- Application layer: 3 use cases (IngestDailyRates, GetLatestRates, GetRatesByDate)
- Infrastructure: FrankfurterClient adapter, DrizzleExchangeRateRepository, ingestion job (`make job-ingest`)
- Dashboard SSR shows live ticker cards for 29 currencies
- 18 tests (domain, application, HTTP)

---

## 2026-04-19 — Phase 1: Project Skeleton

**Summary:** Initial monorepo structure with all tooling, Docker environment, CI pipeline, and docs.

**Endpoints added:**
- `GET /health` — health check
- `GET /` — SSR dashboard (placeholder)

**Other:**
- Monorepo setup: pnpm workspaces, Biome, Lefthook, TypeScript strict
- Docker: multi-stage Dockerfile, docker-compose with Postgres
- CI: GitHub Actions pipeline (lint + typecheck + test)
- OpenAPI 3.1 spec as source of truth
- Documentation: architecture, API, runbook, ADRs, roadmap
