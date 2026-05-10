# Future Data Providers

> **Status (2026-05-10)** — Phases 1–11 deployed on Render + Neon. Frankfurter (FX), CoinGecko (crypto), FRED + ECB (macro) live. Phase 12 candidates have validated integration specs ready to build.

---

## Roadmap snapshot

| # | Provider | Domain | Status | Effort | Service / Port | Spec |
|---|----------|--------|--------|--------|----------------|------|
| 1 | **ESIOS** | Spanish electricity | Spec ready | Medium | New `esios-go` / 8120 | [esios-integration.md](./esios-integration.md) |
| 2 | **CNMV** | Spanish funds NAV | Spec ready | High | New `cnmv-go` / 8130 | [cnmv-integration.md](./cnmv-integration.md) |
| 3 | **BdE** | Spanish rates | Spec ready | Low | Extends `macro-go` / 8110 | [bde-integration.md](./bde-integration.md) |
| 4 | **EIOPA** | EU insurance aggregates | Researched | Low–Medium | TBD | [comparison-providers-research.md](./comparison-providers-research.md) |
| 5 | **Xetra** | EU ETFs catalog | Researched | Low | TBD | [comparison-providers-research.md](./comparison-providers-research.md) |
| 6 | **ENTSO-E** | EU electricity | Researched | Medium | Could extend `esios-go` | [comparison-providers-research.md](./comparison-providers-research.md) |
| — | Equities, commodities | Markets | Backlog | — | — | — |

**Recommended sequence:** BdE (lowest effort, validates multi-source pattern) → ESIOS (high visual value, hourly data) → CNMV (highest effort, original project goal).

---

## 1. ESIOS — Spanish Electricity

### At a glance

| Field | Value |
|-------|-------|
| Provider | Red Eléctrica de España (REE) |
| Base URL | `https://api.esios.ree.es` |
| Auth | Header `x-api-key: <token>` |
| Token request | Email `consultasios@ree.es` (1–3 days) |
| Format | JSON (versioned via `Accept: application/vnd.esios-api-v1+json`) |
| Rate limit | Undocumented — recommend `time.Sleep(2s)` between calls |
| Granularity | Hourly (some series 5/10/15 min) |
| Cost | Free, attribution required |
| Service | New `esios-go` on port **8120** |
| DB schema | `esios_series` + `esios_observations` (TIMESTAMPTZ, UTC) |

### Endpoints consumed

| Method | Path | Use |
|--------|------|-----|
| GET | `/indicators` | Catalog discovery (one-off) |
| GET | `/indicators/{id}?start_date&end_date&time_trunc&geo_ids[]` | Main ingest |

**Required headers**

```
x-api-key: <TOKEN>
Accept: application/json; application/vnd.esios-api-v1+json
Content-Type: application/json
```

### Geo IDs

| `geo_id` | System |
|----------|--------|
| 8741 | España / Península (MVP only) |
| 8742 | Canarias |
| 8743 | Baleares |
| 8744 | Ceuta |
| 8745 | Melilla |

### Series tracked

**Tier 1 — MVP (5 indicators, all `geo_id=8741`)**

| ID | Name | Category | Unit | Frequency |
|----|------|----------|------|-----------|
| 1001 | Término facturación PVPC 2.0TD | `pricing` | EUR/MWh | hourly |
| 600 | Demanda real | `demand` | MW | hourly |
| 10211 | OMIE precio horario final | `pricing` | EUR/MWh | hourly |
| 1293 | Generación programada PBF total | `generation` | MW | hourly |
| 10355 | Factor emisiones CO2 | `emissions` | tCO2/MWh | hourly |

**Tier 2 — post-MVP**

| ID | Name | Category |
|----|------|----------|
| 1739 | Precio energía excedentaria autoconsumo | `pricing` |
| 1900 | Desglose peaje 2.0TD MAG | `pricing` |
| 2108 | Mecanismo gas-cap | `pricing` |
| 10070 | Generación hidráulica (PBF) | `generation` |
| 10071 | Generación nuclear (PBF) | `generation` |
| 10072 | Generación ciclo combinado (PBF) | `generation` |
| 10073 | Generación eólica (PBF) | `generation` |
| 10074 | Generación solar PV (PBF) | `generation` |
| 1159 | Generación eólica medida | `generation` |

