import {
  date,
  index,
  numeric,
  pgTable,
  serial,
  timestamp,
  unique,
  varchar,
} from 'drizzle-orm/pg-core';

export const exchangeRates = pgTable(
  'exchange_rates',
  {
    id: serial('id').primaryKey(),
    baseCurrency: varchar('base_currency', { length: 3 }).notNull(),
    quoteCurrency: varchar('quote_currency', { length: 3 }).notNull(),
    rate: numeric('rate', { precision: 16, scale: 6 }).notNull(),
    date: date('date').notNull(),
    createdAt: timestamp('created_at').defaultNow().notNull(),
  },
  (table) => [
    unique('uq_exchange_rates_pair_date').on(table.baseCurrency, table.quoteCurrency, table.date),
    index('idx_exchange_rates_base_date').on(table.baseCurrency, table.date),
  ],
);
