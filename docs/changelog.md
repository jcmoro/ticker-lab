# Changelog

Reverse-chronological log of significant changes to Ticker Lab.

---

## 2026-05-17 — Tier 2 item 11: handler integration tests (crypto-go, macro-go)

**Summary:** Closed the handler-coverage gap between `esios-go`/`cnmv-go` (already well-tested) and `crypto-go`/`macro-go` (shape-only assertions). 9 new tests covering ordering, days-filter semantics, default fallbacks, error paths and aggregated value computation.

**`crypto-go` (4 new):**
- `TestLatestEndpoint_OrderingByMarketCap` — seeds 3 prices at `2099-12-31` with mixed `market_cap_eur`, asserts handler returns them in DESC order. Filters response by `integ-ord-*` prefix to tolerate concurrent test rows at the same MAX date.
- `TestHistoryEndpoint_FiltersByDays` — seeds rows at `CURRENT_DATE - {50, 25, 5}` for `integ-hist-coin`, exercises `?days=30|60|7` and asserts counts `{2, 3, 1}`.
- `TestHistoryEndpoint_InvalidDaysDefaultsTo90` — `?days=abc|-5|0` all collapse to 90 per the handler's `parsed > 0` guard.
- `TestHistoryEndpoint_UnknownCoinReturnsEmpty` — unknown `id` is 200 + `count: 0`, not 404.

**`macro-go` (5 new):**
- `TestIndicatorsEndpoint_CategoryFilter` — seeds 2 series across distinct categories, asserts `?category=` narrows correctly.
- `TestIndicatorsEndpoint_LatestAndPrevComputed` — 2 observations seeded, validates `latest_value` / `latest_date` / `prev_value` from the LATERAL joins in `FindIndicators`.
- `TestHistoryEndpoint_FiltersByDays` — `days=30|90|365` over observations at `CURRENT_DATE - {200, 60, 10}` returns `{1, 2, 3}`.
- `TestHistoryEndpoint_MissingParamsReturn400` — direct handler call (bypassing the mux) with empty `source`/`id` returns ProblemDetails with `code=MISSING_PARAMS`.
- `TestHistoryEndpoint_UnknownSeriesReturnsEmpty` — unknown series ⇒ 200 + `count: 0`.

**Isolation:** tests cleanup via prefixed `coin_id` / `source` literals (`integ-*`, `test-cat`, `test-lat`, `test-hist`) so they're safe against concurrent runs and against real ingest seed data.

**Counts:** crypto-go 7→11 tests, macro-go 9→14 tests. Total Go ~80.

---

## 2026-05-17 — Tier 2 item 9: total_size on remaining list responses

**Summary:** Two list endpoints that previously returned `total_size: null` now report the count for the requested filter on the first page. `cnmv-go` `/api/v1/funds/{isin}/nav-observations` and `esios-go` `/api/v1/electricity/indicators/{indicator_id}/geos/{geo_id}/observations` now compute `COUNT(*)` matching the (resource, start_date, end_date) filter when `page_token` is empty and embed it in `httpx.Page`. Subsequent pages omit `total_size` to avoid an extra COUNT roundtrip per request.

**Why "first page only":** AIP-158 leaves `total_size` optional, and these endpoints use a date-cursor that overrides `start_date` on subsequent pages. Recomputing the absolute total on every page would either require encoding the original filter range in the cursor (extra surface) or returning the misleading "remaining count after cursor" (semantic drift). Page-1-only delivers AIP-correct semantics where it matters (the user typically reads `total_size` once) without the cost on hot paths.

**New repository methods:**
- `cnmv-go`: `CountNAVObservations(ctx, isin, start, end) (int64, error)`
- `esios-go`: `CountObservations(ctx, indicatorID, geoID, start, end) (int64, error)`