### Endpoints exposed

```
GET /health
GET /api/v1/electricity/indicators?category=pricing&page_size=&page_token=
GET /api/v1/electricity/indicators/{indicator_id}/geos/{geo_id}/observations?start_date=&end_date=&page_size=&page_token=
```

### Schema

```sql
CREATE TABLE esios_series (
    indicator_id   INTEGER NOT NULL,
    geo_id         INTEGER NOT NULL,
    name           VARCHAR(200) NOT NULL,
    short_name     VARCHAR(50)  NOT NULL,
    category       VARCHAR(50)  NOT NULL,
    unit           VARCHAR(20)  NOT NULL,
    geo_name       VARCHAR(50)  NOT NULL,
    frequency      VARCHAR(10)  NOT NULL DEFAULT 'hourly',
    last_synced_at TIMESTAMPTZ,
    PRIMARY KEY (indicator_id, geo_id)
);

CREATE TABLE esios_observations (
    id            BIGSERIAL    PRIMARY KEY,
    indicator_id  INTEGER      NOT NULL,
    geo_id        INTEGER      NOT NULL,
    value         NUMERIC(20,6) NOT NULL,
    datetime_utc  TIMESTAMPTZ  NOT NULL,
    UNIQUE (indicator_id, geo_id, datetime_utc)
);
```

### Phases

1. Skeleton + ESIOS client + DB + tests
2. Backfill (chunked monthly) + GitHub Actions cron `30 5 * * *`
3. Dashboard SSR `/electricity` (amber accent) + `/electricity/:id/:geo` (Chart.js)
4. Tier 2 series + OpenAPI + changelog + architecture

→ **Full spec:** [esios-integration.md](./esios-integration.md)

---

## 2. CNMV — Spanish Investment Funds

### At a glance

| Field | Value |
|-------|-------|
| Provider | Comisión Nacional del Mercado de Valores |
| Listing page | `cnmv.es/portal/Publicaciones/Descarga-Informacion-Individual.aspx` |
| Download URL pattern | `cnmv.es/webservices/verdocumento/ver?e={OPAQUE_TOKEN}` |
| Auth | None (public) |
| Format | ZIP → XML + XSD per month |
| Cadence | **Monthly** ZIP containing **daily** NAVs of the month |
| Publication latency | ~10–15 days after month-end |
| Volume (snapshot 202511) | 1,441 entities, 3,112 ISINs, ZIP 2.2 MB, FONDMENS XML 15 MB |
| Annual rows | ~820k NAV observations |
| Service | New `cnmv-go` on port **8130** |
| DB schema | `cnmv_funds` + `cnmv_nav_observations` |

### Critical particularities

- **No daily-only file** — only monthly ZIP with daily NAVs inside.
- **URLs are tokenized and non-deterministic** — must scrape HTML with regex to find each month's link.
- **`VL_DiaN = 0` means non-trading day or missing data** — must filter. Days that don't exist in the month (Feb 30/31, Nov 31) also come as 0.
- **Pension plans NOT in CNMV** — DGSFP only publishes quarterly DECs. Out of scope.

### Files used (MVP)

| File | Cadence | Content |
|------|---------|---------|
| `FONDREGISTRO_YYYYMM.xml` | Monthly | Fund identification, gestora, depositario, ISIN, ETF flag |
| `FONDMENS_YYYYMM.xml` | Monthly | NAV/partícipes/patrimonio per day of month |

### Files deferred (post-MVP, FONDTRIM)

Quarterly file required for: vocación inversora (category), TER, comisión gestión/depósito, comisión suscripción/reembolso, official rentabilidad. Schema reserves these fields from day one — no migration needed when added.

### Endpoints exposed

