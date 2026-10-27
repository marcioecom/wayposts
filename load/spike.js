// Pico súbito: 10x a taxa base por 1 minuto. Mede fila, autoscaling e quanto
// tempo o sistema leva para voltar ao normal.
// Ex.: BASE_RATE=100 SPIKE_FACTOR=10 k6 run load/spike.js
import { mixed, thresholds, summaryStats } from './lib.js';

const base = Number(__ENV.BASE_RATE || 100);
const factor = Number(__ENV.SPIKE_FACTOR || 10);

export const options = {
  scenarios: {
    spike: {
      executor: 'ramping-arrival-rate',
      startRate: 0,
      timeUnit: '1s',
      preAllocatedVUs: Number(__ENV.PREALLOC_VUS || 200),
      maxVUs: Number(__ENV.MAX_VUS || 4000),
      stages: [
        { target: base, duration: '30s' },
        { target: base * factor, duration: '10s' },
        { target: base * factor, duration: __ENV.SPIKE_DURATION || '60s' },
        { target: base, duration: '30s' },
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
