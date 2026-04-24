# Macro Indicators Integration — FRED & ECB

## Resumen

Integración con dos fuentes de datos macroeconómicos:

1. **FRED** (Federal Reserve Economic Data) — datos macro de EEUU: GDP, CPI, desempleo, tipos de interés, empleo
2. **ECB Data Portal** — datos macro de la Eurozona: HICP (inflación), tipos de interés del BCE, GDP eurozona

Ambas APIs son gratuitas, bien documentadas y con datos de alta calidad institucional.

---

## FRED API

### Información general

| Campo | Valor |
|-------|-------|
| Base URL | `https://api.stlouisfed.org/fred/` |
| Auth | API key como query param (`&api_key=XXX`) |
| Registro | https://fred.stlouisfed.org/docs/api/api_key.html (gratuito) |
| Rate limit | 120 requests/minuto |
| Formatos | JSON, XML, Excel, CSV (param `file_type`) |
| Documentación | https://fred.stlouisfed.org/docs/api/fred/ |

### Endpoints principales

#### `GET /fred/series`
Metadata de una serie (nombre, frecuencia, unidades, fechas disponibles).

```
https://api.stlouisfed.org/fred/series?series_id=CPIAUCSL&api_key=XXX&file_type=json
```

Response:
```json
{
  "seriess": [{
    "id": "CPIAUCSL",
    "title": "Consumer Price Index for All Urban Consumers: All Items in U.S. City Average",
    "frequency": "Monthly",
    "units": "Index 1982-1984=100",
    "observation_start": "1947-01-01",
    "observation_end": "2026-03-01",
    "seasonal_adjustment": "Seasonally Adjusted"
  }]
}
```

#### `GET /fred/series/observations`
Datos históricos de una serie. Este es el endpoint principal para ingesta.

```
https://api.stlouisfed.org/fred/series/observations?series_id=CPIAUCSL&api_key=XXX&file_type=json&observation_start=2020-01-01
```

Parámetros:

| Param | Tipo | Descripción |
|-------|------|-------------|
| `series_id` | String | **Requerido**. ID de la serie (ej: `CPIAUCSL`) |
| `api_key` | String | **Requerido**. API key registrada |
| `file_type` | String | `json`, `xml`, `xls`, `txt` (default: xml) |
| `observation_start` | String | Fecha inicio (YYYY-MM-DD) |
| `observation_end` | String | Fecha fin (YYYY-MM-DD) |
| `frequency` | String | Agregación: `d`, `w`, `bw`, `m`, `q`, `sa`, `a` |
| `units` | String | Transformación: `lin` (nivel), `chg` (cambio), `ch1` (cambio desde hace 1 año), `pch` (% cambio), `pc1` (% cambio interanual), `pca` (% cambio anualizado), `cch` (cambio compuesto), `cca` (cambio compuesto anualizado), `log` (log natural) |
| `sort_order` | String | `asc` o `desc` |
| `limit` | Integer | Max observaciones (default: 100000) |
| `offset` | Integer | Paginación |

Response:
```json
{
  "observation_start": "2020-01-01",
  "observation_end": "2026-04-23",
  "count": 75,
  "observations": [
    { "date": "2020-01-01", "value": "257.971" },
    { "date": "2020-02-01", "value": "258.678" }
  ]
}
```

#### `GET /fred/series/search`
Buscar series por texto.

```
https://api.stlouisfed.org/fred/series/search?search_text=inflation&api_key=XXX&file_type=json
```

### Series relevantes (EEUU)

| Serie ID | Nombre | Frecuencia | Categoría |
|----------|--------|------------|-----------|
| `CPIAUCSL` | Consumer Price Index (All Urban Consumers) | Mensual | Inflación |
| `PCEPI` | PCE Price Index | Mensual | Inflación |
| `UNRATE` | Unemployment Rate | Mensual | Empleo |
| `U6RATE` | U-6 Unemployment Rate (broad) | Mensual | Empleo |
| `PAYEMS` | Total Nonfarm Payrolls | Mensual | Empleo |
| `GDP` | Gross Domestic Product (nominal) | Trimestral | PIB |
| `GDPC1` | Real GDP (inflation-adjusted) | Trimestral | PIB |
| `FEDFUNDS` | Federal Funds Effective Rate | Mensual | Tipos de interés |
| `DFF` | Federal Funds Rate (daily) | Diaria | Tipos de interés |
| `DGS10` | 10-Year Treasury Constant Maturity Rate | Diaria | Tipos de interés |
| `DGS2` | 2-Year Treasury Constant Maturity Rate | Diaria | Tipos de interés |
| `T10Y2Y` | 10Y-2Y Treasury Spread | Diaria | Tipos de interés |
| `CSUSHPINSA` | Case-Shiller Home Price Index | Mensual | Vivienda |
| `PPIACO` | Producer Price Index (All Commodities) | Mensual | Precios |
| `M2SL` | M2 Money Supply | Mensual | Oferta monetaria |

