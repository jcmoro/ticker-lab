# CNMV Fichero Público — Integración fondos de inversión españoles

## Resumen

Integración con la **CNMV** (Comisión Nacional del Mercado de Valores) para ingesta de valores liquidativos (NAV) de fondos de inversión españoles a través de los **Ficheros Públicos de Información Individual de IIC**.

A diferencia de Frankfurter (FX) o FRED/ECB (macro) que ofrecen APIs JSON, la CNMV publica ficheros mensuales en formato ZIP que contienen XML + XSD. Cada fichero mensual incluye **valores liquidativos diarios** del mes, partícipes y patrimonio para todos los fondos registrados en España.

**Ámbito CNMV**:
- Fondos de inversión (FI), Fondos hedge (FHF)
- SICAV, SICAV hedge (SHF)
- Compartimentos y clases de cada IIC

**Fuera de scope** (importante):
- **Planes de pensiones** son regulados por **DGSFP** (no CNMV). DGSFP no expone equivalente al `FONDMENS`: sólo informes estadístico-contables trimestrales (DECs). Documentado como gap conocido en `future-providers.md`.

---

## CNMV Fichero Público — Información general

| Campo | Valor |
|-------|-------|
| Página de descarga | `https://www.cnmv.es/portal/Publicaciones/Descarga-Informacion-Individual.aspx` |
| Patrón URL de descarga | `https://www.cnmv.es/webservices/verdocumento/ver?e={TOKEN}` |
| Auth | **No requiere** (acceso público) |
| Rate limit | No documentado (uso razonable) |
| Periodicidad | **Mensual** (un ZIP por mes) |
| Formato | ZIP → XML + XSD + PDF explicativo |
| Latencia de publicación | ~10–15 días tras fin de mes (fichero 202511 publicado 12-dic-2025) |
| Marco regulatorio | Circular 4/2008 CNMV |
| Términos de uso | Difusión pública gratuita. Atribución a CNMV recomendada |
| Documentación | https://www.cnmv.es/DocPortal/Estadisticas/Descarga-Informacion-Individual/Periodicidad-Descripcion.pdf |

### Particularidad crítica: URLs tokenizadas

A diferencia de FRED o ECB, **las URLs de descarga NO son deterministas**. Cada enlace mensual tiene un token opaco (`?e=gNnBCYLm1OSoE41Jxc7q6KZYksC08FGXDZqhBuQ1yuq...`) que se renderiza en el HTML de la página de descarga. Implicaciones:

- No se puede construir una URL del tipo `…/FONDMENS_202511.zip`.
- El cliente debe **scrapear el HTML** de la página `Descarga-Informacion-Individual.aspx`, extraer los enlaces `<a id="…lnkZip" href="…verdocumento/ver?e=TOKEN">`, mapearlos al mes correspondiente y descargar.
- La extracción se hace con regex sobre el HTML (no se necesita parser DOM completo). Patrón:
  ```html
  <a id="ctl00_ContentPrincipal_grdDescargas_ctlNN_lnkZip"
     href="https://www.cnmv.es/webservices/verdocumento/ver?e=TOKEN"
     title="Noviembre" target="_blank">
  ```
- El response al token devuelve `Content-Type: application/zip` y `Content-Disposition: inline; filename=DOC.ZIP`.

### Ficheros disponibles (catálogo CNMV)

