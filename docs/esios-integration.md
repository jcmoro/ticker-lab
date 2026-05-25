# ESIOS Integration — Spanish Electricity Data

## Resumen

Integración con la **API e·sios** de Red Eléctrica de España (REE) para incorporar datos del sistema eléctrico español al dashboard de Ticker Lab.

**Cobertura funcional:**

- Precio de la energía (PVPC peaje 2.0TD por hora)
- Demanda eléctrica peninsular y por sistemas (Canarias, Baleares, Ceuta, Melilla)
- Mix de generación por tecnología (eólica, solar PV, solar térmica, hidráulica, nuclear, ciclo combinado, carbón, cogeneración)
- Precio mercado spot OMIE
- Factor de emisiones de CO2

**Características clave:**

- API gratuita, requiere token personal (no auto-servicio: vía email a `consultasios@ree.es`)
- Datos horarios para series de precios y generación
- Datos en tiempo real (hasta 5/10/15 minutos de granularidad para algunas series)
- Datos públicos con atribución a Red Eléctrica de España (REE)
- Una sola fuente — encaja en el patrón genérico ya usado en `macro-go`

---

## ESIOS API

### Información general

| Campo | Valor |
|-------|-------|
| Base URL producción | `https://api.esios.ree.es` |
| Base URL pre-producción | `https://apip.esios.ree.es` |
| Auth | Header `x-api-key: <token>` (legacy: `Authorization: Token token="<token>"`, todavía compatible) |
| Solicitud de token | Email a `consultasios@ree.es` (asunto: "Personal token request") |
| Formato | JSON (versionado vía `Accept: application/json; application/vnd.esios-api-v1+json`) |
| Documentación | https://api.esios.ree.es/doc/ |
| Rate limit | No documentado oficialmente; comunidad recomienda esperar entre 1s y 4s entre requests |
| Términos de uso | Datos públicos. Atribución obligatoria a "Red Eléctrica de España (REE) — e·sios" |

### Headers requeridos

Todos los endpoints autenticados requieren los tres headers:

```
x-api-key: <TU_TOKEN>
Accept: application/json; application/vnd.esios-api-v1+json
Content-Type: application/json
```

> **Nota de migración**: REE migró el header `Authorization: Token token="..."` al header `x-api-key: ...`. Ambos siguen siendo compatibles, pero el nuevo formato (`x-api-key`) es el oficial y debe usarse en código nuevo.

### Endpoints principales

#### `GET /indicators` — Catálogo de indicadores

Lista todos los indicadores disponibles (>1.500). Útil una sola vez para descubrir IDs.

```bash
curl "https://api.esios.ree.es/indicators" \
  -H "x-api-key: $ESIOS_API_KEY" \
  -H "Accept: application/json; application/vnd.esios-api-v1+json" \
  -H "Content-Type: application/json"
```

Response (truncada):
```json
{
  "indicators": [
    { "id": 1001, "name": "Término de facturación de energía activa del PVPC 2.0TD", "short_name": "PVPC T. 2.0TD" },
    { "id": 600,  "name": "Demanda real",                                            "short_name": "Demanda real" }
  ],
  "meta": { "size": 1547 }
}
```

Existen variantes con filtros por taxonomía (`?taxonomy_terms=`) y búsqueda por texto (`?text=`); validar al implementar si se necesitan.

#### `GET /indicators/{id}` — Datos de un indicador

Endpoint principal de ingesta. Devuelve metadata + array de valores temporales.

```bash
curl "https://api.esios.ree.es/indicators/1001?start_date=2026-05-01T00:00&end_date=2026-05-09T23:59&time_trunc=hour" \
  -H "x-api-key: $ESIOS_API_KEY" \
  -H "Accept: application/json; application/vnd.esios-api-v1+json" \
  -H "Content-Type: application/json"
```

Parámetros:

| Param | Tipo | Descripción |
|-------|------|-------------|
| `start_date` | ISO8601 | Inicio rango (`YYYY-MM-DDTHH:MM` o con offset `+02:00`/`Z`) |
| `end_date` | ISO8601 | Fin rango |
| `time_trunc` | String | `five_minutes`, `ten_minutes`, `fifteen_minutes`, `hour`, `day`, `month`, `year` |
| `time_agg` | String | `sum`, `average` (cómo agregar a la granularidad de `time_trunc`) |
| `geo_ids[]` | Integer[] | Filtrar por sistema eléctrico (8741=Península, 8742=Canarias, 8743=Baleares, 8744=Ceuta, 8745=Melilla) |
| `geo_agg` | String | `sum`, `average` (agregación entre geografías) |
| `geo_trunc` | String | `country`, `electric_system`, `autonomous_community`, `province`, `electric_subsystem` |
| `locale` | String | `es` o `en` (afecta a `name`, `short_name`, `geo_name`) |

