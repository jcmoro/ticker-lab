# Runbook

## Local Development

### First-time setup

```bash
make setup
```

Copies `.env.example` to `.env`, builds Docker images, and installs dependencies.

### Start development

```bash
make dev
```

- Dashboard: http://localhost:3000
- Health check: http://localhost:3000/health
- Exchange rates API: http://localhost:3000/api/v1/exchange-rates/latest

### Stop

```bash
make down
```

### Full cleanup

```bash
make clean
```

Removes containers, volumes, and `node_modules`.

## Quality Checks

```bash
make ci        # Runs lint + typecheck + test
make lint      # Biome linter
make format    # Biome formatter (auto-fix)
make typecheck # TypeScript
make test      # All tests (18 tests across domain, application, HTTP)
```

## Database

### Run migrations

```bash
make db-migrate
```

### Generate a new migration after editing schema.ts

```bash
docker compose run --rm api pnpm --filter @ticker-lab/api db:generate
```

### Seed development data

```bash
make db-seed
```

### Connect to Postgres directly

```bash
docker compose exec db psql -U ticker -d ticker_lab
```

### Useful queries

```sql
-- Count ingested rates
SELECT COUNT(*) FROM exchange_rates;

-- Latest rates for EUR
SELECT quote_currency, rate, date
FROM exchange_rates
WHERE base_currency = 'EUR'
ORDER BY date DESC, quote_currency
LIMIT 30;

-- Distinct dates ingested
SELECT DISTINCT date FROM exchange_rates ORDER BY date DESC;
```

## Data Ingestion

### Manual ingestion

```bash
make job-ingest
```

Fetches latest ECB exchange rates from Frankfurter API for EUR and saves to the database. Safe to run multiple times — uses upsert (ON CONFLICT UPDATE).

### Verify ingestion worked

```bash
curl http://localhost:3000/api/v1/exchange-rates/latest | jq
```

Should return 29 currency rates with the latest ECB business day date.

## Troubleshooting

### Containers won't start

```bash
make clean && make setup
```

### Database connection errors

Check that the `db` container is healthy:

```bash
docker compose ps
```

### Port 3000 already in use

```bash
lsof -i :3000
# Kill the process, or change API_PORT in .env
```

### Ingestion returns 0 rates

- Frankfurter API may be down — check https://api.frankfurter.dev/v1/latest
- Weekends/holidays: ECB doesn't publish rates on non-business days. The API returns the last available date.

### Dashboard shows "No rates available yet"

Run `make job-ingest` to populate the database with exchange rate data.

### Migration fails

If the database schema is out of sync:

```bash
# Check current migration state
docker compose exec db psql -U ticker -d ticker_lab -c "SELECT * FROM drizzle.__drizzle_migrations;"

# Nuclear option: reset everything (destroys data)
make clean && make setup && make db-migrate && make job-ingest
```
