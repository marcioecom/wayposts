# Arquitetura do Wayposts

Laboratório de carga para um feed social no estilo do board da Stone. O objetivo é
medir onde o desenho rompe e produzir números defensáveis sobre fanout, leitura de
timeline e frescor.

- Escopo funcional: post com texto e até 4 imagens, like, follow, Home Timeline,
  Profile Timeline, delete soft e nudge de novos posts por WebSocket.
- Fora de escopo: replies, reposts, busca, DM, notificações persistidas, ranking.
- Vocabulário: [`CONTEXT.md`](../CONTEXT.md). Decisões: [`docs/adr/`](adr/).

## 1. Requisitos e premissas

| Requisito | Valor |
|---|---|
| DAU | 20.000.000 |
| Posts por usuário/dia | 10 |
| Aberturas de app por usuário/dia | 10 |
| Seguidores por autor | power-law: mediana ~150, p99 ~5k, máx 1M, perfil realista |
| Likes por post, premissa | ~5 |
| Posts com mídia, premissa | ~20% |
| Conexões WS concorrentes | 5 a 10% do DAU |

| Derivado | Média | Pico, ~4× |
|---|---|---|
| Posts/s | 2.315 | 9.260 |
| Leituras de timeline/s | 2.315 | 9.260 |
| Likes/s | 11.574 | 46.296 |
| Fanout em ZADD/s | ~463k | ~1,85M |
| Uploads de mídia/s | ~463 | ~1.852 |

## 2. Componentes

| Serviço | Responsabilidade | Dados que possui |
|---|---|---|
| `gateway` | REST, auth JWT, rate limit, WebSocket e nudge | nada, stateless |
| `post` | criar/ler/deletar post, presign de mídia, likes, outbox | Postgres `post`; Redis `post:{id}`, `author:{u}`, `likes:{post}` |
| `timeline` | Home Timeline, Profile Timeline, merge e hidratação | Redis `feed:{u}` |
| `social` | usuários, follow/unfollow, contagem e chunks de seguidores | Postgres `social`; Redis `followers:{author}:{chunk}`, `celeb_follows:{u}` |
| `fanout-worker` | consumir `PostCreated` e materializar o feed | nada; usa o mesmo código do `timeline` |

Transporte interno: HTTP/JSON com endpoints de lote, como `GET /internal/posts?ids=`.
O gargalo é o Redis, não o hop entre serviços. gRPC fica como otimização se a medição
mostrar que o hop importa.

## 3. Fluxos

### 3.1 Criar post

```
cliente → gateway → post
  1. POST /media/presign            → URL assinada (R2/S3), sem bytes no serviço
  2. cliente faz PUT direto no bucket
  3. POST /posts {texto, media[]}   → HEAD valida objeto
  4. transação: INSERT posts + INSERT outbox
  5. relay do outbox → RabbitMQ fanout.<hash(author)>
  6. 201 Created (não espera fanout)
```

### 3.2 Fanout

```
fanout-worker ← fanout.<shard>
  1. lê PostCreated {postID, authorID}
  2. social.GetFollowerCount(author)
  3. se count > threshold (default 10k): ack e fim  ← celebridade, lida no merge
  4. senão: para cada chunk de 1k seguidores (social.GetFollowerChunk):
       pipeline ZADD feed:{follower} score=postID member=postID
       ZREMRANGEBYRANK feed:{follower} 0 -501        ← cap 500
  5. ack no fim; erro → retry com backoff → DLQ
```

### 3.3 Ler Home Timeline

```
cliente → gateway → timeline
  1. ZREVRANGEBYSCORE feed:{u} (cursor) -inf LIMIT 0 40
  2. para cada celebridade em celeb_follows:{u}:
       ZREVRANGEBYSCORE author:{celeb} (cursor) -inf LIMIT 0 20
       filtra post.created_at >= follow_at
  3. merge k-way por postID desc + dedupe
  4. adiciona os últimos posts do próprio autor (read-your-writes)
  5. hidrata em pipeline: HGETALL post:{id} + SMISMEMBER likes:{id} {u}
     miss → post.GetPosts(ids) em lote
     deleted → pula e remove do feed preguiçosamente
  6. devolve 20 itens + cursor (último postID)
```

### 3.4 Like

```
cliente → gateway → post
  script Lua: SADD likes:{post} {user}; se 1 → HINCRBY post:{id} likes 1
  evento em buffer → flush em lote (500 eventos ou 100ms) → INSERT likes ON CONFLICT
```

### 3.5 Nudge

```
gateway, a cada 30s, varre as conexões em lotes:
  ZREVRANGE feed:{u} 0 0 WITHSCORES
  se head > last_seen → ZCOUNT (last_seen, +inf) → WS {type:"new_posts", count}
```

O fanout não conhece o WebSocket. Qualquer gateway serve qualquer conexão.

## 4. Modelo de dados

### Postgres

| Schema | Tabela | Índices relevantes |
|---|---|---|
| `post` | `posts(id, author_id, text, media, created_at, deleted, like_count)` | PK `id`, `(author_id, id desc)` |
| `post` | `likes(user_id, post_id)` | PK `(user_id, post_id)` |
| `post` | `outbox(id, event_type, payload, created_at, published_at)` | `(published_at, id)` |
| `social` | `users(id, handle, display_name)` | PK `id`, unique `handle` |
| `social` | `follows(follower_id, followee_id, created_at)` | PK `(follower_id, followee_id)`, `(followee_id)` |

