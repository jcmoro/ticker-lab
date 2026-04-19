import type { FastifyInstance } from 'fastify';
import type { ExchangeRate } from '../../../domain/exchange-rate/ExchangeRate.js';
import type { HistoryPoint } from '../../../domain/exchange-rate/ExchangeRateRepository.js';

interface ExchangeRatesDeps {
  getLatestRates: { execute(baseCurrency: string): Promise<ExchangeRate[]> };
  getRatesByDate: { execute(baseCurrency: string, date: string): Promise<ExchangeRate[]> };
  getRateHistory: {
    execute(
      baseCurrency: string,
      quoteCurrency: string,
      from: string,
      to: string,
    ): Promise<HistoryPoint[]>;
  };
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

function defaultFrom(): string {
  const d = new Date();
  d.setDate(d.getDate() - 30);
  return d.toISOString().slice(0, 10);
}

function today(): string {
  return new Date().toISOString().slice(0, 10);
}

export function exchangeRateRoutes(deps: ExchangeRatesDeps) {
  return async (server: FastifyInstance): Promise<void> => {
    server.get('/api/v1/exchange-rates/latest', async (request) => {
      const { base = 'EUR' } = request.query as { base?: string };
      const rates = await deps.getLatestRates.execute(base);

      if (rates.length === 0) {
        return { base, date: '', rates: [] };
      }

      return formatResponse(base, rates);
    });

    server.get('/api/v1/exchange-rates/history', async (request) => {
      const {
        base = 'EUR',
        quote,
        from = defaultFrom(),
        to = today(),
      } = request.query as { base?: string; quote?: string; from?: string; to?: string };

      if (!quote) {
        return { base, quote: '', from, to, rates: [] };
      }

      const rates = await deps.getRateHistory.execute(base, quote.toUpperCase(), from, to);

      return { base, quote: quote.toUpperCase(), from, to, rates };
    });

    server.get<{ Params: { date: string } }>('/api/v1/exchange-rates/:date', async (request) => {
      const { date } = request.params;
      const { base = 'EUR' } = request.query as { base?: string };
      const rates = await deps.getRatesByDate.execute(base, date);

      if (rates.length === 0) {
        return { base, date, rates: [] };
      }

      return formatResponse(base, rates);
    });
  };
}
