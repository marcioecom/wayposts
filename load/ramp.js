// Sobe a taxa de chegada até a saturação e segura no topo para confirmar o
// joelho. É o cenário que produz a curva de capacidade.
// Ex.: START_RATE=20 MAX_RATE=1500 RAMP_DURATION=10m k6 run load/ramp.js
import { mixed, thresholds, summaryStats } from './lib.js';

const startRate = Number(__ENV.START_RATE || 20);
const maxRate = Number(__ENV.MAX_RATE || 1000);
const hold = __ENV.HOLD_DURATION || '2m';

export const options = {
  scenarios: {
    ramp: {
      executor: 'ramping-arrival-rate',
      startRate,
      timeUnit: '1s',
      preAllocatedVUs: Number(__ENV.PREALLOC_VUS || 100),
      maxVUs: Number(__ENV.MAX_VUS || 2000),
      stages: [
        { target: startRate, duration: __ENV.WARMUP_DURATION || '1m' },
        { target: maxRate, duration: __ENV.RAMP_DURATION || '10m' },
        { target: maxRate, duration: hold },
      ],
      gracefulStop: '30s',
    },
  },
  thresholds: thresholds(),
  summaryTrendStats: summaryStats,
};

export default function () {
  mixed();
}
