/**
 * Pagination helpers — implements Google AIP-158.
 * See docs/api-design-standards.md §1.3.
 */

export const DEFAULT_PAGE_SIZE = 100;
export const MAX_PAGE_SIZE = 500;

export type PaginationParams = {
  readonly pageSize: number;
  readonly pageToken: string | null;
};

export type PaginatedResponse<T, K extends string> = { readonly next_page_token: string } & {
  readonly [P in K]: readonly T[];
} & { readonly total_size?: number };

export class PaginationError extends Error {
  readonly statusCode = 400;
  readonly code = 'INVALID_ARGUMENT';
  constructor(message: string) {
    super(message);
    this.name = 'PaginationError';
  }
}

type RawQuery = {
  readonly page_size?: string | number;
  readonly page_token?: string;
};

export function parsePaginationParams(
  query: RawQuery,
  options: { defaultSize?: number; maxSize?: number } = {},
): PaginationParams {
  const defaultSize = options.defaultSize ?? DEFAULT_PAGE_SIZE;
  const maxSize = options.maxSize ?? MAX_PAGE_SIZE;

  let pageSize = defaultSize;
  if (query.page_size !== undefined && query.page_size !== '') {
    const n = Number(query.page_size);
    if (!Number.isInteger(n)) {
      throw new PaginationError(`page_size must be an integer, got "${query.page_size}"`);
    }
    if (n < 0) {
      throw new PaginationError(`page_size must be non-negative, got ${n}`);
    }
    pageSize = n === 0 ? defaultSize : Math.min(n, maxSize);
  }

  const rawToken = query.page_token;
  const pageToken = typeof rawToken === 'string' && rawToken.length > 0 ? rawToken : null;

  return { pageSize, pageToken };
}

export function encodeCursor(payload: Record<string, unknown>): string {
  return Buffer.from(JSON.stringify(payload), 'utf8').toString('base64url');
}

export function decodeCursor<T = Record<string, unknown>>(token: string | null): T | null {
  if (!token) return null;
  let json: string;
  try {
    json = Buffer.from(token, 'base64url').toString('utf8');
  } catch {
    throw new PaginationError('page_token is malformed');
  }
  try {
    return JSON.parse(json) as T;
  } catch {
    throw new PaginationError('page_token is malformed');
  }
}

export function buildPaginatedResponse<T, K extends string>(
  itemsKey: K,
  items: readonly T[],
  options: { nextPageToken?: string; totalSize?: number } = {},
): PaginatedResponse<T, K> {
  const base = {
    [itemsKey]: items,
    next_page_token: options.nextPageToken ?? '',
  } as Record<string, unknown>;
  if (options.totalSize !== undefined) {
    base.total_size = options.totalSize;
  }
  return base as PaginatedResponse<T, K>;
}
