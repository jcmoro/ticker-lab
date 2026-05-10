# Banco de España (BdE) Integration — Tipos de interés españoles

## Resumen

Integración con la **API de estadísticas del Banco de España (BIEST)** para incorporar tipos de interés y referencias del mercado hipotecario español a Ticker Lab. Esta fuente complementa a FRED (EEUU) y ECB (eurozona) ya integradas en el servicio `macro-go`, aportando datos específicos de España no presentes en ECB SDW: IRPH, MIBOR y los tipos NEDR/TEDR aplicados por entidades españolas (distintos del agregado MIR de la eurozona).

**Decisión adoptada:** añadir BdE como tercera fuente dentro del servicio `macro-go` existente (no servicio nuevo). Justificación más abajo.

**Verificación contra ECB:** la BdE **no** es redundante. ECB MIR publica el agregado eurozona; BdE publica los tipos NEDR/TEDR de las entidades de crédito españolas (subconjunto Spain-only) y los tipos oficiales de referencia del mercado hipotecario (IRPH, MIBOR — definidos en la Orden EHA/2899/2011 y CBE 5/2012, exclusivos de España). Estos no están en ECB SDW.

---

## Banco de España SDW — info general

| Campo | Valor |
|-------|-------|
| Base URL API | `https://app.bde.es/bierest/resources/srdatosapp/` |
| Auth | **No requiere autenticación** (acceso público) |
| Formato | JSON (único formato del API REST) |
| Encoding | gzip — `Accept-Encoding: gzip` o `curl --compressed` |
| Rate limit | No documentado |
| Documentación | https://www.bde.es/webbe/en/estadisticas/recursos/api-estadisticas-bde.html |
| Contacto | Servicio de Información Estadística — +34 91 338 5651 (L-V 9:00-14:00) |
| Bulk CSV | https://www.bde.es/webbe/en/estadisticas/recursos/descargas-completas.html |

> **Nota**: la URL `bde.es/webbe/...` corresponde al front web; el API REST real está en `app.bde.es/bierest/resources/srdatosapp/`.

### Endpoints

#### `GET /favoritas` — último valor

Devuelve el último dato publicado de una o más series.

```
https://app.bde.es/bierest/resources/srdatosapp/favoritas?idioma=en&series=D_1NBAF472
```

Parámetros:

| Param | Tipo | Descripción |
|-------|------|-------------|
| `idioma` | String | **Requerido**. `es` o `en` |
| `series` | String | **Requerido**. Códigos separados por coma. El carácter `#` se URL-encodea como `%23` |

Response:
```json
[
  {
    "serie": "D_1NBAF472",
    "descripcionCorta": "One-year Euribor",
    "codFrecuencia": "M",
    "decimales": 3,
    "simbolo": "%",
    "tendencia": "+",
    "fechaValor": "2026-04-01T08:15:00Z",
    "valor": 2.747
  }
]
```

Si el código no existe, se devuelve `{ "codigo": "...", "errNum": 404 }` en su posición.

#### `GET /listaSeries` — serie histórica completa

Devuelve metadata + arrays paralelos de fechas y valores. **Endpoint principal para ingesta.**

```
https://app.bde.es/bierest/resources/srdatosapp/listaSeries?idioma=es&series=D_1NBAF472&rango=30M
```

Parámetros:

| Param | Tipo | Descripción |
|-------|------|-------------|
| `idioma` | String | **Requerido**. `es` o `en` |
| `series` | String | **Requerido**. Códigos separados por coma |
| `rango` | String | Opcional. Año concreto (`2024`); mensual: `30M`, `60M`, `MAX`; diario: `3M`, `12M`, `36M`. Validación estricta (rango incompatible con frecuencia → `errNum: 412`) |

Response:
```json
[{
  "serie": "D_1NBAF472",
  "descripcion": "Euribor a un año",
  "codFrecuencia": "M",
  "decimales": 3,
  "simbolo": "%",
  "fechaInicio": "1999-01-01T09:15:00Z",
  "fechaFin": "2026-04-01T08:15:00Z",
  "fechas":  ["2026-04-01T08:15:00Z", "2026-03-01T09:15:00Z", ...],
  "valores": [2.747, 2.565, ...]
}]
```

Notas:
- `fechas[i]` corresponde a `valores[i]` (mismo índice). Orden descendente por defecto.
- Mensual ⇒ fecha = primer día del mes.
- Diaria ⇒ día de referencia.
- Trimestral ⇒ primer día del primer mes del trimestre.
- Anual ⇒ 1 de enero.
- El offset `T08:15:00Z`/`T09:15:00Z` varía con DST; truncar a `YYYY-MM-DD`.

