import { afterEach, describe, expect, it, vi } from 'vitest';
import type { ExchangeRate } from '../../../domain/exchange-rate/ExchangeRate.js';
import { buildServer } from '../server.js';

const sampleRates: ExchangeRate[] = [
  { baseCurrency: 'EUR', quoteCurrency: 'GBP', rate: 0.8561, date: '2026-04-17' },
  { baseCurrency: 'EUR', quoteCurrency: 'USD', rate: 1.1358, date: '2026-04-17' },
];

function createStubDeps() {
  return {
    getLatestRates: { execute: vi.fn().mockResolvedValue(sampleRates) },
    getRatesByDate: { execute: vi.fn().mockResolvedValue([]) },
    getRateHistory: {
      execute: vi.fn().mockResolvedValue([
        { date: '2026-04-15', rate: 1.13 },
        { date: '2026-04-16', rate: 1.134 },
        { date: '2026-04-17', rate: 1.1358 },
      ]),
    },
    convertCurrency: { execute: vi.fn().mockResolvedValue(null) },
  };
}

describe('GET / (dashboard)', () => {
  it('returns HTML with exchange rates and links to detail', async () => {
    const server = await buildServer(createStubDeps());

    const response = await server.inject({ method: 'GET', url: '/' });

    expect(response.statusCode).toBe(200);
    expect(response.headers['content-type']).toContain('text/html');
    expect(response.body).toContain('GBP');
    expect(response.body).toContain('British Pound');
    expect(response.body).toContain('0.8561');
    expect(response.body).toContain('href="/rates/GBP"');
    expect(response.body).toContain('href="/rates/USD"');

    await server.close();
  });

  it('returns HTML with empty state when no rates', async () => {
    const deps = {
      ...createStubDeps(),
      getLatestRates: { execute: vi.fn().mockResolvedValue([]) },
    };
    const server = await buildServer(deps);

    const response = await server.inject({ method: 'GET', url: '/' });

    expect(response.statusCode).toBe(200);
    expect(response.body).toContain('No rates available yet');

    await server.close();
  });
});

describe('GET /rates/:quote (detail page)', () => {
  it('returns HTML with Chart.js and history data', async () => {
    const server = await buildServer(createStubDeps());

    const response = await server.inject({ method: 'GET', url: '/rates/USD' });

    expect(response.statusCode).toBe(200);
    expect(response.body).toContain('EUR / USD');
    expect(response.body).toContain('US Dollar');
    expect(response.body).toContain('1.1358');
    expect(response.body).toContain('chart.umd.min.js');
    expect(response.body).toContain('rateChart');

    await server.close();
  });
});

describe('GET /crypto (crypto page)', () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('renders without error when API returns prices with missing numeric fields', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        json: () =>
          Promise.resolve({
            date: '2026-04-17',
            prices: [
              {
                coin_id: 'bitcoin',
                symbol: 'BTC',
                name: 'Bitcoin',
                price_eur: undefined,
                price_usd: undefined,
                change_24h: undefined,
              },
            ],
          }),
      }),
    );
    const server = await buildServer(createStubDeps());

    const response = await server.inject({ method: 'GET', url: '/crypto' });

    expect(response.statusCode).toBe(200);
    expect(response.body).toContain('Bitcoin');
    expect(response.body).toContain('0.00%');

    await server.close();
  });

  it('renders empty state when crypto service is unreachable', async () => {
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new Error('ECONNREFUSED')));
    const server = await buildServer(createStubDeps());

    const response = await server.inject({ method: 'GET', url: '/crypto' });

    expect(response.statusCode).toBe(200);
    expect(response.body).toContain('No crypto data available');

    await server.close();
  });
});

describe('GET /crypto/:id (crypto detail page)', () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('renders without error when coin has missing numeric fields', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        json: () =>
          Promise.resolve({
            prices: [
              {
                coin_id: 'bitcoin',
                symbol: 'BTC',
                name: 'Bitcoin',
                price_eur: undefined,
                price_usd: undefined,
                change_24h: undefined,
              },
            ],
          }),
      }),
    );
    const server = await buildServer(createStubDeps());

    const response = await server.inject({ method: 'GET', url: '/crypto/bitcoin' });

    expect(response.statusCode).toBe(200);
    expect(response.body).toContain('Bitcoin');
    expect(response.body).toContain('0.00%');

    await server.close();
  });
});