Retenção de posts: 30 dias no laboratório. Sem retenção, 200M posts/dia × ~1KB estoura
o disco em uma semana.

### Redis

| Chave | Tipo | Conteúdo | Retenção |
|---|---|---|---|
| `feed:{u}` | ZSET | postID com score = postID | cap 500 |
| `author:{u}` | ZSET | últimos 200 postIDs | cap 200 |
| `post:{id}` | HASH | corpo + contadores + `deleted` | TTL 24h + LRU |
| `likes:{post}` | SET | userIDs que curtiram | TTL 7d + LRU |
| `followers:{author}:{chunk}` | SET | até 1k seguidores | TTL 24h |
| `celeb_follows:{u}` | ZSET | celebridade → follow_at | mantido no follow |

LRU e TTL resolvem problemas diferentes. 200M posts/dia × 1KB × 7 dias dá ~1,4TB de
corpos, e 30B de likes em 30 dias. O cache guarda o subconjunto quente com `allkeys-lru`
e `maxmemory` definido, e o resto cai para o Postgres. TTL sozinho não fecha a conta.

## 5. Limites conhecidos

- **Memória do feed.** 20M × 500 × ~32B ≈ 320GB; com cap 200, ~128GB. É o maior custo
  fixo do desenho e o motivo de o cap ser variável de experimento.
- **Cache de corpos.** Dimensionado por `maxmemory`, não por retenção. A taxa de hit
  vira métrica do relatório.
- **Celebridades.** O índice `celeb_follows` é escrito no follow; mudar o threshold
  exige reconstrução. Aceitável no laboratório, documentado como limitação.
- **Delete.** A remoção do feed é preguiçosa, no read. Não há fanout de delete; o custo
  seria o mesmo do post.
- **Likes.** Janela de perda de até 100ms no buffer antes do flush. Likes não
  persistidos nessa janela desaparecem em crash do `post`.
- **Threshold.** Autor que cruza o limite continua com posts antigos no feed de quem já
  o seguia; só os novos deixam de ser empurrados.

## 6. SLOs e saturação

| Operação | p99 | Observação |
|---|---|---|
| `POST /posts` | ≤ 300ms | não inclui fanout |
| `GET /timeline` | ≤ 150ms | 20 itens, hidratação e merge |
| `POST/DELETE /likes` | ≤ 100ms | script Lua |
| Fanout lag | p95 ≤ 2s / p99 ≤ 10s | principal indicador arquitetural |
| Erros 5xx | < 0,1% | — |
| Nudge | ≤ 30s | comportamento de produto |

Saturação é o primeiro patamar em que um SLO quebra por 5 min seguidos ou o erro passa
de 1%. O relatório publica vazão por vCPU, ponto de joelho e a curva de latência.

## 7. Plano de testes

Modelo aberto, com taxa de chegada independente da latência, usando k6. Modelo fechado
esconde o joelho por coordinated omission.

| Cenário | Objetivo |
|---|---|
| Steady | linha de base e estabilidade |
| Ramp | encontrar o joelho |
| Spike de 2× em 30s | fila, backlog e recuperação |
| Soak de 1h a 70% do joelho | vazamento, GC, crescimento de memória |
| Rajada de celebridade | validar o merge sob o pior caso |
| Cache frio / restart do gateway | reconstrução e reconnect storm |
| Fanout com threshold 1k/10k/100k | curva de custo push vs merge |

Escalas: 1:1000, com 20k usuários, para smoke; 1:20 a 1:100, ou 1M a 200k usuários, para
saturação, com dois fatores para provar linearidade.

Os cenários, variáveis e a metodologia de leitura da curva estão em
[`load/README.md`](load/README.md).

## 8. Infra em duas fases

**Fase 0: orbit em 1:1000.** Namespace `wayposts` no cluster atual. Reaproveita o
operador CloudNativePG, Prometheus/Grafana/Loki e o Traefik. Valkey e RabbitMQ sobem
dedicados no namespace, com uma réplica cada, para não compartilhar memória nem chaves
com o Echo. Serve para validar deploy, contrato, métricas e pipeline de dados, não para
concluir capacidade.

Status: deployado em 27/10/2026 no worker de Helsinki, enquanto o de Falkenstein está em
~89% de CPU requests. 20k usuários e 4,03M arestas; smoke de 50 iters/s com 0% de erro,
p99 de 34ms e fanout lag p99 de 218ms; rampa até 200 iters/s sem erro. Manifests e
runbook em [`deploy/k8s/README.md`](../deploy/k8s/README.md).

**Fase 1: nós temporários.** 2 a 3 CX33/CCX13 na mesma região do gerador, k3s efêmero ou
node pool temporário, gerador em VPS separada. Cada experimento sobe o ambiente, mede e
destrói. Os charts são os mesmos, com values diferentes.

O gerador nunca roda no mesmo nó dos serviços medidos, e nunca sobre WAN, porque RTT de
~200ms mascararia qualquer p99.

## 9. O que o relatório precisa provar

1. Fanout híbrido sustenta o perfil extremo sem violar o SLO de frescor dos demais?
2. Onde está o joelho do fanout em ZADD/s por vCPU e quantos shards/workers ele exige?
3. Qual o custo do merge de celebridades no p99 da timeline, por número de celebridades seguidas?
4. Qual a taxa de hit do cache de corpos e quanto ela custa em latência e em Postgres?
5. Qual o threshold ótimo, entre 1k, 10k e 100k, para o grafo realista?
6. Quanto de memória o feed consome por usuário e como o cap afeta retenção percebida?
