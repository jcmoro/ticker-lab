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

### 5. Update Documentation

* Update `docs/changelog.md` if endpoints, schemas, or infrastructure changed
* Update `docs/api.md` if API surface changed
* Update `docs/architecture.md` if layers, data flow, or DB schema changed
* Update `docs/runbook.md` if new operational procedures or troubleshooting scenarios
* Update `README.md` if quick start, commands, or project structure changed

### 6. Deliver

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

## Documentation Standards (based on DR_0012)

### General Principles

* **Spec as source of truth:** OpenAPI YAML is the contract. Code, types, and documentation derive from it — never the other way around.
* **AI-consumable:** all fields must have non-trivial `description` and `examples`. Descriptions must include domain context, not just restate the field name. This applies to OpenAPI specs, JSON Schemas, and database documentation.
* **README.md mandatory at every `docs/` subdirectory.** Each README serves as index and context for its folder.
* **`docs/changelog.md` mandatory.** Reverse chronological. Updated when: adding/modifying/removing an endpoint, changing a request/response schema, changing database schema. Each entry: date, title, summary, affected endpoints, schema changes.

### Documentation Structure

```
docs/
├── README.md                  # Project overview and doc index
├── changelog.md               # Reverse-chronological change log
├── architecture.md            # Architecture overview and diagrams
├── api.md                     # API navigation guide
├── specs/
│   └── openapi/
│       └── v1/openapi.yaml    # Versioned OpenAPI spec (or root openapi.yaml)
│   └── schemas/               # JSON Schemas (requests/, responses/, shared/)
├── database/
│   └── schema.md              # ER diagram, tables, JSONB schemas
│   └── dependency-map.md      # Service dependencies (DBs, APIs, brokers)
├── decisions/                 # ADRs
│   └── ADR-NNN-short-desc.md
├── operations/
│   ├── runbook.md             # Oncall/SRE procedures
│   ├── troubleshooting.md     # Resolved problems knowledge base
│   └── development.md         # Local dev setup guide
├── future-providers.md
└── future-features.md
```

### OpenAPI Spec Rules

* **Format:** YAML only (not JSON — more readable, allows comments)
* **Version:** OpenAPI 3.1.0
* **Mandatory `info` fields:** `title`, `version`, `description` (must include domain context and service purpose), `contact`
* **Mandatory per endpoint:** `summary`, `description` (with business context), `operationId` (camelCase), `tags` (at least one domain tag), all `parameters` with description + example + schema, `requestBody` with description and complete example, all possible `responses` (2xx, 4xx, 5xx) with examples
* **Mandatory per schema:** `title`, `description`, `required` list, every property with `description`, `type`, `example`, and constraints (`format`, `enum`, `minLength`, `maxLength`, `minimum`, `maximum`) when applicable
* **Error format:** RFC 9457 `ProblemDetails` (`application/problem+json`) with fields: `type` (URI), `title`, `status`, `detail`, `code` (SCREAMING_SNAKE_CASE)

### JSON Schema Rules

* If data crosses a boundary (network, process, JSONB field), it must have a JSON Schema
* Every schema must have: `$schema`, `$id`, `title`, `description`
* Every property must have: `description`, `examples`
* Explicit `required` — never assume all fields are mandatory
* Schemas in `shared/` are reusable via `$ref`

### ADR Rules

* Path: `docs/decisions/ADR-NNN-short-description.md`
* NNN: sequential (001, 002...), description: kebab-case, max 5 words
* **Required fields:** Status (Proposed/Accepted/Deprecated/Superseded), Date, Context, Decision, Alternatives Considered (table), Consequences (positive/negative)
* **Write an ADR when:** choosing tech stack, choosing architecture patterns, making infrastructure decisions, deviating from established conventions
* Maintain an index in `docs/decisions/README.md`

### Operational Documentation

* **Runbook** (`docs/operations/runbook.md`): oriented to incident response. Incidents coded as INC-NNN with: symptoms, diagnosis, common causes (table), resolution (numbered steps). Commands must be copy-paste ready. Always include rollback as last option.
* **Troubleshooting** (`docs/operations/troubleshooting.md`): knowledge base of resolved problems. Each entry coded as PROB-NNN with: date, severity, symptoms, root cause (mandatory — "restarted and it worked" is insufficient), applied solution, prevention (mandatory). Reverse chronological.
* **Development** (`docs/operations/development.md`): local setup guide.

### Database Documentation

* Document: ER diagram (Mermaid preferred), table list with descriptions, key fields (PK, FK, indexes), JSONB fields linking to their JSON Schema
* Every JSONB field without a linked schema is a violation
* Maintain a dependency map: databases, external APIs consumed, services depended on

### Self-Verification (Documentation)

When delivering changes that affect documentation, verify:

* README.md exists and updated at affected `docs/` directories
* `docs/changelog.md` has entry if endpoints or schemas changed
* OpenAPI spec has descriptions + examples on all fields
* Schemas have `description` and `examples` on every property
* ADR written if a significant architectural decision was made
* Runbook updated if new failure modes introduced

---

## Communication Rules

* Assume reader understands DDD and Hexagonal architecture
* Do not explain architecture unless requested
* Focus on implementation and correctness
* Keep responses concise and technical
* Respond in the language the user uses
