import { describe, expect, it, vi } from 'vitest';
import type { ExchangeRate } from '../../domain/exchange-rate/ExchangeRate.js';
import type { ExchangeRateRepository } from '../../domain/exchange-rate/ExchangeRateRepository.js';
import { ConvertCurrency } from './ConvertCurrency.js';

const sampleRates: ExchangeRate[] = [
  { baseCurrency: 'EUR', quoteCurrency: 'USD', rate: 1.18, date: '2026-04-17' },
  { baseCurrency: 'EUR', quoteCurrency: 'GBP', rate: 0.87, date: '2026-04-17' },
  { baseCurrency: 'EUR', quoteCurrency: 'JPY', rate: 187.72, date: '2026-04-17' },
];

function stubRepository(rates: ExchangeRate[]): ExchangeRateRepository {
  return {
    save: vi.fn(),
    findLatest: vi.fn().mockResolvedValue(rates),
    findByDate: vi.fn(),
    findHistory: vi.fn(),
  };
}

describe('ConvertCurrency', () => {
  it('converts EUR to USD', async () => {
    const uc = new ConvertCurrency(stubRepository(sampleRates));
    const result = await uc.execute('EUR', 'USD', 100);

    expect(result).not.toBeNull();
    expect(result?.from).toBe('EUR');
    expect(result?.to).toBe('USD');
    expect(result?.result).toBe(118);
  });

  it('converts USD to EUR (inverse)', async () => {
    const uc = new ConvertCurrency(stubRepository(sampleRates));
    const result = await uc.execute('USD', 'EUR', 118);

    expect(result).not.toBeNull();
    expect(result?.result).toBe(100);
  });

  it('converts GBP to JPY (cross-rate via EUR)', async () => {
    const uc = new ConvertCurrency(stubRepository(sampleRates));
    const result = await uc.execute('GBP', 'JPY', 1);

    expect(result).not.toBeNull();
    // GBP→JPY = EUR/JPY / EUR/GBP = 187.72 / 0.87 ≈ 215.77
    expect(result?.result).toBeGreaterThan(215);
    expect(result?.result).toBeLessThan(216);
  });

  it('returns rate 1 for same currency', async () => {
    const uc = new ConvertCurrency(stubRepository(sampleRates));
    const result = await uc.execute('EUR', 'EUR', 50);

    expect(result?.rate).toBe(1);
    expect(result?.result).toBe(50);
  });

  it('returns null for unknown currency', async () => {
    const uc = new ConvertCurrency(stubRepository(sampleRates));
    const result = await uc.execute('EUR', 'XYZ', 100);

    expect(result).toBeNull();
  });

  it('returns null when no rates available', async () => {
    const uc = new ConvertCurrency(stubRepository([]));
    const result = await uc.execute('EUR', 'USD', 100);

    expect(result).toBeNull();
  });
});
