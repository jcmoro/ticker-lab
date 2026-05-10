# Tech Debt Analysis & Test Expansion Plan

**Date:** 2026-05-10
**Scope:** Full monorepo (Node API + 3 Go services). Pre-Phase 12 baseline.
**Method:** Code audit across 10 dimensions, file:line citations.

---

## Executive summary

| # | Dimension | Severity | Status |
|---|-----------|---------:|--------|
| 1 | Code duplication in Go services | **High** | 60–80 lines of HTTP boilerplate replicated 3× |
| 2 | Zero integration tests for repositories & providers | **Medium** | CLAUDE.md mandates real-Postgres tests; not happening |
| 3 | `render.yaml` declares 1 service, production runs 4 | **Medium** | Manual setup drift, not in IaC |
| 4 | Go services lack graceful shutdown | **Medium** | `log.Fatal` on signal, in-flight requests dropped |
| 5 | Migration race: each Go service migrates on startup | **Medium** | Concurrent deploys could conflict |
| 6 | Service URL coupling — internal hostnames leak to browser | **Medium** | Root cause of `/converter` Go-engine regression (2026-05-10); structural fix = API gateway pattern |
| 7 | `dashboard.ts` 295 lines, 11 route handlers | Medium | Hard to test in isolation |
| 8 | Raw `Error` throws (8 locations) instead of domain exceptions | Low | Domain `errors.ts` exists, infra doesn't use it |
| 9 | Go uses unstructured `log` (text), Node uses Pino (JSON) | Low | Inconsistent observability |
| 10 | OpenAPI ↔ code aligned; deps healthy; no security smells | — | Good |

**Architecture is sound** — hexagonal layers respected, no `any` leaks, no SQL injection, no leaked secrets. The pain points are operational consistency and test depth, not design.

**Estimated remediation effort:** ~15–20 hours focused work (Tier 1 + half Tier 2 below). High ROI before adding 2 more Go services in Phase 12.

---

## Findings by dimension

### 1. Architecture compliance — Mostly OK

- No layer violations in `apps/api/src/`. Domain remains infrastructure-free.
- Views (SSR templates) contain no business logic.
- **Raw `Error` throws (8 locations)** that should use domain exceptions:
  - `apps/api/src/main.ts:14` — config error
  - `apps/api/src/infrastructure/providers/FrankfurterClient.ts:37, 63` — external API errors
  - `apps/api/src/infrastructure/http/routes/dashboard.ts:15` — fetch error
  - `apps/crypto-go/main.go:17, 86` — `log.Fatal()` on init failures
  - `apps/macro-go/main.go:17, 45, 53` — `log.Fatal()` on init failures
- Domain `errors.ts` exists and is used in domain/application; infrastructure bypasses it.

**Fix:** create `ConfigError`, `ExternalAPIError` domain types; catch and transform at HTTP layer to RFC 9457 ProblemDetails.

### 2. Code quality — Duplication is the biggest friction

**Quantified Go duplication:**

| Pattern | crypto-go | converter-go | macro-go | Phase 12 will add |
|---------|:---------:|:------------:|:--------:|:-----------------:|
| CORS middleware | ✓ | ✓ | ✓ | esios-go, cnmv-go |
| Health handler | ✓ | (—) | ✓ | esios-go, cnmv-go |
| ProblemDetails struct | ✓ | ✓ | ✓ | esios-go, cnmv-go |
| Migrate-on-startup pattern | ✓ | (—) | ✓ | esios-go, cnmv-go |
| Repository upsert pattern | ✓ | (—) | ✓ | esios-go, cnmv-go |

**Long file:** `apps/api/src/infrastructure/http/routes/dashboard.ts` — 295 lines, 11 route handlers, repetitive try/catch around external fetches.

**Magic values:**
- `dashboard.ts:48` — `Math.min(Number(days) || 90, 365 * 5)`
- `crypto-go/main.go:104` — `time.Sleep(15 * time.Second)`
- `macro-go/main.go:94` — `time.Sleep(500 * time.Millisecond)`

**Fix priority:** before Phase 12, extract `apps/internal/httpx/` (Go shared package) with CORS, health, problem details writer, migration pattern. Cuts ~60 lines per new service.