**Tests added:**
- `TestObservationsEndpoint_TotalSizeOnFirstPageOnly` (esios-go) — seeds 5 observations, paginates at size=2, asserts `total_size=5` on page 1 and `total_size` absent on page 2.
- `TestNavObservationsEndpoint_TotalSizeOnFirstPageOnly` (cnmv-go) — analogous for NAV observations.

**Quality gates:** full test suites green in both services.

---

## 2026-05-17 — Tier 2 item 12: migration advisory lock

**Summary:** Wrapped `Migrate(ctx)` in the 4 Go services (`crypto-go`, `macro-go`, `esios-go`, `cnmv-go`) with `pg_advisory_lock` so concurrent boots can't race on `CREATE TABLE IF NOT EXISTS` / `CREATE INDEX IF NOT EXISTS`. Each service uses a distinct lock key (12001–12004) so they don't block each other.

**Implementation pattern:**
- `r.pool.Acquire(ctx)` to pin a session.
- `SELECT pg_advisory_lock($1)` with the per-service key.
- Deferred `pg_advisory_unlock` runs before the connection is released (defers LIFO), so the next caller of that pooled connection doesn't inherit the lock.
- The existing DDL executes on the locked connection.

**Why session-scoped (not transaction-scoped):** the DDL contains multiple statements that we don't want to wrap in a transaction (Postgres serializes DDL automatically per statement). Session lock + connection acquire keeps the scope narrow without forcing a transaction.

**Test added:** `TestMigrate_ConcurrentSafe` in `apps/crypto-go/main_test.go` — runs Migrate from 8 goroutines against a shared pool; all must return `nil`. Validates the lock pattern; the other 3 services use the identical pattern with only the key constant changed.

**Quality gates:** vet + tests green in all 4 services (test suites: crypto-go 13, macro-go 6, esios-go 9, cnmv-go 10).

---

## 2026-05-17 — Tier 2 item 14: graceful shutdown + slog in Go services

**Summary:** New shared helper `httpx.Run(ctx, addr, handler, shutdownTimeout)` wires `http.Server` with `srv.Shutdown` and `slog` lifecycle messages. All 5 Go services (`converter-go`, `crypto-go`, `macro-go`, `esios-go`, `cnmv-go`) replaced their `log.Fatal(http.ListenAndServe(...))` boot with `signal.NotifyContext(SIGINT, SIGTERM)` + `httpx.Run`. Render sends SIGTERM during redeploys; in-flight requests now have 15 s to drain.

**New file:** `apps/internal/httpx/server.go` — `Run` returns the listen error if the bind fails, or the `Shutdown` error if draining exceeded the timeout, or `nil` on clean shutdown. Uses `slog` (default text handler) for `server listening`, `shutdown signal received, draining`, `graceful shutdown failed`, `server stopped cleanly`.

**Tests added:** `TestRun_ShutdownOnCtxCancel` (start, GET /ping, cancel ctx, verify clean return within 3 s) and `TestRun_ListenError` (port already bound, expect `*net.OpError`). Bound to ephemeral ports (`127.0.0.1:0`) to avoid flakes.

**Scope kept tight:** ingestion CLI paths still use `log.Printf` (human-readable output for `make job-*` and `prod-*` targets). Migrating those to `slog` adds churn without HTTP-observability value.

**Quality gates:** `go vet` + `go test` green in all 5 services and `httpx`. Docker prod builds for `esios-go` and `cnmv-go` verified.

---

## 2026-05-17 — Tier 2 item 13: domain exceptions

**Summary:** Replaced 8 raw `throw new Error()` in `apps/api/src` with three named exception classes co-located in `apps/api/src/infrastructure/errors.ts`. Aligns with CLAUDE.md rule "Exceptions must be domain-specific (never throw raw Error)" and matches the existing precedent in `domain/exchange-rate/errors.ts` and `PaginationError`.

