# API

## Contract

The OpenAPI 3.1 spec lives at `apps/api/openapi.yaml`. This is the **source of truth** for all HTTP endpoints.

## Viewing the Spec

During development, the raw YAML is the reference. A Swagger UI integration may be added later.

## Generating Types

```bash
make openapi-generate
```

This generates TypeScript types into `packages/shared/src/generated/api.ts`.

## Endpoints

### System

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Health check — returns `{ status: "ok", timestamp }` |

### Exchange Rates (Phase 2)

_Endpoints will be defined in the OpenAPI spec before implementation._

## Conventions

- All endpoints return JSON unless serving SSR HTML
- Error responses follow a consistent shape (defined in OpenAPI)
- Dates in ISO 8601 format
- Pagination via `?page=&limit=` query params when needed
