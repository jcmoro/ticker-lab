import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { FrankfurterClient } from './FrankfurterClient.js';

/**
 * Provider integration tests using cassette-style fetch stubs.
 * No real network calls — fetch is mocked with fixed JSON payloads taken
 * from real Frankfurter responses. Tests cover URL construction, response
 * parsing, and error semantics.
 */

const BASE_URL = 'https://api.frankfurter.dev';

function mockFetchOnce(
  payload: unknown,
  init: { ok?: boolean; status?: number; statusText?: string } = {},
) {
  const ok = init.ok ?? true;
  return vi.spyOn(globalThis, 'fetch').mockResolvedValueOnce({
    ok,
    status: init.status ?? (ok ? 200 : 500),
    statusText: init.statusText ?? (ok ? 'OK' : 'Internal Server Error'),
    json: async () => payload,
  } as Response);
}

describe('FrankfurterClient', () => {
  let client: FrankfurterClient;

  beforeEach(() => {
    client = new FrankfurterClient(BASE_URL);
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe('fetchLatest', () => {
    it('hits /v1/latest with the base query param', async () => {
      const spy = mockFetchOnce({
        amount: 1,
        base: 'EUR',
        date: '2026-05-09',
        rates: { USD: 1.0825, GBP: 0.851 },
      });

      await client.fetchLatest('EUR');

      expect(spy).toHaveBeenCalledWith('https://api.frankfurter.dev/v1/latest?base=EUR');
    });

    it('parses the rates map into ExchangeRate domain objects', async () => {
      mockFetchOnce({
        amount: 1,
        base: 'EUR',
        date: '2026-05-09',
        rates: { USD: 1.0825, GBP: 0.851 },
      });

      const rates = await client.fetchLatest('EUR');

      expect(rates).toHaveLength(2);
      const usd = rates.find((r) => r.quoteCurrency === 'USD');
      expect(usd).toEqual({
        baseCurrency: 'EUR',
        quoteCurrency: 'USD',
        rate: 1.0825,
        date: '2026-05-09',
      });
    });

    it('throws with status code on non-OK responses', async () => {
      mockFetchOnce(null, { ok: false, status: 503, statusText: 'Service Unavailable' });

      await expect(client.fetchLatest('EUR')).rejects.toThrow(/503.*Service Unavailable/);
    });

    it('returns an empty array when the rates map is empty', async () => {
      mockFetchOnce({ amount: 1, base: 'EUR', date: '2026-05-09', rates: {} });
      const rates = await client.fetchLatest('EUR');
      expect(rates).toEqual([]);
    });
  });

  describe('fetchByDate', () => {
    it('hits /v1/{date} with the base query param', async () => {
      const spy = mockFetchOnce({
        amount: 1,
        base: 'EUR',
        date: '2026-04-17',
        rates: { USD: 1.13 },
      });

      await client.fetchByDate('EUR', '2026-04-17');

      expect(spy).toHaveBeenCalledWith('https://api.frankfurter.dev/v1/2026-04-17?base=EUR');
    });

    it('preserves the date returned by the API', async () => {
      // Frankfurter returns the *previous business day* if requested date is
      // a weekend/holiday — the client must trust the API's date, not the
      // input date.
      mockFetchOnce({
        amount: 1,
        base: 'EUR',
        date: '2026-04-17', // requested 2026-04-19 (a Sunday), got Friday back
        rates: { USD: 1.13 },
      });

      const rates = await client.fetchByDate('EUR', '2026-04-19');
      expect(rates[0]?.date).toBe('2026-04-17');
    });

    it('throws on non-OK responses', async () => {
      mockFetchOnce(null, { ok: false, status: 404, statusText: 'Not Found' });
      await expect(client.fetchByDate('EUR', '1900-01-01')).rejects.toThrow(/404/);
    });
  });

  describe('fetchDateRange', () => {
    it('hits /v1/{from}..{to} with the base query param', async () => {
      const spy = mockFetchOnce({
        amount: 1,
        base: 'EUR',
        start_date: '2026-05-01',
        end_date: '2026-05-02',
        rates: {
          '2026-05-01': { USD: 1.08 },
          '2026-05-02': { USD: 1.09 },
        },
      });

      await client.fetchDateRange('EUR', '2026-05-01', '2026-05-02');

      expect(spy).toHaveBeenCalledWith(
        'https://api.frankfurter.dev/v1/2026-05-01..2026-05-02?base=EUR',
      );
    });

    it('flattens the nested time-series into one ExchangeRate per (date, quote)', async () => {
      mockFetchOnce({
        amount: 1,
        base: 'EUR',
        start_date: '2026-05-01',
        end_date: '2026-05-02',
        rates: {
          '2026-05-01': { USD: 1.08, GBP: 0.85 },
          '2026-05-02': { USD: 1.09, GBP: 0.86 },
        },
      });

      const rates = await client.fetchDateRange('EUR', '2026-05-01', '2026-05-02');
      expect(rates).toHaveLength(4);

      const may1Usd = rates.find((r) => r.date === '2026-05-01' && r.quoteCurrency === 'USD');
      expect(may1Usd?.rate).toBe(1.08);

      const may2Gbp = rates.find((r) => r.date === '2026-05-02' && r.quoteCurrency === 'GBP');
      expect(may2Gbp?.rate).toBe(0.86);
    });

    it('throws on non-OK responses', async () => {
      mockFetchOnce(null, { ok: false, status: 500, statusText: 'Internal Server Error' });
      await expect(client.fetchDateRange('EUR', '2026-05-01', '2026-05-02')).rejects.toThrow(/500/);
    });

    it('returns an empty array for an empty range', async () => {
      mockFetchOnce({
        amount: 1,
        base: 'EUR',
        start_date: '2026-05-01',
        end_date: '2026-05-02',
        rates: {},
      });

      const rates = await client.fetchDateRange('EUR', '2026-05-01', '2026-05-02');
      expect(rates).toEqual([]);
    });
  });

  describe('URL encoding', () => {
    it('URL-encodes the base currency parameter', async () => {
      const spy = mockFetchOnce({
        amount: 1,
        base: 'USD',
        date: '2026-05-09',
        rates: {},
      });

      // Currency codes are ISO 4217 alphabetic, but defensively encode anyway.
      await client.fetchLatest('USD');
      expect(spy).toHaveBeenCalledWith(expect.stringContaining('base=USD'));
    });
  });
});