### Notas de uso
- Los valores vienen como strings (hay que parsear a float)
- El valor `"."` indica dato no disponible
- El param `units=pch` devuelve directamente el % cambio mensual (útil para CPI → inflación)
- Con `units=pc1` se obtiene el cambio interanual directamente
- 120 QPM es muy generoso — no necesita rate limiting agresivo

---

## ECB Data Portal API

### Información general

| Campo | Valor |
|-------|-------|
| Base URL | `https://data-api.ecb.europa.eu/service/` |
| Auth | **No requiere autenticación** (acceso público) |
| Rate limit | No documentado explícitamente (uso razonable) |
| Protocolo | Solo HTTPS |
| Formatos | JSON (`jsondata`), CSV (`csvdata`), SDMX-ML 2.1 |
| Estándar | SDMX 2.1 RESTful |
| Documentación | https://data.ecb.europa.eu/help/api/overview |

### Estructura de consultas

```
https://data-api.ecb.europa.eu/service/data/{DATAFLOW}/{KEY}?{PARAMS}
```

- **DATAFLOW**: ID del dataset (ej: `EXR`, `ICP`, `FM`)
- **KEY**: Dimensiones separadas por puntos (ej: `M.USD.EUR.SP00.A`)
- **PARAMS**: Filtros y formato

### Parámetros de consulta

| Param | Tipo | Descripción |
|-------|------|-------------|
| `startPeriod` | String | Fecha inicio (ISO 8601: `YYYY`, `YYYY-MM`, `YYYY-MM-DD`) |
| `endPeriod` | String | Fecha fin |
| `updatedAfter` | String | Solo datos actualizados desde (ISO 8601 URL-encoded) |
| `detail` | String | `full`, `dataonly`, `serieskeysonly`, `nodata` |
| `firstNObservations` | Integer | Limitar primeras N observaciones |
| `lastNObservations` | Integer | Limitar últimas N observaciones |
| `format` | String | `csvdata`, `jsondata`, `structurespecificdata`, `genericdata` |

### Sintaxis de keys

- Dimensiones separadas por `.` en el orden definido por el DSD del dataset
- Wildcard: omitir dimensión (ej: `M..EUR.SP00.A` = todas las monedas)
- OR: usar `+` (ej: `M.USD+GBP.EUR.SP00.A`)

### Dataflows relevantes (Eurozona)

| Dataflow ID | Nombre | Descripción |
|-------------|--------|-------------|
| `ICP` | Indices of Consumer Prices | HICP — inflación eurozona |
| `IRS` | Interest Rate Statistics | Tipos de interés del BCE |
| `EST` | Euro Short-Term Rate | ESTR (sustituto del EONIA) |
| `MIR` | MFI Interest Rate Statistics | Tipos de interés bancarios |
| `EXR` | Exchange Rates | Tipos de cambio (ya cubierto por Frankfurter) |
| `BSI` | Balance Sheet Items | Agregados monetarios (M1, M2, M3) |

GDP y desempleo de la eurozona:

| Dataflow ID | Nombre | Descripción |
|-------------|--------|-------------|
| `MNA` / `JDF_MNA_*` | National Accounts | PIB eurozona (múltiples variantes) |
| `IESS_PUB` | Labour Force Survey | Indicadores de empleo |

### Ejemplos de consultas

```bash
# HICP inflación eurozona (mensual, índice general, todas las áreas)
https://data-api.ecb.europa.eu/service/data/ICP/M.U2.N.000000.4.ANR?startPeriod=2020-01&format=jsondata

# Tipo de interés BCE - main refinancing operations
https://data-api.ecb.europa.eu/service/data/FM/B.U2.EUR.4F.KR.MRR_FR.LEV?format=jsondata

# Euro Short-Term Rate (ESTR)
https://data-api.ecb.europa.eu/service/data/EST/B.EU000A2X2A25.WT?startPeriod=2024-01&format=jsondata

# Tipo de cambio EUR/USD mensual
https://data-api.ecb.europa.eu/service/data/EXR/M.USD.EUR.SP00.A?startPeriod=2020-01&format=csvdata
```

