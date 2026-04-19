import type { ExchangeRate } from './ExchangeRate.js';

export interface ExchangeRateRepository {
  save(rates: ExchangeRate[]): Promise<void>;
  findLatest(baseCurrency: string): Promise<ExchangeRate[]>;
  findByDate(baseCurrency: string, date: string): Promise<ExchangeRate[]>;
}
