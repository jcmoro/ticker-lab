# Future Features

Roadmap of features for Ticker Lab, ordered by priority.

Items marked ~~strikethrough~~ are already implemented.

---

## ~~Phase 6 — Migrate to Render + Neon~~ ✓
- ~~Fly.io trial expires ~2026-04-26~~
- ~~Move app to Render (free tier, Docker, auto-deploy from GitHub)~~
- ~~Move Postgres to Neon (free serverless Postgres, 0.5GB, scale-to-zero)~~
- ~~Update Makefile, runbook, CI workflows~~
- ~~Discard fly.toml after migration~~

## ~~Phase 7 — Historical Data & Charts~~ ✓
- ~~Backfill historical exchange rates from Frankfurter (`GET /v1/2020-01-01..2026-04-17`)~~
- ~~New endpoint: `GET /api/v1/exchange-rates/history?base=EUR&quote=USD&from=&to=`~~
- ~~Chart.js via `<script>` tag (no frontend framework) for time-series visualization~~
- ~~Dashboard: sparklines per currency, detail view with full chart~~
- ~~Configurable date ranges (1W, 1M, 3M, 1Y, ALL)~~

## ~~Phase 8 — Second Provider (CoinGecko / Crypto)~~ ✓
- ~~Validates hexagonal architecture scales to multiple bounded contexts~~
- ~~New bounded context: `crypto` with its own provider, repository, endpoints~~
- ~~CoinGecko free tier (no API key): BTC, ETH, SOL, top 20 by market cap~~
- ~~`GET /api/v1/crypto/latest` and `GET /api/v1/crypto/history`~~
- ~~Dashboard section for crypto prices~~

## ~~Phase 9 — Currency Converter~~ ✓
- ~~`GET /api/v1/convert?from=EUR&to=USD&amount=100`~~
- ~~Uses existing exchange rate data (no external call needed)~~
- ~~Widget in dashboard with input fields~~
- ~~Cross-rate calculation (e.g., GBP→JPY via EUR)~~

## ~~Phase 9b — Currency Converter in Go (microservice)~~ ✓
- ~~Reimplement the converter as a standalone Go microservice~~
- ~~Reads from the same Neon Postgres (or consumes the Ticker Lab REST API)~~
- ~~Validates polyglot architecture: same DB, different runtimes~~
- ~~Go chi or stdlib router, minimal dependencies~~
- ~~Separate Docker container, own Render service~~
- ~~Interesting experiment: compare Node vs Go for the same use case~~

## Phase 10 — Hotel Catalog (RateHawk)

Nueva línea de negocio: catálogo de hoteles vía la Content API de RateHawk (Emerging Travel Group). Microservicio Go (`ratehawk-go`) que ingesta y sirve datos de hoteles. Alcance inicial: sandbox (4 países).

### API de RateHawk (Content API)

| Endpoint | Método | Rate Limit | Función |
|----------|--------|------------|---------|
| `/api/content/v1/filter_values` | GET | 60 QPM | Países, idiomas, tipos de hotel disponibles |
| `/api/content/v1/hotel_ids_by_filter/` | POST | 60 QPM | Obtener HIDs filtrados por país/tipo/estrellas |
| `/api/content/v1/hotel_content_by_ids/` | POST | 1200 QPM | Contenido completo de hoteles por HIDs |

- Documentación: https://docs.emergingtravel.com/docs/content-api/retrieve-hotel-ids-by-filter/
- Auth: HTTP Basic (`KEY_ID:API_KEY`)
- Sandbox: `https://api-sandbox.worldota.net` (países: 59, 153, 189, 201)
- Producción: `https://api.worldota.net`
- Soporta `updated_since` para sync incremental

### Estructura del servicio (`apps/ratehawk-go/`)

Sigue el patrón exacto de `crypto-go`: Go stdlib + pgx, CLI + HTTP, Docker multi-stage.