### Notas de uso
- No requiere API key — completamente público
- El formato JSON (`jsondata`) devuelve SDMX-JSON, que es más verboso que un JSON convencional (estructura con dimensiones + observaciones indexadas)
- CSV (`csvdata`) es más fácil de parsear para ingesta
- Las keys de las dimensiones se descubren consultando el DSD del dataflow
- El endpoint de dataflows (`/service/dataflow`) lista todos los datasets disponibles

---

## Indicadores propuestos para Ticker Lab

### Tier 1 — Indicadores esenciales (MVP)

| Indicador | Fuente | Serie/Dataflow | Frecuencia |
|-----------|--------|----------------|------------|
| CPI (inflación EEUU) | FRED | `CPIAUCSL` (units=pc1) | Mensual |
| HICP (inflación eurozona) | ECB | `ICP` | Mensual |
| Desempleo EEUU | FRED | `UNRATE` | Mensual |
| Federal Funds Rate | FRED | `FEDFUNDS` | Mensual |
| 10Y Treasury Yield | FRED | `DGS10` | Diaria |
| Tipo BCE (MRR) | ECB | `FM` (MRR_FR) | Puntual (decisiones BCE) |
| GDP EEUU (real) | FRED | `GDPC1` | Trimestral |

### Tier 2 — Indicadores adicionales

| Indicador | Fuente | Serie/Dataflow | Frecuencia |
|-----------|--------|----------------|------------|
| 2Y Treasury Yield | FRED | `DGS2` | Diaria |
| 10Y-2Y Spread (inversión curva) | FRED | `T10Y2Y` | Diaria |
| PCE Price Index | FRED | `PCEPI` | Mensual |
| Nonfarm Payrolls | FRED | `PAYEMS` | Mensual |
| Euro Short-Term Rate (ESTR) | ECB | `EST` | Diaria |
| M2 Money Supply | FRED | `M2SL` | Mensual |
| Case-Shiller Home Prices | FRED | `CSUSHPINSA` | Mensual |

---

## Decisiones de diseño

### Lenguaje: Go
Consistente con converter-go y crypto-go. Mismos patrones: stdlib HTTP client, pgx, CLI + HTTP server. Sin dependencias adicionales.

### Modelo genérico
Todos los indicadores comparten la misma estructura (source + series_id + date + valor numérico). Una tabla de metadata define qué series se rastrean y a qué categoría pertenecen. Esto permite añadir series nuevas sin cambiar código — solo configuración.

### ECB en CSV, no JSON
El formato SDMX-JSON del ECB es muy verboso (estructura de dimensiones + observaciones indexadas). Usar `?format=csvdata` devuelve un CSV plano que se parsea trivialmente con `encoding/csv` de stdlib.

### Rate limiting con Sleep
Mismo patrón que crypto-go con CoinGecko: `time.Sleep` entre requests. FRED (120 QPM → 500ms), ECB (precaución → 1s).

### Sync incremental
- FRED: `observation_start` con la fecha del último dato almacenado
- ECB: `startPeriod` con la fecha del último dato almacenado

---

## Plan de implementación

### Estructura del servicio (`apps/macro-go/`)

```
apps/macro-go/
  go.mod              # module macro-go, go 1.25, pgx/v5
  go.sum
  main.go             # CLI (ingest, ingest-ecb, backfill) + HTTP server (:8110)
  models.go           # Observation, Indicator, HistoryPoint, SeriesMeta, Config
  fred.go             # Cliente FRED: API key auth, FetchObservations
  ecb.go              # Cliente ECB: sin auth, CSV parser, FetchDataflow
  repository.go       # Migrate, SaveObservations, FindIndicators, FindHistory
  handlers.go         # GET /health, /api/v1/macro/indicators, /api/v1/macro/{source}/{id}/history
  main_test.go        # Tests: health, CORS, repository, handlers

docker/macro-go/
  Dockerfile          # Multi-stage: builder → dev → prod (copia de crypto-go)
```

### Schema de base de datos

```sql
-- Metadata de series rastreadas (configuración)
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
CREATE INDEX IF NOT EXISTS idx_macro_obs_date ON macro_observations(date);
```

