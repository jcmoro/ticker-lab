# API Design Standards

**Source:** Google API Improvement Proposals (AIPs) — `https://google.aip.dev/` (formerly `cloud.google.com/apis/design`).
**Status:** Adopted with explicit deviations.
**Date:** 2026-05-10.

This document distills the subset of Google's API Design Guide that is **binding** for Ticker Lab, plus the deviations we accept on purpose. It complements `CLAUDE.md` and `decisions/001-tech-stack.md`.

---

## 1. Rules adopted

### 1.1 Resource-oriented design (AIP-121)

- Design around **nouns (resources)**, not verbs.
- Apply the **standard methods** where applicable: List, Get, Create, Update, Delete (AIPs 131–135).
- Custom methods are reserved for operations that don't fit the CRUD pattern (e.g., currency conversion).

### 1.2 Resource naming (AIP-122)

- Collection segments are **plural nouns** in the URL path.
- Resource hierarchy alternates `collection/resource_id/sub_collection/sub_resource_id`.
- Resource IDs follow RFC-1034 where possible: `^[a-z]([a-z0-9-]{0,61}[a-z0-9])?$`. Stable, opaque to clients.

### 1.3 List method (AIP-132 + AIP-158 pagination)

- Use `GET /collection`. Query params for filters.
- **Pagination is mandatory from inception** — adding it later breaks compatibility:
  - Request: `page_size` (int, optional), `page_token` (string, optional).
  - Response: top-level `items` array (or named after resource) + `next_page_token` (string). Empty `next_page_token` signals end.
  - `page_size` defaults to a sensible value when omitted; clamps to a maximum.
  - `page_token` is opaque, URL-safe; clients must not parse it.
  - Negative `page_size` → 400 with `INVALID_ARGUMENT` semantic.
- Optional `total_size` (int, may be an estimate).

### 1.4 Get method (AIP-131)

- `GET /collection/{id}` returns the resource.
- 404 if not found, with structured error body.

### 1.5 Custom methods (AIP-136)

- For non-CRUD operations, use `:verb` suffix on resource or collection:
  - `GET /api/v1/exchange-rates:convert?from=…&to=…&amount=…`
  - `GET /api/v1/exchange-rates:latest`
- Convention: `:` separates the resource path from the action verb. Verb is `lowerCamelCase`.

### 1.6 Versioning (AIP-185)

- URL path versioning: `/api/v1/...`.
- Bump to `v2` only on breaking changes; backward-compatible additions stay in `v1`.

### 1.7 Naming conventions (AIP-190 + AIP-140)

- **JSON field casing: `snake_case`** (Ticker Lab convention — see deviation §2.1 below).
- Resource type names in code: `UpperCamelCase`.
- Method/operation names in code: `UpperCamelCase`.
- Field names in JSON: `snake_case` (`indicator_id`, `latest_at`, `next_page_token`).
- Timestamps: end with `_at` (e.g., `created_at`, `synced_at`). See §2.2 below.
- Avoid abbreviations except universally understood ones (`id`, `url`, `api`).

### 1.8 Standard fields (AIP-148, partially adopted)

- Time fields: RFC 3339, UTC (e.g., `2026-05-10T18:00:00Z`).
- Resource identifier field: `id` (string), exposed in responses.
- Timestamps used in time-series tables: `date` (calendar day, ISO 8601 `YYYY-MM-DD`) where granularity is daily.

### 1.9 Errors (RFC 9457 ProblemDetails — see §2.3 deviation)

- Content type: `application/problem+json`.
- Required fields: `type` (URI), `title`, `status`, `detail`, `code` (SCREAMING_SNAKE_CASE).
- HTTP status mapping:
  - 400 → `INVALID_ARGUMENT`
  - 401 → `UNAUTHENTICATED`
  - 403 → `PERMISSION_DENIED`
  - 404 → `NOT_FOUND`
  - 409 → `ALREADY_EXISTS` / `FAILED_PRECONDITION`
  - 429 → `RESOURCE_EXHAUSTED`
  - 5xx → `INTERNAL` / `UNAVAILABLE`
