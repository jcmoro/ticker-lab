# Future Data Providers

Providers to integrate after the Frankfurter MVP is complete.

## Priority 1 — Investment Funds & Pension Plans

The original goal of Ticker Lab. Requires research into available public APIs:

- **CNMV (Spain)** — Comisión Nacional del Mercado de Valores. Publishes fund data but API access unclear.
- **Morningstar** — Has data on Spanish funds/plans but requires paid API.
- **Inverco** — Spanish fund/pension industry association. Publishes statistics, scraping may be needed.
- **Bolsas y Mercados Españoles (BME)** — May have fund NAV data.

### Challenges
- No single free, reliable API for Spanish investment funds
- May require scraping or combining multiple sources
- NAV (Net Asset Value) updates are typically daily, fits our daily ingestion model

## Priority 2 — Crypto

- **CoinGecko** (free tier) — prices, market cap, historical data. No API key for basic usage.

## Priority 3 — Macro Indicators

- **FRED** (Federal Reserve Economic Data) — free API key, US macro data (GDP, CPI, unemployment, interest rates)
- **ECB Statistical Data Warehouse** — Eurozone macro data
- **Banco de España** — Spanish economic indicators
- **World Bank Open Data** — global indicators

## Priority 4 — Equities & Indices

- **Yahoo Finance** (unofficial) — delayed stock/index data, fragile API
- **Alpha Vantage** (free tier) — 25 requests/day, 500/month
- **Twelve Data** (free tier) — 800 API credits/day

## Priority 5 — Commodities

- Available through FRED, Alpha Vantage, or dedicated commodity APIs
- Gold, oil, natural gas, agricultural commodities
