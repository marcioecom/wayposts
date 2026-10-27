// Helpers compartilhados pelos cenários k6.
//
// Modelo aberto: quem define a carga é a taxa de chegada (iterations/s), não o
// número de VUs. Assim a latência alta não reduz a carga sozinha e o joelho da
// curva aparece na taxa em que o sistema satura.

import http from 'k6/http';
import encoding from 'k6/encoding';
import { check } from 'k6';

export const BASE = __ENV.BASE || 'http://localhost:8080';
export const USERS = Number(__ENV.USERS || 2000);
export const CELEBS = Number(__ENV.CELEBS || 3);
export const TIMELINE_LIMIT = Number(__ENV.TIMELINE_LIMIT || 20);
export const ENFORCE = __ENV.ENFORCE !== '0';

// SLOs do laboratório (docs/architecture.md). A carga roda contra o gateway,
// então a latência medida aqui inclui proxy + serviço + Redis/Postgres.
export const sloThresholds = {
  'http_req_failed': ['rate<0.001'],
  'http_req_duration{kind:home}': ['p(99)<150'],
  'http_req_duration{kind:profile}': ['p(99)<150'],
  'http_req_duration{kind:post}': ['p(99)<300'],
  'http_req_duration{kind:like}': ['p(99)<100'],
  'http_req_duration{kind:presign}': ['p(99)<150'],
  'http_req_duration{kind:upload}': ['p(99)<1000'],
};

export const summaryStats = ['avg', 'min', 'med', 'p(90)', 'p(95)', 'p(99)', 'max'];

export function thresholds() {
  return ENFORCE ? sloThresholds : {};
}

// randomUser sorteia um ID válido do seed. Math.random distribui melhor que
// __VU/__ITER quando o número de VUs é pequeno e a taxa é alta.
export function randomUser() {
  return 1 + Math.floor(Math.random() * USERS);
}

export function randomAuthor() {
  return 1 + Math.floor(Math.random() * USERS);
}

function params(kind, uid) {
  return {
    headers: {
      Authorization: `Bearer ${uid}`,
      'Content-Type': 'application/json',
    },
    tags: { kind },
    timeout: '10s',
  };
}

function ok(res, kind) {
  check(res, {
    [`${kind} 2xx`]: (r) => r.status >= 200 && r.status < 300,
  });
  return res;
}

export function home(uid = randomUser()) {
  const res = http.get(
    `${BASE}/v1/timeline/home?limit=${TIMELINE_LIMIT}`,
    params('home', uid),
  );
  return ok(res, 'home');
}

export function profile(author = randomAuthor(), uid = randomUser()) {
  const res = http.get(
    `${BASE}/v1/timeline/profile/${author}?limit=${TIMELINE_LIMIT}`,
    params('profile', uid),
  );
  return ok(res, 'profile');
}

export function createPost(uid = randomUser(), body = 'carga') {
  const payload = JSON.stringify({
    body: `${body} ${Math.random().toString(36).slice(2, 10)}`,
  });
  const res = http.post(`${BASE}/v1/posts`, payload, params('post', uid));
  return ok(res, 'post');
}

// createPostWithMedia exercita presign + PUT direto no bucket + validação por
// HEAD. Fica atrás de MEDIA=1 porque triplica as chamadas por post.
const PNG = 'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8DwHwAFAAH/q842iQAAAABJRU5ErkJggg==';

export function createPostWithMedia(uid = randomUser()) {
  const presign = http.post(
    `${BASE}/v1/media/presign`,
    JSON.stringify({ items: [{ content_type: 'image/png', size: 70 }] }),
    params('presign', uid),
  );
  ok(presign, 'presign');
  if (presign.status !== 200) {
    return presign;
  }

  const item = JSON.parse(presign.body).items[0];
  const upload = http.put(item.url, encoding.b64decode(PNG, 'std'), {
    headers: item.headers,
    tags: { kind: 'upload' },
    timeout: '10s',
  });
  ok(upload, 'upload');
  if (upload.status < 200 || upload.status >= 300) {
    return upload;
  }

  const res = http.post(
    `${BASE}/v1/posts`,
    JSON.stringify({ body: 'post com mídia', media: [item.key] }),
    params('post', uid),
  );
  return ok(res, 'post');
}

// likeFirstPost lê a timeline e curte o primeiro item. Custa duas chamadas, mas
// usa um post que o usuário realmente veria.
export function likeFirstPost(uid = randomUser()) {
  const feed = home(uid);
  if (feed.status !== 200) {
    return feed;
  }
  const items = JSON.parse(feed.body).items || [];
  if (items.length === 0) {
    return feed;
  }
  const pid = items[Math.floor(Math.random() * items.length)].id;
  const res = http.post(`${BASE}/v1/posts/${pid}/likes`, null, params('like', uid));
  return ok(res, 'like');
}

let mediaEnabled = __ENV.MEDIA === '1';

// mixed é a distribuição padrão: leitura domina, escrita é minoria. Ajuste via
// HOME_PCT/PROFILE_PCT/POST_PCT se o cenário pedir outra mistura.
export function mixed() {
  const homePct = Number(__ENV.HOME_PCT || 0.85);
  const profilePct = Number(__ENV.PROFILE_PCT || 0.05);
  const postPct = Number(__ENV.POST_PCT || 0.07);
  const r = Math.random();

  if (r < homePct) {
    return home();
  }
  if (r < homePct + profilePct) {
    return profile();
  }
  if (r < homePct + profilePct + postPct) {
    return mediaEnabled ? createPostWithMedia() : createPost();
  }
  return likeFirstPost();
}

export { http, check };
