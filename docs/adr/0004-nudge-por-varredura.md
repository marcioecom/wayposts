# 0004: Nudge de novos posts por varredura no gateway

- Status: aceito
- Data: 2026-10-25

## Contexto

O produto precisa do aviso "Mostrar N posts" quando a Home Timeline recebe posts
enquanto o usuário está com a tela aberta. O aviso carrega apenas um contador, não o
conteúdo. Gerá-lo dentro do fanout exigiria consultar presença por seguidor por post,
somando ~460k `GET`/s aos ~460k `ZADD`/s. É caro para entregar um booleano, e acoplaria
o write path ao WebSocket.

## Decisão

- O gateway mantém as conexões WS e autentica no handshake com JWT.
- Uma varredura periódica, 30 s por default e configurável, percorre as conexões em
  lotes por shard, sem um timer ou goroutine por conexão.
- Para cada conexão: `ZREVRANGE feed:{u} 0 0 WITHSCORES` para o head. Se `head >
  last_seen`, envia `{type: "new_posts", count}`. O `count` sai de `ZCOUNT` com o
  cursor do cliente.
- `last_seen` chega no handshake. Depois disso o hub avança o cursor para o head
  observado a cada varredura, e o cliente reconecta com o último ID que carregou.
- O fanout não conhece o WebSocket. Qualquer instância do gateway serve qualquer
  conexão, sem presença roteada e sem sticky session.

## Consequências

- O write path permanece independente do WS, e não há custo de nudge por post.
- O custo do nudge é proporcional a conexões, não a posts × seguidores: 2M conexões /
  30 s ≈ 66k leituras/s no cluster, ~1.7k/s por gateway de 50k conexões.
- Escala horizontal do gateway é trivial: mais instâncias, mais conexões.
- A latência do aviso pode chegar a 30 s. É o comportamento aceito do produto.
- Reinício do gateway gera reconexão em massa. O cliente precisa de backoff com jitter.
- O nudge pode chegar para um post já visto, por corrida entre fetch e varredura. O
  cliente descarta por ID.
- A varredura lê só `feed:{u}`, então posts de celebridade, que o merge traz na leitura,
  não disparam nudge. Aceito na v1, porque o caminho de merge já não materializa o post.
  Se isso virar requisito, a saída é um índice `celeb_head:{u}` mantido pelo social.
- A varredura precisa ser observada: conexões por instância, custo de Redis por
  varredura e latência do ciclo.