**New classes** (`apps/api/src/infrastructure/errors.ts`):
- `MissingEnvVarError(name)` — used in 5 boot sites (`main.ts`, `persistence/migrate.ts`, `persistence/db.ts`, `jobs/ingest.ts`, `jobs/backfill.ts`) for `DATABASE_URL` not set.
- `FrankfurterApiError(status, statusText)` — used twice in `FrankfurterClient` for non-OK upstream responses.
- `DownstreamFetchError(url, status)` — used in `dashboard.ts` `fetchService` helper that fans out SSR calls to crypto/macro/esios/cnmv services.

**Messages preserved** to keep backward-compatibility with existing tests asserting on regex (`/503.*Service Unavailable/`, `/404/`, `/500/`).

**No handler changes:** `errorHandler` keeps falling back to 500 for these — boot errors crash the process before Fastify, and downstream/provider errors as 500 is acceptable for now. Promoting to 502 Bad Gateway is a follow-up if desired.

**Tests:** 81 Node tests pass (unchanged).

---

## 2026-05-17 — CI: daily cron for BdE, ESIOS, CNMV ingestion

**Summary:** Extended `.github/workflows/ingest.yml` to cover the three Phase 12 providers. BdE runs as a new step inside the existing `macro` job (no new secret needed); ESIOS and CNMV run as new dedicated jobs. ESIOS gracefully no-ops with a workflow warning if `ESIOS_API_KEY` is missing, so the daily run keeps green until the REE token arrives.

**Workflow changes:**
- `macro` job — new step `Ingest BdE` (`apps/macro-go && go run . ingest-bde`)
- New job `esios` — runs `apps/esios-go && go run . ingest`; reads `ESIOS_API_KEY` secret; skips with `::warning::` if unset
- New job `cnmv` — runs `apps/cnmv-go && go run . ingest`; idempotent monthly upsert (current + previous month), kept on daily cron for consistency

**Secrets required (environment `prod`):** `DATABASE_URL` (all), `FRED_API_KEY` (macro), `ESIOS_API_KEY` (esios, optional).

**API surface:** no change.
**Schema:** no change.

---

## 2026-05-11 — Phase 12 (CNMV): Spanish investment funds microservice

**Summary:** New bounded context — Spanish investment funds from CNMV's monthly public-information files. Go microservice (`cnmv-go`, port 8130) scrapes the HTML listing page for tokenized ZIP URLs, downloads the archive, streams the XML, and exposes AIP-aligned REST endpoints with cursor pagination.

**Real-data validation:** Smoke-tested end-to-end against the live November 2025 file:
- FONDREGISTRO: 2.2 MB → **3,112 funds** parsed
- FONDMENS: 15.3 MB → **88,227 NAV observations** parsed
- Full ingest (download + parse + upsert) in ~5 seconds locally
- Endpoint `/api/v1/funds?gestora=MARCH` returns real MARCH ASSET MANAGEMENT funds with NAV, patrimonio, change_pct

**New service:** `apps/cnmv-go/`
- `GET /health`
- `GET /api/v1/funds?tipo=&gestora=&q=&page_size=&page_token=` — list with `next_page_token` + exact `total_size`
- `GET /api/v1/funds/{isin}` — detail (ISIN validated against ISO 6166 `^[A-Z]{2}[A-Z0-9]{9}[0-9]$`)
- `GET /api/v1/funds/{isin}/nav-observations?start_date=&end_date=&page_size=&page_token=` — time-series with date-cursor pagination
- `./cnmv-go ingest` — current + previous month, idempotent upsert
- `./cnmv-go backfill [fromYear]` — monthly walk from `fromYear` (default 2020) to now, ~2s pause between months

**Schema (new):**
- `cnmv_funds` (PK `isin CHAR(12)`) — catalog with gestora, depositario, ETF flag, currency, and reserved-for-FONDTRIM columns (TER, comisión gestión/depósito, vocación inversora)
- `cnmv_nav_observations` (UNIQUE `(isin, date)`) — NAV/partícipes/patrimonio per (ISIN, day)

