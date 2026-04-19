# Architecture

## Overview

Ticker Lab follows a **hexagonal architecture** (ports & adapters) with manual dependency injection.

```
┌─────────────────────────────────────────────────┐
│                   Fastify SSR                    │  ← Views (templates)
├─────────────────────────────────────────────────┤
│                   HTTP Routes                    │  ← Infrastructure
├─────────────────────────────────────────────────┤
│                   Use Cases                      │  ← Application
├─────────────────────────────────────────────────┤
│            Domain (entities, ports)              │  ← Domain (zero deps)
├─────────────────────────────────────────────────┤
│   Frankfurter Client  │  Drizzle Repository     │  ← Infrastructure adapters
├───────────────────────┼─────────────────────────┤
│   Frankfurter API     │  PostgreSQL             │  ← External systems
└───────────────────────┴─────────────────────────┘
```

## Layers

### Domain (`src/domain/`)
Pure business logic. Entities, value objects, and **port interfaces**. Zero dependencies on frameworks or infrastructure.

**Current contents:**
- `exchange-rate/ExchangeRate.ts` — entity with factory function and validation (currency codes, rate positivity, date format)
- `exchange-rate/ExchangeRateProvider.ts` — port interface for external data sources
- `exchange-rate/ExchangeRateRepository.ts` — port interface for persistence
- `exchange-rate/errors.ts` — domain errors (InvalidCurrencyError, InvalidRateError, InvalidDateFormatError, RatesNotFoundError)

### Application (`src/application/`)
Use cases (commands and queries). Depends only on domain ports. Orchestrates business operations.

**Current contents:**
- `exchange-rate/IngestDailyRates.ts` — command: fetches rates from provider, saves to repository
- `exchange-rate/GetLatestRates.ts` — query: retrieves most recent rates from repository
- `exchange-rate/GetRatesByDate.ts` — query: retrieves rates for a specific date

### Infrastructure (`src/infrastructure/`)
Adapters that implement domain ports: HTTP server, database repositories, external API clients, scheduled jobs.

**Current contents:**
- `http/server.ts` — Fastify server setup with view engine and route registration
- `http/routes/health.ts` — health check endpoint
- `http/routes/exchange-rates.ts` — REST endpoints for exchange rate data
- `http/routes/dashboard.ts` — SSR dashboard route
- `persistence/schema.ts` — Drizzle ORM schema (exchange_rates table)
- `persistence/db.ts` — database connection
- `persistence/DrizzleExchangeRateRepository.ts` — implements ExchangeRateRepository port with upsert and queries
- `providers/FrankfurterClient.ts` — implements ExchangeRateProvider port, fetches from ECB via Frankfurter API
- `jobs/ingest.ts` — standalone script for daily ingestion (`make job-ingest`)

### Views (`src/views/`)
Eta templates rendered server-side by Fastify. Thin presentation layer — no business logic. Consumes API responses only.

**Current contents:**
- `layouts/main.eta` — base HTML layout (light theme, nav bar, Inter font)
- `pages/dashboard.eta` — ticker card grid with flags and currency names
- `pages/rate-detail.eta` — Chart.js chart with period selector
- `pages/converter.eta` — currency converter with Node/Go toggle

## Data Flow

### Ingestion (daily job)
```
make job-ingest
  → IngestDailyRates.execute("EUR")
    → FrankfurterClient.fetchLatest("EUR")
      → GET https://api.frankfurter.dev/v1/latest?base=EUR
    → createExchangeRate() for each rate (validation)
    → DrizzleExchangeRateRepository.save(rates)
      → INSERT ... ON CONFLICT UPDATE (upsert)
```

### REST API query
```
GET /api/v1/exchange-rates/latest?base=EUR
  → exchangeRateRoutes handler
    → GetLatestRates.execute("EUR")
      → DrizzleExchangeRateRepository.findLatest("EUR")
        → SELECT latest date, then SELECT rates for that date
    → format as { base, date, rates: [{ currency, rate }] }
```

### SSR dashboard
```
GET /
  → dashboardRoutes handler
    → GetLatestRates.execute("EUR")
    → reply.viewAsync("pages/dashboard", { rates, date })
      → Eta renders HTML with ticker card grid
```

## Go Microservices

Two standalone Go services share the same Neon Postgres database with the Node.js monolith.

```
┌──────────────────┐  ┌──────────────────┐  ┌──────────────────┐
│   Node (Fastify)  │  │ Go (converter)   │  │  Go (crypto)     │
│   :3000           │  │   :8080          │  │   :8090          │
│ exchange rates    │  │ /api/v1/go/      │  │ /api/v1/crypto/  │
│ converter (node)  │  │   convert        │  │   latest         │
│ dashboard SSR     │  │                  │  │   {id}/history   │
└────────┬─────────┘  └────────┬─────────┘  └────────┬─────────┘
         │                     │                      │
         └─────────────┬───────┴──────────────────────┘
                       │
                ┌──────┴───────┐
                │ Neon Postgres │
                │ (shared DB)   │
                └──────────────┘
```

### Converter Go (`apps/converter-go/`)
- Reimplements currency converter in Go
- Same logic, same contract (`ConversionResponse` with `engine` field)
- Frontend toggle lets you compare Node vs Go response times

### Crypto Go (`apps/crypto-go/`)
- Second bounded context: crypto prices from CoinGecko
- Own table (`crypto_prices`), auto-migrated on startup
- Ingestion via CLI: `./crypto-go ingest`
- Top 20 coins: BTC, ETH, SOL, BNB, XRP, ADA, DOGE, AVAX, DOT, POL, LINK, UNI, ATOM, LTC, FIL, APT, ARB, OP, NEAR, ICP

### Polyglot principles
- **Shared database:** all services read/write the same Neon Postgres
- **Independent deploy:** each service is a separate Render instance
- **Same OpenAPI spec:** all endpoints documented in one `openapi.yaml`
- **Independent lifecycle:** Go services auto-migrate their own tables

## Dependency Injection

Manual wiring in `main.ts` (composition root). No DI framework.

```
main.ts creates:
  postgres client → drizzle db
  → DrizzleExchangeRateRepository(db)
  → GetLatestRates(repository)
  → GetRatesByDate(repository)
  → GetRateHistory(repository)
  → ConvertCurrency(repository)
  → buildServer({ getLatestRates, getRatesByDate, getRateHistory, convertCurrency })
```

Server dependencies use structural typing (interfaces with `execute` method), not concrete class references. This allows tests to provide stubs without importing implementation classes.

## OpenAPI First

The `openapi.yaml` file is the source of truth for the HTTP contract. TypeScript types are generated from it via `openapi-typescript`. Handlers validate against this contract.

## Database Schema

```
exchange_rates
├── id              SERIAL PRIMARY KEY
├── base_currency   VARCHAR(3) NOT NULL
├── quote_currency  VARCHAR(3) NOT NULL
├── rate            NUMERIC(16,6) NOT NULL
├── date            DATE NOT NULL
├── created_at      TIMESTAMP DEFAULT NOW()
├── UNIQUE(base_currency, quote_currency, date)
└── INDEX(base_currency, date)
```