### 3. Test coverage — Repository and provider layers untested

| Layer | Files | Tests | Gap |
|-------|-------|-------|-----|
| Domain (Node) | `ExchangeRate.test.ts` | Unit | ✓ |
| Application (Node) | `GetLatestRates`, `IngestDailyRates`, `ConvertCurrency` | Unit (mocked repo) | ✓ |
| HTTP routes (Node) | `exchange-rates.test.ts`, `dashboard.test.ts`, `health.test.ts`, `api-docs.test.ts`, `metrics.test.ts` | Functional (Supertest) | Partial — error paths thin |
| **Persistence (Drizzle)** | — | **Zero** | **Critical** |
| **Providers (Frankfurter)** | — | **Zero** | **Critical** |
| **Templates (Eta)** | — | **Zero** | Medium |
| Go handlers | `main_test.go` × 3 | HTTP + CORS only | Repository + clients untested |

**CLAUDE.md mandate violated:** "No mocks for the database — always test against real Postgres via Testcontainers." Reality: Node tests mock the repository; CI uses a Postgres service for Go tests but Node integration tests don't run against it.

**Flakiness:** none observed. No `sleep`s in tests, no fixed `Date.now()`.

### 4. OpenAPI ↔ code drift — Aligned

- 8 endpoints + 2 health checks in spec, all registered in code.
- All schemas have `title`, `description`, `required`, `examples` (per CLAUDE.md / DR_0012).
- Error responses use RFC 9457 ProblemDetails consistently.
- Generated TS types match handlers. No drift.
- SSR routes (`/`, `/rates/:quote`, `/converter`, `/macro`, etc.) intentionally not in spec — view layer.

### 5. Dependencies — Healthy

- Fastify 5, Drizzle 0.40, TypeScript 5.8, Vitest 3.1, Biome 2 — all current.
- pgx/v5 in Go services — current.
- pnpm workspace coherent, no duplicate versions.
- No known CVE shapes.
- **Suggestion:** add Renovate or Dependabot to keep current.

### 6. Operational debt — Real drift

#### `render.yaml`
Declares only `tickerlab` (Node API). Production also runs:
- `tickerlab-crypto` → port 8090
- `tickerlab-converter` → port 8080
- `tickerlab-macro` → port 8110

Created manually in Render dashboard, not in IaC. **Fix before Phase 12:** add all services to `render.yaml` with explicit env vars.

#### Graceful shutdown
- Node API: handles SIGTERM/SIGINT, closes server + DB ✓ (`apps/api/src/main.ts:47-55`)
- Go services: **none** — `log.Fatal` on signal drops in-flight requests.

#### Service URL coupling — internal hostnames leak to the browser
`dashboard.ts:76, 84, 145` reads `process.env.GO_CONVERTER_URL` / `CRYPTO_GO_URL` / `MACRO_GO_URL` and either fetches server-side (crypto, macro) or **passes the URL into the rendered HTML** for client-side fetch (converter, see `views/pages/converter.eta:59`). Because `docker-compose.yml` set the env vars to internal Docker hostnames (`http://converter-go:8080`, `http://crypto-go:8090`, `http://macro-go:8110`), the converter page emitted `var GO_URL = 'http://converter-go:8080'` which the browser cannot resolve — the Go engine button silently failed.

