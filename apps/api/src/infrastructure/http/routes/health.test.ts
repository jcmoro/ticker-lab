import { describe, expect, it, vi } from 'vitest';
import { buildServer } from '../server.js';

const stubDeps = {
  getLatestRates: { execute: vi.fn().mockResolvedValue([]) },
  getRatesByDate: { execute: vi.fn().mockResolvedValue([]) },
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
