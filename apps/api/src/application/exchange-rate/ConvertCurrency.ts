import type { ExchangeRateRepository } from '../../domain/exchange-rate/ExchangeRateRepository.js';

export interface ConversionResult {
  from: string;
  to: string;
  amount: number;
  rate: number;
  result: number;
  date: string;
  engine: string;
}

export class ConvertCurrency {
  constructor(private readonly repository: ExchangeRateRepository) {}

  async execute(from: string, to: string, amount: number): Promise<ConversionResult | null> {
    if (from === to) {
      return {
        from,
        to,
        amount,
        rate: 1,
        result: amount,
        date: new Date().toISOString().slice(0, 10),
        engine: 'node',
      };
    }

    const rates = await this.repository.findLatest('EUR');
    if (rates.length === 0) return null;

    const date = rates[0]?.date ?? '';
    const rateMap = new Map(rates.map((r) => [r.quoteCurrency, r.rate]));
    rateMap.set('EUR', 1);

    const fromRate = rateMap.get(from);
    const toRate = rateMap.get(to);

    if (fromRate === undefined || toRate === undefined) return null;

    const rate = toRate / fromRate;
    const result = Math.round(amount * rate * 100) / 100;

    return {
      from,
      to,
      amount,
      rate: Math.round(rate * 1_000_000) / 1_000_000,
      result,
      date,
      engine: 'node',
    };
  }
}
