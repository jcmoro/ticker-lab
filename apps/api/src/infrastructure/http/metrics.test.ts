import { describe, expect, it, vi } from 'vitest';
import { buildServer } from './server.js';

const stubDeps = {
  getLatestRates: { execute: vi.fn().mockResolvedValue([]) },
  getRatesByDate: { execute: vi.fn().mockResolvedValue([]) },
  getRateHistory: { execute: vi.fn().mockResolvedValue([]) },
};

describe('GET /metrics', () => {
  it('returns uptime and request counts', async () => {
    const server = await buildServer(stubDeps);

    // Make a request first to generate metrics
    await server.inject({ method: 'GET', url: '/health' });

    const response = await server.inject({
      method: 'GET',
      url: '/metrics',
    });

    expect(response.statusCode).toBe(200);

    const body = JSON.parse(response.body) as {
      uptime: number;
      requests: { total: number; byRoute: Record<string, number> };
      timestamp: string;
    };

    expect(body.uptime).toBeGreaterThan(0);
    // /health + /metrics itself = at least 2 (onRequest fires before handler)
    expect(body.requests.total).toBeGreaterThanOrEqual(2);
    expect(body.requests.byRoute['GET /health']).toBe(1);
    expect(body.requests.byRoute['GET /metrics']).toBe(1);
    expect(body.timestamp).toBeDefined();

    await server.close();
  });
});
