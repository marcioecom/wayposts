# 0002: Feed materializado como conjunto de IDs, hidratado em lote

- Status: aceito
- Data: 2026-10-25

## Contexto

A Home Timeline entrega ~20 posts por abertura, a ~2.3k aberturas/s, com pico de ~9k/s.
Denormalizar o corpo do post dentro do feed multiplica a escrita pelo número de
seguidores: 200M posts/dia × 200 seguidores ≈ 40 TB/dia. Também transforma um like ou
um delete em invalidação em cascata por todos os seguidores. O feed precisa de ordem
estável, do mais recente para o mais antigo, e paginação por cursor.

## Decisão

- `feed:{user}`: ZSET no Redis Cluster. `member` = postID, um Snowflake int64, e
  `score` = postID. Cap de 500 entradas, sem TTL.
- `post:{id}`: HASH com autor, texto, mídia, `created_at`, contador de likes e flag de
  deletado. TTL de 24h com eviction LRU. Sete dias de corpos a 200M posts/dia × ~1KB
  dariam ~1,4 TB.
- Hidratação: `ZREVRANGEBYSCORE` para 20 IDs, pipeline de `HGETALL`, misses em lote no
  Postgres. Posts deletados são pulados e removidos do feed de forma preguiçosa.
- `author:{user}`: ZSET com os últimos 200 posts do autor. Serve a Profile Timeline, o
  merge do próprio autor, para read-your-writes, e o merge de celebridades.
- `celeb_follows:{user}`: ZSET de celebridade para `follow_at`. Lista o que puxar no
  merge e aplica o filtro `post.created_at >= follow_at`.

## Consequências

- Escrita de post é O(1) no corpo; like e delete tocam um único objeto.
- Paginação por cursor vira `ZREVRANGEBYSCORE` com `max = cursor`, sem OFFSET.
- Memória previsível: 20M usuários × 500 entradas × ~32 bytes ≈ 300 GB de RAM, ~120 GB
  com cap 200. O cap é variável de experimento.
- Read path paga N+1 controlado, com 20 hidratações, e depende da taxa de hit do cache.
- Corpo com TTL menor que o feed exige fallback ao Postgres e um caminho de miss que
  precisa ser medido, não assumido.
- Cap fixo faz usuário inativo perder o histórico do feed. Ao voltar, ele vê os últimos
  N posts.
