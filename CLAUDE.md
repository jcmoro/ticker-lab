# CLAUDE.md

This file defines how Claude Code must operate when working in this repository.

---

## Context Priority

When rules conflict, prioritize in this order:

1. Non-Negotiable Rules
2. Execution Protocol
3. Self-Verification
4. Quality Gates
5. Remaining documentation

Explicit user instructions always override this file.

---

## Project Overview

**Ticker Lab** — personal experiment: a financial data dashboard that ingests public economic data daily and displays it as a ticker-style dashboard.

**Current scope:** exchange rates from the ECB via [Frankfurter API](https://frankfurter.dev) as the first data provider. Future providers (investment funds, pension plans, crypto, macro indicators) are documented in `/docs/future-providers.md`.

### Stack

| Layer | Technology |
|-------|-----------|
| Runtime | Node.js 24 LTS |
| Language | TypeScript (strict mode) |
| Backend | Fastify 5 |
| Frontend | Fastify SSR (lightweight templates, decoupled — replaceable by SPA later) |
| Database | PostgreSQL 16 + Drizzle ORM |
| API contract | OpenAPI 3.1 (YAML, source of truth) |
| Testing | Vitest (unit + functional) + Supertest (HTTP) + Testcontainers (Postgres) |
| Linting/Formatting | Biome |
| Git hooks | Lefthook (pre-commit, commit-msg, pre-push) |
| Containers | Docker + docker-compose (dev & prod) |
| CI | GitHub Actions |
| Monorepo | pnpm workspaces |
| Deploy | Fly.io (tickerlab.fly.dev) |

### Infrastructure

* PostgreSQL 16 (via Docker, both dev and prod)
* Docker-only execution environment — no local Node.js required
* Nginx for static asset serving in production (frontend container)

---

## Runtime Rules

* All execution happens inside Docker containers
* TypeScript `strict: true` mandatory — no `any`, no implicit types
* Prefer `readonly` properties and immutable data structures
* Prefer union types / enums over string constants
* No `any` or `unknown` casts without explicit justification
* Exceptions must be domain-specific (never throw raw `Error`)
* All async code must handle errors explicitly (no unhandled rejections)

---

## Common Commands

All commands executed via `make`. No direct `npm`/`pnpm` commands outside containers.

### Environment

* `make setup` — install dependencies and initialize environment
* `make dev` — start development environment (docker-compose up)
* `make down` — stop all containers
* `make clean` — remove containers, volumes, node_modules

### Testing

* `make test` — run all tests
* `make test-unit` — unit tests only
* `make test-functional` — functional/integration tests only

### Quality

* `make lint` — run Biome linter
* `make format` — run Biome formatter
* `make typecheck` — TypeScript type checking
* `make ci` — full CI pipeline locally (lint + typecheck + test)

### Contracts

* `make openapi-generate` — generate TypeScript types from OpenAPI spec
* OpenAPI spec location: `apps/api/openapi.yaml`

### Database

* `make db-migrate` — run Drizzle migrations
* `make db-seed` — seed development data

### Jobs

* `make job-ingest` — manually trigger daily data ingestion

Git hooks (via Lefthook) enforce: linting, formatting, type checking, conventional commits, and tests on push.

---

## Architecture

### Design Approach

Hexagonal architecture (ports & adapters) with manual dependency injection. No DI framework — composition root in `main.ts` wires everything explicitly.

**OpenAPI first:** the `openapi.yaml` file is the source of truth for the HTTP contract. TypeScript types are generated from it. Handlers validate against this contract.

**Fastify SSR:** the frontend is rendered server-side by Fastify using lightweight templates (e.g., eta/nunjucks). This keeps the frontend decoupled — it consumes the same API contract and can be replaced by a React SPA or any other frontend later without touching the backend.

### Bounded Contexts

* **Exchange Rates** — ingestion and serving of ECB exchange rate data (Frankfurter provider)
* _(Future: Funds, Pensions, Crypto, Macro — see `/docs/future-providers.md`)_

### Layer Structure (Mandatory)

```
apps/api/src/
├── domain/           # Entities, value objects, ports (interfaces)
│                     # ZERO dependencies on infrastructure or frameworks
├── application/      # Use cases (commands/queries)
│                     # Depends only on domain ports
├── infrastructure/   # Adapters: FrankfurterClient, DrizzleRepository, FastifyServer
│                     # Implements domain ports
├── views/            # SSR templates (Fastify rendering)
│                     # Consumes API responses, contains NO business logic
└── main.ts           # Composition root — manual DI wiring
```

### Architectural Patterns

* **Hexagonal (Ports & Adapters):** domain defines ports (interfaces), infrastructure provides adapters
* **CQRS (light):** separate command (ingest/write) and query (read) paths
* **Repository pattern:** data access abstracted behind domain-defined interfaces
* **OpenAPI-first:** contract defined before implementation, types generated from spec
* **SSR as thin view layer:** templates render data, all logic lives in the API

---

## Non-Negotiable Rules

* Respect bounded context boundaries
* No cross-context shortcuts
* Domain layer has zero infrastructure/framework dependencies
* View/SSR layer contains no business logic
* Repositories accessed only from Application/Infrastructure layers
* Maintain CQRS separation (read vs write paths)
* Prefer consistency over novelty
* Changes must be minimal and focused
* All code must be reachable from Docker — no "works on my machine" paths
* OpenAPI spec must be updated before or together with any API change

---

## Approach (Agent Operating Rules)

* Understand context before making changes
* Inspect relevant existing files first
* Prefer minimal edits over rewriting files
* Re-check files before modifying if context may have changed
* Follow existing architectural patterns
* Keep solutions simple and explicit
* Provide concise, implementation-focused responses
* Do not expose internal reasoning unless explicitly requested
* Avoid conversational filler
* Language: respond in the same language the user writes in

---

## Execution Protocol (Mandatory)

Follow this sequence for every task:

### 1. Understand

* Identify bounded context
* Identify affected layer(s)
* Read relevant files
* Confirm architectural constraints

### 2. Plan

* Determine minimal required changes
* Extend existing patterns
* Identify required tests and contract impact

### 3. Implement

* Apply focused modifications
* Respect layer boundaries
* Prefer editing existing code
* Update OpenAPI spec if API surface changes

### 4. Validate

* Ensure quality gates pass
* Verify OpenAPI contract when affected
* Confirm architecture rules remain satisfied

### 5. Deliver

* Provide concise summary of changes
* Avoid unnecessary explanations

---

## Planning Constraints

* Do not implement before understanding existing patterns
* Assume repository already contains the correct architectural solution
* Prefer consistency over introducing new abstractions

---

## Self-Verification (Mandatory Before Delivering)

Before considering work complete, verify:

### Architecture

* No layer violations
* No cross-context dependency introduced
* Domain remains infrastructure-free

### Code Quality

* Matches patterns already used in the context
* No unnecessary abstractions
* Minimal modification surface
* TypeScript strict — no `any` leaks

### Contracts

* OpenAPI valid if affected
* DTOs consistent with schemas
* Generated types regenerated if spec changed

### Testing

* Required unit or functional tests exist
* Tests validate behavior, not implementation details
* Tests run inside Docker (Testcontainers for Postgres)

If any rule fails -> revise before delivering.

---

## Overengineering Guard

Reject solutions that:

* Introduce new architectural patterns without precedent in this repo
* Add abstractions not required by the task
* Modify multiple bounded contexts unnecessarily
* Replace working implementations without explicit reason
* Add DI frameworks, decorators, or metaprogramming
* Introduce frontend complexity beyond what SSR templates require

---

## Testing Strategy

* **Unit tests:** domain entities, value objects, use cases — pure functions, no I/O
* **Functional tests:** HTTP endpoints via Supertest against real Fastify instance
* **Integration tests:** repository adapters against real Postgres via Testcontainers
* **No mocks for the database** — always test against real Postgres
* Tests collocated next to source: `*.test.ts` alongside `*.ts`

---

## Quality Gates (Required Before Completion)

All must be green:

* `make lint` — no Biome errors
* `make format` — no formatting issues
* `make typecheck` — no TypeScript errors
* `make test` — all tests pass
* OpenAPI spec valid (if changed)

Work is not complete until all gates pass.

---

## Communication Rules

* Assume reader understands DDD and Hexagonal architecture
* Do not explain architecture unless requested
* Focus on implementation and correctness
* Keep responses concise and technical
* Respond in the language the user uses