Response (indicador 1001, truncada):
```json
{
  "indicator": {
    "id": 1001,
    "name": "Término de facturación de energía activa del PVPC 2.0TD",
    "short_name": "PVPC T. 2.0TD",
    "magnitud": [{ "id": 23, "name": "Precio €/MWh" }],
    "tiempo":   [{ "id": 4,  "name": "Hora" }],
    "geos": [
      { "geo_id": 8741, "geo_name": "España"   },
      { "geo_id": 8742, "geo_name": "Canarias" }
    ],
    "values_updated_at": "2026-05-09T20:46:21.000+02:00",
    "values": [
      {
        "value": 87.43,
        "datetime":     "2026-05-09T00:00:00.000+02:00",
        "datetime_utc": "2026-05-08T22:00:00.000Z",
        "geo_id": 8741,
        "geo_name": "España"
      },
      {
        "value": 88.11,
        "datetime":     "2026-05-09T01:00:00.000+02:00",
        "datetime_utc": "2026-05-08T23:00:00.000Z",
        "geo_id": 8741,
        "geo_name": "España"
      }
    ]
  }
}
```

> **Importante**: el array `values` contiene **una entrada por (timestamp, geo_id)**. Para un indicador con 5 sistemas eléctricos y granularidad horaria, un día son 24 × 5 = 120 entradas. Hay que agrupar/filtrar por `geo_id` al persistir.

### Granularidad temporal y volumen

| `time_trunc` | Puntos por día | Puntos por año |
|--------------|----------------|----------------|
| `hour` | 24 | ~8.760 |
| `day` | 1 | 365 |
| `month` | — | 12 |

- Para series de precios y generación se usará `time_trunc=hour`.
- **Sin paginación nativa**: la API no documenta `offset/limit` en este endpoint. Para evitar respuestas grandes y posibles timeouts, se recomienda fragmentar por meses en backfill (1 mes ≈ 720 puntos × 5 geos = 3.600 entradas — manejable). Validar al implementar el límite real si se intentan rangos > 1 año.

### Códigos de error

| Status | Significado | Acción |
|--------|-------------|--------|
| `200 OK` | Datos válidos | Procesar |
| `401 Unauthorized` | Token ausente o inválido | Abortar ingest, alertar |
| `403 Forbidden` | Token revocado o sin permisos | Marcar serie inactiva, log |
| `404 Not Found` | Indicador no existe | Eliminar de configuración |
| `429 Too Many Requests` | Rate limit excedido | Backoff exponencial; subir `time.Sleep` |
| `5xx` | Error servidor REE | Reintento backoff (3 intentos, 2s/4s/8s) |

> **Validar al implementar**: la API no documenta formato estándar de error. Loguear cuerpo crudo cuando `status >= 400` y ajustar parser si REE devuelve JSON estructurado.

### Términos de uso y atribución

- Datos públicos — uso permitido para fines analíticos y de visualización.
- **Atribución obligatoria**: cada vista que muestre datos derivados debe incluir "Fuente: Red Eléctrica de España — e·sios (https://www.esios.ree.es)".
- Sin distinción explícita comercial/no-comercial en la API pública, pero el token es nominal: respetar el uso razonable.

---

## Indicadores propuestos para Ticker Lab

### Tier 1 — Esenciales (MVP)

Verificado contra el catálogo ESIOS en vivo el 2026-05-25. **Los `geo_id` son por indicador**, no intercambiables: los indicadores de demanda/generación usan la taxonomía de **sistema eléctrico** (8741=Península, 8742-8745=islas), mientras que los **precios de mercado** (mercado SPOT diario) usan la taxonomía de **país** (1=Portugal, 2=Francia, 3=España).

