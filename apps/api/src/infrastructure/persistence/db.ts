import { drizzle } from 'drizzle-orm/postgres-js';
import postgres from 'postgres';
import { MissingEnvVarError } from '../errors.js';
import * as schema from './schema.js';

const connectionString = process.env.DATABASE_URL;
if (!connectionString) {
  throw new MissingEnvVarError('DATABASE_URL');
}

const client = postgres(connectionString);
export const db = drizzle(client, { schema });