Decisiones:
- Tabla `macro_series` define qué se rastrea + categoría para agrupar en dashboard
- Tabla `macro_observations` almacena todos los datapoints (genérica)
- `UNIQUE(source, series_id, date)` previene duplicados + permite upsert
- No hay JSONB — los datos macro son escalares simples (fecha + valor)

### Tipos Go (`models.go`)

```go
// Serie registrada (metadata de configuración)
type SeriesMeta struct {
    Source     string `json:"source"`
    SeriesID   string `json:"series_id"`
    Name       string `json:"name"`
    Frequency  string `json:"frequency"`
    Unit       string `json:"unit"`
    Category   string `json:"category"`
}

// Observación cruda (del API o DB)
type Observation struct {
    Source   string  `json:"source"`
    SeriesID string  `json:"series_id"`
    Value    float64 `json:"value"`
    Date     string  `json:"date"`
}

// Indicador con último valor (para listado)
type Indicator struct {
    Source      string  `json:"source"`
    SeriesID    string  `json:"series_id"`
    Name        string  `json:"name"`
    Category    string  `json:"category"`
    Unit        string  `json:"unit"`
    LatestValue float64 `json:"latest_value"`
    LatestDate  string  `json:"latest_date"`
    PrevValue   float64 `json:"prev_value,omitempty"`
    Change      float64 `json:"change,omitempty"`
}

// Punto histórico (para charts)
type HistoryPoint struct {
    Date  string  `json:"date"`
    Value float64 `json:"value"`
}

// Configuración de series Tier 1
var fredSeries = []SeriesMeta{
    {Source: "fred", SeriesID: "CPIAUCSL", Name: "CPI (All Urban Consumers)", Frequency: "monthly", Unit: "index", Category: "inflation"},
    {Source: "fred", SeriesID: "UNRATE", Name: "Unemployment Rate", Frequency: "monthly", Unit: "percent", Category: "employment"},
    {Source: "fred", SeriesID: "FEDFUNDS", Name: "Federal Funds Rate", Frequency: "monthly", Unit: "percent", Category: "interest_rates"},
    {Source: "fred", SeriesID: "DGS10", Name: "10-Year Treasury Yield", Frequency: "daily", Unit: "percent", Category: "interest_rates"},
    {Source: "fred", SeriesID: "GDPC1", Name: "Real GDP", Frequency: "quarterly", Unit: "billions_usd", Category: "gdp"},
}

var ecbSeries = []SeriesMeta{
    {Source: "ecb", SeriesID: "ICP", Name: "HICP (Eurozone Inflation)", Frequency: "monthly", Unit: "percent", Category: "inflation"},
    {Source: "ecb", SeriesID: "FM_MRR", Name: "ECB Main Refinancing Rate", Frequency: "monthly", Unit: "percent", Category: "interest_rates"},
}

type HealthResponse struct { ... }   // patrón crypto-go
type ProblemDetails struct { ... }   // patrón crypto-go (RFC 7807)
```

### Cliente FRED (`fred.go`)

```go
type FREDClient struct {
    baseURL    string           // https://api.stlouisfed.org/fred
    apiKey     string
    httpClient *http.Client     // 30s timeout
}

func NewFREDClient(apiKey string) *FREDClient

// Obtiene observaciones de una serie desde una fecha
func (c *FREDClient) FetchObservations(seriesID string, startDate string) ([]Observation, error)
```

Flujo de `FetchObservations`:
1. GET `/fred/series/observations?series_id=CPIAUCSL&api_key=XXX&file_type=json&observation_start=2020-01-01`
2. Parsear response JSON: `{ observations: [{ date: "2020-01-01", value: "257.971" }] }`
3. Filtrar valores `"."` (dato no disponible)
4. Convertir `value` de string a float64
5. Retornar `[]Observation`

### Cliente ECB (`ecb.go`)

```go
type ECBClient struct {
    baseURL    string           // https://data-api.ecb.europa.eu/service
    httpClient *http.Client     // 30s timeout
}

func NewECBClient() *ECBClient

// Obtiene datos de un dataflow en formato CSV
func (c *ECBClient) FetchDataflow(dataflow, key string, startPeriod string) ([]Observation, error)
```

Flujo de `FetchDataflow`:
1. GET `/data/{dataflow}/{key}?startPeriod=2020-01&format=csvdata`
2. Parsear CSV con `encoding/csv` (stdlib)
3. Columnas relevantes: `TIME_PERIOD` (fecha) y `OBS_VALUE` (valor)
4. Convertir a `[]Observation`