| ID | Nombre oficial REE | Categoría | Frecuencia | Unidad | Geo | Notas |
|-----|--------|-----------|------------|--------|-----|-------|
| `1001` | Término de facturación de energía activa del PVPC 2.0TD | `pricing` | Horaria | €/MWh | 8741 (España peninsular) | Tarifa regulada |
| `1293` | Demanda real | `demand` | Horaria (agregada de 5 min) | MW | 8741 | Demanda peninsular real |
| `600`  | Precio mercado SPOT diario | `pricing` | Horaria (agregada de 15 min) | €/MWh | **3** (país España) | Precio mayorista MIBEL; sin desagregación por sistema eléctrico |
| `10211` | Precio horario final (suma de componentes) | `pricing` | Horaria | €/MWh | 8741 | Precio final unificado (no es OMIE puro: incluye ajustes) |
| `10355` | CO2 Asociado Generación T.Real | `emissions` | Horaria (agregada de 5 min) | tCO2/MWh | 8741 | Factor de emisiones en tiempo real |

### Tier 2 — Adicionales (post-MVP)

| ID | Nombre | Categoría | Frecuencia | Unidad |
|-----|--------|-----------|------------|--------|
| `1739` | Precio energía excedentaria autoconsumo | `pricing` | Horaria | €/MWh |
| `1900` | Desglose peaje por defecto 2.0TD MAG | `pricing` | Horaria | €/MWh |
| `2108` | Mecanismo de ajuste a la producción (gas-cap) | `pricing` | Horaria | €/MWh |
| `10073` | Generación programada eólica (PBF) | `generation` | Horaria | MW |
| `10074` | Generación programada solar fotovoltaica (PBF) | `generation` | Horaria | MW |
| `10071` | Generación programada nuclear (PBF) | `generation` | Horaria | MW |
| `10072` | Generación programada ciclo combinado (PBF) | `generation` | Horaria | MW |
| `10070` | Generación programada hidráulica (PBF) | `generation` | Horaria | MW |
| `1159` | Generación eólica medida | `generation` | Horaria | MW |

> **Nota**: los IDs 10070-10074 corresponden al patrón "PBF por tecnología" observado en la comunidad. **Validar al implementar** consultando `GET /indicators?text=programada` o el catálogo completo, ya que algunos IDs varían según el tipo de programa (PBF, P48, en tiempo real). Mantener `models.go` como única fuente para añadir/quitar series sin tocar código.

### Geo IDs útiles

ESIOS no usa un único espacio de `geo_id`. Cada indicador declara qué taxonomía publica:

**Sistema eléctrico (usado por demanda y generación):**

| `geo_id` | Sistema eléctrico |
|----------|-------------------|
| `8741` | España / Península |
| `8742` | Canarias |
| `8743` | Baleares |
| `8744` | Ceuta |
| `8745` | Melilla |

**País (usado por precios de mercado mayorista — SPOT diario, intradiario):**

| `geo_id` | País |
|----------|------|
| `1` | Portugal |
| `2` | Francia |
| `3` | España |

> **Verificar al añadir un indicador nuevo**: probar `GET /indicators/{id}?start_date=...&end_date=...` *sin* `geo_ids[]` y mirar los `geo_id` que aparecen en `values[]`. No asumir que 8741 funciona para todo.

---

## Decisiones de diseño

### Lenguaje: Go

Consistente con `crypto-go`, `converter-go`, `macro-go`. Mismos patrones: stdlib HTTP, pgx/v5, CLI + HTTP server. Sin dependencias adicionales.

### Modelo genérico (alineado con `macro-go`)

Tablas `esios_series` (metadata) + `esios_observations` (timeseries con `(indicator_id, geo_id, datetime_utc)` único). Esto permite:

- Añadir/quitar indicadores sólo con cambios en `models.go` (configuración declarativa).
- Reutilizar el patrón de upsert de `macro-go`.
- Soportar múltiples sistemas eléctricos por indicador sin duplicar tablas.

### Granularidad horaria, almacenada como TIMESTAMPTZ

A diferencia de `macro-go` (datos diarios/mensuales con `DATE`), ESIOS publica datos horarios. El campo de tiempo será `TIMESTAMPTZ` (UTC), almacenando `datetime_utc` para evitar ambigüedad con horario de verano peninsular.

### Sólo Península en MVP

Para el dashboard, filtrar `geo_id = 8741` reduce volumen 5× y simplifica la primera versión. Los demás sistemas son fáciles de habilitar más adelante (la columna `geo_id` ya existe).

### Rate limiting con `time.Sleep`

