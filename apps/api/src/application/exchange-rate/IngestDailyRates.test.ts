import { describe, expect, it, vi } from 'vitest';
import type { ExchangeRate } from '../../domain/exchange-rate/ExchangeRate.js';
import type { ExchangeRateProvider } from '../../domain/exchange-rate/ExchangeRateProvider.js';
import type { ExchangeRateRepository } from '../../domain/exchange-rate/ExchangeRateRepository.js';
import { IngestDailyRates } from './IngestDailyRates.js';

function stubProvider(rates: ExchangeRate[]): ExchangeRateProvider {
  return {
    fetchLatest: vi.fn().mockResolvedValue(rates),
    fetchByDate: vi.fn().mockResolvedValue(rates),
  };
}

function stubRepository(): ExchangeRateRepository & { save: ReturnType<typeof vi.fn> } {
  return {
    save: vi.fn().mockResolvedValue(undefined),
    findLatest: vi.fn().mockResolvedValue([]),
    findByDate: vi.fn().mockResolvedValue([]),
  };
}

const sampleRates: ExchangeRate[] = [
  { baseCurrency: 'EUR', quoteCurrency: 'USD', rate: 1.1358, date: '2026-04-17' },
  { baseCurrency: 'EUR', quoteCurrency: 'GBP', rate: 0.8561, date: '2026-04-17' },
];

describe('IngestDailyRates', () => {
  it('fetches rates from provider and saves to repository', async () => {
    const provider = stubProvider(sampleRates);
    const repository = stubRepository();
    const useCase = new IngestDailyRates(provider, repository);

    const result = await useCase.execute('EUR');

    expect(provider.fetchLatest).toHaveBeenCalledWith('EUR');
    expect(repository.save).toHaveBeenCalledWith(sampleRates);
    expect(result).toEqual({ count: 2, date: '2026-04-17' });
  });

  it('returns zero count when provider returns no rates', async () => {
    const provider = stubProvider([]);
    const repository = stubRepository();
    const useCase = new IngestDailyRates(provider, repository);

    const result = await useCase.execute('EUR');

    expect(repository.save).not.toHaveBeenCalled();
    expect(result).toEqual({ count: 0, date: '' });
  });
});
