# ADR-001: Technology Stack

**Status:** Accepted
**Date:** 2026-04-19

## Context

Ticker Lab is a personal experiment to build a financial data dashboard. The developer is backend-focused, values SOLID principles and robustness, and has a strict 0€ budget.

## Decision

| Component | Choice | Rationale |
|-----------|--------|-----------|
| Runtime | Node.js 24 LTS | Current LTS, long support window |
| Language | TypeScript (strict) | Type safety, IDE support, catches errors early |
| Backend | Fastify 5 | Lightweight, fast, first-class TS support, native OpenAPI integration |
| Frontend | Fastify SSR + Eta | Minimal complexity, no separate build step, decoupled for future swap |
| Database | PostgreSQL 16 + Drizzle | Rock-solid, free, excellent for time-series financial data. Drizzle for explicit SQL and great TS types |
| Testing | Vitest + Supertest + Testcontainers | Fast, modern, real database in tests |
| Quality | Biome | All-in-one linter + formatter, faster than ESLint + Prettier |
| Monorepo | pnpm workspaces | Simple, fast, no extra tooling needed |
| Containers | Docker + docker-compose | Reproducible environments, required by project constraints |
| CI | GitHub Actions | Free for public repos, Docker-native |
| Deploy | Fly.io | Free tier, Docker-native, managed Postgres, HTTPS included |

## Alternatives Considered

- **NestJS** over Fastify: too much ceremony (decorators, DI framework) for a personal experiment
- **Prisma** over Drizzle: heavier (binary engine), less explicit SQL generation
- **Next.js** for frontend: SSR/SSG overkill when data updates daily and the developer is backend-focused
- **ESLint + Prettier** over Biome: two tools instead of one, slower

## Consequences

- All team members (currently one) need Docker installed
- No local Node.js required for running the app
- IDE TypeScript support still works via local node_modules