Mismo patrón que `macro-go`. ESIOS no documenta rate limit oficial; la comunidad usa entre 1s y 4s. Empezamos con **`time.Sleep(2 * time.Second)`** entre indicadores y validamos en producción.

### Sync incremental

- Almacenar `last_synced_at` (timestamp de la última observación obtenida por serie).
- En cada `ingest`, enviar `start_date = last_synced_at` y `end_date = now()`.
- Upsert por clave `(indicator_id, geo_id, datetime_utc)` para tolerancia a reentregas.
- Backfill ignora `last_synced_at` y trocea por meses (`start_date=2020-01-01`, iterar mes a mes).

### Token: secret obligatorio en runtime

Sin token no se puede ingestar. El servicio:
- En CLI (`ingest`/`backfill`): falla rápido si `ESIOS_API_KEY` no está definido.
- En modo HTTP server: arranca igualmente (sólo necesita DB para servir lecturas).

---

## Plan de implementación

### Estructura del servicio (`apps/esios-go/`)

```
apps/esios-go/
  go.mod              # module esios-go, go 1.25, pgx/v5
  go.sum
  main.go             # CLI (ingest, backfill) + HTTP server (:8120)
  models.go           # Observation, Indicator, HistoryPoint, SeriesMeta
  esios.go            # Cliente ESIOS: x-api-key auth, FetchIndicator, parseo values
  repository.go       # Migrate, SeedSeries, SaveObservations, FindIndicators, FindHistory
  handlers.go         # GET /health, /api/v1/electricity/indicators, /api/v1/electricity/indicators/{id}/geos/{geo}/observations
  main_test.go        # Tests: health, CORS, repository, handlers
  testdata/
    indicator_1001.json   # Fixture para tests offline del parser

docker/esios-go/
  Dockerfile          # Multi-stage builder → dev → prod (copia de macro-go)
```

### Schema de base de datos

```sql
CREATE TABLE IF NOT EXISTS esios_series (
    indicator_id   INTEGER NOT NULL,
    geo_id         INTEGER NOT NULL,
    name           VARCHAR(200) NOT NULL,
    short_name     VARCHAR(50)  NOT NULL,
    category       VARCHAR(50)  NOT NULL,
    unit           VARCHAR(20)  NOT NULL,
    geo_name       VARCHAR(50)  NOT NULL,
    frequency      VARCHAR(10)  NOT NULL DEFAULT 'hourly',
    last_synced_at TIMESTAMPTZ,
    PRIMARY KEY (indicator_id, geo_id)
);

CREATE TABLE IF NOT EXISTS esios_observations (
    id             BIGSERIAL    PRIMARY KEY,
    indicator_id   INTEGER      NOT NULL,
    geo_id         INTEGER      NOT NULL,
    value          NUMERIC(20,6) NOT NULL,
    datetime_utc   TIMESTAMPTZ  NOT NULL,
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    UNIQUE (indicator_id, geo_id, datetime_utc)
);

CREATE INDEX IF NOT EXISTS idx_esios_obs_series ON esios_observations(indicator_id, geo_id, datetime_utc DESC);
CREATE INDEX IF NOT EXISTS idx_esios_obs_dt     ON esios_observations(datetime_utc DESC);
```

### Tipos Go (`models.go`)

