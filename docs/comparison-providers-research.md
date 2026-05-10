# Comparison Providers — API Research

**Date:** 2026-05-10
**Scope:** Public APIs for insurance comparison, investment product comparison, and utility (electricity / gas / internet) comparison data, focused on Spain and the EU.
**Purpose:** Identify viable free/freemium data sources to expand Ticker Lab beyond exchange rates, crypto, and macro indicators.

Verdicts:
- **Promising** — usable today, free or free-tier, public API or stable file feed.
- **Possible** — usable with caveats (PDF parsing, file scraping with session auth, fragile undocumented endpoints).
- **Not viable** — paid, TOS-blocked, or legally risky.

---

## 1. Insurance

### Bottom line
No public API for **consumer-facing premium quotes** in Spain. Real per-product pricing will only become viable when **FIDA** (Financial Data Access Regulation, the open-insurance equivalent of PSD2) is enforced — expected adoption mid-2026, implementation late 2027.

What is ingestible today: **aggregate sector statistics** from EU/Spain regulators.

### Regulators & industry bodies

| Provider | URL | Auth | Data | Update | Verdict |
|---|---|---|---|---|---|
| **EIOPA Insurance Statistics** | `eiopa.europa.eu/tools-and-data/insurance-statistics` | None | EU/EEA aggregated: balance sheet, premiums, claims, SCR, asset exposures by country | Quarterly + annual | **Promising** |
| **EIOPA via data.europa.eu** | `data.europa.eu/data/datasets/eiopa-insurance-statistics-group-annual` | None | Same data, machine-readable. SPARQL endpoint + REST Registry API | Synced w/ EIOPA | **Promising** (best entry point) |
| **EIOPA Catastrophe Data Hub** | `eiopa.europa.eu/tools-and-data/catastrophe-data-hub` | None | Historical EU catastrophe losses | Periodic | Possible (niche) |
| **DGSFP Registros Públicos** | `rrpp.dgsfp.mineco.es` | None | Registry of Spanish insurers & mediators (no pricing). Excel/PDF | Continuous | Possible (catalog only) |
| **DGSFP "Servicios"** | `sededgsfp.gob.es` | Cert / contract | Mediators, pension funds, solvency data | — | Not viable (B2G integration) |
| **UNESPA Publications** | `unespa.es/que-hacemos/publicaciones` | None | Sector-wide premium volume, claims aggregates | Quarterly + annual | Possible (PDFs only) |
| **Consorcio de Compensación de Seguros** | `consorseguros.es` | None | Extraordinary risks, agricultural insurance, FIVA (insured vehicles) | Annual | Possible (PDF/XLSX) |
| **datos.gob.es (tag: seguros)** | `datos.gob.es/en/catalogo?tags=seguros` | API key optional | CKAN-style discovery layer | Varies | Promising as discovery |

### Comparison sites
| Provider | Verdict |
|---|---|
| Rastreator, Acierto, Comparaiso | **Not viable** — broker model, no public API, TOS forbids scraping, GDPR exposure on personalized quotes |
| Insurify, Policygenius (US ref) | **Not viable** — partnership-only |
| Herald (heraldai.com) | Aggregator of B2B insurance APIs — **paid** |

### Watchlist
- **FIDA Regulation** — trilogue began Apr 2025, expected adoption mid-2026, implementation late 2027 with 24-month compliance window. Phase 2 explicitly covers motor insurance + investment products via standardized APIs (EBA/EIOPA spec).
- Track `eiopa.europa.eu/browse/digitalisation-and-financial-innovation/open-insurance_en` for sandbox/pilot announcements.

---

## 2. Investment Products

### Bottom line
Multiple free official sources cover Spanish funds, pension plans, EU ETFs, and Spanish deposit rates. Comparison of fees and NAV is fully feasible without paid data.

### Funds & pension plans (Spain)

| Provider | URL | Auth | Data | Update | Verdict |
|---|---|---|---|---|---|
| **CNMV Fichero Público** | `cnmv.es/Portal/Consultas/IIC/FondosFichero.aspx`, `…/PlanesPensionesFichero.aspx` | None | NAV diario, TER, comisiones, holdings trimestrales, gestoras | Daily NAV, quarterly holdings | **Promising** |
| **Inverco** | `inverco.es/estadisticas` | None | Aggregated AUM, monthly stats by category, no per-fund NAV | Monthly | Possible (aggregate only) |
| **Morningstar** | — | Paid (enterprise) | NAV, ratings, fees, holdings | — | Not viable (free tier blocked) |
| **Quefondos** | `quefondos.com` | Scraping only | NAV, fees, performance ~3,000 funds | — | Not viable (TOS) |
| **Rankia** | — | None | Editorial only | — | Not viable (no data) |