```
GET /health
GET /api/v1/funds?tipo=FI&gestora=BBVA&q=indexa&page_size=&page_token=
GET /api/v1/funds/{isin}
GET /api/v1/funds/{isin}/nav-observations?start_date=&end_date=&page_size=&page_token=
```

### Schema

```sql
CREATE TABLE cnmv_funds (
    isin                       CHAR(12) PRIMARY KEY,
    tipo                       VARCHAR(8)   NOT NULL,    -- FI | FHF | SICAV | SHF
    numero_registro            BIGINT       NOT NULL,
    numero_compartimento       BIGINT       NOT NULL DEFAULT 0,
    numero_clase               BIGINT       NOT NULL DEFAULT 0,
    denominacion               VARCHAR(200) NOT NULL,
    is_etf                     BOOLEAN      NOT NULL DEFAULT FALSE,
    gestora_nombre             VARCHAR(200),
    gestora_grupo              VARCHAR(200),
    depositario_nombre         VARCHAR(200),
    depositario_grupo          VARCHAR(200),
    currency                   CHAR(3)      NOT NULL DEFAULT 'EUR',
    -- Reserved for FONDTRIM phase:
    vocacion_inversora         VARCHAR(50),
    ter                        NUMERIC(8, 4),
    comision_gestion           NUMERIC(8, 4),
    comision_deposito          NUMERIC(8, 4),
    last_seen_period           CHAR(6),
    created_at                 TIMESTAMP DEFAULT NOW(),
    updated_at                 TIMESTAMP DEFAULT NOW()
);

CREATE TABLE cnmv_nav_observations (
    id          BIGSERIAL PRIMARY KEY,
    isin        CHAR(12)       NOT NULL,
    date        DATE           NOT NULL,
    nav         NUMERIC(20, 6) NOT NULL,
    participes  BIGINT,
    patrimonio  NUMERIC(20, 2),
    UNIQUE (isin, date)
);
```

### Implementation notes

- HTML scraping with regex (no DOM parser needed).
- Streaming XML parser (`encoding/xml.Decoder.Token`) keeps memory <100 MB for 15 MB FONDMENS.
- User-Agent: `ticker-lab-cnmv-go/1.0 (+https://tickerlab.fly.dev)` — some CNMV endpoints 403 without realistic UA.
- Sync incremental: download current + previous month each run (covers late publications). Idempotent via upsert.
- Backfill: 2020 → present, ~72 ZIPs, ~5M rows, ~30 min execution.

### Phases

1. Skeleton + HTML scraper + `ParseRegistro` + funds endpoint
2. `ParseMens` streaming + NAV history endpoint + `ingest` subcommand
3. `backfill` subcommand + GitHub Actions cron `30 8 * * *`
4. Dashboard SSR `/funds` (search, filters, top movers) + `/funds/:isin` (Chart.js)
5. (optional) `FONDTRIM` parser → populate TER, vocación, comisiones
6. (optional) `/funds/compare?isins=...` with calculated returns + UI

→ **Full spec:** [cnmv-integration.md](./cnmv-integration.md)

---

## 3. BdE — Banco de España rates

### At a glance

| Field | Value |
|-------|-------|
| Provider | Banco de España (Statistics Web Service / BIEST) |
| Base URL | `https://app.bde.es/bierest/resources/srdatosapp/` |
| Auth | None (public) |
| Format | JSON gzip (Go decompresses automatically) |
| Rate limit | Undocumented — recommend `time.Sleep(1s)` |
| Cost | Free |
| Integration approach | **Extends existing `macro-go`** (no new service) |
| DB schema | Reuses `macro_series` + `macro_observations` with `source="bde"` |

### Endpoints consumed

| Method | Path | Use |
|--------|------|-----|
| GET | `/favoritas?idioma=es&series=ID` | Latest value |
| GET | `/listaSeries?idioma=es&series=ID&rango=30M` | Historical timeseries |

**`rango` is enum, not arbitrary dates**: monthly = `30M`, `60M`, `MAX`; daily = `3M`, `12M`, `36M`. Sync strategy: always fetch last N months and rely on upsert.

