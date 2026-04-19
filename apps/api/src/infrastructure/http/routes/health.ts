import type { FastifyInstance } from 'fastify';
import type { Sql } from 'postgres';

export function healthRoutes(db?: Sql) {
  return async (server: FastifyInstance): Promise<void> => {
    server.get('/health', async () => {
      return {
        status: 'ok' as const,
        timestamp: new Date().toISOString(),
      };
    });

    server.get('/ready', async (_request, reply) => {
      let dbStatus: 'ok' | 'error' = 'error';

      if (db) {
        try {
          await db`SELECT 1`;
          dbStatus = 'ok';
        } catch {
          dbStatus = 'error';
        }
      }

      const status = dbStatus === 'ok' ? 'ready' : 'not_ready';
      const statusCode = status === 'ready' ? 200 : 503;

      return reply.status(statusCode).send({
        status,
        checks: { database: dbStatus },
        timestamp: new Date().toISOString(),
      });
    });
  };
}
