import { describe, expect, it, vi } from 'vitest';
import type { ExchangeRate } from '../../../domain/exchange-rate/ExchangeRate.js';
import { buildServer } from '../server.js';

const sampleRates: ExchangeRate[] = [
  { baseCurrency: 'EUR', quoteCurrency: 'GBP', rate: 0.8561, date: '2026-04-17' },
  { baseCurrency: 'EUR', quoteCurrency: 'USD', rate: 1.1358, date: '2026-04-17' },
];

const stubDeps = {
  getLatestRates: { execute: vi.fn().mockResolvedValue(sampleRates) },
  getRatesByDate: { execute: vi.fn().mockResolvedValue(sampleRates) },
};

describe('Exchange Rates API', () => {
  describe('GET /api/v1/exchange-rates/latest', () => {
    it('returns latest rates with default base EUR', async () => {
      const server = await buildServer(stubDeps);

      const response = await server.inject({
        method: 'GET',
        url: '/api/v1/exchange-rates/latest',
      });

      expect(response.statusCode).toBe(200);

      const body = JSON.parse(response.body) as { base: string; date: string; rates: unknown[] };
      expect(body.base).toBe('EUR');
      expect(body.date).toBe('2026-04-17');
      expect(body.rates).toHaveLength(2);

      await server.close();
    });

    it('passes base query parameter to use case', async () => {
      const server = await buildServer(stubDeps);

      await server.inject({
        method: 'GET',
        url: '/api/v1/exchange-rates/latest?base=USD',
      });

      expect(stubDeps.getLatestRates.execute).toHaveBeenCalledWith('USD');

      await server.close();
    });

    it('returns empty rates when no data exists', async () => {
      const deps = {
        getLatestRates: { execute: vi.fn().mockResolvedValue([]) },
        getRatesByDate: { execute: vi.fn().mockResolvedValue([]) },
      };
      const server = await buildServer(deps);

      const response = await server.inject({
        method: 'GET',
        url: '/api/v1/exchange-rates/latest',
      });

      const body = JSON.parse(response.body) as { rates: unknown[] };
      expect(body.rates).toHaveLength(0);

      await server.close();
    });
  });

  describe('GET /api/v1/exchange-rates/:date', () => {
    it('returns rates for a specific date', async () => {
      const server = await buildServer(stubDeps);

      const response = await server.inject({
        method: 'GET',
        url: '/api/v1/exchange-rates/2026-04-17',
      });

      expect(response.statusCode).toBe(200);
      expect(stubDeps.getRatesByDate.execute).toHaveBeenCalledWith('EUR', '2026-04-17');

      await server.close();
    });
  });
});
