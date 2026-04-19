import type { ExchangeRate } from './ExchangeRate.js';

export interface ExchangeRateProvider {
  fetchLatest(baseCurrency: string): Promise<ExchangeRate[]>;
  fetchByDate(baseCurrency: string, date: string): Promise<ExchangeRate[]>;
  fetchDateRange(baseCurrency: string, from: string, to: string): Promise<ExchangeRate[]>;
}