```go
type SeriesMeta struct {
    IndicatorID int    `json:"indicator_id"`
    GeoID       int    `json:"geo_id"`
    Name        string `json:"name"`
    ShortName   string `json:"short_name"`
    Category    string `json:"category"`
    Unit        string `json:"unit"`
    GeoName     string `json:"geo_name"`
    Frequency   string `json:"frequency"`
}

type Observation struct {
    IndicatorID int       `json:"indicator_id"`
    GeoID       int       `json:"geo_id"`
    Value       float64   `json:"value"`
    DatetimeUTC time.Time `json:"datetime_utc"`
}

type Indicator struct {
    IndicatorID int       `json:"indicator_id"`
    GeoID       int       `json:"geo_id"`
    Name        string    `json:"name"`
    ShortName   string    `json:"short_name"`
    Category    string    `json:"category"`
    Unit        string    `json:"unit"`
    GeoName     string    `json:"geo_name"`
    LatestValue float64   `json:"latest_value"`
    LatestAt    time.Time `json:"latest_at"`
    PrevValue   float64   `json:"prev_value,omitempty"`
    Change      float64   `json:"change,omitempty"`
}

type HistoryPoint struct {
    DatetimeUTC time.Time `json:"datetime_utc"`
    Value       float64   `json:"value"`
}

var tier1Series = []SeriesMeta{
    {IndicatorID: 1001,  GeoID: 8741, Name: "PVPC 2.0TD",                     ShortName: "PVPC",      Category: "pricing",    Unit: "EUR/MWh",  GeoName: "España", Frequency: "hourly"},
    {IndicatorID: 600,   GeoID: 8741, Name: "Demanda real",                    ShortName: "Demanda",   Category: "demand",     Unit: "MW",       GeoName: "España", Frequency: "hourly"},
    {IndicatorID: 10211, GeoID: 8741, Name: "OMIE — precio horario final",     ShortName: "OMIE",      Category: "pricing",    Unit: "EUR/MWh",  GeoName: "España", Frequency: "hourly"},
    {IndicatorID: 1293,  GeoID: 8741, Name: "Generación programada PBF total", ShortName: "Gen total", Category: "generation", Unit: "MW",       GeoName: "España", Frequency: "hourly"},
    {IndicatorID: 10355, GeoID: 8741, Name: "Factor emisiones CO2",            ShortName: "CO2",       Category: "emissions",  Unit: "tCO2/MWh", GeoName: "España", Frequency: "hourly"},
}
```

### Cliente ESIOS (`esios.go`)

```go
type ESIOSClient struct {
    baseURL    string
    apiKey     string
    httpClient *http.Client
}

func NewESIOSClient(apiKey string) *ESIOSClient {
    return &ESIOSClient{
        baseURL:    "https://api.esios.ree.es",
        apiKey:     apiKey,
        httpClient: &http.Client{Timeout: 30 * time.Second},
    }
}

func (c *ESIOSClient) FetchIndicator(
    ctx context.Context,
    indicatorID int, geoID int,
    startDate, endDate time.Time, timeTrunc string,
) ([]Observation, error)
```

Flujo:
1. URL: `GET {baseURL}/indicators/{indicatorID}?start_date={ISO}&end_date={ISO}&time_trunc={tt}&geo_ids[]={geoID}`.
2. Headers: `x-api-key`, `Accept: application/json; application/vnd.esios-api-v1+json`, `Content-Type: application/json`.
3. Si `status != 200`, error con cuerpo crudo.
4. Decode JSON al struct `indicatorResponse{ Indicator{ Values []{ Value, DatetimeUTC, GeoID } } }`.
5. Filtrar por `GeoID == geoID` (defensa en profundidad — REE a veces ignora el filtro).
6. Mapear a `[]Observation`.

### Repository (`repository.go`)

```go
type Repository struct{ pool *pgxpool.Pool }

func (r *Repository) Migrate(ctx context.Context) error
func (r *Repository) SeedSeries(ctx context.Context, series []SeriesMeta) error
func (r *Repository) SaveObservations(ctx context.Context, obs []Observation) error
func (r *Repository) UpdateLastSynced(ctx context.Context, indicatorID, geoID int, ts time.Time) error
func (r *Repository) GetLastSynced(ctx context.Context, indicatorID, geoID int) (time.Time, error)
func (r *Repository) FindIndicators(ctx context.Context, category string) ([]Indicator, error)
func (r *Repository) FindHistory(ctx context.Context, indicatorID, geoID int, days int) ([]HistoryPoint, error)
```

`SaveObservations` — batch upsert:

```sql
INSERT INTO esios_observations (indicator_id, geo_id, value, datetime_utc)
VALUES ($1, $2, $3, $4), ($5, $6, $7, $8), ...
ON CONFLICT (indicator_id, geo_id, datetime_utc)
DO UPDATE SET value = EXCLUDED.value
```

`FindIndicators` — último valor + valor anterior con `LATERAL`:

```sql
SELECT s.indicator_id, s.geo_id, s.name, s.short_name, s.category, s.unit, s.geo_name,
       o.value AS latest_value, o.datetime_utc AS latest_at,
       prev.value AS prev_value
FROM esios_series s
LEFT JOIN LATERAL (
    SELECT value, datetime_utc FROM esios_observations
    WHERE indicator_id = s.indicator_id AND geo_id = s.geo_id
    ORDER BY datetime_utc DESC LIMIT 1
) o ON true
LEFT JOIN LATERAL (
    SELECT value FROM esios_observations
    WHERE indicator_id = s.indicator_id AND geo_id = s.geo_id AND datetime_utc < o.datetime_utc
    ORDER BY datetime_utc DESC LIMIT 1
) prev ON true
WHERE ($1 = '' OR s.category = $1)
ORDER BY s.category, s.name;
```

