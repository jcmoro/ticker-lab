import { buildServer } from './infrastructure/http/server.js';

const start = async (): Promise<void> => {
  const server = await buildServer();

  const port = Number(process.env.API_PORT ?? 3000);
  const host = process.env.API_HOST ?? '0.0.0.0';

  await server.listen({ port, host });
};

start().catch((err: unknown) => {
  console.error('Failed to start server:', err);
  process.exit(1);
});
