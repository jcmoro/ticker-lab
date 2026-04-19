# API

## Contract

The OpenAPI 3.1 spec lives at `apps/api/openapi.yaml`. This is the **source of truth** for all HTTP endpoints.

**Interactive docs:** http://localhost:3000/api/docs (ReDoc)
**Raw spec:** http://localhost:3000/api/openapi.yaml

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
| GET | `/api/v1/exchange-rates/latest` | Latest exchange rates for a base currency |
| GET | `/api/v1/exchange-rates/:date` | Exchange rates for a specific date |

**Query parameters:**

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `base` | string | `EUR` | ISO 4217 base currency code (3 letters) |

**Response format:**

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

Returns an empty `rates` array if no data is available for the requested base/date.

### SSR Dashboard

| Method | Path | Description |
|--------|------|-------------|
| GET | `/` | HTML dashboard showing EUR exchange rates as ticker cards |

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

Exchange rates come from the [Frankfurter API](https://frankfurter.dev), which provides ECB (European Central Bank) reference rates. Rates are updated once per business day (no weekends/holidays). Currently 29 currencies against EUR.

## Conventions

- All API endpoints return JSON
- SSR routes (`/`) return HTML
- Dates in ISO 8601 format (YYYY-MM-DD)
- Currency codes follow ISO 4217 (3-letter uppercase)
