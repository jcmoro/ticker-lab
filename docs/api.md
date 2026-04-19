# API

## Contract

The OpenAPI 3.1 spec lives at `apps/api/openapi.yaml`. This is the **source of truth** for all HTTP endpoints.

**Live:** https://tickerlab.onrender.com/api/docs (ReDoc)
**Local:** http://localhost:3000/api/docs
**Raw spec:** `/api/openapi.yaml`

## Generating Types

```bash
make openapi-generate
```

Generates TypeScript types into `packages/shared/src/generated/api.ts`.

## Endpoints

### System

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Health check — returns `{ status: "ok", timestamp }` |
| GET | `/ready` | Readiness check — verifies DB connectivity, returns 200/503 |
| GET | `/api/docs` | Interactive API documentation (ReDoc) |
| GET | `/api/openapi.yaml` | Raw OpenAPI 3.1 spec |
| GET | `/metrics` | Request counts, uptime, per-route stats (JSON) |

### Exchange Rates

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/exchange-rates/latest` | Latest rates for a base currency (`?base=EUR`) |
| GET | `/api/v1/exchange-rates/:date` | Rates for a specific date (`?base=EUR`) |
| GET | `/api/v1/exchange-rates/history` | Time series (`?quote=USD&from=2025-01-01&to=2026-04-17`) |
| GET | `/api/v1/convert` | Currency converter — Node.js (`?from=GBP&to=JPY&amount=1000`) |
| GET | `/api/v1/go/convert` | Currency converter — Go engine (same params, separate service) |

### SSR Pages

| Method | Path | Description |
|--------|------|-------------|
| GET | `/` | Dashboard — 30 currencies with flags, names, links to detail |
| GET | `/rates/:quote` | Detail — Chart.js line chart, period selector (30d/90d/180d/365d) |
| GET | `/converter` | Interactive converter — dropdowns with flags, swap, live result |

## Response Formats

### Exchange rates

```json
{
  "base": "EUR",
  "date": "2026-04-17",
  "rates": [
    { "currency": "USD", "rate": 1.1797 },
    { "currency": "GBP", "rate": 0.87168 }
  ]
}
```

### History

```json
{
  "base": "EUR",
  "quote": "USD",
  "from": "2025-01-01",
  "to": "2026-04-17",
  "rates": [
    { "date": "2025-01-02", "rate": 1.0358 },
    { "date": "2025-01-03", "rate": 1.0412 }
  ]
}
```

### Conversion

```json
{
  "from": "GBP",
  "to": "JPY",
  "amount": 1000,
  "rate": 215.770115,
  "result": 215770.12,
  "date": "2026-04-17"
}
```

Cross-rates (e.g., GBP to JPY) are calculated via EUR: `rate = EUR/JPY / EUR/GBP`.

## Error Format

Errors follow RFC 9457 ProblemDetails (`application/problem+json`):

```json
{
  "type": "https://tickerlab.dev/problems/not-found",
  "title": "Not Found",
  "status": 404,
  "detail": "No exchange rates found for EUR on 2026-04-20",
  "code": "RATES_NOT_FOUND"
}
```

## Data Source

Exchange rates come from the [Frankfurter API](https://frankfurter.dev), which provides ECB (European Central Bank) reference rates. Rates are updated once per business day (no weekends/holidays). 30 currencies supported (EUR + 29 quote currencies).

Historical data backfilled from 2024-01-01 (~17,500 rate records).

## Conventions

- All API endpoints return JSON
- SSR routes return HTML
- Dates in ISO 8601 format (YYYY-MM-DD)
- Currency codes follow ISO 4217 (3-letter uppercase)
- Amounts rounded to 2 decimal places, rates to 6
