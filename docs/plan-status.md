# Plan Status

> **Last updated:** 2026-05-17
>
> Live tracker for the post-Phase-11 work. Per-PR completion, test counts,
> and what's still pending. Pair with [`tech-debt-analysis.md`](./tech-debt-analysis.md)
> (rationale) and [`changelog.md`](./changelog.md) (chronological).

---

## Tier 1 — Phase 12 prep · 100% ✓

| # | Item                                                                       | Status |
| - | -------------------------------------------------------------------------- | :----: |
| 1 | Adopt [`api-design-standards.md`](./api-design-standards.md) as binding   |   ✓    |
| 2 | Adjust Phase 12 specs to AIP-122/132/158 URLs                              |   ✓    |
| 3 | Pagination helper Node + Go (AIP-158)                                      |   ✓    |
| 4 | `render.yaml` declares all services                                        |   ✓    |
| 5 | `apps/internal/httpx` Go shared package + service wiring                   |   ✓    |
| 6 | Drizzle repository integration tests — *caught real upsert bug*           |   ✓    |
| 7 | FrankfurterClient cassette tests                                           |   ✓    |

---

## Phase 12 — implementation

| Step | Provider | Service                     | Code | DB data (local)             | SSR pages                       |
| :--: | -------- | --------------------------- | :--: | --------------------------- | ------------------------------- |
| 1    | **BdE**  | extends `macro-go`          |  ✓   | 1,315 obs (10 series)       | reuses `/macro` + Spanish Rates |
| 2    | **ESIOS** | `esios-go` :8120           |  ✓   | 0 — *blocked on REE token*  | ✓ `/electricity` (empty state)  |
| 3    | **CNMV** | `cnmv-go` :8130             |  ✓   | 3,150 funds + 179k NAV obs  | ✓ `/funds` + detail (Chart.js)  |

---

## Tests

|                          | Before | Now    |   Δ   |
| ------------------------ | -----: | -----: | ----: |
| Node (Vitest)            |     35 |     81 |  +46  |
| Go (6 modules incl. httpx) |   21 |   ~70  |  +49  |
| **Total**                | **56** | **~151** | **+95** |

Quality gates: Biome ✓ · TypeScript strict ✓ · 6 Go modules vet/test ✓

---

## Issues surfaced & resolved during this sprint

Classified by category. The pre-existing production bugs are the
high-value finds — the rest is normal sprint iteration.

### Pre-existing production bugs caught by new work (2)

| # | Where                                 | Detected by              | Impact in production                                          | Fix                                       |
| - | ------------------------------------- | ------------------------ | ------------------------------------------------------------- | ----------------------------------------- |
| 1 | `DrizzleExchangeRateRepository.save()` | PR #5 integration test  | Daily ECB re-ingest silently never updated existing rates — `set: { rate: column.rate }` self-assigned | `set: { rate: sql\`excluded.rate\` }`   |
| 2 | `/converter` Go-engine button silent failure | manual smoke      | Browser fetched `http://converter-go:8080` (internal Docker hostname leaked into HTML); failed since the service was deployed | `docker-compose.yml`: `GO_CONVERTER_URL=http://localhost:8080` |

### Bugs I introduced this sprint and fixed before commit (3)

| # | Where                                              | When introduced       | Fix                                          |
| - | -------------------------------------------------- | --------------------- | -------------------------------------------- |
| 3 | CI failed with `relation "exchange_rates" does not exist` | PR #5 (new test)   | `migrate()` in `beforeAll`                   |
| 4 | All 5 Dockerfiles built binary at wrong path        | PR #3 (`-o /app/svc`) | `-o ./svc` (matches WORKDIR)                 |
| 5 | CNMV `last_seen_period` tagged with requested year (vs XML's `FechaDatos`) | CNMV PR | Trust `FechaDatos` from `ParseRegistro`     |

### External / environment quirks (2)

| # | Where                              | Resolution                                                          |
| - | ---------------------------------- | ------------------------------------------------------------------- |
| 6 | `TestSaveAndFindLatest` was flaky against real seeded DB | Test bug, not production: use far-future date (2099-12-31) and clean up via `t.Cleanup` |
| 7 | `app.bde.es` closes connections without a recognizable User-Agent | Set `User-Agent: Mozilla/5.0 (compatible; ticker-lab-macro-go/1.0; …)` |

---

## Pending

### Immediate (no extra cost)

| Item                                                  | Effort  | Blocking? |
| ----------------------------------------------------- | ------- | --------- |
| Wait for ESIOS token (email REE)                     | 1–3 d   | only for ESIOS smoke |
| Update `README.md` (test count stale, missing doc links) | ~10 min | no       |
| CNMV production backfill (Neon, ~5M rows / ~30 min)   | 30 min  | no        |

### Tier 2 polish (post-Phase 12, ~15–19 h)

| #   | Item                                                                 | Effort |
| --- | -------------------------------------------------------------------- | -----: |
| 8   | Plan `/api/v2/` migration for verb-shaped v1 endpoints              | 2 h    |
| 8b  | API gateway pattern (resolves service-URL leak documented in §6 of audit) | 4–6 h |
| 9   | Add `total_size` to remaining list responses                         | 1 h    |
| 10  | Eta template rendering tests                                         | 2–3 h  |
| 11  | Go handler integration tests (crypto-go, macro-go)                  | 4–6 h  |
| 12  | ✓ Migration advisory lock (`pg_advisory_lock`) (2026-05-17)          | done   |
| 13  | ✓ Domain exceptions — replaced 8 raw `throw new Error()` (2026-05-17) | done   |
| 14  | ✓ Graceful shutdown + `slog` in Go services (2026-05-17)             | done   |
| C   | ✓ GitHub Actions cron for BdE, ESIOS, CNMV daily ingest (2026-05-17) | done   |
| —   | SOCREGISTRO/SOCTRIM parser for SICAVs (CNMV quarterly)               | 4–6 h  |

### Hosting migration ([ADR-003](./decisions/003-hosting-strategy.md), separate cycle)

| Item                                                | Effort |
| --------------------------------------------------- | -----: |
| Confirm ADR-003 status (Proposed → Accepted)        | —      |
| Provision Hetzner VPS + DNS + Traefik (or Dokploy)  | 2 h    |
| `pg_dump` Neon → Postgres in VPS                    | 1 h    |
| GitHub Actions SSH-deploy (replace Render webhook)  | 2 h    |
| Cutover DNS + decommission Render + Neon            | 1 h    |

### v2 release window (Tier 3, separate cycle, ~21–26 h)

- Schema validation in CI (reject non-`snake_case` JSON, lists without pagination, errors without `code`)
- Chaos tests, performance benchmarks, Renovate, index audit
- Migrate v1 verb endpoints to v2 resource-style + sunset

---

## Recommended next action

**Tier 2 item C done (2026-05-17).** `ingest.yml` now covers BdE (inside the
`macro` job), ESIOS (skips if `ESIOS_API_KEY` unset) and CNMV. When the REE
token arrives, adding the secret is the only action required.

Next candidates: `README.md` update (10 min), Tier 2 item 8b (API gateway,
4–6 h, resolves the converter-URL leak class of bug for good), or item 13
(domain exceptions, 2 h).
