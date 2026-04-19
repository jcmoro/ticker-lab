import type { FastifyInstance, FastifyReply, FastifyRequest } from 'fastify';
import type { ExchangeRate } from '../../../domain/exchange-rate/ExchangeRate.js';

export function dashboardRoutes(getLatestRates: {
  execute(baseCurrency: string): Promise<ExchangeRate[]>;
}) {
  return async (server: FastifyInstance): Promise<void> => {
    server.get('/', async (_request: FastifyRequest, reply: FastifyReply) => {
      const rates = await getLatestRates.execute('EUR');
      const date = rates[0]?.date ?? 'N/A';

      return reply.viewAsync('pages/dashboard', {
        title: 'Ticker Lab',
        rates: rates.map((r) => ({ currency: r.quoteCurrency, rate: r.rate })),
        date,
        updatedAt: new Date().toISOString(),
      });
    });
  };
}
