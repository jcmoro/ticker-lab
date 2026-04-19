# Changelog

Reverse-chronological log of significant changes to Ticker Lab.

---

## 2026-04-19 — Phase 6: Migrate to Render + Neon

**Summary:** Migrated from Fly.io (7-day trial) to permanently free hosting.

**Infrastructure:**
- App: Render free tier (native Node, auto-deploy from GitHub)
- Database: Neon free tier (serverless Postgres, Frankfurt EU)
- URL: https://tickerlab.onrender.com
- Fly.io apps destroyed

**Other:**
- `render.yaml` for Render Blueprint
- `entrypoint.sh` for migration + start
- GitHub Actions ingest workflow runs directly against Neon (no SSH needed)
- Makefile updated: fly-* commands replaced with prod-* commands
- 17,536 historical rates migrated to Neon

---

## 2026-04-19 — Phase 7: Historical Data & Charts

**Summary:** Time series data with interactive Chart.js charts and date range selection.

**Endpoints added:**
- `GET /api/v1/exchange-rates/history?base=EUR&quote=USD&from=&to=` — time series for a currency pair

**Pages added:**
- `GET /rates/:quote` — detail page with Chart.js line chart, period selector (30d/90d/180d/365d)
- Dashboard ticker cards now link to their detail pages

**Other:**
- Backfill job (`make job-backfill`) — fetches historical rates from Frankfurter in 90-day chunks
- 17,536 historical rates backfilled from 2024-01-01
- Domain: `HistoryPoint` type, `findHistory` on repository port, `fetchDateRange` on provider port
- Application: `GetRateHistory` use case
- OpenAPI spec v0.5.0 with HistoryResponse + HistoryPoint schemas
- 26 tests (2 new: history endpoint, detail page)

---

## 2026-04-19 — Phase 5: Deployment

**Summary:** Production-ready deployment to Fly.io with automated CI/CD and daily ingestion cron.

**Infrastructure:**
- `fly.toml` — Fly.io config (Madrid region, shared-cpu-1x, 256MB, health/readiness checks)
- Production Dockerfile updated: includes migrations, OpenAPI spec, views
- `release_command` runs DB migrations automatically on deploy
- Migration script (`migrate.ts`) uses drizzle-orm programmatic migrate

**CI/CD:**
- `.github/workflows/deploy.yml` — auto-deploy on push to main (after CI passes)
- `.github/workflows/ingest.yml` — daily ECB ingestion cron (Mon-Fri 16:30 UTC)

**Docs:**
- Runbook updated with full Fly.io setup, deploy, rollback, and production operations

---

## 2026-04-19 — Phase 4: Observability

**Summary:** Structured metrics, graceful shutdown, startup banner.

**Endpoints added:**
- `GET /metrics` — request counts (total + per-route), uptime (JSON)

**Other:**
- Graceful shutdown: handles SIGTERM/SIGINT, closes Fastify server and Postgres connection
- Startup log line with version and Node.js version
- Global onRequest hook tracks request counts per route
- OpenAPI spec updated to v0.4.0
- 22 tests (1 new: metrics endpoint)

---

## 2026-04-19 — Phase 3: API Docs, Readiness & Error Handling

**Summary:** Polish HTTP surface — interactive API documentation, readiness probe, structured error responses.

**Endpoints added:**
- `GET /ready` — readiness check with DB connectivity verification (200/503)
- `GET /api/docs` — interactive API documentation via ReDoc
- `GET /api/openapi.yaml` — raw OpenAPI spec endpoint

**Other:**
- Error handler returns RFC 9457 ProblemDetails (`application/problem+json`) for domain errors
- OpenAPI spec updated to v0.3.0 with ReadinessResponse schema
- 21 tests (3 new: readiness, OpenAPI spec, ReDoc page)

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
