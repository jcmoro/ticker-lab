# Runbook

## Local Development

### First-time setup

```bash
make setup
```

This copies `.env.example` to `.env`, builds Docker images, and installs dependencies.

### Start development

```bash
make dev
```

Opens: http://localhost:3000 (dashboard) and http://localhost:3000/health (healthcheck).

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
make format    # Biome formatter
make typecheck # TypeScript
make test      # All tests
```

## Database

```bash
make db-migrate  # Run pending migrations
make db-seed     # Seed development data
```

### Connecting to Postgres directly

```bash
docker compose exec db psql -U ticker -d ticker_lab
```

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
