import { drizzle } from 'drizzle-orm/postgres-js';
import postgres from 'postgres';
import { DrizzleExchangeRateRepository } from '../persistence/DrizzleExchangeRateRepository.js';
import * as schema from '../persistence/schema.js';
import { FrankfurterClient } from '../providers/FrankfurterClient.js';

const BASE_CURRENCY = 'EUR';
const CHUNK_DAYS = 90;

function addDays(dateStr: string, days: number): string {
  const d = new Date(dateStr);
  d.setDate(d.getDate() + days);
  return d.toISOString().slice(0, 10);
}

async function main(): Promise<void> {
  const databaseUrl = process.env.DATABASE_URL;
  if (!databaseUrl) {
    throw new Error('DATABASE_URL environment variable is required');
  }

  const frankfurterUrl = process.env.FRANKFURTER_BASE_URL ?? 'https://api.frankfurter.dev';
  const fromDate = process.argv[2] ?? '2024-01-01';
  const toDate = process.argv[3] ?? new Date().toISOString().slice(0, 10);

  const client = postgres(databaseUrl);
  const db = drizzle(client, { schema });
  const provider = new FrankfurterClient(frankfurterUrl);
  const repository = new DrizzleExchangeRateRepository(db);

  console.log(`Backfilling ${BASE_CURRENCY} rates from ${fromDate} to ${toDate}...`);

  let chunkStart = fromDate;
  let totalRates = 0;

  while (chunkStart < toDate) {
    const chunkEnd =
      addDays(chunkStart, CHUNK_DAYS) < toDate ? addDays(chunkStart, CHUNK_DAYS) : toDate;

    console.log(`  Fetching ${chunkStart} .. ${chunkEnd}`);
    const rates = await provider.fetchDateRange(BASE_CURRENCY, chunkStart, chunkEnd);

    if (rates.length > 0) {
      await repository.save(rates);
      totalRates += rates.length;
      console.log(`  Saved ${rates.length} rates`);
    }

    chunkStart = addDays(chunkEnd, 1);
  }

  console.log(`Done: ${totalRates} total rates backfilled.`);
  await client.end();
}

main().catch((err: unknown) => {
  console.error('Backfill failed:', err);
  process.exit(1);
});
