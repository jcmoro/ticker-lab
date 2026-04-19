import { createExchangeRate, type ExchangeRate } from '../../domain/exchange-rate/ExchangeRate.js';
import type { ExchangeRateProvider } from '../../domain/exchange-rate/ExchangeRateProvider.js';

interface FrankfurterResponse {
  amount: number;
  base: string;
  date: string;
  rates: Record<string, number>;
}

export class FrankfurterClient implements ExchangeRateProvider {
  constructor(private readonly baseUrl: string) {}

  async fetchLatest(baseCurrency: string): Promise<ExchangeRate[]> {
    const url = `${this.baseUrl}/v1/latest?base=${encodeURIComponent(baseCurrency)}`;
    return this.fetchRates(url);
  }

  async fetchByDate(baseCurrency: string, date: string): Promise<ExchangeRate[]> {
    const url = `${this.baseUrl}/v1/${date}?base=${encodeURIComponent(baseCurrency)}`;
    return this.fetchRates(url);
  }

  private async fetchRates(url: string): Promise<ExchangeRate[]> {
    const response = await fetch(url);

    if (!response.ok) {
      throw new Error(`Frankfurter API error: ${response.status} ${response.statusText}`);
    }

    const data = (await response.json()) as FrankfurterResponse;

    return Object.entries(data.rates).map(([currency, rate]) =>
      createExchangeRate({
        baseCurrency: data.base,
        quoteCurrency: currency,
        rate,
        date: data.date,
      }),
    );
  }
}
