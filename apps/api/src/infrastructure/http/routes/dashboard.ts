import type { FastifyInstance, FastifyReply, FastifyRequest } from 'fastify';
import type { ExchangeRate } from '../../../domain/exchange-rate/ExchangeRate.js';
import type { HistoryPoint } from '../../../domain/exchange-rate/ExchangeRateRepository.js';
import { enrichRates, getAllCurrencies, getCurrencyMeta } from '../currency-meta.js';

interface DashboardDeps {
  getLatestRates: { execute(baseCurrency: string): Promise<ExchangeRate[]> };
  getRateHistory: {
    execute(base: string, quote: string, from: string, to: string): Promise<HistoryPoint[]>;
  };
}

const RETRY_ATTEMPTS = 5;
const RETRY_DELAY_MS = 5000;

async function fetchWithRetry(url: string): Promise<Response> {
  for (let attempt = 1; attempt <= RETRY_ATTEMPTS; attempt++) {
    const res = await fetch(url);
    if (res.ok) return res;
    if (attempt < RETRY_ATTEMPTS) {
      await new Promise((resolve) => setTimeout(resolve, RETRY_DELAY_MS));
    }
  }
  throw new Error(`Failed to fetch ${url} after ${RETRY_ATTEMPTS} attempts`);
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

    server.get('/converter', async (_request: FastifyRequest, reply: FastifyReply) => {
      const goBaseUrl = process.env.GO_CONVERTER_URL ?? 'http://localhost:8080';
      return reply.viewAsync('pages/converter', {
        title: 'Currency Converter',
        currencies: getAllCurrencies(),
        goBaseUrl,
      });
    });

    const macroBaseUrl = process.env.MACRO_GO_URL ?? 'http://localhost:8110';

    server.get('/macro', async (_request: FastifyRequest, reply: FastifyReply) => {
      try {
        const res = await fetchWithRetry(`${macroBaseUrl}/api/v1/macro/indicators`);
        const data = (await res.json()) as { count: number; indicators: MacroIndicator[] };
        return reply.viewAsync('pages/macro', {
          title: 'Macro Indicators',
          count: data.count,
          indicators: data.indicators ?? [],
        });
      } catch {
        return reply.viewAsync('pages/macro', {
          title: 'Macro Indicators',
          count: 0,
          indicators: [],
          macroBaseUrl,
        });
      }
    });

    server.get<{ Params: { source: string; id: string } }>(
      '/macro/:source/:id',
      async (
        request: FastifyRequest<{ Params: { source: string; id: string } }>,
        reply: FastifyReply,
      ) => {
        const { source, id } = request.params;
        const { days = '365' } = request.query as { days?: string };

        try {
          const res = await fetchWithRetry(
            `${macroBaseUrl}/api/v1/macro/${source}/${id}/history?days=${days}`,
          );
          const data = (await res.json()) as MacroHistory;

          return reply.viewAsync('pages/macro-detail', {
            title: data.name ?? id,
            source,
            series_id: id,
            name: data.name ?? id,
            unit: findIndicatorUnit(source, id),
            frequency: findIndicatorFreq(source, id),
            points: data.points ?? [],
            days,
          });
        } catch {
          return reply.viewAsync('pages/macro-detail', {
            title: id,
            source,
            series_id: id,
            name: id,
            unit: '',
            frequency: '',
            points: [],
            days,
          });
        }
      },
    );

    const cryptoBaseUrl = process.env.CRYPTO_GO_URL ?? 'http://localhost:8090';

    server.get('/crypto', async (_request: FastifyRequest, reply: FastifyReply) => {
      try {
        const res = await fetchWithRetry(`${cryptoBaseUrl}/api/v1/crypto/latest`);
        const data = (await res.json()) as { date: string; prices: CryptoPrice[] };
        return reply.viewAsync('pages/crypto', {
          title: 'Crypto',
          prices: sanitizeCryptoPrices(data.prices),
          date: data.date ?? 'N/A',
        });
      } catch {
        return reply.viewAsync('pages/crypto', {
          title: 'Crypto',
          prices: [],
          date: 'N/A',
          cryptoBaseUrl,
        });
      }
    });

    server.get<{ Params: { id: string } }>(
      '/crypto/:id',
      async (request: FastifyRequest<{ Params: { id: string } }>, reply: FastifyReply) => {
        const coinId = request.params.id;
        const { days = '90' } = request.query as { days?: string };

        try {
          const [latestRes, historyRes] = await Promise.all([
            fetchWithRetry(`${cryptoBaseUrl}/api/v1/crypto/latest`),
            fetchWithRetry(`${cryptoBaseUrl}/api/v1/crypto/${coinId}/history?days=${days}`),
          ]);

          const latest = (await latestRes.json()) as { prices: CryptoPrice[] };
          const history = (await historyRes.json()) as {
            prices: { date: string; price: number }[];
          };

          const coin = sanitizeCryptoPrices(latest.prices).find(
            (p: CryptoPrice) => p.coin_id === coinId,
          );

          return reply.viewAsync('pages/crypto-detail', {
            title: coin?.name ?? coinId,
            coin: coin ?? {
              coin_id: coinId,
              symbol: '',
              name: coinId,
              price_eur: 0,
              price_usd: 0,
              change_24h: 0,
            },
            history: history.prices ?? [],
            days,
          });
        } catch {
          return reply.viewAsync('pages/crypto-detail', {
            title: coinId,
            coin: {
              coin_id: coinId,
              symbol: '',
              name: coinId,
              price_eur: 0,
              price_usd: 0,
              change_24h: 0,
            },
            history: [],
            days,
          });
        }
      },
    );
  };
}

