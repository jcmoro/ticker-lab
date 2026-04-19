import type { ExchangeRate } from '../../domain/exchange-rate/ExchangeRate.js';
import type { ExchangeRateRepository } from '../../domain/exchange-rate/ExchangeRateRepository.js';

export class GetLatestRates {
  constructor(private readonly repository: ExchangeRateRepository) {}

  async execute(baseCurrency: string): Promise<ExchangeRate[]> {
    return this.repository.findLatest(baseCurrency);
  }
}