**Key technical decisions:**
- **HTML scraping with regex** — CNMV download URLs are tokenized (`?e=OPAQUE_TOKEN`) and rotate; the listing page is the only deterministic discovery surface. Regex matches `_lnkZip` ids + Spanish month names in the `title` attribute.
- **XML streaming via `DecodeElement` per `<Entidad>` / `<Clase>`** — the 15 MB FONDMENS doesn't fit comfortably in a single decode tree, but per-clase blocks keep memory <100 MB. `daysFloat[32]` / `daysInt[32]` arrays decode the 31 day-tagged fields without declaring 31 struct members each.
- **`VL_DiaN = 0` filtered** as non-trading day; days beyond the month's length (Feb 30/31, Apr 31, ...) also dropped using `time.Date` arithmetic.
- **FONDTRIM (quarterly) deferred** — TER, comisiones, vocación inversora live there; schema already reserves the columns to avoid future migration.
- **AIP-158 pagination from inception** — critical because 3,112 ISINs would overflow without pagination.

**Out of scope (DGSFP):** Pension plans are regulated by DGSFP, not CNMV. DGSFP publishes only quarterly DECs, not daily NAV files. No equivalent ingestion path exists today.

**Tests added (8):** `TestHealthEndpoint`, `TestParseRegistro`, `TestParseMens`, `TestParseMens_ShortMonthDropsExtraDays`, `TestListMonthlyZips_HTMLScraping`, `TestUpsertAndFind`, `TestFundDetailEndpoint_InvalidISIN`, `TestFundsEndpoint_PaginationCursor`. Fixtures `testdata/FONDREGISTRO_sample.xml` and `testdata/FONDMENS_sample.xml` based on the real 202511 file structure.

**Operational:**
- `make job-cnmv` / `make job-cnmv-backfill`
- `docker-compose.yml` — new `cnmv-go` on port 8130
- `render.yaml` — new `tickerlab-cnmv` web service
- `.env.example` — `CNMV_GO_URL`

**Pending follow-ups (separate PR):**
- SSR pages `/funds` and `/funds/{isin}` with search/filter and Chart.js NAV history
- Backfill 2020-presente in production (~5M rows, ~30 min)
- FONDTRIM parser for fees and categories

---

## 2026-05-11 — Phase 12 (ESIOS): Spanish electricity microservice

**Summary:** New bounded context — Spanish electricity data from REE's e·sios API. Go microservice (`esios-go`, port 8120) ingests hourly observations and serves them via AIP-aligned REST endpoints with mandatory cursor pagination (per `api-design-standards.md`).

**New service:** `apps/esios-go/`
- `GET /health`
- `GET /api/v1/electricity/indicators?category=&page_size=&page_token=` — list with `next_page_token` + `total_size`
- `GET /api/v1/electricity/indicators/{indicator_id}/geos/{geo_id}/observations?start_date=&end_date=&page_size=&page_token=` — nested resource (AIP-122) with cursor over `datetime_utc`
- `./esios-go ingest` — incremental sync since `last_synced_at`
- `./esios-go backfill` — 2020 → now, chunked monthly to dodge undocumented response-size limits

**Tier 1 indicators (5, all geo_id=8741 Peninsula):**
- `1001` PVPC 2.0TD (pricing, EUR/MWh)
- `600` Demanda real (demand, MW)
- `10211` OMIE precio horario final (pricing, EUR/MWh)
- `1293` Generación programada PBF total (generation, MW)
- `10355` Factor emisiones CO2 (emissions, tCO2/MWh)

**Schema (new):**
- `esios_series` (PK `(indicator_id, geo_id)`) — catalog of tracked series
- `esios_observations` (UNIQUE `(indicator_id, geo_id, datetime_utc)`) — `TIMESTAMPTZ` for hourly granularity, stored in UTC