```
apps/ratehawk-go/
  go.mod              # module ratehawk-go, go 1.25, pgx/v5
  go.sum
  main.go             # CLI (full-sync, incremental-sync, sync-country) + HTTP server (:8100)
  models.go           # Hotel, HotelSummary, HotelFilter, Config, ProblemDetails
  ratehawk.go         # Cliente HTTP: Basic Auth, FetchHotelIDs, FetchHotelContent
  repository.go       # Migrate, SaveHotels, FindHotels, FindHotelByHID, LogSync
  handlers.go         # GET /health, /api/v1/hotels, /api/v1/hotels/{hid}
  main_test.go        # Tests: health, CORS, repository, handlers

docker/ratehawk-go/
  Dockerfile          # Multi-stage: builder → dev → prod (copia de crypto-go)
```

### Schema de base de datos

```sql
-- Tabla principal de hoteles
CREATE TABLE IF NOT EXISTS ratehawk_hotels (
    hid              INTEGER PRIMARY KEY,          -- ID numérico de RateHawk
    name             VARCHAR(500) NOT NULL,
    kind             VARCHAR(50) NOT NULL,          -- hotel, apartment, hostel, etc.
    star_rating      SMALLINT NOT NULL DEFAULT 0,   -- 0-5
    country_code     VARCHAR(10) NOT NULL,
    region_name      VARCHAR(200) NOT NULL DEFAULT '',
    address          TEXT NOT NULL DEFAULT '',
    latitude         NUMERIC(10, 7),
    longitude        NUMERIC(10, 7),
    check_in_time    VARCHAR(10) DEFAULT '',
    check_out_time   VARCHAR(10) DEFAULT '',
    images           JSONB DEFAULT '[]',            -- [{url, category_slug}]
    amenity_groups   JSONB DEFAULT '[]',            -- [{group_name, amenities}]
    room_groups      JSONB DEFAULT '[]',            -- [{rg_ext, name, room_amenities}]
    description      JSONB DEFAULT '[]',            -- [{title, paragraphs}]
    metapolicy       JSONB DEFAULT '{}',            -- políticas del hotel
    serp_filters     JSONB DEFAULT '[]',            -- filtros de búsqueda
    payment_methods  JSONB DEFAULT '[]',
    facts            JSONB DEFAULT '{}',            -- year_built, rooms_number, etc.
    updated_at       TIMESTAMP NOT NULL DEFAULT NOW(),
    created_at       TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_rh_hotels_country ON ratehawk_hotels(country_code);
CREATE INDEX IF NOT EXISTS idx_rh_hotels_kind ON ratehawk_hotels(kind);
CREATE INDEX IF NOT EXISTS idx_rh_hotels_star ON ratehawk_hotels(star_rating);

-- Log de sincronizaciones para observabilidad
CREATE TABLE IF NOT EXISTS ratehawk_sync_log (
    id               SERIAL PRIMARY KEY,
    sync_type        VARCHAR(20) NOT NULL,          -- full / incremental
    hotels_fetched   INTEGER NOT NULL DEFAULT 0,
    hotels_saved     INTEGER NOT NULL DEFAULT 0,
    started_at       TIMESTAMP NOT NULL,
    completed_at     TIMESTAMP,
    error            TEXT,
    status           VARCHAR(20) NOT NULL DEFAULT 'running'
);
```

Decisiones de diseño:
- `hid` como PK (identificador único de RateHawk en todas las llamadas API)
- Campos escalares buscables como columnas con índices
- Datos ricos/anidados como JSONB (images, amenities, rooms, metapolicy) — son datos de visualización
- `json.RawMessage` en Go para capturar JSONB sin definir structs profundos

### Cliente API (`ratehawk.go`)

```go
type RateHawkClient struct {
    baseURL    string
    keyID      string
    apiKey     string
    httpClient *http.Client  // 30s timeout
}

func NewRateHawkClient(cfg Config) *RateHawkClient
func (c *RateHawkClient) FetchHotelIDs(country int) ([]int, error)
func (c *RateHawkClient) FetchHotelIDsUpdatedSince(country int, since string) ([]int, error)
func (c *RateHawkClient) FetchHotelContent(hids []int, language string) ([]Hotel, error)
```

