# 0005: IDs de 53 bits compatíveis com score de ZSET

- Status: aceito
- Data: 2026-10-25

## Contexto

O feed ordena e pagina por `score = postID`. O score de um ZSET no Redis é `float64`,
com inteiros exatos apenas até 2^53. Um Snowflake padrão de 63 bits perde os bits
baixos ao virar score: posts do mesmo milissegundo colidem e a paginação por cursor,
com `ZREVRANGEBYSCORE ... max = cursor`, passa a pular ou repetir posts. O mesmo limite
de 2^53 vale para números em JSON/JavaScript, o que quebraria o cliente web.

## Decisão

Layout de 53 bits, gerado no `post` service:

| Bits | Campo | Observação |
|---|---|---|
| 41 | milissegundos desde 2026-01-01 | ≈70 anos de horizonte |
| 6 | node ID | 64 geradores, vindo de configuração |
| 6 | sequência | 64 IDs/ms por nó = 64k IDs/s por instância |

- `score = member = ID` no feed. O cursor de paginação é o próprio ID.
- `created_at` é derivado do ID. Não há coluna de ordenação concorrente.
- Relógio para trás: o gerador espera o último timestamp emitido. Estouro de sequência
  avança para o próximo milissegundo.

## Consequências

- Ordenação exata e merge k-way por comparação de inteiros.
- ID seguro em JSON/JS sem `BigInt`; 8 bytes por entrada no ZSET.
- Limite de 64 instâncias geradoras e 64k IDs/s por instância, folgado para o pico
  estimado de ~9k posts/s. Ainda assim, é um teto que precisa aparecer no runbook.
- A epoch customizada de 2026-01-01 precisa estar documentada e é incompatível com
  Snowflake de biblioteca padrão.
- Skew de relógio entre nós pode gerar IDs fora de ordem. A estratégia de espera precisa
  ser testada, não assumida.
