# ADR-003: Hosting Strategy — Hetzner Self-Hosted Multi-Project

**Status:** Proposed
**Date:** 2026-05-10
**Supersedes:** Section of ADR-001 referencing Fly.io (already migrated to Render + Neon as part of Phase 6)

---

## Context

Three forces are pushing us to re-evaluate the current Render + Neon free-tier setup:

1. **Render free-tier limits at scale.** Phase 12 adds two new Go microservices (`esios-go`, `cnmv-go`) on top of the existing API + 3 Go services + Postgres. Each new service compounds cold-start latency and consumes a slot in the free plan.
2. **Multi-project ambition.** Beyond Ticker Lab we want to host:
   - Future Dockerized side projects (same stack pattern).
   - A **WordPress + WooCommerce** site (PHP + MariaDB), unrelated to Ticker Lab but needing the same infra.
3. **Budget shift.** ADR-001 fixed budget at **0 €/month**. Hosting all three workloads with always-on availability requires accepting a small recurring cost. We are willing to spend up to ~**€15/month** total if it removes cold starts and consolidates everything under one operational model.

Current production state (verified 2026-05-10):
- Render: `tickerlab` (Node API), plus manually-created services for crypto-go, converter-go, macro-go.
- Neon: free Postgres, Frankfurt EU. Connection string injected as `DATABASE_URL`.
- Cron: GitHub Actions runs ingestion against Neon directly.
- Cold starts on Render free tier (~30s after 15 min idle) addressed for crypto via client-side wake-up ping; same workaround would scale poorly to 5+ services.

---

## Decision

**Proposed:** migrate to a single **Hetzner Cloud VPS** in Falkenstein (DE) hosting all current and future workloads via Docker Compose, behind a self-managed reverse proxy with automatic TLS.

**Initial sizing:** **CX33** (4 vCPU shared, 8 GB RAM, 80 GB NVMe — €6.49/month) or **CPX32** (4 AMD vCPU, 8 GB RAM, 160 GB NVMe — €7.99/month). Headroom for 7+ Ticker Lab containers + WordPress + MariaDB + reverse proxy. Scale up to CCX13 (€15.99/month, dedicated vCPU) if WordPress traffic creates CPU contention.