### Why integrate (not redundant with ECB)

| Concept | ECB SDW | BdE | Overlaps? |
|---------|---------|-----|-----------|
| ECB policy rates (MRR, DFR) | `FM` | `D_DNBCEB72` | Yes — use ECB |
| ESTR | `EST` | `D_1NBAS572` | Yes — use ECB |
| Euribor (any tenor) | Not exposed | `D_1NBAF472`, `D_DNBAF172` | **No — BdE only** |
| **IRPH** (Spain official mortgage ref) | None | `D_1T9H0000` | **No — BdE only** |
| **MIBOR** (legacy ES) | None | `D_1E723706.EUR` | **No — BdE only** |
| **NEDR/TEDR Spain banks** | Aggregate EU only | `DN_1TI2T*` | **No — Spain-only** |

### Series tracked (category `spanish_rates`)

**Tier 1 — Mortgage reference rates (monthly)**

| Code | Name |
|------|------|
| `D_1NBAF472` | Euribor 12m — referencia hipotecaria |
| `D_1NBAE972` | Euribor 6m — referencia hipotecaria |
| `D_1NBAD972` | Euribor 3m — referencia hipotecaria |
| `D_1NBAC972` | Euribor 1m — referencia hipotecaria |
| `D_1T9H0000` | IRPH — préstamos hipotecarios > 3 años |
| `D_1T9H0011` | IRS 5 años — referencia hipotecaria |

**Tier 2 — Euribor daily**

| Code | Name |
|------|------|
| `D_DNBAF172` | Euribor 12m (daily) |
| `D_DNBAS172`–`D_DNBAE172` | Euribor 1w / 1m / 3m / 6m (daily) |

**Tier 3 — TIPI Spain-only NEDR/TEDR (monthly)**

| Code | Name |
|------|------|
| `DN_1TI2T0135` | Préstamos hogares — vivienda (TEDR) |
| `DN_1TI2T0138` | Préstamos hogares — consumo (TEDR) |
| `DN_1TI2T0144` | Préstamos sociedades no financ. (TEDR) |

### Code changes (extends `macro-go`)

```
apps/macro-go/
  bde.go         # NEW — BIEST client, JSON parser
  models.go      # MODIFIED — add bdeSeries, bdeDefaultRange
  main.go        # MODIFIED — add `ingest-bde` subcommand
  main_test.go   # MODIFIED — TestBDEDateNormalization, TestBDEParsing
```

No schema migration. No new Dockerfile. No new port. No new env var.

### Phases

1. `bde.go` client + Tier 1 + Tier 2 series + `ingest-bde` subcommand + tests
2. Tier 3 series + `macro.eta` template label "Spanish Rates"
3. GitHub Actions cron + changelog + architecture update

→ **Full spec:** [bde-integration.md](./bde-integration.md)

---

## 4. EIOPA — EU Insurance Aggregates

### At a glance

| Field | Value |
|-------|-------|
| Provider | European Insurance and Occupational Pensions Authority |
| Access point | `data.europa.eu/data/datasets/eiopa-insurance-statistics-group-annual` |
| Auth | None |
| Format | CSV / SPARQL / Registry REST API |
| Cadence | Quarterly + annual |
| Cost | Free, public domain |
| Coverage | EU/EEA aggregated: balance sheet, premiums, claims, SCR, asset exposures, country breakdowns |

### What's available now vs. later

| | Now | Later (FIDA, ~2027) |
|---|---|---|
| Aggregate sector statistics | ✓ via EIOPA | ✓ |
| Per-product premium quotes | ✗ no public API | ✓ when FIDA enforced |
| Personalized comparisons | ✗ TOS + GDPR blockers | ✓ via standardized APIs |

**FIDA (Financial Data Access Regulation)** — open insurance equivalent of PSD2. Trilogue began Apr 2025, expected adoption mid-2026, implementation late 2027 with 24-month compliance window. Phase 2 explicitly covers motor insurance and investment products.

