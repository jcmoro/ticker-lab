# Future Features

Roadmap of features for Ticker Lab, ordered by priority.

Items marked ~~strikethrough~~ are already implemented.

---

## Phase 6 — Migrate to Render + Neon
- Fly.io trial expires ~2026-04-26
- Move app to Render (free tier, Docker, auto-deploy from GitHub)
- Move Postgres to Neon (free serverless Postgres, 0.5GB, scale-to-zero)
- Update Makefile, runbook, CI workflows
- Discard fly.toml after migration

## Phase 7 — Historical Data & Charts
- Backfill historical exchange rates from Frankfurter (`GET /v1/2020-01-01..2026-04-17`)
- New endpoint: `GET /api/v1/exchange-rates/history?base=EUR&quote=USD&from=&to=`
- Chart.js via `<script>` tag (no frontend framework) for time-series visualization
- Dashboard: sparklines per currency, detail view with full chart
- Configurable date ranges (1W, 1M, 3M, 1Y, ALL)

## Phase 8 — Second Provider (CoinGecko / Crypto)
- Validates hexagonal architecture scales to multiple bounded contexts
- New bounded context: `crypto` with its own provider, repository, endpoints
- CoinGecko free tier (no API key): BTC, ETH, SOL, top 20 by market cap
- `GET /api/v1/crypto/latest` and `GET /api/v1/crypto/history`
- Dashboard section for crypto prices

## Phase 9 — Currency Converter
- `GET /api/v1/convert?from=EUR&to=USD&amount=100`
- Uses existing exchange rate data (no external call needed)
- Widget in dashboard with input fields
- Cross-rate calculation (e.g., GBP→JPY via EUR)

## Phase 9b — Currency Converter in Go (microservice)
- Reimplement the converter as a standalone Go microservice
- Reads from the same Neon Postgres (or consumes the Ticker Lab REST API)
- Validates polyglot architecture: same DB, different runtimes
- Go chi or stdlib router, minimal dependencies
- Separate Docker container, own Render service
- Interesting experiment: compare Node vs Go for the same use case

## Phase 10 — Macro Indicators (FRED / ECB)
- Third bounded context: `macro`
- FRED API (free key): US interest rates, CPI, unemployment, GDP
- ECB Statistical Data Warehouse: Eurozone rates, inflation
- `GET /api/v1/macro/indicators` and `GET /api/v1/macro/indicators/:id/history`
- Dashboard section for macro data

## Phase 11 — Alerts
- `POST /api/v1/alerts` — create threshold alert (e.g., "EUR/USD > 1.10")
- `GET /api/v1/alerts` — list user's alerts
- Cron evaluates alerts after each ingestion
- Notification via webhook, email, or Telegram

## Phase 12 — Investment Funds & Pension Plans
- Original project goal (see `docs/future-providers.md` for API research)
- CNMV/Inverco scraping or Morningstar if viable
- NAV (Net Asset Value) tracking, daily updates
- `GET /api/v1/funds/latest` and `GET /api/v1/funds/:isin/history`

---

## Additional Features (unordered)

### Comparisons
- Compare multiple currencies/assets on the same chart
- Percentage change view (normalized to a base date)

### Currency Heatmap
- Grid showing all EUR cross-rates color-coded by daily change (green/red)

### Exportar CSV/JSON
- `GET /api/v1/exchange-rates/export?format=csv&from=&to=`
- Download historical data for offline analysis

### Widget Embeddable
- `GET /api/v1/widget/ticker` — returns an HTML/JS snippet
- Embed exchange rate ticker in external websites via `<iframe>` or `<script>`

### Telegram / Discord Bot
- `/rate EUR USD` — query exchange rates from chat
- `/alert EUR USD > 1.10` — create alert from chat
- Uses the existing API

### Portfolio Tracker
- Define positions (bought X EUR of USD at rate Y on date Z)
- Track P&L evolution over time
- Requires authentication

### PWA (Progressive Web App)
- Service worker for offline dashboard access
- Push notifications for alerts
- Installable on mobile home screen

### ~~API Documentation (ReDoc)~~ ✓
- ~~Serve interactive API docs at `/api/docs`~~

### Real-time Updates
- Move from daily cron to more frequent polling (hourly, every 15min)
- WebSocket or SSE for live dashboard updates
- Consider rate limits of each provider

### Multi-language (i18n)
- Spanish and English initially
- Number formatting locale-aware (1.234,56 vs 1,234.56)

### React SPA Frontend
- Replace Fastify SSR with React + Vite when interactivity demands it
- TanStack Query for data fetching
- Tailwind + shadcn/ui for components
- The API contract (OpenAPI) remains the same — frontend is fully decoupled

### Authentication
- Optional login for personalized dashboards/alerts
- OAuth2 or magic link (no passwords)

### Observability (advanced)
- Prometheus-format `/metrics` endpoint
- Grafana dashboard for monitoring ingestion jobs
- Alertmanager for ingestion failures
