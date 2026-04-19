import path from 'node:path';
import { fileURLToPath } from 'node:url';
import view from '@fastify/view';
import { Eta } from 'eta';
import Fastify from 'fastify';
import type { Sql } from 'postgres';
import type { ExchangeRate } from '../../domain/exchange-rate/ExchangeRate.js';
import { errorHandler } from './error-handler.js';
import { apiDocsRoutes } from './routes/api-docs.js';
import { dashboardRoutes } from './routes/dashboard.js';
import { exchangeRateRoutes } from './routes/exchange-rates.js';
import { healthRoutes } from './routes/health.js';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const viewsDir = path.join(__dirname, '..', '..', 'views');

export interface ServerDependencies {
  getLatestRates: { execute(baseCurrency: string): Promise<ExchangeRate[]> };
  getRatesByDate: { execute(baseCurrency: string, date: string): Promise<ExchangeRate[]> };
  db?: Sql;
}

export async function buildServer(deps: ServerDependencies) {
  const server = Fastify({
    logger:
      process.env.NODE_ENV === 'development' ? { transport: { target: 'pino-pretty' } } : true,
  });

  server.setErrorHandler(errorHandler);

  const eta = new Eta({ views: viewsDir, cache: process.env.NODE_ENV === 'production' });

  await server.register(view, {
    engine: { eta },
    root: viewsDir,
  });

  await server.register(healthRoutes(deps.db));
  await server.register(apiDocsRoutes);
  await server.register(exchangeRateRoutes(deps));
  await server.register(dashboardRoutes(deps.getLatestRates));

  return server;
}