- Auth: `req.SetBasicAuth(c.keyID, c.apiKey)`
- Rate limiting con `time.Sleep` (patrón crypto-go): 1s entre IDs requests, 50ms entre content requests
- Error wrapping: `fmt.Errorf("ratehawk ...: %w", err)`

### Subcomandos CLI

| Comando | Función |
|---------|---------|
| `./ratehawk-go` | Servidor HTTP en `:8100` |
| `./ratehawk-go full-sync` | Sync completo: IDs por país → content en batches de 100 → upsert |
| `./ratehawk-go incremental-sync` | Usa `updated_since` desde último sync exitoso |
| `./ratehawk-go sync-country 59` | Sync de un solo país |

### Endpoints HTTP

| Endpoint | Descripción |
|----------|-------------|
| `GET /health` | Health check (engine: "go-ratehawk") |
| `GET /api/v1/hotels?country=&kind=&star_rating=&search=&limit=&offset=` | Listado paginado |
| `GET /api/v1/hotels/{hid}` | Detalle completo |

### Dashboard SSR

- `/hotels` — Grid de tarjetas con filtros (país, tipo, estrellas, búsqueda) + paginación
- `/hotels/:hid` — Ficha detalle: galería de imágenes, amenities, habitaciones, check-in/out, descripción
- Integración en `dashboard.ts` con `fetchWithRetry` (mismo patrón que crypto)
- Link "Hotels" en navegación principal

### Infraestructura a modificar

| Fichero | Cambio |
|---------|--------|
| `docker-compose.yml` | Servicio `ratehawk-go` (puerto 8100, target: dev) |
| `.env.example` | `RATEHAWK_KEY_ID`, `RATEHAWK_API_KEY`, `RATEHAWK_BASE_URL`, `RATEHAWK_GO_URL` |
| `Makefile` | ratehawk-go en `go-vet`/`go-test` + targets `job-hotels-sync`, `job-hotels-incremental` |
| `apps/api/.../dashboard.ts` | Rutas `/hotels` y `/hotels/:hid` |
| `apps/api/.../main.eta` | Link "Hotels" en nav |
| GitHub Actions CI | Vet + test ratehawk-go |
| GitHub Actions ingest | Cron diario `incremental-sync` |

### Fases de implementación

**Fase 1 — Esqueleto Go + DB + HTTP:**
Ficheros Go (models, repository, handlers, main), Dockerfile, docker-compose, Makefile. Validar: `make go-ci`.

**Fase 2 — Cliente API + Sync:**
`ratehawk.go`, subcomandos CLI, sync log. Validar: test manual contra sandbox (requiere credenciales).

**Fase 3 — Dashboard SSR:**
Rutas en dashboard.ts, templates hotels.eta y hotel-detail.eta, navegación. Validar: `make ci`.

**Fase 4 — CI/CD + Docs:**
GitHub Actions, changelog, architecture docs. Validar: CI green.

## Phase 11 — Macro Indicators (FRED / ECB)

Nuevo bounded context: indicadores macroeconómicos de EEUU (FRED) y Eurozona (ECB). Microservicio Go (`macro-go`) que ingesta series económicas y las sirve en el dashboard. Dos fuentes de datos, modelo genérico.

### Fuentes de datos

| API | Auth | Rate Limit | Formato | Datos |
|-----|------|------------|---------|-------|
| FRED (`api.stlouisfed.org/fred/`) | API key (query param, gratuita) | 120 QPM | JSON | EEUU: CPI, desempleo, tipos, GDP |
| ECB (`data-api.ecb.europa.eu/service/`) | Sin auth (público) | No documentado | CSV (`csvdata`) | Eurozona: HICP, tipos BCE, ESTR |

- Documentación detallada: `docs/macro-indicators-integration.md`

### Series Tier 1 (MVP)