**Auth:** `x-api-key` header. Token obtained via email to `consultasios@ree.es` (~1–3 days). The service starts and serves reads without it; only `ingest`/`backfill` require it.

**Implementation notes:**
- Defensive geo_id filter — REE occasionally ignores `geo_ids[]` and returns every system; client re-filters.
- Backfill chunks by month (~720 hourly points × geo) — `/indicators/{id}` has no documented pagination.
- Reuses `apps/internal/httpx` for CORS, `WriteJSON`, `WriteProblem`, `ParsePagination`, `Page`.

**Tests added (8):** `TestHealthEndpoint`, `TestCorsMiddleware`, `TestFetchIndicator_FixtureParsing`, `TestFetchIndicator_NonOKStatus`, `TestSeedAndFindIndicators`, `TestSaveObservationsUpsertsValue`, `TestIndicatorsEndpointPagination`, `TestIndicatorsEndpoint_NegativePageSizeReturns400`. Fixture `testdata/indicator_1001.json` recorded from a real ESIOS response shape.

**Operational:**
- `make job-esios` / `make job-esios-backfill`
- `docker-compose.yml` — new `esios-go` service on port 8120
- `render.yaml` — new `tickerlab-esios` web service with build filter
- `.env.example` — `ESIOS_API_KEY` and `ESIOS_GO_URL`

**Pending follow-ups (separate PR):**
- SSR pages `/electricity` and `/electricity/:id/:geo` (Chart.js)
- GitHub Actions cron for daily ingest
- Smoke test against real API once a token is provisioned

---

## 2026-05-10 — Phase 12 (BdE): Spanish rates as third macro source

**Summary:** `macro-go` extended with a third upstream — Banco de España (BIEST) — adding 10 Spain-specific rate series under the new `spanish_rates` category. No new service; same schema; reuses existing `/api/v1/macro/...` endpoints.

**Series added (category `spanish_rates`):**
- Tier 1 — Mortgage reference (monthly): Euribor 1m / 3m / 6m / 12m, IRPH, IRS 5y
- Tier 2 — Euribor daily (`D_DNBAF172`)
- Tier 3 — TIPI Spain-only NEDR/TEDR (monthly): préstamos hogares vivienda, consumo; sociedades no financ.

**New code:** `apps/macro-go/bde.go` (`BDEClient`, gap detection, ISO-8601 → `YYYY-MM-DD` normalization). `bdeSeries` and `bdeDefaultRange` in `models.go`. `ingest-bde` subcommand and BdE block in `backfill`.

**Schema changes:** none. Reuses `macro_series` + `macro_observations` with `source='bde'`.

**Endpoints affected:** none new. `/api/v1/macro/indicators?category=spanish_rates` now returns the 10 BdE series. `/macro/bde/{id}/history` works through the existing generic handler.

**Operational:**
- `make job-macro-ingest-bde` — ingest only BdE.
- `make job-macro-ingest` — now runs FRED + ECB + BdE in sequence.
- `make seed-dev` — includes BdE in the macro step.

**Tests added (4):** `TestBDEDateNormalization`, `TestBDEParsing`, `TestBDEErrorResponse`, `TestBDEGapDetection`. Smoke-validated against the real API: 1,315 observations across 10 series.

**Gotcha worth recording:** `app.bde.es` drops requests without a recognizable User-Agent (returns EOF mid-handshake). The client sets `Mozilla/5.0 (compatible; ticker-lab-macro-go/1.0; ...)`.

---

## 2026-04-24 — Phase 11: Macro Indicators (FRED & ECB)

**Summary:** New bounded context — macro economic indicators from FRED (US) and ECB (Eurozone). Go microservice (`macro-go`) ingests 14 series and serves them via REST + SSR dashboard.

