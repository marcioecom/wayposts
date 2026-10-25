# 0007: Listas de seguidores em cache chunked

- Status: aceito
- Data: 2026-10-25

## Contexto

O fanout precisa de ~460k IDs de seguidores por segundo na média, 2.3k posts/s × 200, e
~1.8M/s no pico. É uma consulta por post retornando ~200 linhas, não 460k consultas, mas
a soma das linhas e a memória do grafo, ~4B arestas, tornam a origem da lista uma
decisão de capacidade, não de conveniência. Posts de contas com 9k seguidores também não
podem monopolizar um worker por muito tempo.

## Decisão

- Chaves `followers:{author}:{chunk}`: SETs de 1k IDs, com TTL de 24 h.
- Miss reconstrói a partir do Postgres com `singleflight`, uma query por autor, não uma
  por worker concorrente. Follow/unfollow atualiza o chunk aberto com `SADD`/`SREM`;
  mudança estrutural invalida e reconstrói.
- O worker processa chunk por chunk, acumulando `ZADD` em pipeline, e só dá `ack` no fim.
- A origem é uma interface plugável, com implementação direta no Postgres para servir de
  baseline no mesmo harness de carga.

## Consequências

- Uma leitura por chunk em vez de uma por seguidor; ~2.3k `SMEMBERS`/s na média.
- Duração de tarefa limitada por chunk: um post grande não trava a fila inteira.
- Consistência de follow/unfollow fica no cache, sem esperar TTL.
- A memória cresce com as arestas dos autores ativos. É o custo que o baseline em
  Postgres não paga.
- Miss em rajada, com cache reiniciado, vira thundering herd no Postgres sem
  `singleflight`.
- Divergência entre cache e tabela é um modo de falha silencioso. Precisa de
  reconciliação periódica e de uma métrica de tamanho divergente.

## Implementação

- `Follow`/`Unfollow` incrementam `followers_gen:{author}` antes de tocar nos chunks. A
  reconstrução lê a geração, consulta o Postgres, grava os chunks e relê a geração. Se
  mudou, descarta o resultado e tenta de novo, até 3 vezes. Sem isso, um follow
  concorrente à reconstrução desapareceria do cache até o TTL.
- `followers_meta:{author}` guarda o número de chunks e expira 1 h antes deles, então o
  meta sempre expira primeiro e a próxima leitura reconstrói em vez de ler um conjunto
  parcial.
- A paginação é aplicada em memória sobre a lista ordenada. É aceitável porque o fanout
  verifica o contador e nem lista autores acima do threshold. O pior caso é o próprio
  threshold, 10k IDs no laboratório.
- O `followers_meta` também cacheia o caso "sem seguidores", com 0 chunks, evitando
  consultar o Postgres a cada post de autor novo.
- `wayposts_follower_cache_total{result}` separa `hit`, `miss`, `rebuild`, `stale_retry`
  e `error`.