### Cadencia de actualización

- **Diaria** para series Euribor mercado monetario UEM (`D_DNBAF172`, etc.) — L-V tras publicación EMMI.
- **Mensual** para tipos oficiales de referencia hipotecario (Euribor referencia, IRPH, MIBOR) — días 2-3 del mes siguiente, oficializados en BOE ~día 18-20.
- **Mensual** para tipos NEDR/TEDR de entidades españolas (TIPI) — publicación con ~2 meses de retraso (datos marzo disponibles en mayo).

---

## Series relevantes

### Tier 1 — Tipos oficiales de referencia hipotecario (mensuales)

Definidos en Orden EHA/2899/2011 y CBE 5/2012. Series exclusivas de España.

| Código BdE | Descripción | Frecuencia | Categoría |
|------------|-------------|------------|-----------|
| `D_1NBAS972` | Euríbor 1 semana — referencia oficial mercado hipotecario | Mensual | `spanish_rates` |
| `D_1NBAC972` | Euríbor 1 mes — referencia oficial mercado hipotecario | Mensual | `spanish_rates` |
| `D_1NBAD972` | Euríbor 3 meses — referencia oficial mercado hipotecario | Mensual | `spanish_rates` |
| `D_1NBAE972` | Euríbor 6 meses — referencia oficial mercado hipotecario | Mensual | `spanish_rates` |
| `D_1NBAF472` | Euríbor 12 meses — referencia oficial mercado hipotecario | Mensual | `spanish_rates` |
| `D_1T9H0000` | **IRPH** — Tipo medio préstamos hipotecarios > 3 años, vivienda libre, conjunto EC España | Mensual | `spanish_rates` |
| `D_1E723706.EUR` | **MIBOR** a 1 año (préstamos formalizados antes de 1-ene-2000) | Mensual | `spanish_rates` (legacy) |
| `D_1T9H0011` | IRS 5 años — referencia oficial hipotecaria | Mensual | `spanish_rates` |
| `D_1T9H0004` | Rendimiento Deuda Pública 2-6 años — referencia hipotecaria | Mensual | `spanish_rates` |

### Tier 2 — Euribor diario (mercado monetario UEM, feed Refinitiv)

| Código BdE | Descripción | Frecuencia |
|------------|-------------|------------|
| `D_DNBAS172` | Euríbor 1 semana (diario) | Diaria |
| `D_DNBAC172` | Euríbor 1 mes (diario) | Diaria |
| `D_DNBAD172` | Euríbor 3 meses (diario) | Diaria |
| `D_DNBAE172` | Euríbor 6 meses (diario) | Diaria |
| `D_DNBAF172` | Euríbor 12 meses (diario) | Diaria |

### Tier 3 — TIPI: tipos aplicados por entidades españolas (Spain-only NEDR/TEDR)

Estas series son el delta principal frente a ECB MIR (que sólo publica el agregado eurozona).

| Código BdE | Descripción | Frecuencia |
|------------|-------------|------------|
| `DN_1TI2T0135` | Préstamos a hogares — vivienda — nuevas operaciones (TEDR) | Mensual |
| `DN_1TI2T0138` | Préstamos a hogares — consumo — nuevas operaciones (TEDR) | Mensual |
| `DN_1TI2T0141` | Préstamos a hogares — otros fines — nuevas operaciones | Mensual |
| `DN_1TI2T0144` | Préstamos a sociedades no financieras — total — nuevas operaciones | Mensual |
| `DN_1TI2T0116` | Tarjetas de crédito de pago aplazado — hogares | Mensual |

(Códigos extraídos del catálogo `be1903.csv` dentro de `SB_TIIF.zip`. Existen muchas más series TIPI; estas cinco son el subset más representativo.)

---

## Comparación con ECB SDW — ¿es redundante?

| Concepto | ECB SDW | BdE BIEST | ¿Solapa? |
|----------|---------|-----------|----------|
| Tipo de intervención BCE (MRR, DFR) | `FM` dataflow | `D_DNBCEB72` etc. | Sí — usar ECB |
| ESTR | `EST` | `D_1NBAS572` | Sí — usar ECB (mejor cobertura) |
| Euribor 12m | No publica directamente | `D_1NBAF472`, `D_DNBAF172` | **No solapa** — ECB no expone Euribor en SDW |
| HICP eurozona | `ICP` | (BdE no publica HICP) | N/A |
| MIR — agregado eurozona | `MIR` dataflow | — | — |
| **NEDR/TEDR España solo** | No (agregado UE) | `DN_1TI2T*` | **No solapa** — Spain-specific |
| **IRPH** (referencia oficial hipotecaria ES) | No | `D_1T9H0000` | **No solapa** — definido por ley española |
| **MIBOR** (legacy ES) | No | `D_1E723706.EUR` | **No solapa** — exclusivo ES |

