import type {
  ExchangeRateRepository,
  HistoryPoint,
} from '../../domain/exchange-rate/ExchangeRateRepository.js';

export class GetRateHistory {
  constructor(private readonly repository: ExchangeRateRepository) {}

  async execute(
    baseCurrency: string,
    quoteCurrency: string,
    from: string,
    to: string,
  ): Promise<HistoryPoint[]> {
    return this.repository.findHistory(baseCurrency, quoteCurrency, from, to);
  }
}
