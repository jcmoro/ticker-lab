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

## Production (Render + Neon)

### Architecture

- **App:** Render free tier (Node.js native runtime, auto-deploy from GitHub)
- **Database:** Neon free tier (serverless Postgres, Frankfurt EU)
- **Cron:** GitHub Actions (Mon-Fri 16:30 UTC, runs ingestion directly against Neon)

### Deploy

Deployments happen automatically on push to `main`. Render detects the push, builds, and deploys. To trigger manually, use the Render dashboard or:

```bash
make deploy  # requires RENDER_DEPLOY_HOOK env var
```

### Required secrets

**Render environment variables** (set in Render dashboard):
- `DATABASE_URL` — Neon connection string
- `NODE_ENV` = `production`
- `API_PORT` = `3000`
- `API_HOST` = `0.0.0.0`
- `FRANKFURTER_BASE_URL` = `https://api.frankfurter.dev`

**GitHub repository secrets** (Settings → Secrets):
- `DATABASE_URL` — same Neon connection string (used by ingestion cron)
- `RENDER_DEPLOY_HOOK` — (optional) Render deploy hook URL

### Daily ingestion

ECB exchange rates are ingested automatically Mon-Fri at 16:30 UTC via GitHub Actions cron (`.github/workflows/ingest.yml`). To trigger manually:

```bash
# From GitHub Actions
gh workflow run "Daily Ingestion"

# Or locally against production DB
DATABASE_URL="<neon-connection-string>" make prod-ingest
```

### Backfill historical data

```bash
DATABASE_URL="<neon-connection-string>" make prod-backfill
```

### Crypto ingestion

```bash
# Local
make job-crypto

# Production
DATABASE_URL="<neon-connection-string>" make prod-crypto
```

### Connect to production database

```bash
DATABASE_URL="<neon-connection-string>" make prod-db
# or directly:
psql "<neon-connection-string>"
```

### View logs

Render dashboard → tickerlab → Logs. No CLI log access on free tier.

### URLs

- Dashboard: https://tickerlab.onrender.com
- Health: https://tickerlab.onrender.com/health
- Readiness: https://tickerlab.onrender.com/ready
- API docs: https://tickerlab.onrender.com/api/docs
- Metrics: https://tickerlab.onrender.com/metrics
- Crypto: https://tickerlab.onrender.com/crypto
- Crypto API: https://tickerlab-crypto.onrender.com/api/v1/crypto/latest

### Cold starts

Render free tier spins down after 15 minutes of inactivity. First request after that takes ~30 seconds (cold start). This is expected for a personal experiment.

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
