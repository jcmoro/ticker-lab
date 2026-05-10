import { describe, expect, it } from 'vitest';
import {
  buildPaginatedResponse,
  DEFAULT_PAGE_SIZE,
  decodeCursor,
  encodeCursor,
  MAX_PAGE_SIZE,
  PaginationError,
  parsePaginationParams,
} from './pagination.js';

describe('parsePaginationParams', () => {
  it('returns defaults when query is empty', () => {
    expect(parsePaginationParams({})).toEqual({
      pageSize: DEFAULT_PAGE_SIZE,
      pageToken: null,
    });
  });

  it('parses a string page_size', () => {
    expect(parsePaginationParams({ page_size: '50' })).toEqual({
      pageSize: 50,
      pageToken: null,
    });
  });

  it('parses a numeric page_size', () => {
    expect(parsePaginationParams({ page_size: 25 })).toEqual({
      pageSize: 25,
      pageToken: null,
    });
  });

  it('clamps page_size to MAX_PAGE_SIZE', () => {
    expect(parsePaginationParams({ page_size: 9999 }).pageSize).toBe(MAX_PAGE_SIZE);
  });

  it('honors custom maxSize', () => {
    expect(parsePaginationParams({ page_size: 9999 }, { maxSize: 200 }).pageSize).toBe(200);
  });

  it('uses default for page_size=0', () => {
    expect(parsePaginationParams({ page_size: 0 }).pageSize).toBe(DEFAULT_PAGE_SIZE);
  });

  it('throws on negative page_size', () => {
    expect(() => parsePaginationParams({ page_size: -1 })).toThrow(PaginationError);
  });

  it('throws on non-integer page_size', () => {
    expect(() => parsePaginationParams({ page_size: '12.5' })).toThrow(PaginationError);
  });

  it('throws on non-numeric page_size', () => {
    expect(() => parsePaginationParams({ page_size: 'abc' })).toThrow(PaginationError);
  });

  it('extracts page_token when present', () => {
    expect(parsePaginationParams({ page_token: 'opaque' }).pageToken).toBe('opaque');
  });

  it('treats empty page_token as null', () => {
    expect(parsePaginationParams({ page_token: '' }).pageToken).toBeNull();
  });

  it('PaginationError exposes code and statusCode for the error handler', () => {
    try {
      parsePaginationParams({ page_size: -1 });
    } catch (e) {
      expect(e).toBeInstanceOf(PaginationError);
      const err = e as PaginationError;
      expect(err.code).toBe('INVALID_ARGUMENT');
      expect(err.statusCode).toBe(400);
    }
  });
});

describe('cursor encode/decode', () => {
  it('round-trips an object', () => {
    const payload = { last_id: 'ES0138841038', last_date: '2026-05-08' };
    const token = encodeCursor(payload);
    expect(decodeCursor(token)).toEqual(payload);
  });

  it('returns null for null token', () => {
    expect(decodeCursor(null)).toBeNull();
  });

  it('throws on malformed base64', () => {
    expect(() => decodeCursor('not!valid!base64==')).toThrow(PaginationError);
  });

  it('throws on valid base64 but invalid JSON', () => {
    const corrupt = Buffer.from('not json', 'utf8').toString('base64url');
    expect(() => decodeCursor(corrupt)).toThrow(PaginationError);
  });

  it('produces URL-safe tokens (no /, +, =)', () => {
    const token = encodeCursor({ a: 'x'.repeat(200) });
    expect(token).not.toMatch(/[/+=]/);
  });
});

describe('buildPaginatedResponse', () => {
  it('shapes a basic response', () => {
    const r = buildPaginatedResponse('items', [1, 2, 3]);
    expect(r).toEqual({ items: [1, 2, 3], next_page_token: '' });
  });

  it('includes next_page_token when provided', () => {
    const r = buildPaginatedResponse('funds', [], { nextPageToken: 'abc' });
    expect(r.next_page_token).toBe('abc');
  });

  it('includes total_size when provided', () => {
    const r = buildPaginatedResponse('funds', [], { totalSize: 3112 });
    expect(r).toMatchObject({ total_size: 3112 });
  });

  it('omits total_size when not provided', () => {
    const r = buildPaginatedResponse('funds', []);
    expect(Object.hasOwn(r, 'total_size')).toBe(false);
  });

  it('AIP-132 names items field after the resource', () => {
    const r = buildPaginatedResponse('indicators', [{ id: 1001 }]);
    expect(r).toHaveProperty('indicators');
    expect((r as { indicators: readonly unknown[] }).indicators).toHaveLength(1);
  });
});
