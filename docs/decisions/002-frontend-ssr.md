# ADR-002: Fastify SSR Instead of React SPA

**Status:** Accepted
**Date:** 2026-04-19

## Context

The project needs a web interface to display ticker data. The developer is backend-focused and wants to minimize frontend complexity while keeping the door open for a richer frontend later.

## Decision

Use Fastify's built-in SSR capabilities with Eta templates instead of a separate React/Vite application.

The view layer lives in `apps/api/src/views/` and renders HTML server-side. It contains **no business logic** — it only formats and displays data received from the API layer.

## Rationale

- **Simplicity:** no frontend build toolchain (no Vite, no webpack, no bundler)
- **Single runtime:** one Node.js process serves both API and HTML
- **Daily data:** exchange rates update once per day — no need for client-side reactivity
- **Decoupled:** templates consume the same API contract as any future SPA would. Swapping to React later requires no backend changes.
- **Backend-friendly:** Eta templates are close to plain HTML, natural for a backend developer

## Consequences

- No client-side JavaScript framework (vanilla JS only if needed for interactivity)
- Charts require a server-side rendering approach or lightweight client-side library (e.g., Chart.js via `<script>` tag)
- If interactive features grow significantly, this decision should be revisited (see `docs/future-features.md` → "React SPA Frontend")
- The `apps/web/` workspace is not needed — may be created later if a SPA is introduced
