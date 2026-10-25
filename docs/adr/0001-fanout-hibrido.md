# 0001: Fanout híbrido para Home Timelines

- Status: aceito
- Data: 2026-10-25

## Contexto

20M DAU × 10 posts/dia ≈ 2.3k posts/s (pico estimado ~9k/s). O custo do fanout push
cresce com tráfego × seguidores. O grafo é power-law, com marcas e lojistas: uma conta
com 10M de seguidores gera 10M operações por post. Uma mensagem dessas na mesma fila
dos posts comuns bloqueia o consumo e viola o frescor do feed para todos os usuários.

## Decisão

Fanout híbrido com threshold configurável (default: 10.000 seguidores).

- **Abaixo do threshold.** O worker empurra o post para o feed de cada seguidor, no modo
  push.
- **Acima do threshold.** Não há fanout. O post vive na fonte de verdade e o read path
  busca os posts recentes das celebridades seguidas, em cache, e faz k-way merge com o
  feed empurrado.
- O threshold é variável de experimento. O relatório compara o custo de push e de merge
  nos cortes de 1k, 10k e 100k seguidores.

## Consequências

- O custo de escrita deixa de crescer com o tamanho da audiência das maiores contas.
- A memória do feed não é consumida pelos seguidores de celebridades.
- O read path passa a ter duas fontes: exige merge, dedupe e paginação por cursor entre
  o feed empurrado e as listas de posts das celebridades.
- O merge só é barato com ID monotônico gerado por Snowflake. Sem isso, vira ordenação
  em memória.
- Follow/unfollow de celebridade tem efeito imediato na leitura, sem backfill.

## Alternativas descartadas

- **Push puro.** 10M ZADD por post de marca e head-of-line blocking na fila.
- **Pull puro.** Até 500 autores por abertura de timeline, leitura caríssima.
- **Três faixas com usuários ativos.** Adiado. Reduz memória, mas introduz timeline fria
  e backfill, mais um modo de falha para o laboratório explicar.
