import type { FastifyInstance } from 'fastify';
import type { ExchangeRate } from '../../../domain/exchange-rate/ExchangeRate.js';

interface ExchangeRatesDeps {
  getLatestRates: { execute(baseCurrency: string): Promise<ExchangeRate[]> };
  getRatesByDate: { execute(baseCurrency: string, date: string): Promise<ExchangeRate[]> };
}

function formatResponse(
  baseCurrency: string,
  rates: { date: string; quoteCurrency: string; rate: number }[],
) {
  const date = rates[0]?.date ?? '';
  return {
    base: baseCurrency,
    date,
    rates: rates.map((r) => ({ currency: r.quoteCurrency, rate: r.rate })),
  };
}

export function exchangeRateRoutes(deps: ExchangeRatesDeps) {
  return async (server: FastifyInstance): Promise<void> => {
    server.get('/api/v1/exchange-rates/latest', async (request) => {
      const { base = 'EUR' } = request.query as { base?: string };
      const rates = await deps.getLatestRates.execute(base);

      if (rates.length === 0) {
        return {
          base,
          date: '',
          rates: [],
        };
      }

      return formatResponse(base, rates);
    });

    server.get<{ Params: { date: string } }>('/api/v1/exchange-rates/:date', async (request) => {
      const { date } = request.params;
      const { base = 'EUR' } = request.query as { base?: string };
      const rates = await deps.getRatesByDate.execute(base, date);

      if (rates.length === 0) {
        return {
          base,
          date,
          rates: [],
        };
      }

      return formatResponse(base, rates);
    });
  };
}
