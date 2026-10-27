// Carga constante com a mistura padrão (leitura dominante).
// Ex.: RATE=200 DURATION=5m k6 run load/steady.js
import { mixed, thresholds, summaryStats } from './lib.js';

const rate = Number(__ENV.RATE || 50);
const duration = __ENV.DURATION || '2m';

export const options = {
  scenarios: {
    steady: {
      executor: 'constant-arrival-rate',
      rate,
      timeUnit: '1s',
      duration,
      preAllocatedVUs: Number(__ENV.PREALLOC_VUS || 50),
      maxVUs: Number(__ENV.MAX_VUS || 1000),
      gracefulStop: '30s',
    },
  },
  thresholds: thresholds(),
  summaryTrendStats: summaryStats,
};

export default function () {
  mixed();
}
