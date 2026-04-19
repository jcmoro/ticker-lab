import { describe, expect, it } from 'vitest';
import { buildServer } from '../server.js';

describe('GET /health', () => {
  it('returns 200 with ok status', async () => {
    const server = await buildServer();

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
