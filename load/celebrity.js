// Cenário de celebridade: escritas de contas acima do threshold (não
// materializadas) e leituras de quem as segue (caminho de merge).
// Ex.: CELEBS=3 RATE=100 k6 run load/celebrity.js
import { createPost, home, thresholds, summaryStats, CELEBS } from './lib.js';

const rate = Number(__ENV.RATE || 50);
const duration = __ENV.DURATION || '3m';
const writePct = Number(__ENV.CELEB_WRITE_PCT || 0.1);

export const options = {
  scenarios: {
    celebrity: {
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
  if (Math.random() < writePct) {
    const celeb = 1 + Math.floor(Math.random() * CELEBS);
    createPost(celeb, 'anúncio');
    return;
  }
  home();
}