Configuración de keys por serie ECB:

| Serie | Dataflow | Key |
|-------|----------|-----|
| HICP eurozona | `ICP` | `M.U2.N.000000.4.ANR` |
| Tipo BCE (MRR) | `FM` | `B.U2.EUR.4F.KR.MRR_FR.LEV` |

### Repository (`repository.go`)

```go
type Repository struct {
    pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository
func (r *Repository) Migrate(ctx context.Context) error
func (r *Repository) SeedSeries(ctx context.Context, series []SeriesMeta) error
func (r *Repository) SaveObservations(ctx context.Context, obs []Observation) error
func (r *Repository) UpdateLastSynced(ctx context.Context, source, seriesID string) error
func (r *Repository) FindIndicators(ctx context.Context, category string) ([]Indicator, error)
func (r *Repository) FindHistory(ctx context.Context, source, seriesID string, days int) ([]HistoryPoint, error)
func (r *Repository) GetLastSyncDate(ctx context.Context, source, seriesID string) (string, error)
```

`SaveObservations` — batch upsert (patrón crypto-go):
```sql
INSERT INTO macro_observations (source, series_id, value, date)
VALUES ($1, $2, $3, $4), ...
ON CONFLICT (source, series_id, date) DO UPDATE SET value = EXCLUDED.value
```

`FindIndicators` — último valor + valor anterior para calcular cambio:
```sql
SELECT s.source, s.series_id, s.name, s.category, s.unit,
       o.value AS latest_value, o.date::text AS latest_date,
       prev.value AS prev_value
FROM macro_series s
LEFT JOIN LATERAL (
    SELECT value, date FROM macro_observations
    WHERE source = s.source AND series_id = s.series_id
    ORDER BY date DESC LIMIT 1
) o ON true
LEFT JOIN LATERAL (
    SELECT value FROM macro_observations
    WHERE source = s.source AND series_id = s.series_id AND date < o.date
    ORDER BY date DESC LIMIT 1
) prev ON true
WHERE ($1 = '' OR s.category = $1)
ORDER BY s.category, s.name
```

### Handlers (`handlers.go`)

```go
func handleHealth(w http.ResponseWriter, _ *http.Request)
func handleIndicators(repo *Repository) http.HandlerFunc      // GET /api/v1/macro/indicators?category=
func handleHistory(repo *Repository) http.HandlerFunc          // GET /api/v1/macro/{source}/{id}/history?days=365
func writeJSON(w http.ResponseWriter, status int, data any)
```

Response de `/api/v1/macro/indicators`:
```json
{
  "count": 7,
  "indicators": [
    {
      "source": "fred",
      "series_id": "CPIAUCSL",
      "name": "CPI (All Urban Consumers)",
      "category": "inflation",
      "unit": "index",
      "latest_value": 319.084,
      "latest_date": "2026-03-01",
      "prev_value": 318.642,
      "change": 0.14
    }
  ]
}
```

Response de `/api/v1/macro/fred/CPIAUCSL/history?days=365`:
```json
{
  "source": "fred",
  "series_id": "CPIAUCSL",
  "name": "CPI (All Urban Consumers)",
  "days": 365,
  "count": 12,
  "points": [
    { "date": "2025-04-01", "value": 314.069 },
    { "date": "2025-05-01", "value": 314.927 }
  ]
}
```

### Subcomandos CLI (`main.go`)

```go
func main() {
    // Env: DATABASE_URL (required), FRED_API_KEY (required for ingest), PORT (default 8110)
    // DB pool + migrate + seed series metadata

    switch os.Args[1] {
    case "ingest":      runIngestFRED(repo, fredClient)     // sync incremental FRED
    case "ingest-ecb":  runIngestECB(repo, ecbClient)       // sync incremental ECB
    case "backfill":    runBackfill(repo, fredClient, ecbClient) // histórico completo
    default:            // HTTP server
    }
}
```

`runIngestFRED`:
1. Para cada serie en `fredSeries`:
   - Obtener `last_synced` de `macro_series`
   - `FetchObservations(seriesID, lastDate)`
   - `SaveObservations(observations)`
   - `UpdateLastSynced(source, seriesID)`
   - Sleep 500ms
2. Log total de observaciones guardadas

`runIngestECB`: mismo flujo con `ecbSeries` y sleep 1s.

