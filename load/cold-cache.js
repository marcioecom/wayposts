// Cache frio: leitura pura logo depois de limpar o cache de corpo no Redis.
// Os feeds continuam válidos; o que falta é `post:{id}`, então toda hidratação
// vai ao Postgres. Como a leitura não reescreve o hash, o caminho frio se
// sustenta durante a execução.
// Rode `make flush-posts` antes e mantenha a taxa baixa: o objetivo é medir o
// custo do miss, não a saturação.
// Ex.: RATE=20 DURATION=2m k6 run load/cold-cache.js
import { home, thresholds, summaryStats } from './lib.js';

const rate = Number(__ENV.RATE || 20);
const duration = __ENV.DURATION || '2m';

export const options = {
  scenarios: {
    cold: {
      executor: 'constant-arrival-rate',
      rate,
      timeUnit: '1s',
      duration,
      preAllocatedVUs: Number(__ENV.PREALLOC_VUS || 30),
      maxVUs: Number(__ENV.MAX_VUS || 500),
      gracefulStop: '30s',
    },
  },
  thresholds: thresholds(),
  summaryTrendStats: summaryStats,
};

export default function () {
  home();
}
