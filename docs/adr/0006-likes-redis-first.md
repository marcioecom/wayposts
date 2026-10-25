# 0006: Likes no Redis com persistência em lote

- Status: aceito
- Data: 2026-10-25

## Contexto

Assumindo ~5 likes por post, são ~1B likes/dia: ~11.6k/s na média e ~46k/s no pico,
cinco vezes a vazão de posts. Diferente de posts, likes se concentram. Um post viral
recebe dezenas de milhares de likes no mesmo contador, e uma linha
`UPDATE ... SET likes = likes + 1` no Postgres serializa todas as escritas. Além do
contador, o read path precisa responder "este usuário curtiu este post?" para 20 posts
por página, e a operação precisa ser idempotente sob retry e duplo toque.

## Decisão

- **Escrita.** Script Lua atômico no Redis. `SADD likes:{post} {user}` e, se retornar 1,
  `HINCRBY post:{id} likes 1`. Unlike é o espelho, com `SREM` e `HINCRBY -1`.
- **Persistência.** Um worker consome o fluxo de likes e grava em lote na tabela
  `likes(user_id, post_id)` do Postgres, com `ON CONFLICT DO NOTHING`. A tabela é a
  fonte da verdade para reconstrução do contador.
- **Leitura do contador.** Vem no próprio `post:{id}` já hidratado pelo feed, sem
  consulta extra.
- **"Curti?".** `SMISMEMBER` no mesmo pipeline de hidratação, com 20 comandos por
  leitura.
- **TTL** de 7 dias nos SETs de likers, com eviction LRU. Trinta dias dariam ~30B
  entradas. Miss de "curti?" cai para uma query em lote no Postgres.

## Consequências

- Sem linha quente: o contador de um post viral é um `HINCRBY` em memória.
- Idempotência garantida pelo `SADD` dentro do script; retry é inofensivo.
- O contador exibido pode ficar à frente da tabela `likes` pela janela do flush em lote.
  Em restart do Redis, ele é reconstruído a partir da tabela.
- São ~20 comandos extras por leitura de timeline, ~180k/s no pico, dentro do pipeline.
- Posts com milhões de likers ocupam 8 a 16 MB de SET cada. O TTL e a expiração de posts
  antigos limitam o total.
- Perda do Redis implica likes não persistidos na janela do flush. A divergência precisa
  ser aceitável e documentada, não descoberta.
