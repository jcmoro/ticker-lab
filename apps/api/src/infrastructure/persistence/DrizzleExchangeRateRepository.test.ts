import { drizzle } from 'drizzle-orm/postgres-js';
import postgres from 'postgres';
import { afterAll, afterEach, beforeAll, describe, expect, it } from 'vitest';
import { DrizzleExchangeRateRepository } from './DrizzleExchangeRateRepository.js';
import * as schema from './schema.js';

/**
 * Integration tests against real Postgres. Required by CLAUDE.md
 * ("No mocks for the database — always test against real Postgres").
 *
 * Isolation strategy: every row written by these tests uses base_currency
 * = 'ZZZ' (RFC 4217 reserved for testing) and date >= '2099-01-01', so the
 * suite can run alongside seeded dev data without cross-contamination.
 * afterEach deletes everything in that namespace.
 */

const TEST_BASE = 'ZZZ';
const FUTURE = (n: number) => `2099-01-${String(n).padStart(2, '0')}`;

const dbUrl = process.env.DATABASE_URL;
const describeIfDb = dbUrl ? describe : describe.skip;

describeIfDb('DrizzleExchangeRateRepository (integration)', () => {
  let client: ReturnType<typeof postgres>;
  let db: ReturnType<typeof drizzle<typeof schema>>;
  let repo: DrizzleExchangeRateRepository;

  beforeAll(() => {
    client = postgres(dbUrl as string, { max: 2 });
    db = drizzle(client, { schema });
    repo = new DrizzleExchangeRateRepository(db);
  });

  afterEach(async () => {
    await client`DELETE FROM exchange_rates WHERE base_currency = ${TEST_BASE}`;
  });

  afterAll(async () => {
    await client.end();
  });

  describe('save', () => {
    it('is a no-op for an empty array', async () => {
      await expect(repo.save([])).resolves.toBeUndefined();
    });

    it('inserts new rates', async () => {
      await repo.save([
        { baseCurrency: TEST_BASE, quoteCurrency: 'USD', rate: 1.1, date: FUTURE(1) },
        { baseCurrency: TEST_BASE, quoteCurrency: 'GBP', rate: 0.85, date: FUTURE(1) },
      ]);

      const rows = await repo.findByDate(TEST_BASE, FUTURE(1));
      expect(rows).toHaveLength(2);
      expect(rows.map((r) => r.quoteCurrency).sort()).toEqual(['GBP', 'USD']);
    });

    it('upserts on (base, quote, date) unique conflict', async () => {
      await repo.save([
        { baseCurrency: TEST_BASE, quoteCurrency: 'USD', rate: 1.0, date: FUTURE(2) },
      ]);
      await repo.save([
        { baseCurrency: TEST_BASE, quoteCurrency: 'USD', rate: 1.5, date: FUTURE(2) },
      ]);

      const rows = await repo.findByDate(TEST_BASE, FUTURE(2));
      expect(rows).toHaveLength(1);
      expect(rows[0]?.rate).toBe(1.5);
    });
  });

  describe('findLatest', () => {
    it('returns an empty array when no rates exist for the base', async () => {
      expect(await repo.findLatest(TEST_BASE)).toEqual([]);
    });

    it('returns the rates of the most recent date for the given base', async () => {
      await repo.save([
        { baseCurrency: TEST_BASE, quoteCurrency: 'USD', rate: 1.1, date: FUTURE(1) },
        { baseCurrency: TEST_BASE, quoteCurrency: 'USD', rate: 1.2, date: FUTURE(3) },
        { baseCurrency: TEST_BASE, quoteCurrency: 'GBP', rate: 0.85, date: FUTURE(3) },
      ]);

      const latest = await repo.findLatest(TEST_BASE);
      expect(latest).toHaveLength(2);
      expect(latest.every((r) => r.date === FUTURE(3))).toBe(true);
    });

    it('does not include rates from other base currencies', async () => {
      await repo.save([
        { baseCurrency: TEST_BASE, quoteCurrency: 'USD', rate: 1.1, date: FUTURE(4) },
      ]);

      // EUR rates from real seed data must not leak in.
      const latest = await repo.findLatest(TEST_BASE);
      expect(latest.every((r) => r.baseCurrency === TEST_BASE)).toBe(true);
    });
  });

  describe('findByDate', () => {
    it('returns rates ordered by quote_currency', async () => {
      await repo.save([
        { baseCurrency: TEST_BASE, quoteCurrency: 'USD', rate: 1.1, date: FUTURE(5) },
        { baseCurrency: TEST_BASE, quoteCurrency: 'AUD', rate: 1.6, date: FUTURE(5) },
        { baseCurrency: TEST_BASE, quoteCurrency: 'GBP', rate: 0.85, date: FUTURE(5) },
      ]);

      const rows = await repo.findByDate(TEST_BASE, FUTURE(5));
      expect(rows.map((r) => r.quoteCurrency)).toEqual(['AUD', 'GBP', 'USD']);
    });

    it('returns an empty array for a date with no rates', async () => {
      expect(await repo.findByDate(TEST_BASE, FUTURE(6))).toEqual([]);
    });
  });

  describe('findHistory', () => {
    it('returns rates within an inclusive date range, ascending', async () => {
      await repo.save([
        { baseCurrency: TEST_BASE, quoteCurrency: 'USD', rate: 1.1, date: FUTURE(1) },
        { baseCurrency: TEST_BASE, quoteCurrency: 'USD', rate: 1.2, date: FUTURE(2) },
        { baseCurrency: TEST_BASE, quoteCurrency: 'USD', rate: 1.3, date: FUTURE(3) },
      ]);

      const history = await repo.findHistory(TEST_BASE, 'USD', FUTURE(1), FUTURE(3));
      expect(history.map((p) => p.date)).toEqual([FUTURE(1), FUTURE(2), FUTURE(3)]);
      expect(history.map((p) => p.rate)).toEqual([1.1, 1.2, 1.3]);
    });

    it('excludes rates outside the range', async () => {
      await repo.save([
        { baseCurrency: TEST_BASE, quoteCurrency: 'USD', rate: 1.1, date: FUTURE(1) },
        { baseCurrency: TEST_BASE, quoteCurrency: 'USD', rate: 1.2, date: FUTURE(2) },
        { baseCurrency: TEST_BASE, quoteCurrency: 'USD', rate: 1.3, date: FUTURE(3) },
      ]);

      const history = await repo.findHistory(TEST_BASE, 'USD', FUTURE(2), FUTURE(2));
      expect(history).toHaveLength(1);
      expect(history[0]?.date).toBe(FUTURE(2));
    });

    it('filters by quote currency', async () => {
      await repo.save([
        { baseCurrency: TEST_BASE, quoteCurrency: 'USD', rate: 1.1, date: FUTURE(1) },
        { baseCurrency: TEST_BASE, quoteCurrency: 'GBP', rate: 0.85, date: FUTURE(1) },
      ]);

      const usd = await repo.findHistory(TEST_BASE, 'USD', FUTURE(1), FUTURE(1));
      const gbp = await repo.findHistory(TEST_BASE, 'GBP', FUTURE(1), FUTURE(1));
      expect(usd).toHaveLength(1);
      expect(usd[0]?.rate).toBe(1.1);
      expect(gbp).toHaveLength(1);
      expect(gbp[0]?.rate).toBe(0.85);
    });

    it('returns an empty array when no data exists in the range', async () => {
      expect(await repo.findHistory(TEST_BASE, 'USD', FUTURE(1), FUTURE(3))).toEqual([]);
    });
  });
});
