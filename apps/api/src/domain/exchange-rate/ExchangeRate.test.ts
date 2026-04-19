import { describe, expect, it } from 'vitest';
import { createExchangeRate } from './ExchangeRate.js';
import { InvalidCurrencyError, InvalidDateFormatError, InvalidRateError } from './errors.js';

describe('createExchangeRate', () => {
  const validParams = {
    baseCurrency: 'EUR',
    quoteCurrency: 'USD',
    rate: 1.1358,
    date: '2026-04-17',
  };

  it('creates a valid exchange rate', () => {
    const rate = createExchangeRate(validParams);

    expect(rate.baseCurrency).toBe('EUR');
    expect(rate.quoteCurrency).toBe('USD');
    expect(rate.rate).toBe(1.1358);
    expect(rate.date).toBe('2026-04-17');
  });

  it('normalizes currency codes to uppercase', () => {
    const rate = createExchangeRate({ ...validParams, baseCurrency: 'eur', quoteCurrency: 'usd' });

    expect(rate.baseCurrency).toBe('EUR');
    expect(rate.quoteCurrency).toBe('USD');
  });

  it('returns a frozen object', () => {
    const rate = createExchangeRate(validParams);
    expect(Object.isFrozen(rate)).toBe(true);
  });

  it('throws InvalidCurrencyError for invalid base currency', () => {
    expect(() => createExchangeRate({ ...validParams, baseCurrency: 'EU' })).toThrow(
      InvalidCurrencyError,
    );
  });

  it('throws InvalidCurrencyError for invalid quote currency', () => {
    expect(() => createExchangeRate({ ...validParams, quoteCurrency: 'USDX' })).toThrow(
      InvalidCurrencyError,
    );
  });

  it('throws InvalidRateError for zero rate', () => {
    expect(() => createExchangeRate({ ...validParams, rate: 0 })).toThrow(InvalidRateError);
  });

  it('throws InvalidRateError for negative rate', () => {
    expect(() => createExchangeRate({ ...validParams, rate: -1.5 })).toThrow(InvalidRateError);
  });

  it('throws InvalidRateError for Infinity', () => {
    expect(() => createExchangeRate({ ...validParams, rate: Number.POSITIVE_INFINITY })).toThrow(
      InvalidRateError,
    );
  });

  it('throws InvalidDateFormatError for invalid date', () => {
    expect(() => createExchangeRate({ ...validParams, date: '17-04-2026' })).toThrow(
      InvalidDateFormatError,
    );
  });
});