| Indicador | Fuente | Serie ID | Frecuencia |
|-----------|--------|----------|------------|
| CPI (inflación EEUU) | FRED | `CPIAUCSL` | Mensual |
| HICP (inflación eurozona) | ECB | `ICP` | Mensual |
| Desempleo EEUU | FRED | `UNRATE` | Mensual |
| Federal Funds Rate | FRED | `FEDFUNDS` | Mensual |
| 10Y Treasury Yield | FRED | `DGS10` | Diaria |
| Tipo BCE (MRR) | ECB | `FM` | Puntual |
| GDP EEUU (real) | FRED | `GDPC1` | Trimestral |

### Estructura del servicio (`apps/macro-go/`)

```
apps/macro-go/
  go.mod              # module macro-go, go 1.25, pgx/v5
  go.sum
  main.go             # CLI (ingest, ingest-ecb, backfill) + HTTP server (:8110)
  models.go           # Indicator, HistoryPoint, SeriesMeta, Config, ProblemDetails
  fred.go             # Cliente FRED: API key auth, FetchSeries, FetchSeriesObservations
  ecb.go              # Cliente ECB: sin auth, CSV parser, FetchDataflow
  repository.go       # Migrate, SaveObservations, FindIndicators, FindHistory
  handlers.go         # GET /health, /api/v1/macro/indicators, /api/v1/macro/{source}/{id}/history
  main_test.go        # Tests: health, CORS, repository, handlers

docker/macro-go/
  Dockerfile          # Multi-stage: builder → dev → prod (copia de crypto-go)
```

### Schema de base de datos

```sql
-- Metadata de series rastreadas
CREATE TABLE IF NOT EXISTS macro_series (
    source       VARCHAR(10) NOT NULL,      -- 'fred' o 'ecb'
    series_id    VARCHAR(50) NOT NULL,      -- 'CPIAUCSL', 'ICP', etc.
    name         VARCHAR(200) NOT NULL,
    frequency    VARCHAR(10) NOT NULL,      -- 'daily', 'monthly', 'quarterly'
    unit         VARCHAR(50) DEFAULT '',    -- 'percent', 'index', 'billions_usd'
    category     VARCHAR(50) NOT NULL,      -- 'inflation', 'employment', 'interest_rates', 'gdp'
    last_synced  TIMESTAMP,
    PRIMARY KEY (source, series_id)
);

-- Observaciones (datos históricos)
CREATE TABLE IF NOT EXISTS macro_observations (
    id           SERIAL PRIMARY KEY,
    source       VARCHAR(10) NOT NULL,
    series_id    VARCHAR(50) NOT NULL,
    value        NUMERIC(20, 6) NOT NULL,
    date         DATE NOT NULL,
    created_at   TIMESTAMP DEFAULT NOW(),
    UNIQUE(source, series_id, date)
);

CREATE INDEX IF NOT EXISTS idx_macro_obs_series ON macro_observations(source, series_id, date);
```

Decisiones:
- Modelo genérico: todos los indicadores comparten estructura (source + series_id + date + value)
- Tabla `macro_series` define qué series se rastrean, con categoría para agrupar en el dashboard
- `UNIQUE(source, series_id, date)` previene duplicados + permite upsert

### Clientes API

**FRED (`fred.go`):**
```go
type FREDClient struct {
    baseURL    string  // https://api.stlouisfed.org/fred
    apiKey     string
    httpClient *http.Client
}
func (c *FREDClient) FetchObservations(seriesID string, start string) ([]Observation, error)
```
- Auth: `&api_key=XXX` como query param
- Response JSON: `{ observations: [{ date, value }] }`
- Valor `"."` = dato no disponible (filtrar)
- Sleep 500ms entre requests (120 QPM)

