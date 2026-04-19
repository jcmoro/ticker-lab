interface CurrencyInfo {
  name: string;
  country: string;
  flag: string;
}

const currencies: Record<string, CurrencyInfo> = {
  AUD: { name: 'Australian Dollar', country: 'Australia', flag: '🇦🇺' },
  BRL: { name: 'Brazilian Real', country: 'Brazil', flag: '🇧🇷' },
  CAD: { name: 'Canadian Dollar', country: 'Canada', flag: '🇨🇦' },
  CHF: { name: 'Swiss Franc', country: 'Switzerland', flag: '🇨🇭' },
  CNY: { name: 'Chinese Yuan', country: 'China', flag: '🇨🇳' },
  CZK: { name: 'Czech Koruna', country: 'Czech Republic', flag: '🇨🇿' },
  DKK: { name: 'Danish Krone', country: 'Denmark', flag: '🇩🇰' },
  EUR: { name: 'Euro', country: 'Eurozone', flag: '🇪🇺' },
  GBP: { name: 'British Pound', country: 'United Kingdom', flag: '🇬🇧' },
  HKD: { name: 'Hong Kong Dollar', country: 'Hong Kong', flag: '🇭🇰' },
  HUF: { name: 'Hungarian Forint', country: 'Hungary', flag: '🇭🇺' },
  IDR: { name: 'Indonesian Rupiah', country: 'Indonesia', flag: '🇮🇩' },
  ILS: { name: 'Israeli Shekel', country: 'Israel', flag: '🇮🇱' },
  INR: { name: 'Indian Rupee', country: 'India', flag: '🇮🇳' },
  ISK: { name: 'Icelandic Krona', country: 'Iceland', flag: '🇮🇸' },
  JPY: { name: 'Japanese Yen', country: 'Japan', flag: '🇯🇵' },
  KRW: { name: 'South Korean Won', country: 'South Korea', flag: '🇰🇷' },
  MXN: { name: 'Mexican Peso', country: 'Mexico', flag: '🇲🇽' },
  MYR: { name: 'Malaysian Ringgit', country: 'Malaysia', flag: '🇲🇾' },
  NOK: { name: 'Norwegian Krone', country: 'Norway', flag: '🇳🇴' },
  NZD: { name: 'New Zealand Dollar', country: 'New Zealand', flag: '🇳🇿' },
  PHP: { name: 'Philippine Peso', country: 'Philippines', flag: '🇵🇭' },
  PLN: { name: 'Polish Zloty', country: 'Poland', flag: '🇵🇱' },
  RON: { name: 'Romanian Leu', country: 'Romania', flag: '🇷🇴' },
  SEK: { name: 'Swedish Krona', country: 'Sweden', flag: '🇸🇪' },
  SGD: { name: 'Singapore Dollar', country: 'Singapore', flag: '🇸🇬' },
  THB: { name: 'Thai Baht', country: 'Thailand', flag: '🇹🇭' },
  TRY: { name: 'Turkish Lira', country: 'Turkey', flag: '🇹🇷' },
  USD: { name: 'US Dollar', country: 'United States', flag: '🇺🇸' },
  ZAR: { name: 'South African Rand', country: 'South Africa', flag: '🇿🇦' },
};

const fallback: CurrencyInfo = { name: '', country: '', flag: '' };

export function getCurrencyMeta(code: string): CurrencyInfo {
  return currencies[code] ?? fallback;
}

export function getAllCurrencies(): Array<{ code: string } & CurrencyInfo> {
  return Object.entries(currencies)
    .map(([code, info]) => ({ code, ...info }))
    .sort((a, b) => a.code.localeCompare(b.code));
}

export function enrichRates(rates: { currency: string; rate: number }[]) {
  return rates.map((r) => ({
    ...r,
    ...getCurrencyMeta(r.currency),
  }));
}
