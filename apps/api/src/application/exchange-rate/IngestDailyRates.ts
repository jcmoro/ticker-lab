import type { ExchangeRateProvider } from '../../domain/exchange-rate/ExchangeRateProvider.js';
import type { ExchangeRateRepository } from '../../domain/exchange-rate/ExchangeRateRepository.js';

export class IngestDailyRates {
  constructor(
    private readonly provider: ExchangeRateProvider,
    private readonly repository: ExchangeRateRepository,
  ) {}

  async execute(baseCurrency: string): Promise<{ count: number; date: string }> {
    const rates = await this.provider.fetchLatest(baseCurrency);

    if (rates.length === 0) {
      return { count: 0, date: '' };
    }

    await this.repository.save(rates);

    return { count: rates.length, date: rates[0]?.date ?? '' };
  }
}
