export class InvalidCurrencyError extends Error {
  constructor(currency: string) {
    super(`Invalid currency code: "${currency}". Must be a 3-letter ISO 4217 code.`);
    this.name = 'InvalidCurrencyError';
  }
}

export class InvalidRateError extends Error {
  constructor(rate: number) {
    super(`Invalid exchange rate: ${rate}. Must be a positive finite number.`);
    this.name = 'InvalidRateError';
  }
}

export class InvalidDateFormatError extends Error {
  constructor(date: string) {
    super(`Invalid date format: "${date}". Must be YYYY-MM-DD.`);
    this.name = 'InvalidDateFormatError';
  }
}

export class RatesNotFoundError extends Error {
  constructor(baseCurrency: string, date?: string) {
    const msg = date
      ? `No exchange rates found for ${baseCurrency} on ${date}`
      : `No exchange rates found for ${baseCurrency}`;
    super(msg);
    this.name = 'RatesNotFoundError';
  }
}
