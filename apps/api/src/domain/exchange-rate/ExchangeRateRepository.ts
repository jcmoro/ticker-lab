import type { ExchangeRate } from './ExchangeRate.js';

export interface HistoryPoint {
  date: string;
  rate: number;
}

export interface ExchangeRateRepository {
  save(rates: ExchangeRate[]): Promise<void>;
  findLatest(baseCurrency: string): Promise<ExchangeRate[]>;
  findByDate(baseCurrency: string, date: string): Promise<ExchangeRate[]>;
  findHistory(
    baseCurrency: string,
    quoteCurrency: string,
    from: string,
    to: string,
  ): Promise<HistoryPoint[]>;
}
