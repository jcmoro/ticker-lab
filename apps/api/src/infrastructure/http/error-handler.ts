import type { FastifyError, FastifyReply, FastifyRequest } from 'fastify';
import { RatesNotFoundError } from '../../domain/exchange-rate/errors.js';

interface ProblemDetails {
  type: string;
  title: string;
  status: number;
  detail: string;
  code: string;
}

function problemDetails(
  status: number,
  title: string,
  detail: string,
  code: string,
): ProblemDetails {
  return {
    type: `https://tickerlab.dev/problems/${title.toLowerCase().replace(/ /g, '-')}`,
    title,
    status,
    detail,
    code,
  };
}

export function errorHandler(error: FastifyError, _request: FastifyRequest, reply: FastifyReply) {
  if (error instanceof RatesNotFoundError) {
    return reply
      .status(404)
      .header('content-type', 'application/problem+json')
      .send(problemDetails(404, 'Not Found', error.message, 'RATES_NOT_FOUND'));
  }

  const status = error.statusCode ?? 500;
  const detail = status >= 500 ? 'An unexpected error occurred' : error.message;

  return reply
    .status(status)
    .header('content-type', 'application/problem+json')
    .send(problemDetails(status, 'Internal Server Error', detail, 'INTERNAL_ERROR'));
}
