import { describe, expect, it, vi } from 'vitest';
import { buildServer } from '../server.js';

const stubDeps = {
  getLatestRates: { execute: vi.fn().mockResolvedValue([]) },
  getRatesByDate: { execute: vi.fn().mockResolvedValue([]) },
};

describe('API Docs', () => {
  describe('GET /api/openapi.yaml', () => {
    it('returns the OpenAPI spec as YAML', async () => {
      const server = await buildServer(stubDeps);

      const response = await server.inject({
        method: 'GET',
        url: '/api/openapi.yaml',
      });

      expect(response.statusCode).toBe(200);
      expect(response.headers['content-type']).toContain('text/yaml');
      expect(response.body).toContain('openapi: 3.1.0');
      expect(response.body).toContain('Ticker Lab API');

      await server.close();
    });
  });

  describe('GET /api/docs', () => {
    it('returns HTML page with ReDoc', async () => {
      const server = await buildServer(stubDeps);

      const response = await server.inject({
        method: 'GET',
        url: '/api/docs',
      });

      expect(response.statusCode).toBe(200);
      expect(response.headers['content-type']).toContain('text/html');
      expect(response.body).toContain('redoc.standalone.js');
      expect(response.body).toContain('/api/openapi.yaml');

      await server.close();
    });
  });
});
