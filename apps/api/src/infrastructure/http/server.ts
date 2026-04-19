import path from 'node:path';
import { fileURLToPath } from 'node:url';
import view from '@fastify/view';
import { Eta } from 'eta';
import Fastify from 'fastify';
import { dashboardRoutes } from './routes/dashboard.js';
import { healthRoutes } from './routes/health.js';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const viewsDir = path.join(__dirname, '..', '..', 'views');

export async function buildServer() {
  const server = Fastify({
    logger:
      process.env.NODE_ENV === 'development' ? { transport: { target: 'pino-pretty' } } : true,
  });

  const eta = new Eta({ views: viewsDir, cache: process.env.NODE_ENV === 'production' });

  await server.register(view, {
    engine: { eta },
    root: viewsDir,
  });

  await server.register(healthRoutes);
  await server.register(dashboardRoutes);

  return server;
}