**Conclusión:** BdE aporta valor real. Ofrece (a) Euribor con resolución diaria que ECB SDW no expone como serie consultable, (b) IRPH y referencias hipotecarias oficiales españolas, y (c) tipos efectivos NEDR/TEDR aplicados por la banca española. **Recomendación: integrar.**

---

## Decisión de integración: Opción A vs Opción B

### Opción A — Añadir BdE al servicio `macro-go` existente (RECOMENDADA)

Añadir un cliente `bde.go` junto a `fred.go` y `ecb.go` dentro de `apps/macro-go/`. Reutilizar `macro_series` + `macro_observations` con `source = "bde"`.

### Opción B — Servicio nuevo `bde-go`

Microservicio independiente con su propia base de datos, Dockerfile, puerto, etc.

### Justificación de Opción A

1. **Mismo modelo de dominio.** BdE expone series temporales escalares idénticas en forma a FRED y ECB: `(source, series_id, date, value)`. La tabla `macro_series` ya existe con la columna `source` precisamente para esto, y `macro_observations` tiene `UNIQUE(source, series_id, date)`. Cero cambios de schema.
2. **Mismo lenguaje, misma stdlib.** El cliente BdE necesita `net/http` + `encoding/json` + `compress/gzip` (auto-handled por Go con `Transport.DisableCompression=false`). Idéntico stack al de `fred.go`.
3. **Misma cadencia operativa.** Sync diario via cron — mismo ciclo que FRED/ECB. Un solo job (`make job-macro-ingest`) puede orquestar las tres fuentes.
4. **Menor superficie operativa.** No nuevo Dockerfile, no nuevo puerto, no nuevo deployment, no nuevas variables de entorno (BdE no requiere auth), no nueva entrada en `dashboard.ts`. La página `/macro` ya agrupa por `category`; basta añadir la categoría `spanish_rates`.
5. **Coherencia con el patrón establecido.** El changelog y `architecture.md` ya documentan macro-go como "macro indicators (multi-source)". Añadir un servicio aparte fragmentaría el dominio sin beneficio.
6. **Reversibilidad.** Si en el futuro BdE crece (más series, mayor frecuencia, parsing CSV de descargas zip), extraerlo a su propio servicio es trivial — el código está aislado en un fichero.

**Trade-offs de Opción B:**
- Único beneficio plausible: aislamiento de fallos. Pero cada fuente ya tiene `continue` en su loop de ingesta, así que ya están aisladas a nivel de proceso.
- Coste: triplicar Dockerfile + compose + Makefile entries + tests de salud + monitorización.

---

## Plan de implementación (Opción A)

### Cambios en schema

**Ninguno.** Reutiliza `macro_series` y `macro_observations` con `source = "bde"`.

La columna `source` en el schema actual es `VARCHAR(10)`. `"bde"` (3 chars) cabe sin problema.

### Estructura del servicio (delta sobre lo existente)

```
apps/macro-go/
  bde.go              # NUEVO — cliente BdE: BIEST API, JSON parser
  models.go           # MODIFICADO — añadir bdeSeries, bdeDefaultRange
  main.go             # MODIFICADO — subcomandos ingest-bde + backfill incluye BdE
  main_test.go        # MODIFICADO — TestBDEDateNormalization, TestBDEParsing
```

### Cliente BdE (`bde.go`)

```go
type BDEClient struct {
    baseURL    string        // https://app.bde.es/bierest/resources/srdatosapp
    httpClient *http.Client  // 30s timeout, gzip auto
}

func NewBDEClient() *BDEClient

func (c *BDEClient) FetchSeries(seriesID string, rango string) ([]Observation, error)
```

Flujo de `FetchSeries`:
1. GET `/listaSeries?idioma=es&series={seriesID}&rango={rango}`
2. Header `Accept-Encoding: gzip` (Go lo añade automáticamente con `Transport` por defecto y descomprime al leer el body).
3. Parsear JSON — array con un único elemento.
4. Si la respuesta tiene `errNum`, devolver error.
5. Iterar `fechas[]` y `valores[]` en paralelo. Truncar fecha ISO 8601 a `YYYY-MM-DD`.
6. Filtrar valores `null` o `_` (gaps históricos).
7. Devolver `[]Observation` con `Source: "bde"`.

