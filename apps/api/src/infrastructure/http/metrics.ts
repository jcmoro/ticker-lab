import type { FastifyInstance } from 'fastify';

export interface RequestMetrics {
  total: number;
  byRoute: Record<string, number>;
}

export function createMetrics(): RequestMetrics {
  return { total: 0, byRoute: {} };
}

export function registerMetricsHook(server: FastifyInstance, metrics: RequestMetrics) {
  server.addHook('onRequest', async (request) => {
    metrics.total++;
    const key = `${request.method} ${request.routeOptions.url ?? request.url}`;
    metrics.byRoute[key] = (metrics.byRoute[key] ?? 0) + 1;
  });
}

export function metricsRoute(metrics: RequestMetrics) {
  return async (server: FastifyInstance): Promise<void> => {
    server.get('/metrics', async () => {
      return {
        uptime: process.uptime(),
        requests: { total: metrics.total, byRoute: metrics.byRoute },
        timestamp: new Date().toISOString(),
      };
    });
  };
}
