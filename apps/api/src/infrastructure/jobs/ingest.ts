import { drizzle } from 'drizzle-orm/postgres-js';
import postgres from 'postgres';
import { IngestDailyRates } from '../../application/exchange-rate/IngestDailyRates.js';
import { DrizzleExchangeRateRepository } from '../persistence/DrizzleExchangeRateRepository.js';
import * as schema from '../persistence/schema.js';
import { FrankfurterClient } from '../providers/FrankfurterClient.js';

const BASE_CURRENCY = 'EUR';

async function main(): Promise<void> {
  const databaseUrl = process.env.DATABASE_URL;
  if (!databaseUrl) {
    throw new Error('DATABASE_URL environment variable is required');
  }

  const frankfurterUrl = process.env.FRANKFURTER_BASE_URL ?? 'https://api.frankfurter.dev';

  const client = postgres(databaseUrl);
  const db = drizzle(client, { schema });

  const provider = new FrankfurterClient(frankfurterUrl);
  const repository = new DrizzleExchangeRateRepository(db);
  const ingest = new IngestDailyRates(provider, repository);

  console.log(`Ingesting exchange rates for ${BASE_CURRENCY}...`);
  const result = await ingest.execute(BASE_CURRENCY);
  console.log(`Done: ${result.count} rates ingested for date ${result.date}`);

  await client.end();
}

main().catch((err: unknown) => {
  console.error('Ingestion failed:', err);
  process.exit(1);
});
