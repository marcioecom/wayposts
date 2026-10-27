// Soak: mesma taxa por muito tempo. Expõe vazamento de memória, crescimento de
// fila, TTL de cache e degradação do Postgres (bloat, conexões).
// Ex.: RATE=80 SOAK=1h k6 run load/soak.js
import { mixed, thresholds, summaryStats } from './lib.js';

const rate = Number(__ENV.RATE || 80);
const soak = __ENV.SOAK || '30m';

export const options = {
  scenarios: {
    soak: {
      executor: 'constant-arrival-rate',
      rate,
      timeUnit: '1s',
      duration: soak,
      preAllocatedVUs: Number(__ENV.PREALLOC_VUS || 80),
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
