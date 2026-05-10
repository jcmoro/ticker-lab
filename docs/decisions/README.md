# Architecture Decision Records (ADRs)

This directory contains the architectural decisions that shape Ticker Lab.

## Index

| #   | Title                                                | Status   | Date       |
| --- | ---------------------------------------------------- | -------- | ---------- |
| [001](001-tech-stack.md)        | Technology Stack                            | Accepted | 2026-04-19 |
| [002](002-frontend-ssr.md)      | Fastify SSR Instead of React SPA            | Accepted | 2026-04-19 |
| [003](003-hosting-strategy.md)  | Hosting Strategy — Hetzner Self-Hosted Multi-Project | Proposed | 2026-05-10 |

## Format

Each ADR follows the structure mandated by `CLAUDE.md`:

- **Status** — Proposed / Accepted / Deprecated / Superseded
- **Date** — when the decision was recorded
- **Context** — what problem is being solved
- **Decision** — what was chosen
- **Alternatives Considered** — table comparing options
- **Consequences** — positive and negative

File naming: `NNN-short-description.md` (sequential, kebab-case, max 5 words).

## When to write an ADR

Per `CLAUDE.md`:

- Choosing a tech stack component
- Choosing an architectural pattern
- Making an infrastructure decision
- Deviating from established conventions