**Stack on the VPS:**
- **Reverse proxy + TLS:** Traefik with Let's Encrypt automation, OR **Dokploy** (open-source self-hosted PaaS, ~0.8% idle CPU) for a managed UI on top of the same primitives.
- **Postgres:** self-hosted in Docker (replaces Neon) — **confirmed**. Single Postgres 16 container shared by all Ticker Lab services; separate MariaDB container for WordPress. No connection pooler (PgBouncer) at current scale; revisit if connection count grows. Backups via `pg_dump` cron to a Hetzner Storage Box (€3.81/month for 1 TB).
- **MariaDB:** separate Docker service for WordPress, isolated from Postgres.
- **CI/CD:** GitHub Actions SSH-deploys (`docker compose pull && up -d`) on push to `main`, replacing Render's auto-deploy webhook.
- **Backups:** Hetzner automatic backups (+20% of VPS price = ~€1.30/mo on CX33), 7 daily snapshots retained.
- **Domain:** purchase a `.com`/`.dev`/`.es` (~€10/year) — required for HTTPS (Let's Encrypt does not issue for `*.onrender.com`-style providers' subdomains anymore in this scenario, and Hetzner doesn't provide vanity subdomains).

**Estimated total monthly cost:** **€10–12** (VPS + backups + domain amortized + Storage Box) for everything: Ticker Lab + WordPress/Woo + 1–2 future projects.

This is a **proposal**, not yet adopted. The decision should be confirmed once Phase 12 services are spec'd to production-ready (current state) and a clear migration window is identified.

---

## Alternatives Considered

| Option | Monthly cost | Always-on | Multi-project | WordPress fit | Verdict |
|--------|-------------:|:---------:|:-------------:|:-------------:|---------|
| **Status quo: Render free + Neon free** | €0 | ✗ (cold starts) | ✗ (per-service free slots) | ✗ (no PHP runtime) | **Rejected for scale.** Works for current Ticker Lab; breaks down with WP and 5+ services |
| **Render Starter ($7/svc/mo) + Neon Launch ($19/mo)** | ~€60+ | ✓ | Per-service billing | ✗ (no PHP) | Rejected — cost escalates linearly per service |
| **Hetzner Cloud CX23 (€3.99)** | ~€8 with backups | ✓ | Tight | ✓ | Marginal — 4 GB RAM tight with 7 containers + WP |
| **Hetzner Cloud CX33 / CPX32** (recommended) | **~€10–12** | ✓ | ✓ | ✓ | **Proposed.** Best fit for sizing and budget |
| **Hetzner Cloud CCX13 (dedicated vCPU)** | ~€18 | ✓ | ✓ | ✓ | Future jump if WP traffic is real |
| **Hetzner Dedicated AX42** | ~€60 + €39 setup | ✓ | ✓ | ✓ | Overkill for personal scale |
| **Vercel + PlanetScale / Supabase** | Pay-per-use | ✓ | Per-deploy | ✗ (no PHP) | Rejected — vendor lock-in, no WP path |
| **Cloudflare Workers + D1** | Pay-per-use | ✓ | ✓ | ✗ (no PHP, no Postgres) | Rejected — incompatible with current stack |
| **DigitalOcean Droplet** | $6 (~€5.50) for 1 GB / $12 for 2 GB | ✓ | ✓ | ✓ | Comparable to Hetzner but pricier and 1 TB egress vs 20 TB |
| **OVH VPS** | €4–8 | ✓ | ✓ | ✓ | Comparable; less predictable network and uglier control panel |
| **AWS EC2 t4g.small** | ~€12 + transfer | ✓ | ✓ | ✓ | Rejected — egress fees, complex pricing, no managed simplicity gain at this size |

### Why Hetzner specifically

- **20 TB included EU traffic** per server vs Render/Neon metered egress. Effectively free network at our scale (<50 GB/month realistic).
- **Falkenstein** datacenter ~35 ms from Spain — better latency than Render's EU regions in practice.
- **Predictable flat pricing.** No per-request, per-build, or per-egress surprises.
- **Simple migration path.** Project is already 100% Dockerized — `docker-compose.yml` runs unchanged on the VPS.
- **Self-hosted Postgres works** at our row counts (CNMV backfill ~5M rows fits comfortably in 80 GB NVMe).

### Why NOT Hetzner

- **No managed Postgres.** Lose Neon's branching, point-in-time recovery, automatic patching. Mitigated by `pg_dump` to Storage Box, but it's a downgrade in DR ergonomics.
- **No managed services in general.** We become responsible for OS patches, kernel updates, fail2ban, firewall rules.
- **Single point of failure.** One VPS = one outage blast radius. Render has implicit per-service isolation.
- **Manual KYC** on new accounts can take 24–48 h — must plan migration window accordingly.
- **April 2026 price hike** (+30–37% on cloud line) shows pricing isn't immutable.
- **Primary IPv4 surcharge** (€0.50/mo) added in 2024. IPv6-only saves it but breaks WP plugin updates from legacy registrars.

---

## Consequences

### Positive

- **Always-on availability** — eliminates cold-start workarounds and wake-up pings.
- **Single operational surface** — one `docker-compose.yml` on one VPS hosts Ticker Lab + WordPress/Woo + future projects. No per-project dashboard navigation.
- **Predictable cost** at ~€10–12/month regardless of traffic spikes (within 20 TB egress).
- **No vendor egress fees** — generous Hetzner traffic policy.
- **Better latency** for Spanish users (Falkenstein vs Render's regions).
- **Multi-tenant pattern reusable** — same VPS pattern serves any future Dockerized project.
- **WP+Woo natively supported** — can't run on Render/Vercel without major rework.
- **Database under our control** — full SQL access, no Neon connection-pool quirks.

### Negative

- **Budget shift** from €0 to ~€10–12/month. Reverses ADR-001's hard 0 € constraint.
- **Operational ownership** — Linux administration, OS updates, security patching, log rotation, fail2ban, SSH hardening become our responsibility.
- **DR is weaker than Neon** — restoring from `pg_dump` is slower than Neon branching.
- **Single point of failure** — VPS outage takes down everything simultaneously. Mitigation: Hetzner backup snapshots + uptime monitoring.
- **Migration effort** — estimated ~1 day:
  - Provision VPS + DNS A/AAAA records.
  - Add Traefik (or Dokploy) reverse-proxy compose service.
  - Migrate secrets from Render env-vars panel to `.env` or Dokploy secret store.
  - GitHub Actions SSH deploy workflow (replaces Render auto-deploy webhook).
  - `pg_dump` from Neon → restore into local Postgres container; update `DATABASE_URL`.
  - Cutover DNS, decommission Render services and Neon DB.
- **No managed certificates** — Let's Encrypt handles 90% of cases automatically via Traefik, but we own renewal monitoring.
- **WordPress hardening** is on us — security plugins, admin URL change, fail2ban, regular updates.

---

## Resolved Decisions

- **2026-05-10 — Postgres self-hosted in Docker (not Neon hybrid).** Rationale: keeps DB local to compute (zero added latency), eliminates a second vendor, gives full SQL control, and Neon's branching/PITR ergonomics aren't load-bearing at this scale. Implications captured in the new "Data Migration" section below.

## Data Migration — Neon → self-hosted Postgres

Concrete steps for cutover (~30 minutes downtime):

1. **Pre-cutover** — provision VPS, bring up Postgres 16 container with persistent volume, verify network reachability.
2. **Freeze writes** — pause GitHub Actions cron jobs (ingest, backfill) to stop new data landing in Neon.
3. **Dump** —
   ```bash
   pg_dump --format=custom --no-owner --no-acl \
     "$NEON_DATABASE_URL" > ticker_lab.dump
   ```
4. **Restore** —
   ```bash
   pg_restore --no-owner --no-acl \
     -d "$VPS_DATABASE_URL" ticker_lab.dump
   ```
5. **Verify** — row counts per table match Neon source; spot-check `exchange_rates`, `crypto_prices`, `macro_observations`.
6. **Cutover** — update `DATABASE_URL` in all services' env (Render → Hetzner stack), redeploy, re-enable cron jobs.
7. **Decommission** — keep Neon free tier alive for 30 days as fallback; delete after.

**Schema management on first deploy:** each Go service runs `Migrate()` on startup. To avoid races, deploy in this order: Postgres container → Node API (creates `exchange_rates`, etc.) → Go services (each creates its own tables). Drizzle migrations on the API side are run via `make db-migrate` before service start.

**Backup cadence (recommended):**
- Hetzner automatic snapshots: 7 daily (built-in at +20% VPS price).
- `pg_dump` cron: nightly to Storage Box, retain 30 days.
- Quarterly DR drill: restore most recent dump into a temporary container and run smoke tests.

## Open Questions

1. **PaaS layer (Dokploy/Coolify) or plain Docker Compose + Traefik?** Dokploy gives a UI and PR previews at ~0.8% idle cost. Plain Compose is leaner and matches the "prefer simplicity" preference. Decision: try plain Compose first; adopt Dokploy if managing multiple projects becomes painful.
2. **Backup target?** Hetzner automatic backups (whole-disk, +20%) vs Storage Box (file-level, €3.81/mo). Likely both: automatic for fast rollback, Storage Box for long-term `pg_dump` retention.
3. **Domain name?** Required for TLS. Pick `.es` for Spanish positioning or generic `.dev`/`.com`. Cost ~€10/year.
4. **Migration window?** Decide whether to migrate before, during, or after Phase 12 implementation. Migrating first reduces operational drift but delays new features.
5. **WordPress site scope?** Confirm traffic estimate (<100 orders/day assumed) — if real traffic is higher, jump straight to CCX13 dedicated vCPU.

---

## References

- [Hetzner Cloud — CX line](https://www.hetzner.com/cloud/cost-optimized)
- [Hetzner Cloud — CPX/CCX lines](https://www.hetzner.com/cloud/regular-performance)
- [Hetzner Storage Box](https://www.hetzner.com/storage/storage-box/)
- [Hetzner April 2026 price adjustment](https://docs.hetzner.com/general/infrastructure-and-availability/price-adjustment/)
- [Dokploy](https://dokploy.com/) — open-source self-hosted PaaS option
- [Coolify](https://coolify.io/) — alternative self-hosted PaaS
- [Ubicloud managed Postgres on Hetzner](https://www.ubicloud.com/blog/open-and-portable-managed-postgresql-avail-hetzner) — if managed DB is later needed
- Current state: [`docs/runbook.md` Production section](../runbook.md#production-render--neon)
- Related: [ADR-001 Technology Stack](./001-tech-stack.md)
