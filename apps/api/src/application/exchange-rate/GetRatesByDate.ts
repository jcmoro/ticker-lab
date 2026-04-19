import type { ExchangeRate } from '../../domain/exchange-rate/ExchangeRate.js';
import type { ExchangeRateRepository } from '../../domain/exchange-rate/ExchangeRateRepository.js';

export class GetRatesByDate {
  constructor(private readonly repository: ExchangeRateRepository) {}

  async execute(baseCurrency: string, date: string): Promise<ExchangeRate[]> {
    return this.repository.findByDate(baseCurrency, date);
  }
}
