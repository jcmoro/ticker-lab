# Ticker Lab

Financial data dashboard that ingests public economic data daily and displays it as a ticker-style dashboard.

**Current scope:** ECB exchange rates via [Frankfurter API](https://frankfurter.dev).

## Stack

| Layer | Technology |
|-------|-----------|
| Backend | Node.js 24 + Fastify 5 + TypeScript |
| Frontend | Fastify SSR (Eta templates) |
| Database | PostgreSQL 16 + Drizzle ORM |
| Contract | OpenAPI 3.1 (source of truth) |
| Quality | Biome + Vitest + Testcontainers |
| Infra | Docker + GitHub Actions + Fly.io |

## Quick Start

```bash
make setup   # Build containers, install dependencies
make dev     # Start development (http://localhost:3000)
make down    # Stop
```

## Commands

```bash
make help          # Show all available targets
make ci            # Run lint + typecheck + test
make lint          # Biome linter
make format        # Biome formatter
make typecheck     # TypeScript checks
make test          # All tests
make db-migrate    # Run migrations
make job-ingest    # Manual data ingestion
```

## Project Structure

```
apps/api/          Backend + SSR frontend (Fastify)
packages/shared/   Shared types (generated from OpenAPI)
docker/            Dockerfiles and Nginx config
docs/              Architecture, API docs, ADRs, roadmap
```

## Documentation

- [Architecture](docs/architecture.md)
- [API](docs/api.md)
- [Runbook](docs/runbook.md)
- [Future Providers](docs/future-providers.md)
- [Future Features](docs/future-features.md)
- [ADR-001: Tech Stack](docs/decisions/001-tech-stack.md)
- [ADR-002: Frontend SSR](docs/decisions/002-frontend-ssr.md)

## License

MIT