Manejo de `rango` para sync incremental:
- BdE no acepta `startPeriod` arbitrario — sólo enums (`30M`, `60M`, `MAX`, etc.).
- **Estrategia:** usar `rango=30M` en sync diario (siempre devuelve los últimos ~30 meses para mensuales, el upsert deduplica). Para diarios usar `rango=3M`.
- Para `backfill` usar `rango=MAX`.

### Configuración de series (`models.go`)

```go
var bdeSeries = []SeriesMeta{
    // Tier 1 — Tipos de referencia oficial mercado hipotecario (mensuales)
    {Source: "bde", SeriesID: "D_1NBAF472", Name: "Euribor 12m (referencia hipotecaria ES)", Frequency: "monthly", Unit: "percent", Category: "spanish_rates"},
    {Source: "bde", SeriesID: "D_1NBAE972", Name: "Euribor 6m (referencia hipotecaria ES)",  Frequency: "monthly", Unit: "percent", Category: "spanish_rates"},
    {Source: "bde", SeriesID: "D_1NBAD972", Name: "Euribor 3m (referencia hipotecaria ES)",  Frequency: "monthly", Unit: "percent", Category: "spanish_rates"},
    {Source: "bde", SeriesID: "D_1NBAC972", Name: "Euribor 1m (referencia hipotecaria ES)",  Frequency: "monthly", Unit: "percent", Category: "spanish_rates"},
    {Source: "bde", SeriesID: "D_1T9H0000", Name: "IRPH (Tipo medio préstamos hipotecarios)", Frequency: "monthly", Unit: "percent", Category: "spanish_rates"},
    {Source: "bde", SeriesID: "D_1T9H0011", Name: "IRS 5 años (referencia hipotecaria ES)",  Frequency: "monthly", Unit: "percent", Category: "spanish_rates"},
    // Tier 2 — Euribor diario
    {Source: "bde", SeriesID: "D_DNBAF172", Name: "Euribor 12m (diario)", Frequency: "daily", Unit: "percent", Category: "spanish_rates"},
    // Tier 3 — Tipos aplicados por entidades españolas (TIPI, Spain-only)
    {Source: "bde", SeriesID: "DN_1TI2T0135", Name: "Préstamos hogares — vivienda (TEDR ES)",  Frequency: "monthly", Unit: "percent", Category: "spanish_rates"},
    {Source: "bde", SeriesID: "DN_1TI2T0138", Name: "Préstamos hogares — consumo (TEDR ES)",   Frequency: "monthly", Unit: "percent", Category: "spanish_rates"},
    {Source: "bde", SeriesID: "DN_1TI2T0144", Name: "Préstamos sociedades no financ. (TEDR ES)", Frequency: "monthly", Unit: "percent", Category: "spanish_rates"},
}

var bdeDefaultRange = map[string]string{
    "daily":     "36M",
    "monthly":   "60M",
    "quarterly": "MAX",
}
```

### Subcomandos CLI (`main.go`)

```go
case "ingest-bde":
    runIngestBDE(repo, NewBDEClient())
    return
```

`runIngestBDE`:
1. Para cada serie en `bdeSeries`:
   - Determinar `rango` según `Frequency` (vía `bdeDefaultRange`)
   - `FetchSeries(seriesID, rango)`
   - `SaveObservations(observations)` (upsert vía `ON CONFLICT`)
   - `UpdateLastSynced(...)`
   - `time.Sleep(1 * time.Second)` (precaución)

`runBackfill`: añadir bloque BdE con `rango=MAX`.

### Tests (`main_test.go`)

Añadir:
- `TestBDEDateNormalization` — `"2026-04-01T08:15:00Z"` → `"2026-04-01"`. Caso edge: gap (`null`/`_`) se filtra.
- `TestBDEParsing` — fixture JSON con array de un objeto y arrays paralelos `fechas`/`valores`.
- `TestBDEErrorResponse` — fixture con `{"errNum": 412, "errMsgUsr": "..."}` produce error.

No se añade test de integración HTTP real (mismo criterio que FRED/ECB — los clientes externos se prueban con fixtures).

### Dashboard SSR