- Codes use `SCREAMING_SNAKE_CASE`, mirroring Google's `google.rpc.Code` enum where possible.

---

## 2. Deliberate deviations from Google AIP

### 2.1 JSON field casing — `snake_case` (not `lowerCamelCase`)

| | Google convention | Ticker Lab |
|---|---|---|
| JSON | `lowerCamelCase` (`nextPageToken`) | **`snake_case`** (`next_page_token`) |

**Why:** snake_case is unambiguous across SQL, OpenAPI schemas, and Postgres column names. The project ingests data from FRED (snake_case), ECB CSVs, CNMV XMLs (mixed) and stores in Postgres — exposing snake_case in JSON keeps a single canonical form across the stack. Cost of conversion to lowerCamelCase outweighs the alignment with Google REST.

**Scope:** all JSON responses across `apps/api`, `apps/crypto-go`, `apps/converter-go`, `apps/macro-go`, `apps/esios-go`, `apps/cnmv-go`.

### 2.2 Timestamp field naming — `_at` suffix (not Google's `_time`)

| | Google convention | Ticker Lab |
|---|---|---|
| Suffix | `create_time`, `update_time`, `delete_time` | **`created_at`, `updated_at`, `synced_at`** |

**Why:** existing Drizzle and Go code already uses `_at`. Aligning would require migration work for negligible benefit.

### 2.3 Error format — RFC 9457 (not `google.rpc.Status`)

**Why:** RFC 9457 is HTTP-native, simpler, well-supported by tooling, and mandated by `CLAUDE.md` Documentation Standards. Google's Status uses Protobuf `Any` and is gRPC-native. We adopt Google's **HTTP status → semantic code** mapping (§1.9) but keep the RFC 9457 envelope.

### 2.4 No proto-first / no gRPC

**Why:** Ticker Lab is OpenAPI-first per ADR-001. Google's guide leans gRPC + Protobuf. We adopt the resource-oriented and naming principles, not the wire format.

---

## 3. Application checklist (per new endpoint)

Before merging a new endpoint:

- [ ] URL uses **plural noun collection**.
- [ ] Resource ID follows RFC-1034 (or documented exception, e.g., ISIN, ISO date).
- [ ] List endpoints implement `page_size` + `page_token` + `next_page_token`.
- [ ] Custom methods use `:verb` suffix.
- [ ] Versioned under `/api/v1/...`.
- [ ] All response fields in `snake_case`.
- [ ] Timestamps use `_at` suffix and RFC 3339 UTC.
- [ ] Errors return `application/problem+json` with `code` in SCREAMING_SNAKE_CASE.
- [ ] OpenAPI spec updated with `description` and `examples` per CLAUDE.md.

---

## 4. References

- [AIP-121 Resource-oriented design](https://google.aip.dev/121)
- [AIP-122 Resource names](https://google.aip.dev/122)
- [AIP-130 Standard methods](https://google.aip.dev/130)
- [AIP-131 Get](https://google.aip.dev/131) / [AIP-132 List](https://google.aip.dev/132) / [AIP-133 Create](https://google.aip.dev/133) / [AIP-134 Update](https://google.aip.dev/134) / [AIP-135 Delete](https://google.aip.dev/135) / [AIP-136 Custom methods](https://google.aip.dev/136)
- [AIP-140 Field names](https://google.aip.dev/140)
- [AIP-148 Standard fields](https://google.aip.dev/148)
- [AIP-158 Pagination](https://google.aip.dev/158)
- [AIP-185 Versioning](https://google.aip.dev/185)
- [AIP-190 Naming conventions](https://google.aip.dev/190)
- [AIP-193 Errors](https://google.aip.dev/193)
- [RFC 9457 ProblemDetails](https://www.rfc-editor.org/rfc/rfc9457.html)
- Mirror entry point: [docs.cloud.google.com/apis/design](https://docs.cloud.google.com/apis/design)
