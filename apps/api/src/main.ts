import { drizzle } from 'drizzle-orm/postgres-js';
import postgres from 'postgres';
import { GetLatestRates } from './application/exchange-rate/GetLatestRates.js';
import { GetRatesByDate } from './application/exchange-rate/GetRatesByDate.js';
import { buildServer } from './infrastructure/http/server.js';
import { DrizzleExchangeRateRepository } from './infrastructure/persistence/DrizzleExchangeRateRepository.js';
import * as schema from './infrastructure/persistence/schema.js';

const start = async (): Promise<void> => {
  const databaseUrl = process.env.DATABASE_URL;
  if (!databaseUrl) {
    throw new Error('DATABASE_URL environment variable is required');
  }

  // Infrastructure
  const client = postgres(databaseUrl);
  const db = drizzle(client, { schema });

  // Repositories
  const exchangeRateRepository = new DrizzleExchangeRateRepository(db);

  // Use cases
  const getLatestRates = new GetLatestRates(exchangeRateRepository);
  const getRatesByDate = new GetRatesByDate(exchangeRateRepository);

  // Server
  const server = await buildServer({ getLatestRates, getRatesByDate, db: client });

  const port = Number(process.env.API_PORT ?? 3000);
  const host = process.env.API_HOST ?? '0.0.0.0';

  await server.listen({ port, host });
};

start().catch((err: unknown) => {
  console.error('Failed to start server:', err);
  process.exit(1);
});