### Suggested integration shape (no spec yet)

- Quarterly cron (low operational burden).
- Could become `eiopa-go` standalone, or extend `macro-go` with `category="insurance"`.
- Decision deferred until selected for build.

→ **Research detail:** [comparison-providers-research.md](./comparison-providers-research.md#1-insurance)

---

## 5. Xetra ETFs

### At a glance

| Field | Value |
|-------|-------|
| Provider | Deutsche Börse / Xetra |
| URL | `xetra.com/xetra-en/instruments/etf-exchange-traded-funds/list-of-tradable-etfs` |
| Auth | None |
| Format | Daily CSV download |
| Coverage | EU-wide ETF reference list (ISIN, TER, replication, AUM) |
| Cost | Free |

### Complementary helper

- **OpenFIGI** (`openfigi.com`) — ISIN ↔ FIGI ↔ ticker normalization. 25 req/min anonymous, 250 with key. Useful when joining Xetra metadata with other sources.

### Suggested integration shape

- Could become a separate `xetra-go` service, or extend `cnmv-go` to cover ETFs alongside Spanish funds.
- Lower priority than CNMV (Spanish funds is the original goal).

→ **Research detail:** [comparison-providers-research.md](./comparison-providers-research.md#2-investment-products)

---

## 6. Backlog (lower priority)

### Equities & Indices

| Provider | Free tier | Notes |
|----------|-----------|-------|
| Alpha Vantage | 25 req/day, 500/month | API key, JSON, decent docs |
| Twelve Data | 800 credits/day | API key |
| Yahoo Finance (unofficial) | Brittle | TOS-restricted, fragile |

### Commodities

Cover via existing FRED series (gold, oil, gas) or add a dedicated commodity provider (Quandl-like). Low priority.

### ENTSO-E (EU electricity)

Free token (1–2 day approval), 400 req/min, 200 GB/month, XML only. Could extend `esios-go` once that service is mature.

---

## 7. Discarded options

| Provider | Reason |
|----------|--------|
| Rastreator, Acierto, Comparaiso | TOS forbids scraping; quotes require PII (GDPR) |
| Insurify, Policygenius | Partnership-only, no public API |
| Morningstar | Paid enterprise tier; free tier scraping forbidden |
| JustETF | Paid B2B; Cloudflare blocks bots |
| Quefondos, Rankia | TOS forbids automated extraction |
| Indexa Capital, MyInvestor, Finizens | Client-only OAuth, not catalog comparison |
| HelpMyCash, iAhorro, Bankimia | Aggressive anti-scraping, affiliate-driven |
| Roams, Rastreator (telco), Kelisto | TOS forbids scraping; free-market retail tariffs unobtainable legally |
| Yahoo Finance (production) | TOS + fragile endpoints |
| BEREC | Annual reports only, no API |

---

## 8. Watch list

| Item | Trigger | Why care |
|------|---------|----------|
| **FIDA implementation** | ~2027 | Standardized open-insurance APIs (per-product, per-customer) — would unlock real insurance comparison |
| **EIOPA Open Insurance pilots** | Ongoing | Sandbox announcements at `eiopa.europa.eu/browse/digitalisation-and-financial-innovation/open-insurance_en` |
| **CNMV FONDTRIM new release** | Quarterly | Triggers Phase 5 of CNMV integration (TER, vocación inversora) |
| **DGSFP open-data initiative** | None known | Pension plan NAV data has no current free source |

---

## References

- High-level research methodology: [comparison-providers-research.md](./comparison-providers-research.md)
- Detailed integration specs: [esios-integration.md](./esios-integration.md), [cnmv-integration.md](./cnmv-integration.md), [bde-integration.md](./bde-integration.md)
- Existing implementations as patterns: [macro-indicators-integration.md](./macro-indicators-integration.md), `apps/macro-go/`, `apps/crypto-go/`
- Architecture rules: [CLAUDE.md](../CLAUDE.md), [decisions/001-tech-stack.md](./decisions/001-tech-stack.md)
