import type { ExchangeRate } from './ExchangeRate.js';

export interface ExchangeRateProvider {
  fetchLatest(baseCurrency: string): Promise<ExchangeRate[]>;
  fetchByDate(baseCurrency: string, date: string): Promise<ExchangeRate[]>;
}
