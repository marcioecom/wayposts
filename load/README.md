# Testes de carga

O harness usa k6 em modelo aberto: a taxa de chegada é fixa e não depende da latência.
Modelo fechado, com N VUs em loop, esconde o joelho, porque quando o sistema fica lento
as VUs diminuem a pressão por conta própria. Esse efeito é o *coordinated omission*. Aqui
a carga continua chegando mesmo com p99 subindo, que é o que interessa para achar
saturação.

O k6 não precisa estar instalado: `scripts/load.sh` roda a imagem oficial em Docker.

## Antes de rodar

1. Suba a infra e os serviços:
   ```sh
   make up
   ./scripts/dev.sh        # em outro terminal
   ```
2. Semeie o grafo com os serviços parados. Rodar `make seed` com o fanout no ar é
   corrida perdida. O número de usuários do seed precisa ser igual ao `USERS` do k6.
3. Aqueça o cache com uma taxa baixa antes de medir, por exemplo
   `RATE=20 DURATION=1m`.

O seeder é power-law: a maioria dos autores tem poucos seguidores, celebridades têm
`CELEB_FOLLOWERS`. Mantenha `MAX_FOLLOWERS` abaixo de `FANOUT_THRESHOLD`, senão o
autor vira celebridade no meio do teste e o caminho muda.

```sh
# smoke 1:1000 (20k usuários)
make seed USERS=20000 FOLLOWS=200 CELEB_FOLLOWERS=12000 MAX_FOLLOWERS=8000
FANOUT_THRESHOLD=10000 ./scripts/dev.sh

# saturação (200k a 1M)
make seed USERS=200000 FOLLOWS=200 CELEB_FOLLOWERS=120000 MAX_FOLLOWERS=8000
```

## Como rodar

```sh
./scripts/load.sh steady
RATE=200 DURATION=5m ./scripts/load.sh steady
START_RATE=50 MAX_RATE=1500 ./scripts/load.sh ramp
```

Dentro do container, `localhost` é o container. Por isso o default de `BASE` é
`http://host.docker.internal:8080`. Para apontar para outro host:

```sh
BASE=http://10.0.0.12:8080 ./scripts/load.sh steady
```

Qualquer variável listada em `scripts/load.sh` é repassada ao k6; flags extras vão
para o `k6 run` depois do nome do cenário.

## Cenários

| Arquivo | Executor | Para que serve |
|---|---|---|
| `steady.js` | taxa constante | linha de base e estabilidade |
| `ramp.js` | taxa crescente + hold | achar o joelho e confirmar o topo |
| `spike.js` | 10× por 60s | fila, backlog e tempo de recuperação |
| `soak.js` | taxa constante longa | vazamento, GC, bloat, TTL, conexões |
| `celebrity.js` | mistura com escritas de celebridade | caminho de merge sob pior caso |
| `cold-cache.js` | leitura pura | custo de hidratação sem cache de corpo |

### Mistura padrão (`steady`, `ramp`, `spike`, `soak`)

| Operação | Peso | Chamadas HTTP por iteração |
|---|---|---|
| Home Timeline | 85% | 1 |
| Profile Timeline | 5% | 1 |
| Criar post | 7% | 1 (3 com `MEDIA=1`) |
| Like | 3% | 2; lê a home e curte um item |

`RATE` é a taxa de iterações por segundo, não de requisições. Com a mistura padrão, 100
iters/s geram ~104 req/s. Ajuste os pesos com `HOME_PCT`, `PROFILE_PCT` e `POST_PCT`; o
like fica com o resto.

### Variáveis

