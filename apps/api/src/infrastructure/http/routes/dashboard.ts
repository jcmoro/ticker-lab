import type { FastifyInstance, FastifyReply, FastifyRequest } from 'fastify';
import type { ExchangeRate } from '../../../domain/exchange-rate/ExchangeRate.js';
import type { HistoryPoint } from '../../../domain/exchange-rate/ExchangeRateRepository.js';
import { enrichRates, getCurrencyMeta } from '../currency-meta.js';

interface DashboardDeps {
  getLatestRates: { execute(baseCurrency: string): Promise<ExchangeRate[]> };
  getRateHistory: {
    execute(base: string, quote: string, from: string, to: string): Promise<HistoryPoint[]>;
  };
}

function daysAgo(days: number): string {
  const d = new Date();
  d.setDate(d.getDate() - days);
  return d.toISOString().slice(0, 10);
}

function today(): string {
  return new Date().toISOString().slice(0, 10);
}

export function dashboardRoutes(deps: DashboardDeps) {
  return async (server: FastifyInstance): Promise<void> => {
    server.get('/', async (_request: FastifyRequest, reply: FastifyReply) => {
      const rates = await deps.getLatestRates.execute('EUR');
      const date = rates[0]?.date ?? 'N/A';

      return reply.viewAsync('pages/dashboard', {
        title: 'Ticker Lab',
        rates: enrichRates(rates.map((r) => ({ currency: r.quoteCurrency, rate: r.rate }))),
        date,
        updatedAt: new Date().toISOString(),
      });
    });

    server.get<{ Params: { quote: string } }>(
      '/rates/:quote',
      async (request: FastifyRequest<{ Params: { quote: string } }>, reply: FastifyReply) => {
        const quote = request.params.quote.toUpperCase();
        const { days = '90' } = request.query as { days?: string };
        const numDays = Math.min(Number(days) || 90, 365 * 5);

        const from = daysAgo(numDays);
        const to = today();

        const [rates, history] = await Promise.all([
          deps.getLatestRates.execute('EUR'),
          deps.getRateHistory.execute('EUR', quote, from, to),
        ]);

        const currentRate = rates.find((r) => r.quoteCurrency === quote)?.rate ?? 0;

        const meta = getCurrencyMeta(quote);

        return reply.viewAsync('pages/rate-detail', {
          quote,
          currentRate: currentRate.toFixed(4),
          currencyName: meta.name,
          flag: meta.flag,
          from,
          to,
          days,
          history,
        });
      },
    );
  };
}
