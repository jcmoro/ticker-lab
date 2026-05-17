import { drizzle } from 'drizzle-orm/postgres-js';
import { migrate } from 'drizzle-orm/postgres-js/migrator';
import postgres from 'postgres';
import { MissingEnvVarError } from '../errors.js';

const databaseUrl = process.env.DATABASE_URL;
if (!databaseUrl) {
  throw new MissingEnvVarError('DATABASE_URL');
}

const client = postgres(databaseUrl, { max: 1 });
const db = drizzle(client);

console.log('Running migrations...');
await migrate(db, { migrationsFolder: './drizzle' });
console.log('Migrations complete.');

await client.end();
