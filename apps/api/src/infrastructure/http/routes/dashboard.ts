import type { FastifyInstance, FastifyReply, FastifyRequest } from 'fastify';

export async function dashboardRoutes(server: FastifyInstance): Promise<void> {
  server.get('/', async (_request: FastifyRequest, reply: FastifyReply) => {
    return reply.viewAsync('pages/dashboard', {
      title: 'Ticker Lab',
      updatedAt: new Date().toISOString(),
    });
  });
}
