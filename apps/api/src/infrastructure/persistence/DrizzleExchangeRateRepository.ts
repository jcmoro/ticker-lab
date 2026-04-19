import { and, desc, eq } from 'drizzle-orm';
import type { PostgresJsDatabase } from 'drizzle-orm/postgres-js';
import type { ExchangeRate } from '../../domain/exchange-rate/ExchangeRate.js';
import type { ExchangeRateRepository } from '../../domain/exchange-rate/ExchangeRateRepository.js';
import * as schema from './schema.js';

export class DrizzleExchangeRateRepository implements ExchangeRateRepository {
  constructor(private readonly db: PostgresJsDatabase<typeof schema>) {}

  async save(rates: ExchangeRate[]): Promise<void> {
    if (rates.length === 0) return;

    await this.db
      .insert(schema.exchangeRates)
      .values(
        rates.map((r) => ({
          baseCurrency: r.baseCurrency,
          quoteCurrency: r.quoteCurrency,
          rate: r.rate.toString(),
          date: r.date,
        })),
      )
      .onConflictDoUpdate({
        target: [
          schema.exchangeRates.baseCurrency,
          schema.exchangeRates.quoteCurrency,
          schema.exchangeRates.date,
        ],
        set: { rate: schema.exchangeRates.rate },
      });
  }

  async findLatest(baseCurrency: string): Promise<ExchangeRate[]> {
    const latestDate = await this.db
      .select({ date: schema.exchangeRates.date })
      .from(schema.exchangeRates)
      .where(eq(schema.exchangeRates.baseCurrency, baseCurrency))
      .orderBy(desc(schema.exchangeRates.date))
      .limit(1);

    if (latestDate.length === 0) return [];

    const dateValue = latestDate[0]?.date;
    if (!dateValue) return [];

    return this.findByDate(baseCurrency, dateValue);
  }

  async findByDate(baseCurrency: string, date: string): Promise<ExchangeRate[]> {
    const rows = await this.db
      .select()
      .from(schema.exchangeRates)
      .where(
        and(
          eq(schema.exchangeRates.baseCurrency, baseCurrency),
          eq(schema.exchangeRates.date, date),
        ),
      )
      .orderBy(schema.exchangeRates.quoteCurrency);

    return rows.map((row) => ({
      baseCurrency: row.baseCurrency,
      quoteCurrency: row.quoteCurrency,
      rate: Number(row.rate),
      date: row.date,
    }));
  }
}
