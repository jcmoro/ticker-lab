FROM node:24-alpine AS base
RUN corepack enable
WORKDIR /app

# ─── Development ──────────────────────────────────────────────
# In dev, everything comes from the bind mount (docker-compose volumes).
# Dependencies are installed via `make setup` on the mounted volume.
FROM base AS dev
CMD ["pnpm", "--filter", "@ticker-lab/api", "dev"]

# ─── Dependencies (production) ────────────────────────────────
FROM base AS deps
COPY package.json pnpm-lock.yaml pnpm-workspace.yaml ./
COPY apps/api/package.json ./apps/api/
COPY packages/shared/package.json ./packages/shared/
RUN pnpm install --frozen-lockfile --prod

# ─── Builder ──────────────────────────────────────────────────
FROM base AS builder
COPY package.json pnpm-lock.yaml pnpm-workspace.yaml ./
COPY apps/api/package.json ./apps/api/
COPY packages/shared/package.json ./packages/shared/
RUN pnpm install --frozen-lockfile
COPY . .
RUN pnpm --filter @ticker-lab/api build

# ─── Production ───────────────────────────────────────────────
FROM node:24-alpine AS prod
WORKDIR /app
ENV NODE_ENV=production
COPY --from=builder /app/apps/api/dist ./dist
COPY --from=builder /app/apps/api/src/views ./views
COPY --from=deps /app/node_modules ./node_modules
EXPOSE 3000
USER node
CMD ["node", "dist/main.js"]
