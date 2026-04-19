import { readFileSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import type { FastifyInstance, FastifyReply, FastifyRequest } from 'fastify';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const specPath = path.join(__dirname, '..', '..', '..', '..', 'openapi.yaml');

export async function apiDocsRoutes(server: FastifyInstance): Promise<void> {
  server.get('/api/openapi.yaml', async (_request: FastifyRequest, reply: FastifyReply) => {
    const spec = readFileSync(specPath, 'utf-8');
    return reply.type('text/yaml').send(spec);
  });

  server.get('/api/docs', async (_request: FastifyRequest, reply: FastifyReply) => {
    return reply.type('text/html').send(`<!DOCTYPE html>
<html>
<head>
  <title>Ticker Lab — API Docs</title>
  <meta charset="utf-8"/>
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <style>body { margin: 0; }</style>
</head>
<body>
  <redoc spec-url="/api/openapi.yaml"></redoc>
  <script src="https://cdn.redoc.ly/redoc/latest/bundles/redoc.standalone.js"></script>
</body>
</html>`);
  });
}
