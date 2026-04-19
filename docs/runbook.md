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

## Production (Fly.io)

### First-time setup

```bash
# Install Fly CLI
curl -L https://fly.io/install.sh | sh

# Login
fly auth login

# Create the app
fly launch --no-deploy

# Create Postgres (free tier)
fly postgres create --name tickerlab-db --region mad --vm-size shared-cpu-1x --volume-size 1

# Attach Postgres (sets DATABASE_URL secret automatically)
fly postgres attach tickerlab-db

# Deploy
fly deploy
```

### Deploy

Deployments happen automatically on push to `main` via GitHub Actions. To deploy manually:

```bash
fly deploy
```

The `release_command` in `fly.toml` runs database migrations before the new version goes live.

### Required secrets

```bash
# DATABASE_URL is set automatically by `fly postgres attach`
# No other secrets needed (Frankfurter API has no key)
```

For GitHub Actions deployment, set the `FLY_API_TOKEN` secret in the repository:

```bash
fly tokens create deploy -x 999999h
# Copy the token to GitHub → Settings → Secrets → FLY_API_TOKEN
```

### Daily ingestion

ECB exchange rates are ingested automatically Mon-Fri at 16:30 UTC via GitHub Actions cron (`.github/workflows/ingest.yml`). To trigger manually:

```bash
# From GitHub Actions
gh workflow run "Daily Ingestion"

# Or via Fly SSH
fly ssh console -C "node dist/infrastructure/jobs/ingest.js"
```

### View logs

```bash
fly logs
fly logs --app tickerlab
```

### Connect to production database

```bash
fly postgres connect -a tickerlab-db
```

### Rollback

```bash
# List recent deployments
fly releases

# Rollback to previous release
fly deploy --image <previous-image-ref>
```

### URLs

- Dashboard: https://tickerlab.fly.dev
- Health: https://tickerlab.fly.dev/health
- Readiness: https://tickerlab.fly.dev/ready
- API docs: https://tickerlab.fly.dev/api/docs
- Metrics: https://tickerlab.fly.dev/metrics

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
