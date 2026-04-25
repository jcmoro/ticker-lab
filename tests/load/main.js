import http from 'k6/http';
import { check, group, sleep } from 'k6';
import { htmlReport } from 'https://raw.githubusercontent.com/benc-uk/k6-reporter/main/dist/bundle.js';
import { textSummary } from 'https://jslib.k6.io/k6-summary/0.1.0/index.js';

// ─── Configuration ─────────────────────────────────────────
const API_BASE = __ENV.API_BASE || 'http://host.docker.internal:3000';
const CRYPTO_BASE = __ENV.CRYPTO_BASE || 'http://host.docker.internal:8090';
const MACRO_BASE = __ENV.MACRO_BASE || 'http://host.docker.internal:8110';

export const options = {
  scenarios: {
    smoke: {
      executor: 'constant-vus',
      vus: 5,
      duration: '30s',
      tags: { test_type: 'smoke' },
    },
    load: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '15s', target: 20 },
        { duration: '30s', target: 20 },
        { duration: '15s', target: 0 },
      ],
      startTime: '35s',
      tags: { test_type: 'load' },
    },
  },
  thresholds: {
    http_req_duration: ['p(95)<500', 'p(99)<1000'],
    http_req_failed: ['rate<0.01'],
  },
};

// ─── Scenarios ─────────────────────────────────────────────

function safeJson(r) {
  try { return JSON.parse(r.body); } catch { return null; }
}

export default function () {
  group('Node API — Exchange Rates', () => {
    const latest = http.get(`${API_BASE}/api/v1/exchange-rates/latest`);
    check(latest, {
      'exchange-rates/latest 200': (r) => r.status === 200,
      'has rates array': (r) => { const d = safeJson(r); return d && d.rates && d.rates.length > 0; },
    });

    const history = http.get(
      `${API_BASE}/api/v1/exchange-rates/history?quote=USD&from=2025-01-01&to=2026-04-01`
    );
    check(history, {
      'exchange-rates/history 200': (r) => r.status === 200,
    });
  });

  group('Node API — Converter', () => {
    const convert = http.get(
      `${API_BASE}/api/v1/convert?from=EUR&to=USD&amount=100`
    );
    check(convert, {
      'convert 200': (r) => r.status === 200,
      'has result': (r) => { const d = safeJson(r); return d && d.result > 0; },
    });
  });

  group('Node API — System', () => {
    const health = http.get(`${API_BASE}/health`);
    check(health, { 'health 200': (r) => r.status === 200 });

    const metrics = http.get(`${API_BASE}/metrics`);
    check(metrics, { 'metrics 200': (r) => r.status === 200 });
  });

  group('Go — Crypto', () => {
    const latest = http.get(`${CRYPTO_BASE}/api/v1/crypto/latest`);
    check(latest, {
      'crypto/latest 200': (r) => r.status === 200,
    });

    const history = http.get(
      `${CRYPTO_BASE}/api/v1/crypto/bitcoin/history?days=30`
    );
    check(history, {
      'crypto/history 200': (r) => r.status === 200,
    });
  });

  group('Go — Macro', () => {
    const indicators = http.get(`${MACRO_BASE}/api/v1/macro/indicators`);
    check(indicators, {
      'macro/indicators 200': (r) => r.status === 200,
      'has indicators': (r) => { const d = safeJson(r); return d && d.count > 0; },
    });

    const history = http.get(
      `${MACRO_BASE}/api/v1/macro/fred/CPIAUCSL/history?days=365`
    );
    check(history, {
      'macro/history 200': (r) => r.status === 200,
    });

    const filtered = http.get(
      `${MACRO_BASE}/api/v1/macro/indicators?category=inflation`
    );
    check(filtered, {
      'macro/inflation 200': (r) => r.status === 200,
    });
  });

  sleep(0.5);
}

// ─── Report ────────────────────────────────────────────────

export function handleSummary(data) {
  return {
    '/results/report.html': htmlReport(data),
    stdout: textSummary(data, { indent: ' ', enableColors: true }),
  };
}