**New service:** `apps/macro-go/`
- `GET /health` — health check
- `GET /api/v1/macro/indicators?category=` — all indicators with latest value, grouped by category
- `GET /api/v1/macro/{source}/{id}/history?days=365` — historical data for an indicator
- `./macro-go ingest` — incremental FRED sync
- `./macro-go ingest-ecb` — incremental ECB sync
- `./macro-go backfill` — full historical backfill (FRED + ECB, from 2000)

**Indicators (14 series):**
- Inflation: CPI, PCE Price Index (FRED), HICP (ECB)
- Employment: Unemployment Rate, Nonfarm Payrolls (FRED)
- Interest Rates: Fed Funds, 10Y Treasury, 2Y Treasury, 10Y-2Y Spread (FRED), ECB MRR, ESTR (ECB)
- GDP: Real GDP (FRED)
- Monetary: M2 Money Supply (FRED)
- Housing: Case-Shiller Home Price Index (FRED)

**Schema changes:**
- `macro_series` table (source, series_id, name, frequency, unit, category)
- `macro_observations` table (source, series_id, value, date) with unique constraint

**Pages added:**
- `GET /macro` — indicators grouped by category with color-coded sections, change badges
- `GET /macro/:source/:id` — Chart.js chart with period selector (3M/6M/1Y/5Y/ALL), emerald theme

**Other:**
- FRED: API key auth, JSON response, 120 QPM
- ECB: no auth, CSV format, public API
- Docker: `docker/macro-go/Dockerfile`, service on port 8110
- GitHub Actions: CI (vet + test), daily ingest cron (FRED + ECB)
- Navigation: "Macro" link added to all pages
- 6 Go tests (health, CORS, indicators, history, save+find)

---

## 2026-04-19 — Phase 9: Currency Converter

**Summary:** Convert between any two of the 30 supported currencies using ECB rates. Supports cross-rates via EUR.

**Endpoints added:**
- `GET /api/v1/convert?from=EUR&to=USD&amount=100` — currency conversion

**Pages added:**
- `GET /converter` — interactive converter with dropdowns (30 currencies with flags), swap button, live result
- Navigation bar added to layout (Rates, Converter, API)

**Other:**
- `ConvertCurrency` use case with cross-rate calculation (e.g., GBP to JPY via EUR)
- OpenAPI spec v0.6.0 with ConversionResponse schema
- 32 tests (6 new: converter use case + endpoint)
- Feature noted for future reimplementation in Go as a separate microservice

---

## 2026-04-20 — Go Quality Gates

**Summary:** Added quality gates for Go microservices (vet + test), integrated into local `make ci` and GitHub Actions CI pipeline.

**Changes:**
- Makefile: `make go-vet`, `make go-test`, `make go-ci` targets; `make ci` now runs Node + Go gates
- Dockerfiles: `dev` stage added to crypto-go and converter-go (Go toolchain available for vet/test)
- docker-compose: Go services now target `dev` stage
- GitHub Actions `ci.yml`: new `go` job (vet + test for both services, with Postgres service)
- crypto-go: fixed test setup — `Migrate()` moved to `getTestPool()` so table exists for all integration tests

**Test count:** 35 Node + 15 Go = **50 tests**

---

## 2026-04-20 — Fix crypto template crash on undefined numeric fields

**Summary:** Fixed `TypeError: Cannot read properties of undefined (reading 'toFixed')` when the Go crypto service returns prices with missing numeric fields.

**Changes:**
- `dashboard.ts`: `sanitizeCryptoPrices()` defaults `price_eur`, `price_usd`, `change_24h` to 0
- Templates: defensive `?? 0` guards on `.toFixed()` calls in `dashboard.eta`, `crypto.eta`, `crypto-detail.eta`
- 3 new tests: crypto page with undefined fields, crypto detail with undefined fields, unreachable crypto service

---

## 2026-04-20 — Crypto Historical Backfill

**Summary:** Backfill historical crypto prices from CoinGecko market_chart API.