Cambio mínimo en `apps/api/`:
- La ruta `/macro` ya agrupa por categoría. La nueva categoría `spanish_rates` aparecerá automáticamente al haber filas en `macro_series`.
- En la plantilla `macro.eta`, añadir el label "Spanish Rates" en el orden de categorías.
- Color accent sugerido: rojo/amarillo (referencia visual a la bandera española) — opcional.

### Variables de entorno

Ninguna nueva. BdE no requiere API key.

### Makefile

```makefile
job-macro-ingest-bde:        ## Ingest BdE Spanish rates
	docker compose run --rm macro-go ./macro-go ingest-bde
```

E incluirlo en el job conjunto:
```makefile
job-macro-ingest:
	docker compose run --rm macro-go ./macro-go ingest
	docker compose run --rm macro-go ./macro-go ingest-ecb
	docker compose run --rm macro-go ./macro-go ingest-bde
```

### Documentación

- `docs/changelog.md` — entrada nueva: "BdE como tercera fuente de datos macro (Spanish rates)".
- `docs/architecture.md` — actualizar sección macro-go para mencionar las tres fuentes.
- Sin cambios en `apps/api/openapi.yaml` (los endpoints `/api/v1/macro/*` ya son genéricos por `source`).

> **Nota AIP (2026-05-10):** los endpoints actuales de `macro-go` (`/api/v1/macro/indicators`, `/api/v1/macro/{source}/{id}/history`) son verb-shaped y sin paginación — quedan flagged en el [audit AIP](./tech-debt-analysis.md#addendum-2026-05-10--api-design-audit-vs-google-aip) para refactorización en el ciclo `/api/v2/` (Tier 2 ítem 8). BdE no introduce nuevos endpoints, así que **hereda** la URL shape actual y el cambio futuro a `v2`. No requiere acción en este servicio.

---

## Fases de implementación

1. **Cliente BdE + ingesta Tier 1 + Tier 2** — `bde.go` con `FetchSeries`, series Tier 1+2 en `bdeSeries`, subcomando `ingest-bde`, bloque backfill, tests parsing+normalización. Validar `make go-ci` + ejecución manual contra DB local.
2. **Tier 3 (TIPI) + dashboard** — series `DN_1TI2T*`, plantilla `macro.eta` con label "Spanish Rates". Re-test backfill completo.
3. **CI/CD + docs** — cron diario en GitHub Actions, `changelog.md` y `architecture.md`.

---

## Hallazgos clave de la investigación

1. **Base URL real validada**: `https://app.bde.es/bierest/resources/srdatosapp/` (no la URL del front web `bde.es/webbe/...`).
2. **Formato:** JSON gzipped — Go lo descomprime automáticamente. No es SDMX.
3. **Sin auth y sin rate limit documentado.** Sleep defensivo de 1s entre requests.
4. **El parámetro `rango` es enum, no fechas arbitrarias.** Cambia la estrategia de sync: traer siempre los últimos N meses y confiar en el upsert.
5. **No es redundante con ECB.** IRPH, MIBOR y los tipos NEDR/TEDR Spain-only no están en ECB SDW.
6. **Recomendación firme: Opción A.** Añadir a `macro-go` con `source="bde"`, sin cambios de schema. Tres fases pequeñas.

---

## Referencias

- [BdE Statistics Web Service (API)](https://www.bde.es/webbe/en/estadisticas/recursos/api-estadisticas-bde.html)
- [BdE Time-series bulk download](https://www.bde.es/webbe/en/estadisticas/recursos/descargas-completas.html)
- [BdE Interest rate statistics](https://www.bde.es/webbe/en/estadisticas/temas/tipos-interes.html)
- [Manual archivos CSV](https://www.bde.es/webbe/es/estadisticas/compartido/docs/manual_archivos_csv.pdf)
- [Catálogo TI (interest rates)](https://www.bde.es/webbe/es/estadisticas/compartido/datos/zip/ti.zip)
- [Catálogo SB_TIIF (MFI rates)](https://www.bde.es/webbe/es/estadisticas/compartido/datos/zip/SB_TIIF.zip)
- [Catálogo SB_TIINTREF (mortgage reference)](https://www.bde.es/webbe/es/estadisticas/compartido/datos/zip/SB_TIINTREF.zip)
- [Cuadro 19.1 PDF — tipos legales, euríbor y referencia](https://www.bde.es/webbe/es/estadisticas/compartido/datos/pdf/a1901.pdf)
- Orden EHA/2899/2011 y CBE 5/2012 — definición legal de tipos oficiales de referencia