| Variável | Default | Cenários | Efeito |
|---|---|---|---|
| `BASE` | `http://host.docker.internal:8080` | todos | URL do gateway |
| `USERS` | `2000` | todos | faixa de IDs sorteada (deve casar com o seed) |
| `CELEBS` | `3` | `celebrity` | quantas celebridades publicam |
| `TIMELINE_LIMIT` | `20` | todos | itens por página |
| `ENFORCE` | `1` | todos | `0` desliga os thresholds de SLO |
| `MEDIA` | `0` | `steady`, `ramp`, `spike`, `soak` | `1` inclui presign + PUT + post com imagem |
| `HOME_PCT` | `0.85` | mistura | fração de leituras de home |
| `PROFILE_PCT` | `0.05` | mistura | fração de leituras de perfil |
| `POST_PCT` | `0.07` | mistura | fração de escritas |
| `RATE` | varia | `steady`, `soak`, `celebrity` | iterações/s |
| `DURATION` | varia | `steady`, `cold-cache`, `celebrity` | duração da carga |
| `START_RATE` / `MAX_RATE` | `20` / `1000` | `ramp` | taxa inicial e topo |
| `WARMUP_DURATION` | `1m` | `ramp` | rampa até a taxa inicial |
| `RAMP_DURATION` | `10m` | `ramp` | subida até o topo |
| `HOLD_DURATION` | `2m` | `ramp` | permanência no topo |
| `BASE_RATE` / `SPIKE_FACTOR` | `100` / `10` | `spike` | patamar e multiplicador |
| `SPIKE_DURATION` | `60s` | `spike` | duração do pico |
| `SOAK` | `30m` | `soak` | duração total |
| `CELEB_WRITE_PCT` | `0.1` | `celebrity` | fração de posts de celebridade |
| `PREALLOC_VUS` / `MAX_VUS` | varia | todos | VUs pré-alocadas e teto |

`MAX_VUS` precisa ser maior que `RATE × p99` (em segundos). Se o p99 é 200ms e a taxa
é 500 iters/s, são ~100 VUs ocupadas ao mesmo tempo; com teto baixo o k6 não consegue
entregar a taxa e a curva mente. O default de cada cenário já é folgado.

## SLOs e thresholds

| Operação | p99 | Tag do k6 |
|---|---|---|
| `POST /v1/posts` | ≤ 300ms | `kind:post` |
| Home/Profile Timeline | ≤ 150ms | `kind:home`, `kind:profile` |
| Like | ≤ 100ms | `kind:like` |
| Presign | ≤ 150ms | `kind:presign` |
| Upload direto no bucket | ≤ 1s | `kind:upload` |
| Erros 5xx | < 0,1% | `http_req_failed` |

Os thresholds são avaliados ao final e o k6 sai com código 99 se algum quebrar.
`ENFORCE=0` remove os thresholds. Use durante a rampa, porque o objetivo ali é observar
a curva, e uma rampa que passa do joelho naturalmente quebra SLO. Rode a taxa escolhida
com `ENFORCE=1` para a medição final.

O k6 mede a latência HTTP vista pelo cliente. Fanout lag não aparece no k6, porque é
medido no servidor, da criação do post até o último `ZADD`. Esse é o indicador
principal da arquitetura. Durante a carga, consulte:

```promql
# p95 e p99 do lag por serviço de fanout
histogram_quantile(0.95, sum by (le) (rate(wayposts_fanout_lag_seconds_bucket[5m])))
histogram_quantile(0.99, sum by (le) (rate(wayposts_fanout_lag_seconds_bucket[5m])))

# vazão de escrita em feeds e mensagens por resultado
sum(rate(wayposts_fanout_feed_writes_total[1m]))
sum by (result) (rate(wayposts_fanout_messages_total[1m]))

# pressão de cache de corpo e de seguidores
sum by (result) (rate(wayposts_post_cache_total[1m]))
sum by (result) (rate(wayposts_follower_cache_total[1m]))
```

Saturação é o primeiro patamar em que um SLO quebra por 5 minutos seguidos ou o erro
passa de 1%.

## Metodologia da curva

1. **Aqueça.** Taxa baixa por 1 a 2 min para o cache de corpos e de seguidores
   estabilizar.
2. **Rampa com `ENFORCE=0`.** Comece em ~20% do joelho esperado e suba por 15 a 20 min.
   Rampa curta aquece o cache no meio da subida e desloca o joelho para cima.
