import { InvalidCurrencyError, InvalidDateFormatError, InvalidRateError } from './errors.js';

export interface ExchangeRate {
  readonly baseCurrency: string;
  readonly quoteCurrency: string;
  readonly rate: number;
  readonly date: string;
}

const CURRENCY_CODE_LENGTH = 3;
const DATE_REGEX = /^\d{4}-\d{2}-\d{2}$/;

export function createExchangeRate(params: {
  baseCurrency: string;
  quoteCurrency: string;
  rate: number;
  date: string;
}): ExchangeRate {
  if (params.baseCurrency.length !== CURRENCY_CODE_LENGTH) {
    throw new InvalidCurrencyError(params.baseCurrency);
  }
  if (params.quoteCurrency.length !== CURRENCY_CODE_LENGTH) {
    throw new InvalidCurrencyError(params.quoteCurrency);
  }
  if (params.rate <= 0 || !Number.isFinite(params.rate)) {
    throw new InvalidRateError(params.rate);
  }
  if (!DATE_REGEX.test(params.date)) {
    throw new InvalidDateFormatError(params.date);
  }

  return Object.freeze({
    baseCurrency: params.baseCurrency.toUpperCase(),
    quoteCurrency: params.quoteCurrency.toUpperCase(),
    rate: params.rate,
    date: params.date,
  });
}
