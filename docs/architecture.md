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

### Application (`src/application/`)
Use cases (commands and queries). Depends only on domain ports. Orchestrates business operations.

### Infrastructure (`src/infrastructure/`)
Adapters that implement domain ports: HTTP server, database repositories, external API clients, scheduled jobs.

### Views (`src/views/`)
Eta templates rendered server-side by Fastify. Thin presentation layer — no business logic. Consumes API responses only.

## Data Flow

1. **Ingestion (daily cron):** Job → `IngestDailyRates` use case → `FrankfurterClient` (port impl) → Domain entity → `ExchangeRateRepository` (port impl) → PostgreSQL
2. **Query (HTTP):** Request → Fastify route → `GetLatestRates` use case → `ExchangeRateRepository` → Response
3. **Dashboard (SSR):** Request → Fastify route → calls API internally → Eta template → HTML response

## Dependency Injection

Manual wiring in `main.ts` (composition root). No DI framework. Constructors receive port interfaces, `main.ts` provides concrete implementations.

## OpenAPI First

The `openapi.yaml` file is the source of truth for the HTTP contract. TypeScript types are generated from it via `openapi-typescript`. Handlers validate against this contract.
