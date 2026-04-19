# Future Features

Features to consider after the MVP ticker dashboard is working.

## Historical Data & Charts
- Store daily snapshots in PostgreSQL
- Recharts or Chart.js for time-series visualization
- Configurable date ranges (1W, 1M, 3M, 1Y, ALL)

## Comparisons
- Compare multiple currencies/assets on the same chart
- Percentage change view (normalized to a base date)

## Alerts
- User-defined thresholds (e.g., "notify when EUR/USD > 1.10")
- Notification channel: email, webhook, or browser push

## Real-time Updates
- Move from daily cron to more frequent polling (hourly, every 15min)
- WebSocket or SSE for live dashboard updates
- Consider rate limits of each provider

## Multi-language (i18n)
- Spanish and English initially
- Number formatting locale-aware (1.234,56 vs 1,234.56)

## React SPA Frontend
- Replace Fastify SSR with React + Vite when interactivity demands it
- TanStack Query for data fetching
- Tailwind + shadcn/ui for components
- The API contract (OpenAPI) remains the same — frontend is fully decoupled

## Authentication
- Optional login for personalized dashboards/alerts
- OAuth2 or magic link (no passwords)

## Observability
- Structured logging with Pino (already in place)
- Prometheus metrics endpoint `/metrics`
- Grafana dashboard for monitoring ingestion jobs

## Mobile
- PWA (Progressive Web App) for mobile access
- Responsive design from the start (SSR templates already support this)