### ETFs (EU)

| Provider | URL | Auth | Data | Update | Verdict |
|---|---|---|---|---|---|
| **JustETF** | — | Paid B2B | — | — | Not viable |
| **BlackRock iShares** | `ishares.com` undocumented JSON | None | NAV, fees, holdings | Daily | Possible (fragile) |
| **Xetra (Deutsche Börse)** | `xetra.com/.../list-of-tradable-etfs` | None | ISIN, TER, replication, AUM | Daily CSV | **Promising** |
| **Borsa Italiana** | `borsaitaliana.it` | None | ETF/ETC list as Excel | Daily | Possible |
| **Euronext** | `live.euronext.com` | Paid for Connect | Undocumented public JSON | — | Possible |
| **OpenFIGI** | `openfigi.com` | Optional key | ISIN ↔ FIGI ↔ ticker | Static metadata | **Promising** (helper) |

### Robo-advisors / fintechs (Spain)
| Provider | Verdict |
|---|---|
| Indexa Capital | Not viable — client-only OAuth API, not for catalog comparison |
| MyInvestor, Finizens, Inbestme, Openbank Invest | Not viable — TOS-restrictive scraping |

### Deposits / fixed income

| Provider | URL | Auth | Data | Update | Verdict |
|---|---|---|---|---|---|
| **Banco de España SDW** | `bde.es/webbe/es/estadisticas/recursos-fb/ws-bde/` | None | TIPI series, Euribor, deposit/loan rates | Daily/monthly | **Promising** |
| **ECB SDW** | `sdw-wsrest.ecb.europa.eu` | None | EU-wide deposit rates | Daily/monthly | Already integrated |
| HelpMyCash, iAhorro, Bankimia | — | — | — | — | Not viable (TOS) |

### Cross-cutting metadata

| Provider | Auth | Free tier | Verdict |
|---|---|---|---|
| **EOD Historical Data** | API key | 20 req/day free, $19.99/mo full | **Promising** as paid fallback |
| **Financial Modeling Prep** | API key | 250 req/day free, $19/mo | Possible (thinner EU coverage) |
| Yahoo Finance (unofficial) | None | Brittle | Not viable |
| Boerse Frankfurt / Stuttgart | None | Daily CSV | Possible |

---

## 3. Utilities

### Bottom line
Strong free APIs for regulated and wholesale electricity/gas prices. **No legal source** for free-market retail tariffs (commercial offers from Iberdrola, Endesa, Naturgy commercializadoras) or per-plan ISP pricing — all major comparators block scraping in ToS.

### Electricity

| Provider | URL | Auth | Data | Update | Verdict |
|---|---|---|---|---|---|
| **REE ESIOS** | `api.esios.ree.es` | API token (free, email request) | PVPC, demand, generation mix, OMIE spot, CO2 factor — ~2,000 indicators | Hourly | **Promising (top pick)** |
| **ENTSO-E Transparency** | `web-api.tp.entsoe.eu/api` | Free token (1–2 day approval) | Day-ahead prices, generation, load, cross-border flows for 39 EU TSOs | Hourly | **Promising** (XML only, 400 req/min, 200 GB/mo) |
| **OMIE** | `omie.es/es/file-access-list` | None | MIBEL daily/intraday market prices | Daily ~13:00 CET | Possible (CSV files, fallback to ESIOS) |
| **CNMC** | `datos.cnmc.es` | None | Annual market reports, retailer share, regulated tariff history | Quarterly/annual | Possible (reference data) |

**Useful ESIOS endpoints (daily ingest):**
- `GET /indicators/1001` — PVPC peaje 2.0TD (default household tariff, hourly EUR/MWh)
- `GET /indicators/600` — Demand real-time (MW)
- `GET /indicators/1293` — Generation mix by technology (renewables share)
- `GET /indicators/10391` — Spot price OMIE daily market
- `GET /indicators/10211` — CO2 emissions factor
- Query params: `start_date`, `end_date` (ISO8601), `time_trunc=hour|day`, `geo_ids=8741` (peninsula)
- Header: `x-api-key: <token>` + `Accept: application/json; application/vnd.esios-api-v1+json`

### Gas