3. **Leia o joelho.** Latência plana e depois um salto súbito indica fila. Se a latência
   já sobe desde o início, olhe o pool de conexões, a CPU de um shard, o `maxmemory` do
   Redis ou o próprio gerador. O gargalo não é a taxa.
4. **Confirme com hold.** Permaneça 5 min no topo; o joelho é o ponto em que o sistema
   não volta a estabilizar.
5. **Meça de novo com `ENFORCE=1`** na taxa imediatamente abaixo do joelho.
6. **Prove linearidade.** Repita a rampa com 2× usuários. A taxa de joelho deve cair
   aproximadamente pela metade se o custo for por usuário; o custo por vCPU, medido em
   iters/s por vCPU, deve ficar estável. Sem esse controle, o número não extrapola.

Registre para cada patamar:

| Taxa em iters/s | p99 home | p99 post | Erros | Fanout lag p95 | vCPU total | vCPU/serviço |
|---|---|---|---|---|---|---|

Extrapolação para 20M DAU: `DAU × iters por usuário por segundo / taxa de joelho`.
O fator de escala do laboratório, 1:1000 ou 1:100, entra no número de usuários, não na
taxa por usuário. Declarar isso no relatório evita a comparação errada.

## Receitas

### Smoke 1:1000

```sh
RATE=50 DURATION=3m ./scripts/load.sh steady
```

### Rampa para saturação

```sh
START_RATE=50 MAX_RATE=1500 WARMUP_DURATION=2m RAMP_DURATION=20m HOLD_DURATION=5m \
  ENFORCE=0 ./scripts/load.sh ramp
```

### Pico

```sh
BASE_RATE=100 SPIKE_FACTOR=10 SPIKE_DURATION=60s ./scripts/load.sh spike
```

Observe o p99 durante o pico e também quando o fanout lag volta ao normal depois.
Backlog que não drena é pior que latência alta.

### Soak

```sh
RATE=80 SOAK=1h ./scripts/load.sh soak
```

Compare o primeiro e o último quarto de hora: memória do Redis, conexões do Postgres,
tamanho das filas e p99 devem estar estáveis. Deriva lenta é vazamento ou bloat.

### Celebridade

```sh
CELEBS=3 CELEB_WRITE_PCT=0.1 RATE=100 ./scripts/load.sh celebrity
```

O post de celebridade não é materializado (`author:{id}` alimenta o merge), então este
cenário mede o custo do merge por leitor. Combine com `TIMELINE_LIMIT` maior para ver
o efeito no p99.

### Cache frio

```sh
make flush-posts
RATE=20 DURATION=2m ./scripts/load.sh cold-cache
```

`flush-posts` apaga `post:{id}`, o corpo, e não apaga `feed:*`, então os feeds continuam
válidos. Como a leitura não reescreve o hash, o caminho frio se sustenta durante toda a
execução. É o custo de ir ao Postgres a cada hidratação. Compare com `steady` na mesma
taxa para isolar o custo do miss.

Não use este cenário para achar saturação: ele mede um modo degradado específico.

## Armadilhas

- `make seed` com os serviços no ar deixa o outbox e o cache em estado inconsistente.
  Pare com `./scripts/stop.sh` primeiro.
- `MEDIA=1` exige o post service com `S3_ENDPOINT=http://host.docker.internal:9000`,
  porque a URL assinada carrega o host. Com `localhost:9000`, o PUT feito de dentro do
  container falha.
- As thresholds são por tag `kind`. Uma operação nova sem tag não entra em nenhum SLO.
- Rode o gerador na mesma rede dos serviços medidos. k6 pela WAN adiciona RTT e contamina
  o p99 inteiro.
- A latência inclui o gateway: proxy, serviço e Redis/Postgres. Para separar o custo do
  proxy, compare `http_req_duration{kind:home}` com a métrica do serviço de timeline.
