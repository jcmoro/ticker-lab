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

### Crypto (Go service, port 8090)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/crypto/latest` | Top 20 crypto prices (EUR + USD, sorted by market cap) |
| GET | `/api/v1/crypto/{id}/history` | Price history for a coin (`?days=90`) |
| GET | `/health` | Go crypto service health check |

**Supported coins:** BTC, ETH, SOL, BNB, XRP, ADA, DOGE, AVAX, DOT, POL, LINK, UNI, ATOM, LTC, FIL, APT, ARB, OP, NEAR, ICP

### Macro Indicators (Go service, port 8110)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/macro/indicators` | All indicators with latest value (`?category=inflation`) |
| GET | `/api/v1/macro/{source}/{id}/history` | Historical data for an indicator (`?days=365`) |
| GET | `/health` | Go macro service health check |

**Sources:** `fred` (FRED API) and `ecb` (ECB Data Portal)

**Series (14):** CPIAUCSL, UNRATE, FEDFUNDS, DGS10, GDPC1, DGS2, T10Y2Y, PCEPI, PAYEMS, M2SL, CSUSHPINSA (FRED) — ICP, FM_MRR, EST (ECB)

**Categories:** `inflation`, `employment`, `interest_rates`, `gdp`, `monetary`, `housing`

### SSR Pages

| Method | Path | Description |
|--------|------|-------------|
| GET | `/` | Dashboard — 30 currencies with flags, names, links to detail |
| GET | `/rates/:quote` | Detail — Chart.js line chart, period selector (30d/90d/180d/365d) |
| GET | `/crypto` | Top 20 crypto prices with 24h change (fetches from Go service) |
| GET | `/crypto/:id` | Crypto detail — Chart.js chart with period selector |
| GET | `/macro` | Macro indicators grouped by category with change badges |
| GET | `/macro/:source/:id` | Macro detail — Chart.js chart, period selector (3M/6M/1Y/5Y/ALL) |
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

### Crypto latest

```json
{
  "date": "2026-04-19",
  "count": 20,
  "prices": [
    {
      "coin_id": "bitcoin",
      "symbol": "BTC",
      "name": "Bitcoin",
      "price_eur": 63587,
      "price_usd": 74658,
      "market_cap_eur": 1272774376670.26,
      "change_24h": -1.0779,
      "date": "2026-04-19"
    }
  ]
}
```

### Crypto history

```json
{
  "coin_id": "bitcoin",
  "days": 90,
  "count": 90,
  "prices": [
    { "date": "2026-01-20", "price": 58234.12 },
    { "date": "2026-01-21", "price": 59123.45 }
  ]
}
```

### Macro indicators

```json
{
  "count": 14,
  "indicators": [
    {
      "source": "fred",
      "series_id": "CPIAUCSL",
      "name": "CPI (All Urban Consumers)",
      "category": "inflation",
      "unit": "index",
      "frequency": "monthly",
      "latest_value": 330.29,
      "latest_date": "2026-03-01",
      "prev_value": 327.46,
      "change": 2.83
    }
  ]
}
```

### Macro history

```json
{
  "source": "fred",
  "series_id": "CPIAUCSL",
  "name": "CPI (All Urban Consumers)",
  "days": 365,
  "count": 12,
  "points": [
    { "date": "2025-04-01", "value": 314.069 },
    { "date": "2025-05-01", "value": 314.927 }
  ]
}
```

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

## Data Sources

| Source | Data | Update frequency |
|--------|------|-----------------|
| [Frankfurter API](https://frankfurter.dev) | ECB exchange rates (30 currencies) | Daily (business days) |
| [CoinGecko API](https://www.coingecko.com/en/api) | Crypto prices (top 20 coins) | On demand (free tier, no API key) |
| [FRED API](https://fred.stlouisfed.org/docs/api/fred/) | US macro indicators (CPI, unemployment, rates, GDP) | Daily (API key required, 120 QPM) |
| [ECB Data Portal](https://data.ecb.europa.eu/help/api/overview) | Eurozone macro indicators (HICP, ECB rates, ESTR) | Daily (public, no auth) |

Historical exchange rates backfilled from 2024-01-01 (~17,500 records).
Macro indicators backfilled from 2000-01-01 (~49,000 records).

## Conventions

- All API endpoints return JSON
- SSR routes return HTML
- Dates in ISO 8601 format (YYYY-MM-DD)
- Currency codes follow ISO 4217 (3-letter uppercase)
- Amounts rounded to 2 decimal places, rates to 6