`runBackfill`: ignora `last_synced`, usa `observation_start=2000-01-01` para FRED y `startPeriod=2000-01` para ECB.

### Docker + Infraestructura

**Dockerfile** (`docker/macro-go/Dockerfile`): idéntico a crypto-go, puerto 8110.

**docker-compose.yml**:
```yaml
macro-go:
  build:
    context: .
    dockerfile: docker/macro-go/Dockerfile
    target: dev
  ports:
    - "8110:8110"
  environment:
    - DATABASE_URL=postgresql://ticker:ticker@db:5432/ticker_lab
    - PORT=8110
    - FRED_API_KEY=${FRED_API_KEY:-}
  depends_on:
    db:
      condition: service_healthy
```

**.env.example**:
```
FRED_API_KEY=
MACRO_GO_URL=http://localhost:8110
```

**Makefile**:
```makefile
# Añadir a go-vet y go-test:
docker compose run --rm macro-go go vet ./...
docker compose run --rm macro-go go test ./...

job-macro-ingest:            ## Ingest FRED + ECB indicators
    docker compose run --rm macro-go ./macro-go ingest
    docker compose run --rm macro-go ./macro-go ingest-ecb

job-macro-backfill:          ## Backfill all macro indicators history
    docker compose run --rm macro-go ./macro-go backfill
```

### Dashboard SSR

**`/macro`** — Indicadores agrupados por categoría:
- Secciones: Inflation, Employment, Interest Rates, GDP
- Cada tarjeta: nombre, último valor con fecha, badge de variación (verde/rojo), sparkline
- Color accent: `#10b981` (emerald) para distinguir de exchange rates (blue), crypto (purple), hotels (amber)

**`/macro/:source/:id`** — Detalle:
- Chart.js con histórico completo
- Selector de período (1M, 3M, 6M, 1Y, 5Y, ALL)
- Metadata: fuente, frecuencia, unidad, última actualización

**Integración** en `dashboard.ts`:
```typescript
const macroBaseUrl = process.env.MACRO_GO_URL ?? 'http://localhost:8110';
// GET /macro → fetchWithRetry → render macro.eta
// GET /macro/:source/:id → fetchWithRetry → render macro-detail.eta
```

### Testing (`main_test.go`)

Mismo patrón que crypto-go:
- `getTestPool(t)` — helper con skip si no hay DATABASE_URL
- `TestHealthEndpoint` — puro httptest, sin DB
- `TestCorsMiddleware` — puro httptest
- `TestSaveAndFindIndicators` — integration: seed series + save observations + find indicators
- `TestFindHistory` — integration: save observations + query history
- `TestIndicatorsEndpoint` — handler con DB
- `TestHistoryEndpoint` — handler con mux para path params
- Cleanup: `DELETE FROM macro_observations WHERE series_id LIKE 'test-%'`

### Fases de implementación

**Fase 1 — Esqueleto Go + DB + HTTP + FRED:**
`go.mod`, `models.go`, `repository.go`, `handlers.go`, `main.go`, `fred.go`. Solo series FRED Tier 1. Dockerfile, docker-compose, Makefile, tests. Validar: `make go-ci`.

**Fase 2 — Cliente ECB + Sync completo:**
`ecb.go` con parseo CSV. Subcomandos `ingest-ecb` y `backfill`. Series ECB (HICP, tipo BCE). Validar: test manual con API key FRED.

**Fase 3 — Dashboard SSR:**
Rutas en `dashboard.ts`, templates `macro.eta` y `macro-detail.eta` (con Chart.js), link en navegación. Validar: `make ci`.

**Fase 4 — Tier 2 + CI/CD + Docs:**
Ampliar con series Tier 2 (DGS2, T10Y2Y, PCEPI, PAYEMS, EST, M2SL, CSUSHPINSA). GitHub Actions (CI + cron diario). Actualizar `docs/changelog.md` y `docs/architecture.md`. Validar: CI green.

---

## Referencias

- [FRED API Docs](https://fred.stlouisfed.org/docs/api/fred/)
- [FRED API Key Registration](https://fred.stlouisfed.org/docs/api/api_key.html)
- [ECB Data Portal API Overview](https://data.ecb.europa.eu/help/api/overview)
- [ECB Data API - Data Queries](https://data.ecb.europa.eu/help/api/data)
- [ECB Dataflows](https://data-api.ecb.europa.eu/service/dataflow)
- Plan de implementación: ver `docs/future-features.md` (Phase 11)