**Reproduced and fixed 2026-05-10:** changed `GO_CONVERTER_URL` in `docker-compose.yml` to `http://localhost:8080` (the host-published port). Crypto and macro work because their fetches are SSR (run inside the API container's network).

**Why this is debt, not just a bug:** the design conflates "URL the API uses to reach a service" with "URL the browser uses to reach a service." They are not the same in any deployment topology (docker compose, Render free tier with separate subdomains, Hetzner with reverse proxy). Phase 12 will add `esios-go` and `cnmv-go` — if either ever needs a client-side fetch (e.g., to keep a chart paginating without round-tripping the SSR), the same bug recurs. The structural fix is item 8b below.

#### Logging
- Node: Fastify logger (Pino, structured JSON) ✓
- Go: `log.Printf` text. Inconsistent. Migrate to `log/slog` (Go 1.21+).

#### HTTP timeouts
- `dashboard.ts:14` uses `AbortSignal.timeout(2000)` ✓
- Go services: no context timeouts on outbound HTTP or DB queries. Risk: hung requests.

### 7. Documentation drift — Good

- `changelog.md` last entry 2026-04-24 (Phase 11) — matches code.
- `architecture.md` mentions all current services.
- ADRs: 001, 002, 003 — recent, status correct. Missing ADRs for Phase 9–11 design decisions (acceptable: commit history covers them).

### 8. Migration & DB hygiene — Race condition risk

- Each Go service calls `Migrate()` in `main()` at startup.
- Concurrent deploys (e.g., Render rolling restart of multiple services) could race on shared schema.
- **Fix:** advisory lock (`SELECT pg_advisory_lock(N)` per service) or move migrations offline.
- Indexes: not verified — some query patterns (e.g., `findLatest` by `(base_currency, date)`) likely benefit from explicit indexes.

### 9. Build & CI — Solid

- `.github/workflows/ci.yml`: Node and Go jobs run in parallel, real Postgres service, efficient pnpm cache.
- Dockerfiles multi-stage where it matters (`docker/api/Dockerfile`).
- Lefthook enforces lint/format/typecheck pre-commit per CLAUDE.md.

### 10. Security smell test — Clean

- No hardcoded secrets, all via env.
- Drizzle parameterized queries. Go raw SQL only with hardcoded values (`base_currency='EUR'`).
- CORS `*` everywhere — acceptable for personal project, document if scope changes.
- Input validation present in handlers.

---

## Test expansion plan

Ranked by ROI. Total Tier 1 + Tier 2 = ~15–20 hours. Do Tier 1 before Phase 12.

### Tier 1 — High impact, low effort (do before Phase 12)

API design / Google AIP items go first: skipping them now ships non-compliant URLs that would need a `v2` cycle to fix later.

| #   | Task                                                                                                                  | Effort           | Why                                                                                                                        |
| --- | --------------------------------------------------------------------------------------------------------------------- | ---------------: | -------------------------------------------------------------------------------------------------------------------------- |
| 1   | **Adopt `api-design-standards.md` as binding reference**                                                              | done             | Captures rules + deviations from Google AIP; referenced from new endpoint reviews                                          |
| 2   | **Adjust Phase 12 specs** (`esios-integration.md`, `cnmv-integration.md`) to Google-style URLs + `:verb` custom methods | 1 h              | Cheaper to fix specs now than to ship non-compliant endpoints and version them out                                         |
| 3   | **Implement pagination helper** (Node + Go shared util) — `page_size` / `page_token` / `next_page_token`              | 3–4 h            | Mandatory from inception per AIP-158; critical for `/funds` (3,112 ISINs); reused by all List endpoints                    |
| 4   | **Update `render.yaml`** with all 4 production services                                                               | 1 h              | Operational debt → IaC. Makes Phase 12 additions trivial                                                                   |
| 5   | **Extract `apps/internal/httpx` Go shared package** + tests once (CORS, health, ProblemDetails, pagination wiring)    | 3–4 h            | Eliminates 60+ lines per new service; Phase 12 inherits clean utils                                                        |
| 6   | **Drizzle repository integration tests** against real Postgres in CI                                                  | 2–3 h            | Closes biggest test gap; CI Postgres already available; validates upsert/conflict/range queries                            |
| 7   | **FrankfurterClient integration test** with recorded HTTP cassette                                                    | 2–3 h            | Catches API contract breakage; provider layer currently 100% untested                                                      |

**Total: ~12–15 hours.**

### Tier 2 — Medium impact, medium effort (next sprint)

| #   | Task                                                                                                                                                | Effort     | Why                                                                                                  |
| --- | --------------------------------------------------------------------------------------------------------------------------------------------------- | ---------: | ---------------------------------------------------------------------------------------------------- |
| 8   | **Plan `/api/v2/` migration** for existing verb-shaped endpoints (`/latest`, `/history`, `/convert`)                                                | 2 h scoping| Keep `v1` for back-compat; introduce `v2` aligned with Google AIP. Coordinate with OpenAPI changes   |
| 8b  | **Adopt API-gateway pattern** — Node API proxies all client-facing calls to downstream Go services; browsers receive only relative `/api/v1/...` URLs. Stops leaking internal hostnames; removes the `GO_CONVERTER_URL` browser-facing env var | 4–6 h      | Root cause of the 2026-05-10 converter regression (see §6). Prevents recurrence as Phase 12 services arrive and any future client-side fetches |
| 9   | **Add `total_size`** (optional, may be estimate) to list responses                                                                                  | 1 h        | Improves client UX; AIP-158 sanctioned                                                               |
| 10  | **Eta template rendering tests** (validate data shapes for `dashboard.eta`, `macro-detail.eta`, `crypto-detail.eta`)                                | 2–3 h      | Catches view/data mismatches before SSR runtime                                                      |
| 11  | **Go handler integration tests** against real Postgres (crypto-go, macro-go)                                                                        | 4–6 h      | Repository + handler coverage end-to-end                                                             |
| 12  | **Migration advisory lock** + idempotent re-run tests                                                                                               | 2 h        | Eliminates race risk between Go services                                                             |
| 13  | **Domain exceptions** + replace 8 raw `Error` throws                                                                                                | 2 h        | Architecture compliance with CLAUDE.md                                                               |
| 14  | **Graceful shutdown** in Go services + structured logging via `slog`                                                                                | 2–3 h      | Production hardening; consistent observability                                                       |

**Total: ~15–19 hours.**

### Tier 3 — Lower priority

| #   | Task                                                                                                                       | Effort  | Why                                                |
| --- | -------------------------------------------------------------------------------------------------------------------------- | ------: | -------------------------------------------------- |
| 15  | **Schema validation in CI** that rejects non-`snake_case` JSON keys, missing pagination on List endpoints, missing `code` on errors | 3 h     | Locks in standards going forward                   |
| 16  | Chaos tests (timeout / 5xx from external APIs)                                                                             | 4–6 h   | Resilience hardening                               |
| 17  | Performance benchmarks (Drizzle queries, Go JSON)                                                                          | 3–4 h   | Baseline for regressions                           |
| 18  | Renovate / Dependabot setup                                                                                                | 1 h     | Keeps deps current                                 |
| 19  | Index audit + explicit migration with required indexes                                                                     | 2 h     | Query performance under growth                     |

---

## Suggested execution order

```
Phase 12-prep (tech-debt sprint, ~12–15 h)
  1. Tier 1.2 — adjust Phase 12 specs to AIP-compliant URLs (1 h)   ← do first, unblocks everything else
  2. Tier 1.3 — pagination helper (3–4 h)
  3. Tier 1.4 — render.yaml fix (1 h)
  4. Tier 1.5 — Go httpx shared package + tests (3–4 h)             ← integrates pagination helper
  5. Tier 1.6 — Drizzle repo integration tests (2–3 h)
  6. Tier 1.7 — FrankfurterClient cassette tests (2–3 h)

Phase 12 implementation (esios-go, cnmv-go, BdE)
  inherits AIP-compliant URLs, pagination helper, clean Go utils, IaC parity

Post-Phase 12 polish (~15–19 h)
  Tier 2 items 8–14 (v2 planning, total_size, integration tests, race fix, exceptions, shutdown/logging)

v2 release window (separate cycle)
  Migrate v1 verb endpoints to v2 resource-oriented style
  Sunset v1 with proper deprecation headers
```

Item 1 goes first because every other Phase 12 task downstream depends on the URL shape (handlers, OpenAPI, dashboard SSR routes). Item 2 (pagination helper) feeds into item 4 (httpx package). The original test/operations items keep their value but follow the API-shape items.

---

## What was NOT analyzed

- Drizzle migration files (`apps/api/migrations/` not read in this audit).
- Lefthook configuration content (assumed from CLAUDE.md description).
- `make ci` actual current pass/fail state (would require running it).
- Index definitions in DB schema (assumed not exhaustive based on query patterns observed).

These should be verified before acting on Tier 2 items 7 (migration lock) and 13 (index audit).

---

## Addendum (2026-05-10) — API design audit vs Google AIP

The Google API Design Guide ([docs.cloud.google.com/apis/design](https://docs.cloud.google.com/apis/design), now hosted at `google.aip.dev`) is adopted as a normative reference. The binding subset, with explicit deviations, is captured in [`api-design-standards.md`](./api-design-standards.md). What follows is the gap audit of the current and planned API surface.

### Compliance summary

| AIP rule | Status | Notes |
|---|---|---|
| Resource-oriented (AIP-121) | **Partial** | `latest`, `history`, `convert` are verb-shaped paths, not resources |
| Resource names plural (AIP-122) | **OK** | `exchange-rates`, `crypto`, `funds`, `indicators` |
| URL casing (AIP-122 prefers `lowerCamelCase`) | **Deviation** | We use `kebab-case` (`exchange-rates`); kept for consistency |
| Get method (AIP-131) | OK | `GET /api/v1/exchange-rates/{date}` ✓ |
| List method (AIP-132) | **Gap** | Lists exist but lack pagination |
| Pagination (AIP-158) | **Critical gap** | No `page_size` / `page_token` / `next_page_token` anywhere |
| Custom methods (AIP-136) | **Gap** | `convert`, `latest`, `history` should use `:verb` syntax |
| Versioning (AIP-185) | OK | `/api/v1/` prefix consistent |
| Field naming (AIP-140) | OK (per our deviation §2.1) | `snake_case` JSON consistently used |
| Timestamps (AIP-142) | OK (per our deviation §2.2) | `_at` suffix, RFC 3339 UTC |
| Errors (AIP-193) | **Compatible** | RFC 9457 envelope + Google semantic codes (deviation §2.3) |

### Concrete endpoint findings

#### Current API

| Endpoint | Issue | Recommended (Google-style) |
|---|---|---|
| `GET /api/v1/exchange-rates/latest` | "latest" is not a resource ID | `GET /api/v1/exchange-rates:latest` (custom method) |
| `GET /api/v1/exchange-rates/history?from=&to=` | Verb in path | `GET /api/v1/exchange-rates?start_date=&end_date=` (List) |
| `GET /api/v1/convert?from=&to=&amount=` | Pure RPC, ambiguous resource | `GET /api/v1/exchange-rates:convert?from=&to=&amount=` |
| `GET /api/v1/crypto/latest` | Same as exchange-rates/latest | `GET /api/v1/crypto:latest` |
| `GET /api/v1/crypto/{id}/history?days=` | "history" is not a child collection | `GET /api/v1/crypto/{id}/observations?page_size=&page_token=` |
| `GET /api/v1/macro/indicators?category=` | No pagination | Add `page_size` + `next_page_token` |
| `GET /api/v1/macro/{source}/{id}/history?days=` | Same as crypto/history | `GET /api/v1/macro/{source}/indicators/{id}/observations` |

#### Phase 12 endpoints (designed before audit)

The specs in `esios-integration.md` and `cnmv-integration.md` need adjustment **before** implementation:

| Planned endpoint | Issue | Adjusted (Google-style) |
|---|---|---|
| `GET /api/v1/electricity/indicators?category=` | No pagination — manageable for 5–14 indicators but inconsistent | Add `page_size` + `next_page_token` |
| `GET /api/v1/electricity/{indicator_id}/{geo_id}/history?days=` | "history" verb-shaped | `GET /api/v1/electricity/indicators/{indicator_id}/geos/{geo_id}/observations?start_date=&end_date=` |
| `GET /api/v1/funds?tipo=&gestora=&q=&limit=50` | **Critical** — 3,112 ISINs without pagination | Replace `limit` with `page_size` + add `page_token` + `next_page_token` |
| `GET /api/v1/funds/{isin}/nav-history?days=365` | "nav-history" verb-shaped | `GET /api/v1/funds/{isin}/nav-observations?start_date=&end_date=&page_size=&page_token=` |

CNMV pagination is **not optional** — the Google rule "pagination must be implemented from inception" applies. Adding it post-launch breaks clients.

### Decision posture

Adopting the Google AIPs is a directional commitment, not a literal port. We keep deliberate deviations (snake_case JSON, RFC 9457 errors, `_at` timestamps) where Ticker Lab's existing conventions match better with Postgres/OpenAPI/HTTP idioms. The win is **resource-oriented URLs and mandatory pagination from inception** — both are cheap to do now and prohibitively expensive to retrofit later.