**ECB (`ecb.go`):**
```go
type ECBClient struct {
    baseURL    string  // https://data-api.ecb.europa.eu/service
    httpClient *http.Client
}
func (c *ECBClient) FetchDataflow(dataflow, key string, start string) ([]Observation, error)
```
- Sin auth
- Formato CSV (`?format=csvdata`) — mucho más fácil de parsear que SDMX-JSON
- Parseo con `encoding/csv` de stdlib
- Sleep 1s entre requests (precaución)

### Subcomandos CLI

| Comando | Función |
|---------|---------|
| `./macro-go` | Servidor HTTP en `:8110` |
| `./macro-go ingest` | Sync incremental FRED (desde last_synced) |
| `./macro-go ingest-ecb` | Sync incremental ECB |
| `./macro-go backfill` | Backfill histórico completo (FRED + ECB) |

### Endpoints HTTP

| Endpoint | Descripción |
|----------|-------------|
| `GET /health` | Health check (engine: "go-macro") |
| `GET /api/v1/macro/indicators` | Todos los indicadores con último valor, agrupados por categoría |
| `GET /api/v1/macro/indicators?category=inflation` | Filtrar por categoría |
| `GET /api/v1/macro/{source}/{series_id}/history?days=365` | Histórico de un indicador |

### Dashboard SSR

- `/macro` — Indicadores agrupados por categoría (Inflation, Employment, Interest Rates, GDP). Cada tarjeta: nombre, último valor, fecha, variación, sparkline
- `/macro/:source/:id` — Detalle con chart histórico (Chart.js), selector de período
- Integración en `dashboard.ts` con `fetchWithRetry` (patrón crypto)

### Infraestructura a modificar

| Fichero | Cambio |
|---------|--------|
| `docker-compose.yml` | Servicio `macro-go` (puerto 8110, target: dev) |
| `.env.example` | `FRED_API_KEY`, `MACRO_GO_URL` |
| `Makefile` | macro-go en `go-vet`/`go-test` + targets `job-macro-ingest`, `job-macro-backfill` |
| `apps/api/.../dashboard.ts` | Rutas `/macro` y `/macro/:source/:id` |
| `apps/api/.../main.eta` | Link "Macro" en nav |
| GitHub Actions CI | Vet + test macro-go |
| GitHub Actions ingest | Cron diario `ingest` + `ingest-ecb` |

### Fases de implementación

**Fase 1 — Esqueleto Go + DB + HTTP + FRED:**
models, repository, handlers, main, `fred.go` con cliente FRED. Solo series FRED Tier 1. Dockerfile, docker-compose, Makefile. Validar: `make go-ci`.

**Fase 2 — Cliente ECB + Sync completo:**
`ecb.go` con parseo CSV. Subcomandos `ingest-ecb` y `backfill`. Añadir series ECB (HICP, tipo BCE). Validar: test manual.

**Fase 3 — Dashboard SSR:**
Rutas en dashboard.ts, templates macro.eta y macro-detail.eta (con Chart.js), navegación. Validar: `make ci`.

**Fase 4 — Tier 2 + CI/CD + Docs:**
Ampliar con series Tier 2, GitHub Actions, changelog, architecture docs. Validar: CI green.

## Phase 12 — Alerts
- `POST /api/v1/alerts` — create threshold alert (e.g., "EUR/USD > 1.10")
- `GET /api/v1/alerts` — list user's alerts
- Cron evaluates alerts after each ingestion
- Notification via webhook, email, or Telegram

## Phase 13 — Investment Funds & Pension Plans
- Original project goal (see `docs/future-providers.md` for API research)
- CNMV/Inverco scraping or Morningstar if viable
- NAV (Net Asset Value) tracking, daily updates
- `GET /api/v1/funds/latest` and `GET /api/v1/funds/:isin/history`

---

## Test Coverage Improvements
- Integration tests for backfill/ingest jobs (end-to-end with test DB)
- HTTP mock tests for FrankfurterClient.fetchDateRange and CoinGeckoClient.FetchHistory/FetchPrices
- Test the full job flow: fetch → save → verify in DB
- ~~CI pipeline for Go tests (`go test ./...` in GitHub Actions)~~ ✓

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