**Changes:**
- `FetchHistory` in CoinGecko client — fetches daily prices for a coin over N days
- `backfill` subcommand: `./crypto-go backfill 365` — iterates top 20 coins with 10s rate limit pause
- Makefile: `make prod-crypto-backfill` for production
- ~3.5 min for 20 coins × 365 days

---

## 2026-04-19 — Automated Crypto Ingestion

**Summary:** Crypto prices now ingested automatically via GitHub Actions cron.

**Changes:**
- `.github/workflows/ingest.yml` updated: two jobs (exchange-rates + crypto)
- Crypto: daily at 08:00 UTC, exchange rates: Mon-Fri 16:30 UTC
- Both jobs reference `environment: prod` for GitHub secrets

---

## 2026-04-19 — Crypto Frontend Integration

**Summary:** Integrated Go crypto microservice into the Node.js SSR dashboard.

**Pages added:**
- `GET /crypto` — top 20 crypto prices with EUR/USD, 24h change (green/red badges), links to detail
- `GET /crypto/:id` — Chart.js chart with period selector (purple theme to distinguish from exchange rates)

**Other:**
- Navigation bar updated: Rates / Crypto / Converter / API
- Node fetches from Go crypto service via `CRYPTO_GO_URL` env var
- docker-compose: Node service connected to Go services via Docker DNS
- Makefile: `make job-crypto` (local) and `make prod-crypto` (production)
- `.env.example` updated with `GO_CONVERTER_URL` and `CRYPTO_GO_URL`

---

## 2026-04-19 — Phase 8: Crypto (CoinGecko, Go)

**Summary:** Second bounded context — top 20 crypto prices from CoinGecko, implemented as a standalone Go microservice.

**New service:** `apps/crypto-go/`
- `GET /api/v1/crypto/latest` — all 20 crypto prices (EUR + USD)
- `GET /api/v1/crypto/{id}/history?days=90` — price history for a coin
- `GET /health` — health check
- `./crypto-go ingest` — fetch prices from CoinGecko and save to DB

**Coins:** BTC, ETH, SOL, BNB, XRP, ADA, DOGE, AVAX, DOT, POL, LINK, UNI, ATOM, LTC, FIL, APT, ARB, OP, NEAR, ICP

**Other:**
- Auto-creates `crypto_prices` table on startup (independent of Drizzle migrations)
- CoinGecko free tier (no API key)
- 6 Go tests (health, latest, history, CORS, save+find)
- Docker: `docker/crypto-go/Dockerfile` (~15MB image)
- docker-compose: crypto-go service on port 8090

---

## 2026-04-19 — Phase 9b: Go Converter Microservice

**Summary:** Currency converter reimplemented as a standalone Go microservice. Both engines (Node + Go) coexist and can be compared side by side.

**New service:**
- `apps/converter-go/` — Go stdlib HTTP server + pgx Postgres driver
- `GET /api/v1/go/convert?from=GBP&to=JPY&amount=1000` — same logic, Go runtime
- CORS enabled for cross-origin calls from the Node frontend
- Docker image: ~15MB (Alpine + static binary)

**Frontend:**
- Converter page now has Node.js / Go / Both toggle
- "Both" mode shows results side by side with response times (ms)
- Each response includes `"engine": "node"` or `"engine": "go"`

**Other:**
- `docker-compose.yml` updated with converter-go service
- `docker/converter-go/Dockerfile` — multi-stage Go build
- OpenAPI spec updated with `/api/v1/go/convert` endpoint
- 32 tests (Node side unchanged)

---

## 2026-04-19 — UI Polish: Light Theme & Currency Info

**Summary:** Redesigned dashboard with light theme, country flags, and currency names.

**Changes:**
- Light theme: white cards, subtle borders, Inter font, blue accents
- Each currency card shows flag emoji + full name (e.g., "US Dollar")
- Navigation bar: Rates, Converter, API links
- `currency-meta.ts` — metadata for 30 currencies (name, country, flag)

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