Según la [documentación oficial de periodicidad](https://www.cnmv.es/DocPortal/Estadisticas/Descarga-Informacion-Individual/Periodicidad-Descripcion.pdf):

| Fichero | Periodicidad | Tipo IIC | Contenido |
|---------|--------------|----------|-----------|
| `FONDREGISTRO` | Mensual | Fondos de inversión | Datos identificativos del fondo, compartimentos y clases |
| `FONDMENS` | Mensual | Fondos de inversión | Valores liquidativos, partícipes y patrimonio **diarios** del mes |
| `FONDTRIM` | Trimestral | Fondos de inversión | Comisiones, gastos y rentabilidades |
| `FONDPATRIMDISVAR` | Trimestral | Fondos de inversión | Distribución del patrimonio y variación |
| `FONDCART` | Trimestral | Fondos de inversión | Cartera de contado (valor a valor) |
| `FONDDERI` | Trimestral | Fondos de inversión | Cartera de derivados |
| `SOCREGISTRO` | Trimestral | SICAV | Datos identificativos de la SICAV |
| `SOCTRIM` | Trimestral | SICAV | Acciones, accionistas, patrimonio, VL, rentabilidad |
| `SOCPATRIMDISVAR` | Trimestral | SICAV | Distribución y variación del patrimonio |
| `SOCCART` | Trimestral | SICAV | Cartera de contado |
| `SOCDERI` | Trimestral | SICAV | Cartera de derivados |

**Decisión MVP**: usar exclusivamente el ZIP mensual de fondos (`FONDMENS` + `FONDREGISTRO`). Trimestrales se evalúan en fase posterior.

---

## Estructura del fichero mensual

### Contenido del ZIP mensual (snapshot 202511 real)

```
FONDREGISTRO_202511.xml          2.187.138 bytes
FONDREGISTRO.xsd                     8.213 bytes
FONDREGISTRO_Documento Explicativo.pdf
FONDMENS_202511.xml             15.306.574 bytes
FONDMENS.xsd                        11.487 bytes
FONDMENS_Documento Explicativo.pdf
```

ZIP comprimido: ~2,2 MB. XML descomprimidos: ~17 MB.

### `FONDREGISTRO` — Maestro de fondos

Estructura (extracto del XSD oficial):

```
FondRegistro
├─ FechaDatos              (YYYYMM, ej. "202511")
└─ Entidad [1..N]
   ├─ Tipo                 (FI | FHF | SICAV | SHF)
   ├─ NumeroRegistro       (long, registro oficial CNMV)
   ├─ Denominacion         (string ≤150)
   ├─ ETF                  (SI | NO)
   ├─ Compartimento [1..N]
   │  ├─ NumeroCompartimento     (0 si no aplica)
   │  ├─ DenominacionCompartimento
   │  └─ Clase [1..N]
   │     ├─ NumeroClase         (0 si no aplica)
   │     ├─ DenominacionClase
   │     └─ ISIN                (string[12], ISO 6166)
   ├─ Gestora
   │  ├─ TipoGestora       (SGIIC | SGC | SAV | …)
   │  ├─ NumeroRegistroGestora
   │  ├─ DenominacionGestora
   │  └─ GrupoGestora
   │     ├─ NumeroGrupoGestora
   │     └─ DenominacionGrupoGestora
   └─ Depositario
      ├─ NumeroRegistroDepositario
      ├─ DenominacionDepositario
      └─ GrupoDepositario { NumeroGrupoDepositario, DenominacionGrupoDepositario }
```

**Sample real** (FONMARCH, FI):

```xml
<Entidad>
  <Tipo>FI</Tipo>
  <NumeroRegistro>9</NumeroRegistro>
  <Denominacion>FONMARCH, FI</Denominacion>
  <ETF>NO</ETF>
  <Compartimento>
    <NumeroCompartimento>0</NumeroCompartimento>
    <DenominacionCompartimento>COMPARTIMENTO 0</DenominacionCompartimento>
    <Clase>
      <NumeroClase>1</NumeroClase>
      <DenominacionClase>CLASE A</DenominacionClase>
      <ISIN>ES0138841038</ISIN>
    </Clase>
    <Clase>
      <NumeroClase>2</NumeroClase>
      <DenominacionClase>CLASE C</DenominacionClase>
      <ISIN>ES0138841004</ISIN>
    </Clase>
  </Compartimento>
  <Gestora>
    <TipoGestora>SGIIC</TipoGestora>
    <NumeroRegistroGestora>190</NumeroRegistroGestora>
    <DenominacionGestora>MARCH ASSET MANAGEMENT, S.G.I.I.C., S.A.U.</DenominacionGestora>
    <GrupoGestora>
      <NumeroGrupoGestora>13040</NumeroGrupoGestora>
      <DenominacionGrupoGestora>BANCA MARCH</DenominacionGrupoGestora>
    </GrupoGestora>
  </Gestora>
  <Depositario>
    <NumeroRegistroDepositario>211</NumeroRegistroDepositario>
    <DenominacionDepositario>BANCO INVERSIS, S.A.</DenominacionDepositario>
    <GrupoDepositario>
      <NumeroGrupoDepositario>13040</NumeroGrupoDepositario>
      <DenominacionGrupoDepositario>BANCA MARCH</DenominacionGrupoDepositario>
    </GrupoDepositario>
  </Depositario>
</Entidad>
```

### `FONDMENS` — Datos diarios del mes

Contiene VL/partícipes/patrimonio por **día del mes** (1..31). Días no hábiles llevan valor `0`.

```
FondMens
├─ FechaDatos              (YYYYMM)
└─ Entidad [1..N]
   ├─ Tipo, NumeroRegistro
   └─ Compartimento [1..N]
      ├─ NumeroCompartimento
      └─ Clase [1..N]
         ├─ NumeroClase, ISIN
         ├─ VLDiario { VL_Dia1, ..., VL_Dia31 }            (decimal, 4 fractionDigits)
         ├─ ParticipesDiario { Participes_Dia1, ..., Participes_Dia31 }
         └─ PatrimonioDiario { Patrimonio_Dia1, ..., Patrimonio_Dia31 }
```

**Sample real** (FONMARCH CLASE A, ES0138841038, noviembre 2025):

```xml
<VLDiario>
  <VL_Dia1>30.5744</VL_Dia1>
  <VL_Dia2>30.5751</VL_Dia2>
  <VL_Dia3>30.5484</VL_Dia3>
  ...
  <VL_Dia30>30.5611</VL_Dia30>
  <VL_Dia31>0</VL_Dia31>
</VLDiario>
```

**Notas críticas**:
- `VL_DiaN = 0` indica día no hábil o sin dato — debe filtrarse.
- Noviembre solo tiene 30 días → `VL_Dia31` aparece como `0`. Misma lógica para febrero.
- La clave única real es `(Tipo, NumeroRegistro, NumeroCompartimento, NumeroClase)`. **El ISIN es la clave funcional** que se expone públicamente.

### Volúmenes reales (snapshot 202511)

| Métrica | Valor |
|---------|-------|
| Entidades (`FONDREGISTRO`) | 1.441 (FI + SICAV + FHF + SHF) |
| ISINs únicos | 3.112 |
| Tamaño ZIP descarga | 2,2 MB |
| Tamaño FONDMENS XML | 15,3 MB |
| Tamaño FONDREGISTRO XML | 2,2 MB |
| Observaciones generadas por mes | ~3.112 ISINs × ~22 días hábiles ≈ 68k filas |
| Observaciones por año | ~820k filas |

Memoria: parsear el XML de 15 MB en streaming consume <100 MB. Compatible con free tier Render (512 MB).

### Campos NO disponibles en FONDMENS/FONDREGISTRO

| Campo | Disponibilidad |
|-------|----------------|
| Divisa | XSD define `TipoDivisa` pero el XML no lo expone. Asumir EUR por defecto en MVP |
| Vocación inversora (categoría) | **Solo en FONDTRIM** (trimestral) |
| Comisión gestión / depósito | **Solo en FONDTRIM** |
| TER / Ratio gastos totales | **Solo en FONDTRIM** |
| Comisión suscripción / reembolso | **Solo en FONDTRIM** |
| Rentabilidad oficial | **Solo en FONDTRIM** (CNMV calcula 1M, 1Y, 3Y, 5Y) |

Implicación: comparación por TER/categoría/rentabilidad oficial requerirá parsear `FONDTRIM` en fase posterior, **o** calcular rentabilidad desde el histórico de NAV. Categoría y comisiones sólo CNMV puede aportarlas.

---

## Decisiones de diseño

### Lenguaje: Go

Consistente con `crypto-go`, `converter-go`, `macro-go`. stdlib + `pgx/v5`. Sin dependencias adicionales.

- `net/http` para descargar ZIP
- `archive/zip` para descomprimir
- `encoding/xml` con `Decoder` en streaming para parsear sin cargar todo el árbol
- `regexp` para extraer tokens del HTML

### Modelo: dos tablas

- `cnmv_funds` — maestro (un ISIN = una fila), poblado desde `FONDREGISTRO`
- `cnmv_nav_observations` — serie temporal de NAV/partícipes/patrimonio por ISIN+fecha

Sigue el patrón macro-go con dominio más rico.

### Parser en streaming

`encoding/xml.Decoder.Token()` permite recorrer el XML sin cargar el árbol completo. Para FONDMENS (15 MB, 3000 ISINs × 31 días × 3 métricas = ~280k elementos terminales), el streaming evita explosión de memoria.

### Sync incremental

Diferente a FRED/ECB: la CNMV publica **un fichero por mes**, no series por endpoint. Estrategia:

- `ingest`: descarga **el ZIP del mes en curso** + el del mes anterior (por si llega tarde el cierre). Upsert idempotente.
- `backfill`: itera sobre todos los meses disponibles desde un año dado (default 2020).

El upsert sobre `UNIQUE (isin, date)` permite re-ejecutar `ingest` sin duplicar datos.

### Almacenamiento de NAV

- `NUMERIC(20, 6)` para VL (XSD permite 4 decimales — margen de seguridad).
- `NUMERIC(20, 2)` para patrimonio.
- `BIGINT` para partícipes.

### Rate limiting

CNMV no documenta rate limit. Se descargan 1–12 ZIPs por ejecución. `time.Sleep(2 * time.Second)` entre descargas.

### User-Agent

Establecer User-Agent identificable (`ticker-lab-cnmv-go/1.0 (+https://tickerlab.fly.dev)`) — algunas páginas CNMV pueden devolver 403 sin User-Agent realista.

---

## Plan de implementación

### Estructura del servicio (`apps/cnmv-go/`)

```
apps/cnmv-go/
  go.mod              # module cnmv-go, go 1.25, pgx/v5
  go.sum
  main.go             # CLI (ingest, backfill) + HTTP server (:8130)
  models.go           # Fund, NAVObservation, FundSummary, NavPoint
  cnmv.go             # CNMVClient: ListMonthlyZips, DownloadZip
  parser.go           # Streaming XML parsers (ParseRegistro, ParseMens)
  repository.go       # Migrate, UpsertFunds, UpsertNAVs, FindFunds, FindNAVHistory
  handlers.go         # GET /health, /api/v1/funds, /api/v1/funds/{isin}, /api/v1/funds/{isin}/nav-observations
  main_test.go        # Tests: health, parser fixtures, repository, handlers
  testdata/
    FONDREGISTRO_sample.xml   # 3-5 entidades reales recortadas
    FONDMENS_sample.xml       # mismas 3-5 entidades, 5 días + algunos en 0

docker/cnmv-go/
  Dockerfile          # Multi-stage: builder → dev → prod (copia de macro-go)
```

### Schema SQL

```sql
CREATE TABLE IF NOT EXISTS cnmv_funds (
    isin                       CHAR(12) PRIMARY KEY,
    tipo                       VARCHAR(8)   NOT NULL,    -- FI | FHF | SICAV | SHF
    numero_registro            BIGINT       NOT NULL,
    numero_compartimento       BIGINT       NOT NULL DEFAULT 0,
    numero_clase               BIGINT       NOT NULL DEFAULT 0,
    denominacion               VARCHAR(200) NOT NULL,
    denominacion_compartimento VARCHAR(200),
    denominacion_clase         VARCHAR(200),
    is_etf                     BOOLEAN      NOT NULL DEFAULT FALSE,
    gestora_nombre             VARCHAR(200),
    gestora_grupo              VARCHAR(200),
    depositario_nombre         VARCHAR(200),
    depositario_grupo          VARCHAR(200),
    currency                   CHAR(3)      NOT NULL DEFAULT 'EUR',
    -- Campos solo disponibles en FONDTRIM (poblado en fase posterior)
    vocacion_inversora         VARCHAR(50),
    ter                        NUMERIC(8, 4),
    comision_gestion           NUMERIC(8, 4),
    comision_deposito          NUMERIC(8, 4),
    last_seen_period           CHAR(6),                   -- último YYYYMM en que apareció
    created_at                 TIMESTAMP DEFAULT NOW(),
    updated_at                 TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_cnmv_funds_tipo     ON cnmv_funds(tipo);
CREATE INDEX IF NOT EXISTS idx_cnmv_funds_gestora  ON cnmv_funds(gestora_nombre);
CREATE INDEX IF NOT EXISTS idx_cnmv_funds_vocacion ON cnmv_funds(vocacion_inversora);

CREATE TABLE IF NOT EXISTS cnmv_nav_observations (
    id            BIGSERIAL PRIMARY KEY,
    isin          CHAR(12)      NOT NULL,
    date          DATE          NOT NULL,
    nav           NUMERIC(20, 6) NOT NULL,
    participes    BIGINT,
    patrimonio    NUMERIC(20, 2),
    created_at    TIMESTAMP DEFAULT NOW(),
    UNIQUE (isin, date)
);

CREATE INDEX IF NOT EXISTS idx_cnmv_nav_isin_date ON cnmv_nav_observations(isin, date DESC);
CREATE INDEX IF NOT EXISTS idx_cnmv_nav_date      ON cnmv_nav_observations(date);
```

Decisiones del schema:
- `cnmv_funds.isin` PK — el ISIN es la clave estable de cara al consumidor.
- `last_seen_period` permite detectar fondos liquidados (no aparecen en último mes).
- Campos de FONDTRIM presentes desde el principio aunque se pueblen en fase 5 — evita migración futura.
- Sin JSONB.

### Tipos Go (`models.go`)

```go
type Fund struct {
    ISIN                       string  `json:"isin"`
    Tipo                       string  `json:"tipo"`
    NumeroRegistro             int64   `json:"numero_registro"`
    NumeroCompartimento        int64   `json:"numero_compartimento"`
    NumeroClase                int64   `json:"numero_clase"`
    Denominacion               string  `json:"denominacion"`
    DenominacionCompartimento  string  `json:"denominacion_compartimento,omitempty"`
    DenominacionClase          string  `json:"denominacion_clase,omitempty"`
    IsETF                      bool    `json:"is_etf"`
    GestoraNombre              string  `json:"gestora_nombre,omitempty"`
    GestoraGrupo               string  `json:"gestora_grupo,omitempty"`
    DepositarioNombre          string  `json:"depositario_nombre,omitempty"`
    DepositarioGrupo           string  `json:"depositario_grupo,omitempty"`
    Currency                   string  `json:"currency"`
    VocacionInversora          string  `json:"vocacion_inversora,omitempty"`
    TER                        float64 `json:"ter,omitempty"`
    ComisionGestion            float64 `json:"comision_gestion,omitempty"`
    ComisionDeposito           float64 `json:"comision_deposito,omitempty"`
    LastSeenPeriod             string  `json:"last_seen_period,omitempty"`
}

type FundSummary struct {
    ISIN          string  `json:"isin"`
    Denominacion  string  `json:"denominacion"`
    GestoraNombre string  `json:"gestora_nombre,omitempty"`
    Tipo          string  `json:"tipo"`
    LatestNAV     float64 `json:"latest_nav"`
    LatestDate    string  `json:"latest_date"`
    PrevNAV       float64 `json:"prev_nav,omitempty"`
    ChangePct     float64 `json:"change_pct,omitempty"`
    Patrimonio    float64 `json:"patrimonio,omitempty"`
    Participes    int64   `json:"participes,omitempty"`
}

type NAVObservation struct {
    ISIN       string
    Date       string  // YYYY-MM-DD
    NAV        float64
    Participes int64
    Patrimonio float64
}

type NavPoint struct {
    Date string  `json:"date"`
    NAV  float64 `json:"nav"`
}
```

### Cliente CNMV (`cnmv.go`)

```go
type CNMVClient struct {
    listURL    string
    httpClient *http.Client
    userAgent  string
}

type MonthlyZip struct {
    Year  int
    Month int    // 1..12
    URL   string // URL completa con token opaco
}

func (c *CNMVClient) ListMonthlyZips(year int) ([]MonthlyZip, error)
func (c *CNMVClient) DownloadZip(url string) (registroXML, mensXML []byte, err error)
```

`ListMonthlyZips`:
1. GET `listURL?ano={year}` (selector de año soportado por la página)
2. Regex sobre HTML:
   ```regex
   <a id="ctl00_ContentPrincipal_grdDescargas_ctl\d+_lnkZip"
      href="(https://www\.cnmv\.es/webservices/verdocumento/ver\?e=[^"]+)"
      title="(Enero|Febrero|...|Diciembre)"
   ```
3. Mapear nombre mes ES → número.

`DownloadZip`:
1. GET URL → bytes del ZIP
2. `zip.NewReader(bytes.NewReader(data), int64(len(data)))`
3. Identificar archivos por prefijo `FONDREGISTRO_` y `FONDMENS_`
4. Leer cada uno completo a `[]byte`

### Parser XML streaming (`parser.go`)

```go
func ParseRegistro(xmlData []byte) ([]Fund, string, error)  // funds, fechaDatos
func ParseMens(xmlData []byte) ([]NAVObservation, error)
```

`ParseMens` (streaming con `encoding/xml.Decoder`):

```go
func ParseMens(xmlData []byte) ([]NAVObservation, error) {
    dec := xml.NewDecoder(bytes.NewReader(xmlData))
    var year, month int
    var currentClase classState  // num, isin
    var navByDay [32]float64
    var partByDay [32]int64
    var patrByDay [32]float64
    var out []NAVObservation

    for {
        tok, err := dec.Token()
        if err == io.EOF { break }
        if err != nil { return nil, err }

        switch t := tok.(type) {
        case xml.StartElement:
            // VL_DiaN, Participes_DiaN, Patrimonio_DiaN — extraer N con strings.TrimPrefix + strconv.Atoi
            // ...
        case xml.EndElement:
            if t.Name.Local == "Clase" && currentClase.isin != "" {
                lastDay := time.Date(year, time.Month(month+1), 0, 0, 0, 0, 0, time.UTC).Day()
                for day := 1; day <= lastDay; day++ {
                    if navByDay[day] == 0 { continue }
                    out = append(out, NAVObservation{
                        ISIN: currentClase.isin,
                        Date: fmt.Sprintf("%04d-%02d-%02d", year, month, day),
                        NAV:  navByDay[day],
                        Participes: partByDay[day],
                        Patrimonio: patrByDay[day],
                    })
                }
            }
        }
    }
    return out, nil
}
```

Validación de fecha real: descartar `day > lastDay` del mes (febrero, 30 días). Usar `time.Date(year, month+1, 0, ...).Day()`.

### Repository (`repository.go`)

```go
type Repository struct{ pool *pgxpool.Pool }

func (r *Repository) Migrate(ctx context.Context) error
func (r *Repository) UpsertFunds(ctx context.Context, funds []Fund, period string) error
func (r *Repository) UpsertNAVs(ctx context.Context, obs []NAVObservation) error
func (r *Repository) FindFunds(ctx context.Context, q FundQuery) ([]FundSummary, error)
func (r *Repository) FindFundByISIN(ctx context.Context, isin string) (*Fund, *FundSummary, error)
func (r *Repository) FindNAVHistory(ctx context.Context, isin string, days int) ([]NavPoint, error)
```

`UpsertFunds` — chunks de 500:
```sql
INSERT INTO cnmv_funds (isin, tipo, numero_registro, ..., last_seen_period)
VALUES ($1, $2, ...), ...
ON CONFLICT (isin) DO UPDATE SET
    denominacion = EXCLUDED.denominacion,
    is_etf = EXCLUDED.is_etf,
    gestora_nombre = EXCLUDED.gestora_nombre,
    last_seen_period = EXCLUDED.last_seen_period,
    updated_at = NOW();
```

`UpsertNAVs` — chunks de 1000:
```sql
INSERT INTO cnmv_nav_observations (isin, date, nav, participes, patrimonio)
VALUES ($1, $2, $3, $4, $5), ...
ON CONFLICT (isin, date) DO UPDATE
SET nav = EXCLUDED.nav,
    participes = EXCLUDED.participes,
    patrimonio = EXCLUDED.patrimonio;
```

`FindFunds` con filtros opcionales:
```sql
SELECT f.isin, f.denominacion, f.gestora_nombre, f.tipo,
       o.nav AS latest_nav, o.date::text AS latest_date,
       o.patrimonio, o.participes,
       prev.nav AS prev_nav,
       CASE WHEN prev.nav > 0 AND prev.nav IS NOT NULL
            THEN ((o.nav - prev.nav) / prev.nav * 100)
            ELSE 0 END AS change_pct
FROM cnmv_funds f
LEFT JOIN LATERAL (
    SELECT nav, date, patrimonio, participes
    FROM cnmv_nav_observations
    WHERE isin = f.isin ORDER BY date DESC LIMIT 1
) o ON true
LEFT JOIN LATERAL (
    SELECT nav FROM cnmv_nav_observations
    WHERE isin = f.isin AND date < o.date
    ORDER BY date DESC LIMIT 1
) prev ON true
WHERE ($1::text IS NULL OR f.tipo = $1)
  AND ($2::text IS NULL OR f.gestora_nombre ILIKE '%' || $2 || '%')
  AND ($3::text IS NULL OR f.denominacion ILIKE '%' || $3 || '%')
ORDER BY o.patrimonio DESC NULLS LAST
LIMIT $4;
```

### Handlers (`handlers.go`)

Routing con `http.ServeMux` (Go 1.22+ path patterns):
```
GET /health
GET /api/v1/funds?tipo=&gestora=&q=&page_size=&page_token=             # list with filters
GET /api/v1/funds/{isin}                                                # detail
GET /api/v1/funds/{isin}/nav-observations?start_date=&end_date=&page_size=&page_token=
```

URL shape follows Google AIP-122/132/158 conventions adopted in [`api-design-standards.md`](./api-design-standards.md): plural resource collections, no verb-shaped path segments, mandatory pagination from inception. Critical here because CNMV publishes ~3,112 ISINs — clients cannot consume the collection without paging.

List response shape:
```json
{
  "funds": [ /* FundSummary objects */ ],
  "next_page_token": "",
  "total_size": 3112
}
```

Validaciones:
- ISIN: regex `^[A-Z]{2}[A-Z0-9]{9}[0-9]$`
- `start_date` / `end_date`: ISO 8601 `YYYY-MM-DD`. Default `end_date = today`, `start_date = today - 365d` if both omitted.
- `page_size` clamp `[1, 500]` (default 100). Negative → 400 `INVALID_ARGUMENT`.
- `page_token` opaque, URL-safe; clients must not parse it. APIs may expire after ~3 days.

Errores como `application/problem+json` (RFC 7807), patrón macro-go.

### Subcomandos CLI

```go
case "ingest":   runIngest(repo, client)               // mes actual + mes anterior
case "backfill": runBackfill(repo, client, fromYear)   // desde 2020
```

`runIngest`:
1. Calcular YYYYMM actual y anterior
2. Para cada mes: `ListMonthlyZips(year)` → encontrar mes → `DownloadZip(url)` → `ParseRegistro` + `ParseMens` → `UpsertFunds` + `UpsertNAVs` → `time.Sleep(2 * time.Second)`

`runBackfill`:
1. Desde `fromYear` (default 2020) hasta año actual
2. Procesar todos los meses encontrados — ~6 años × 12 meses = 72 ZIPs × ~70k filas = ~5M filas, ~30 min de ejecución

### Docker + Infraestructura

**Puerto:** 8130.

**docker-compose.yml**:
```yaml
cnmv-go:
  build:
    context: .
    dockerfile: docker/cnmv-go/Dockerfile
    target: dev
  ports: ["8130:8130"]
  environment:
    - DATABASE_URL=postgresql://ticker:ticker@db:5432/ticker_lab
    - PORT=8130
  depends_on:
    db: { condition: service_healthy }
```

**GitHub Actions** — cron diario `30 8 * * *` (08:30 UTC, tras publicación CNMV). Sobredimensionado para periodicidad mensual; suficiente con cron mensual, pero se mantiene diario por consistencia. Re-ejecuciones idempotentes.

### Dashboard SSR

**`/funds`** — Listado y exploración:
- Buscador (denominación, ISIN, gestora) con debounce
- Filtros: tipo (FI/SICAV), gestora (dropdown top-20)
- Tabla ordenable: ISIN, Denominación, Gestora, NAV, Variación %, Patrimonio
- Top movers (mayor variación últimos 30 días) en sección destacada
- Paginación (50 por página)
- Color accent: `#f59e0b` (amber 500)

**`/funds/:isin`** — Detalle:
- Header: denominación, ISIN, tipo, gestora, depositario, ETF flag
- Métricas: último NAV, variación 1D/1M/1Y (calculada desde histórico)
- Chart.js — histórico NAV con selector 1M/3M/6M/1Y/3Y/5Y/ALL
- Patrimonio actual y partícipes
- Aviso si campos FONDTRIM no están aún disponibles

### Testing

- `TestParseRegistro` — fixture XML reducido (3-5 entidades), valida funds extraídos
- `TestParseMens` — fixture XML, valida NAV observations descartando días 0
- `TestParseMens_FebruaryShortMonth` — valida descarte días 30 y 31
- `TestUpsertFundsAndNAVs` — integration con idempotencia
- `TestFindFundsWithFilters` — filtros tipo/gestora/q
- Cleanup: `DELETE WHERE isin LIKE 'TEST%'`

### Fases de implementación

1. **Esqueleto + parser FONDREGISTRO** — `go.mod`, modelos, cliente HTML scraper, `ParseRegistro`, repository (UpsertFunds + FindFunds), handler `/funds`. Docker.
2. **Parser FONDMENS + NAV observations** — streaming `ParseMens`, `UpsertNAVs`, `FindNAVObservations`, endpoint `/funds/{isin}/nav-observations` with `start_date`/`end_date`/`page_size`/`page_token`, subcomando `ingest`.
3. **Backfill + CI/CD** — subcomando `backfill`, GitHub Actions cron. Backfill 2020–presente.
4. **Dashboard SSR** — rutas `/funds` y `/funds/:isin`, top movers, Chart.js.
5. **FONDTRIM (opcional)** — parser trimestral, poblar TER/categoría/comisiones, sin migración de schema.
6. **Comparador (opcional)** — `/api/v1/funds/compare?isins=...` con rentabilidades calculadas + UI multi-select.

---

## Referencias

- [CNMV — Descarga de información individual de IIC](https://www.cnmv.es/portal/Publicaciones/Descarga-Informacion-Individual.aspx)
- [CNMV — Periodicidad y descripción de ficheros (PDF)](https://www.cnmv.es/DocPortal/Estadisticas/Descarga-Informacion-Individual/Periodicidad-Descripcion.pdf)
- [Circular 4/2008 CNMV — Contenido informes IIC](https://www.boe.es/buscar/act.php?id=BOE-A-2008-19524)
- [Circular 1/2009 CNMV — Categorías por vocación inversora](https://www.boe.es/buscar/act.php?id=BOE-A-2009-2742)
- [Listado completo de fondos de inversión — CNMV](https://www.cnmv.es/portal/consultas/mostrarlistados?id=3)
- [DGSFP — Planes y fondos de pensiones](https://dgsfp.mineco.gob.es/es/Paginas/Planes-y-Fondos-de-Pensiones.aspx) (fuera de scope)
