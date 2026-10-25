import http from 'k6/http';
import { Counter, Trend } from 'k6/metrics';

const successes = new Counter('evaluation_successes');
const throttled = new Counter('evaluation_throttled');
const errors = new Counter('evaluation_errors');
const latency = new Trend('evaluation_wall_ms', true);
const successfulLatency = new Trend('evaluation_success_wall_ms', true);
const rate = Number(__ENV.LOAD_RATE);
export const options = {
  scenarios: { evaluation: { executor: 'constant-arrival-rate', rate, timeUnit: '1s',
    duration: '30s', preAllocatedVUs: 20, maxVUs: 20, gracefulStop: '10s' } },
  summaryTrendStats: ['min', 'med', 'p(95)', 'p(99)', 'max'],
  thresholds: { evaluation_errors: ['count==0'], dropped_iterations: ['count==0'] },
  maxRedirects: 0,
};
export default function () {
  // Zero samples guarantee explicit counters even when an outcome never occurs.
  successes.add(0); throttled.add(0); errors.add(0);
  const started = Date.now();
  const res = http.post(`${__ENV.LOAD_URL}/api/v1/evaluate`, JSON.stringify({
    flagKey: 'load_flag', context: { bucketingKey: `synthetic-${__VU}-${__ITER % 100}` },
  }), { headers: { Authorization: `Bearer ${__ENV.LOAD_TOKEN}`, 'Content-Type': 'application/json' }, timeout: '5s' });
  const elapsed = Date.now() - started;
  latency.add(elapsed);
  if (res.status === 429) { throttled.add(1); return; }
  let body;
  try { body = res.json(); } catch (_) { errors.add(1); return; }
  if (res.status !== 200 || body.flagKey !== 'load_flag' || body.value !== true || body.reason !== 'ENABLED' || body.environment !== __ENV.LOAD_ENV) {
    errors.add(1); return;
  }
  successes.add(1); successfulLatency.add(elapsed);
}
export function handleSummary(data) {
  // Only aggregated metrics; no bodies, headers, context, URLs or environment values.
  const selected = {};
  for (const name of ['http_reqs', 'http_req_duration', 'http_req_failed', 'iterations', 'dropped_iterations',
    'evaluation_successes', 'evaluation_throttled', 'evaluation_errors', 'evaluation_wall_ms', 'evaluation_success_wall_ms']) {
    selected[name] = data.metrics[name]?.values || { count: 0 };
  }
  return { stdout: JSON.stringify(selected) };
}
