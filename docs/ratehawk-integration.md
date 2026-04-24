# RateHawk Integration — Hotel Catalog

## Resumen

Integración con la [Content API de RateHawk](https://docs.emergingtravel.com/docs/content-api/retrieve-hotel-ids-by-filter/) (Emerging Travel Group) para ingestar un catálogo de hoteles y exponerlo en el dashboard de Ticker Lab.

RateHawk es un agregador B2B de hoteles que ofrece acceso a inventario global de alojamiento. La Content API permite obtener información estática de hoteles (no precios ni disponibilidad).

---

## Qué es la Content API

La Content API de RateHawk proporciona datos de catálogo de hoteles: identificadores, nombre, ubicación, estrellas, imágenes, habitaciones, amenidades y políticas. No es una API de reservas ni de precios — es un catálogo estático que se actualiza periódicamente.

### Endpoints utilizados

#### 1. Retrieve Filter Values
- `GET /api/content/v1/filter_values`
- Devuelve los valores válidos para filtrar hoteles: países (con ID numérico), idiomas, tipos de alojamiento, amenidades
- Se usa como primer paso para conocer los filtros disponibles

#### 2. Retrieve Hotel IDs by Filter
- `POST /api/content/v1/hotel_ids_by_filter/`
- Filtra hoteles por país, tipo, estrellas, amenidades, fecha de actualización
- Devuelve arrays de IDs numéricos (`hids`)
- Soporta `updated_since` para obtener solo hoteles modificados desde una fecha (sync incremental)
- Rate limit: 60 requests/minuto

#### 3. Retrieve Hotel Content by IDs
- `POST /api/content/v1/hotel_content_by_ids/`
- Dado un array de HIDs y un idioma, devuelve el contenido completo de cada hotel
- Rate limit: 1200 requests/minuto

### Autenticación

HTTP Basic Auth con `KEY_ID` y `API_KEY` proporcionados por RateHawk.

### Entornos

| Entorno | URL base | Notas |
|---------|----------|-------|
| Sandbox | `https://api-sandbox.worldota.net` | Limitado a países 59, 153, 189, 201 |
| Producción | `https://api.worldota.net` | Acceso completo, requiere certificación |

---

## Modelo de datos del hotel

El contenido de un hotel incluye:

| Campo | Tipo | Descripción |
|-------|------|-------------|
| `hid` | Integer | ID numérico (PK, max 10 dígitos) |
| `name` | String | Nombre del hotel |
| `kind` | String | Tipo: Hotel, Resort, Apartment, Hostel, BNB, Villa, etc. |
| `star_rating` | Integer | 0-5 (0 = sin clasificación) |
| `address` | String | Dirección física |
| `latitude` / `longitude` | Float | Coordenadas geográficas |
| `region` | Object | País, nombre de región, tipo |
| `check_in_time` / `check_out_time` | String | Horarios (HH:MM:SS) |
| `images_ext` | Array | Imágenes categorizadas (exterior, habitaciones, piscina, etc.) |
| `room_groups` | Array | Tipos de habitación con amenidades, categoría, capacidad, vistas |
| `amenity_groups` | Array | Amenidades agrupadas (gratuitas y de pago) |
| `description_struct` | Array | Descripción en párrafos con títulos |
| `metapolicy_struct` | Object | Políticas: mascotas, parking, internet, niños, check-in/out, shuttle |
| `facts` | Object | Datos estructurales: año construcción, pisos, habitaciones, enchufes |
| `payment_methods` | Array | Métodos de pago aceptados |
| `serp_filters` | Array | Filtros de búsqueda (has_parking, has_internet, has_breakfast, etc.) |

---

## Flujo de ingesta

```
1. Obtener HIDs por país
   POST hotel_ids_by_filter/ {country: [59]}
   → [12345, 67890, ...]

2. Dividir HIDs en batches de 100

3. Por cada batch, obtener contenido
   POST hotel_content_by_ids/ {hids: [12345, ...], language: "en"}
   → [{hid, name, kind, images, rooms, ...}, ...]

4. Upsert en PostgreSQL (ON CONFLICT hid DO UPDATE)

5. Registrar resultado en sync_log
```

Para sync incremental, el paso 1 incluye `updated_since` con la fecha del último sync exitoso.

---

## Decisiones de diseño

### Por qué Go
- Consistente con los microservicios existentes (converter-go, crypto-go)
- Valida la arquitectura políglota del proyecto
- stdlib HTTP es suficiente para un cliente API con Basic Auth
- Única dependencia externa: pgx/v5 (PostgreSQL)

### Por qué JSONB para datos ricos
Los campos como `images`, `room_groups`, `amenity_groups` y `metapolicy` tienen estructuras profundamente anidadas (3-4 niveles). Normalizarlos requeriría 5-6 tablas adicionales sin beneficio de consulta — estos datos se almacenan y se devuelven tal cual para renderizado en el frontend.

### Por qué `hid` como PK
El ID numérico de RateHawk (`hid`) es estable, único y referenciado en todas las llamadas API. Usar un ID surrogado (SERIAL) añadiría un mapeo innecesario.

### Rate limiting con Sleep
El mismo enfoque que usa crypto-go con CoinGecko: `time.Sleep` entre requests. Es simple, predecible y suficiente para un job de ingesta batch que no requiere concurrencia.

---

## Limitaciones del sandbox

- Solo 4 países disponibles (IDs: 59, 153, 189, 201)
- Idioma limitado a inglés (`en`)
- Ideal para desarrollo y validación de la integración
- El paso a producción requiere certificación con RateHawk

---

## Referencias

- [Content API Docs](https://docs.emergingtravel.com/docs/content-api/retrieve-hotel-ids-by-filter/)
- [Filter Values](https://docs.emergingtravel.com/docs/content-api/retrieve-filter-values/)
- [Hotel Content by IDs](https://docs.emergingtravel.com/docs/content-api/retrieve-hotels-content-by-ids/)
- Plan de implementación: ver `docs/future-features.md` (Phase 10)