| Provider | URL | Auth | Data | Update | Verdict |
|---|---|---|---|---|---|
| **MIBGAS** | `mibgas.es/en/file-access` | Free account (session cookie) | Daily PVB spot, intraday, monthly products (EUR/MWh) | Daily ~17:00 CET | **Promising** (medium effort, file scraping) |
| **Enagás** | `enagas.es/es/transparencia/sistema-gasista/` | None | Storage levels, LNG terminals, system flows (not retail prices) | Daily | Possible (supply context) |
| **CNMC TUR** | `cnmc.es/ambitos-de-actuacion/energia/mercado-gas` | None | Quarterly Tarifa de Último Recurso (BOE PDF) | Quarterly | Possible (manual update) |

### Internet / broadband

| Provider | URL | Auth | Data | Update | Verdict |
|---|---|---|---|---|---|
| **CNMC** | `data.cnmc.es` + `cnmc.es/estadistica` | None | Operator market share, subscriber counts, average speeds (aggregate) | Quarterly | Possible (market context) |
| **MINETUR Coverage Map** | `avancedigital.mineco.gob.es/banda-ancha/cobertura/` | None | FTTH/HFC/xDSL/4G/5G coverage by municipality | Annual | Possible (CSV + WMS) |
| Roams, Rastreator, Kelisto | — | — | Per-plan tariffs | — | Not viable (TOS) |
| Ofcom (UK) | `ofcom.org.uk/research-and-data` | None | UK-only broadband data | — | Not viable (out of scope) |
| BEREC | — | — | Annual reports only | Annual | Not viable for pricing |

### Recommended ingest plan

| Provider | Frequency | Endpoint | Effort |
|---|---|---|---|
| ESIOS PVPC (1001) | Daily ~21:00 | `/indicators/1001?time_trunc=hour` | Low |
| ESIOS demand + mix (600, 1293) | Daily | indicator endpoints | Low |
| ENTSO-E day-ahead | Daily ~14:00 | `/api?documentType=A44` | Medium (XML) |
| MIBGAS PVB | Daily ~18:00 | File download w/ session | Medium |
| Enagás storage | Weekly | CSV download | Low |
| CNMC market shares | Quarterly | Manual / scheduled | Low |

---

## Phase 12 candidates (ranked)

1. **ESIOS PVPC ingestion** — reuses macro-go pattern, single API key, immediately useful daily ticker.
2. **CNMV funds NAV** — original Ticker Lab goal, official source, daily fits model. Heavier ZIP/XML parsing.
3. **Banco de España deposit rates** — same pattern as FRED/ECB, low effort, complements macro indicators.
4. **EIOPA insurance aggregates** — quarterly cadence, low ops burden, expands into insurance business line.

Decision pending.

---

## Sources

- EIOPA Insurance Statistics — `eiopa.europa.eu/tools-and-data/insurance-statistics_en`
- EIOPA on data.europa.eu — `data.europa.eu/data/datasets/eiopa-insurance-statistics-group-annual`
- data.europa.eu API documentation — `dataeuropa.gitlab.io/data-provider-manual/api-documentation/`
- DGSFP Registros Públicos — `rrpp.dgsfp.mineco.es`
- UNESPA Publications — `unespa.es/que-hacemos/publicaciones`
- Consorcio de Compensación de Seguros — `consorseguros.es/en/`
- datos.gob.es (seguros) — `datos.gob.es/en/catalogo?tags=seguros`
- EIOPA Open Insurance — `eiopa.europa.eu/browse/digitalisation-and-financial-innovation/open-insurance_en`
- FIDA April 2025 Update (Capco) — `capco.com/intelligence/capco-intelligence/financial-data-access-regulation-april-2025-update`
- CNMV Fichero Público — `cnmv.es/portal/Consultas/FicherosPublicos`
- Inverco Estadísticas — `inverco.es/estadisticas`
- Banco de España SDW — `bde.es/webbe/es/estadisticas/recursos-fb/ws-bde/`
- Xetra ETF list — `xetra.com/xetra-en/instruments/etf-exchange-traded-funds/list-of-tradable-etfs`
- OpenFIGI — `openfigi.com`
- ESIOS API — `esios.ree.es/es/pagina/api`
- ENTSO-E Transparency Platform — `transparency.entsoe.eu`
- OMIE file access — `omie.es/es/file-access-list`
- MIBGAS file access — `mibgas.es/en/file-access`
- Enagás transparencia — `enagas.es/es/transparencia/sistema-gasista/`
- CNMC open data — `data.cnmc.es`
- MINETUR broadband coverage — `avancedigital.mineco.gob.es/banda-ancha/cobertura/`
