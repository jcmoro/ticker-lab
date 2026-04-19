import { describe, expect, it, vi } from 'vitest';
import { buildServer } from '../server.js';

const stubDeps = {
  getLatestRates: { execute: vi.fn().mockResolvedValue([]) },
  getRatesByDate: { execute: vi.fn().mockResolvedValue([]) },
  getRateHistory: { execute: vi.fn().mockResolvedValue([]) },
  convertCurrency: { execute: vi.fn().mockResolvedValue(null) },
};

describe('GET /health', () => {
  it('returns 200 with ok status', async () => {
    const server = await buildServer(stubDeps);

    const response = await server.inject({
      method: 'GET',
      url: '/health',
    });

    expect(response.statusCode).toBe(200);

    const body = JSON.parse(response.body) as { status: string; timestamp: string };
    expect(body.status).toBe('ok');
    expect(body.timestamp).toBeDefined();

    await server.close();
  });
});

describe('GET /ready', () => {
  it('returns 503 when no db is provided', async () => {
    const server = await buildServer(stubDeps);

    const response = await server.inject({
      method: 'GET',
      url: '/ready',
    });

    expect(response.statusCode).toBe(503);

    const body = JSON.parse(response.body) as { status: string; checks: { database: string } };
    expect(body.status).toBe('not_ready');
    expect(body.checks.database).toBe('error');

    await server.close();
  });
});