### Handlers HTTP (`handlers.go`)

```
GET /health
GET /api/v1/electricity/indicators?category=pricing&page_size=&page_token=
GET /api/v1/electricity/indicators/{indicator_id}/geos/{geo_id}/observations?start_date=&end_date=&page_size=&page_token=
```

URL shape follows the Google AIP-122/132/158 conventions adopted in [`api-design-standards.md`](./api-design-standards.md): plural resource collections, no verb-shaped path segments, mandatory pagination on every list endpoint.

Response `/api/v1/electricity/indicators`:
```json
{
  "indicators": [
    {
      "indicator_id": 1001, "geo_id": 8741,
      "name": "PVPC 2.0TD", "short_name": "PVPC",
      "category": "pricing", "unit": "EUR/MWh", "geo_name": "España",
      "latest_value": 87.43, "latest_at": "2026-05-09T22:00:00Z",
      "prev_value": 88.11, "change": -0.77
    }
  ],
  "next_page_token": "",
  "total_size": 5
}
```

`next_page_token` is empty when the collection has been fully traversed (AIP-158). `total_size` is optional and may be an estimate.

### Subcomandos CLI

```go
case "ingest":   runIngest(repo, esiosClient)        // últimos N días
case "backfill": runBackfill(repo, esiosClient)      // 2020-01-01 → hoy, mes a mes
```

### Docker e infraestructura

**Puerto:** 8120 (siguiente libre tras macro-go=8110).

**docker-compose.yml**:
```yaml
esios-go:
  build:
    context: .
    dockerfile: docker/esios-go/Dockerfile
    target: dev
  ports: ["8120:8120"]
  environment:
    - DATABASE_URL=postgresql://ticker:ticker@db:5432/ticker_lab
    - PORT=8120
    - ESIOS_API_KEY=${ESIOS_API_KEY:-}
  depends_on:
    db: { condition: service_healthy }
```

**GitHub Actions** — cron diario `30 5 * * *` (05:30 UTC, tras publicación PVPC del día siguiente ~20:15 CET).

### Dashboard SSR

**`/electricity`** — Indicadores agrupados por categoría:
- Secciones: Precios, Demanda, Generación, Emisiones
- Cards con último valor formateado, badge variación, sparkline 24h
- Color accent: `#f59e0b` (amber) para diferenciar de exchange (azul), crypto (morado), macro (emerald)
- Footer atribución obligatoria a REE

**`/electricity/:indicator_id/:geo_id`** — Detalle:
- Chart.js horario, selector 24h/7d/30d/90d/1Y
- Para PVPC: marcar visualmente periodos punta/llano/valle

### Testing

Mismo patrón que `macro-go`:
- `TestParseIndicatorResponse` con fixture `testdata/indicator_1001.json`
- `TestSaveAndFindIndicators` integration con cálculo de `change`
- `TestFindHistory` con observaciones horarias
- IDs negativos para fixtures (`indicator_id < 0`) para cleanup fácil

### Fases de implementación

1. **Esqueleto + cliente + DB** — `go.mod`, modelos, repository, handlers, cliente, Docker, tests.
2. **Backfill + GitHub Actions** — chunking mensual, cron diario.
3. **Dashboard SSR** — rutas `/electricity` y `/electricity/:id/:geo` con Chart.js.
4. **Tier 2 + Documentación** — series adicionales, OpenAPI, changelog, architecture.

---

## Referencias

- [API e·sios — Documentación oficial](https://api.esios.ree.es/doc/)
- [Información de la API — REE](https://www.esios.ree.es/es/pagina/api)
- [Migración header `Authorization` → `x-api-key`](https://github.com/home-assistant/core/issues/84071)
- [aiopvpc — referencia de IDs PVPC](https://github.com/azogue/aiopvpc)
- [SanPen/ESIOS — librería Python con catálogo](https://github.com/SanPen/ESIOS)
- Patrones del repo: `apps/macro-go/`, `apps/crypto-go/`