interface CryptoPrice {
  coin_id: string;
  symbol: string;
  name: string;
  price_eur: number;
  price_usd: number;
  market_cap_eur?: number;
  change_24h: number;
  date?: string;
}

function sanitizeCryptoPrices(prices: CryptoPrice[] | undefined): CryptoPrice[] {
  return (prices ?? []).map((p) => ({
    ...p,
    price_eur: p.price_eur ?? 0,
    price_usd: p.price_usd ?? 0,
    change_24h: p.change_24h ?? 0,
  }));
}

interface MacroIndicator {
  source: string;
  series_id: string;
  name: string;
  category: string;
  unit: string;
  frequency: string;
  latest_value: number;
  latest_date: string;
  prev_value?: number;
  change?: number;
}

interface MacroHistory {
  source: string;
  series_id: string;
  name: string;
  days: number;
  count: number;
  points: { date: string; value: number }[];
}

const macroSeriesMeta: Record<string, { unit: string; freq: string }> = {
  'fred/CPIAUCSL': { unit: 'index', freq: 'monthly' },
  'fred/UNRATE': { unit: 'percent', freq: 'monthly' },
  'fred/FEDFUNDS': { unit: 'percent', freq: 'monthly' },
  'fred/DGS10': { unit: 'percent', freq: 'daily' },
  'fred/GDPC1': { unit: 'billions_usd', freq: 'quarterly' },
  'fred/DGS2': { unit: 'percent', freq: 'daily' },
  'fred/T10Y2Y': { unit: 'percent', freq: 'daily' },
  'fred/PCEPI': { unit: 'index', freq: 'monthly' },
  'fred/PAYEMS': { unit: 'thousands', freq: 'monthly' },
  'fred/M2SL': { unit: 'billions_usd', freq: 'monthly' },
  'fred/CSUSHPINSA': { unit: 'index', freq: 'monthly' },
  'ecb/ICP': { unit: 'percent', freq: 'monthly' },
  'ecb/FM_MRR': { unit: 'percent', freq: 'monthly' },
  'ecb/EST': { unit: 'percent', freq: 'daily' },
};

function findIndicatorUnit(source: string, id: string): string {
  return macroSeriesMeta[`${source}/${id}`]?.unit ?? '';
}

function findIndicatorFreq(source: string, id: string): string {
  return macroSeriesMeta[`${source}/${id}`]?.freq ?? '';
}
