import { describe, expect, it, vi } from 'vitest';
import type { ExchangeRate } from '../../domain/exchange-rate/ExchangeRate.js';
import type { ExchangeRateRepository } from '../../domain/exchange-rate/ExchangeRateRepository.js';
import { GetLatestRates } from './GetLatestRates.js';

const sampleRates: ExchangeRate[] = [
  { baseCurrency: 'EUR', quoteCurrency: 'GBP', rate: 0.8561, date: '2026-04-17' },
  { baseCurrency: 'EUR', quoteCurrency: 'USD', rate: 1.1358, date: '2026-04-17' },
];

function stubRepository(rates: ExchangeRate[]): ExchangeRateRepository {
  return {
    save: vi.fn(),
    findLatest: vi.fn().mockResolvedValue(rates),
    findByDate: vi.fn(),
  };
}

describe('GetLatestRates', () => {
  it('returns rates from repository', async () => {
    const repository = stubRepository(sampleRates);
    const useCase = new GetLatestRates(repository);

    const result = await useCase.execute('EUR');

    expect(repository.findLatest).toHaveBeenCalledWith('EUR');
    expect(result).toEqual(sampleRates);
  });

  it('returns empty array when no rates exist', async () => {
    const repository = stubRepository([]);
    const useCase = new GetLatestRates(repository);

    const result = await useCase.execute('EUR');

    expect(result).toEqual([]);
  });
});
