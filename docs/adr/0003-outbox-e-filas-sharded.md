# 0003: Eventos por outbox transacional e filas sharded

- Status: aceito
- Data: 2026-10-25

## Contexto

O `post` service grava no Postgres e precisa avisar o fanout. Gravar e publicar não são
atômicos: se o publish falha depois do commit, o post existe e nunca é distribuído, sem
erro visível. O fanout é comutativo e idempotente, porque usa `ZADD` com
`score = postID`, então não há requisito de ordenação e a paralelização é livre. O
broker não deve carregar uma mensagem por seguidor, até ~460k msg/s, apenas uma por
post, ~2.3k/s.

## Decisão

- **Outbox transacional** no `post` service. A criação grava `posts` e `outbox` na mesma
  transação. Um relay embutido lê com `FOR UPDATE SKIP LOCKED` a cada ~20 ms e publica.
- **Uma mensagem `PostCreated` por post**, com `postID` e `authorID`.
- **16 filas** `fanout.<shard>`, com `shard = hash(author_id) % 16`. Competing consumers
  por fila; um post de conta grande fica contido no seu shard.
- **At-least-once** com `ack` após o lote de `ZADD`. Duplicata é inofensiva.
- **Retry** com backoff exponencial; após 3 tentativas, DLQ com métrica e alerta.
- **TTL de mensagem de 5 min.** Fanout atrasado além disso é descartado.

## Consequências

- Nenhum post é perdido entre commit e publicação.
- O broker opera em ~2.3k msg/s; o custo real fica nos workers e no Redis.
- O fanout pode ser paralelizado e reexecutado sem efeito colateral.
- Latência extra de 20 a 50 ms entre commit e início do fanout, dentro do SLO de 2 s.
- A tabela `outbox` cresce e precisa de poda. O relay é um ponto único de falha do
  serviço até ser replicado.
- Posts que passem de 5 min sem fanout não são recuperados. Se o worker ficar fora do ar
  além disso, seguidores perdem esses posts. Um job de reconciliação é o próximo passo
  caso isso se torne inaceitável.
- Entrega duplicada é esperada por desenho. Qualquer consumidor futuro precisa ser
  idempotente.
