# Série steady no nó dedicado — 2026-10-28

Série considerada para a curva: o k6 rodou em `orbit-eu-pool-helsinki-k6` (`hel1`,
`10.0.0.5`), separado do nó do stack, `orbit-eu-pool-helsinki-worker1` (`hel1`). O
gerador não compartilha CPU com os serviços e não cruza região.

## Configuração

| Item | Valor |
|---|---|
| Contexto | `orbit-eu-master1` |
| Namespace | `wayposts` |
| Nó do k6 | `orbit-eu-pool-helsinki-k6` (`hel1`, label `wayposts.io/loadgen=true`) |
| Nó do stack | `orbit-eu-pool-helsinki-worker1` (`hel1`, 2 vCPU) |
| Imagem do gateway | `ghcr.io/marcioecom/wayposts:41b7636b1be6d6f190fb74ec79413fc240d1c47c` |
| Seed | 20.000 usuários, 4.033.533 arestas de follow, 3 celebridades |
| `FANOUT_THRESHOLD` | 10.000 |
| Cenário | `steady.js`, 3 minutos por patamar |
| `ENFORCE` | 1 |

## Resultados

| Taxa | Status | Iterações | p99 total | p99 Home | p99 Profile | p99 Post | p99 Like | Descartadas | VUs ativas / teto |
|---:|---|---:|---:|---:|---:|---:|---:|---:|---:|
| 50 | ✓ | 9.001 | 20,59 ms | 20,76 ms | 8,75 ms | 71,63 ms | 11,53 ms | 0 | 3 / 50 |
| 100 | ✓ | 18.001 | 42,49 ms | 44,29 ms | 28,39 ms | 36,69 ms | 22,18 ms | 0 | 7 / 50 |
| 200 | ✓ | 35.996 | 133,57 ms | 135,66 ms | 78,21 ms | 158,23 ms | 65,29 ms | 5 | 32 / 55 |
| 225 | ✗ | 40.340 | 677,37 ms | 701,40 ms | 507,52 ms | 489,75 ms | 278,11 ms | 160 | 162 / 207 |
| 250 | ✗ | 44.666 | 1,35 s | 1,39 s | 1,20 s | 1,11 s | 1,12 s | 335 | 333 / 365 |
| 300 | ✗ | 53.181 | 2,40 s | 2,47 s | 1,91 s | 1,88 s | 1,53 s | 820 | 703 / 728 |
| 400 | ✗ | 55.838 | 6,69 s | 6,84 s | 4,69 s | 4,56 s | 4,43 s | 16.162 | 1.000 / 1.000 |

Os patamares de 225, 250 e 300 usaram `MAX_VUS=2000`. O de 400 usou o default de
1.000 VUs e ficou limitado pelo gerador: entregou ~307 iters/s, com 22,4% das iterações
descartadas. O de 300 é o último patamar limpo acima do joelho, com 1,5% de descarte.

## Reruns do 300 e estabilidade

- **300 com 512Mi:** o k6 foi `OOMKilled` aos 2m15, com 1.746 VUs. O limite de 512Mi
  não segura o gerador quando a latência sobe e o k6 aloca mais VUs.
- **300 com 2Gi e console fechado:** 69,86% de falhas, p99 de 10 s e 11.978 iterações
  descartadas. `post`, `timeline`, `social`, `fanout`, `gateway` e Postgres reiniciaram
  por falhas de liveness/readiness sob carga. O nó ficou em 81% de memória; o Redis
  chegou a 255M/256M com 88.563 chaves evictadas.
- Isso confirma que 300 não é um ponto viável neste deploy. O último patamar que passou
  é 200. O 250 também quebrou o SLO (p99 Home 1,39 s), então não é um fallback seguro
  sem estabilizar o ambiente.

## Leitura da curva

- **O joelho está entre 200 e 225 iters/s.** Em 200, o p99 Home foi 135,66 ms, a 90% do
  alvo de 150 ms. Em 225, saltou para 701,40 ms. A transição é abrupta, típica de fila.
- **A taxa de erro ficou em 0% em todos os patamares.** O sistema absorve a sobrecarga
  como latência, não como falha. O primeiro SLO a quebrar é o de latência, não o de erro.
- **O fanout não é o primeiro gargalo.** Em 400 iters/s, com o stack já saturado, o fanout
  lag ficou em p95 1,47 s e p99 1,90 s, ainda dentro dos alvos de 2 s / 10 s. O cache de
  post seguiu com hit alto e miss próximo de zero. O gargalo está no caminho de leitura
  do stack de 2 vCPU (timeline, post e gateway), não no outbox/fanout.
- **O 400 não é um ponto limpo de taxa.** O teto de 1.000 VUs do `steady.js` fez o k6
  descartar 22,4% das iterações. Para medir 400 de verdade, use `MAX_VUS` acima de
  `RATE × p99`, como 4.000.

## Fanout e cache

Janela de 3 minutos centrada em cada execução, consultada no Prometheus do cluster:

| Taxa | Fanout p95 | Fanout p99 | Escritas/s | Mensagens ok/s | Retry | DLQ | Cache hit/s | Cache miss/s |
|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| 50 | 97,2 ms | 188,8 ms | 788 | 3,4 | 0 | 0 | 885 | 0 |
| 100 | 97,4 ms | 190,8 ms | 1.379 | 7,0 | 0 | 0 | 1.744 | 0 |
| 200 | 180,6 ms | 324,2 ms | 2.620 | 13,7 | 0 | 0 | 3.538 | 0 |
| 225 | 242,8 ms | 451,6 ms | 3.047 | 15,6 | 0 | 0 | 3.701 | 2,7 |
| 250 | 446,4 ms | 883,1 ms | 3.505 | 17,2 | 0 | 0 | 4.145 | 5,5 |
| 300 | 499,9 ms | 951,4 ms | 4.701 | 21,1 | 0 | 0 | 5.150 | 0 |
| 400 | 1.466,1 ms | 1.895,8 ms | 4.429 | 22,1 | 0 | 0 | 5.358 | 0 |

## Limitações

- Os patamares têm 3 minutos. A definição do projeto pede 5 minutos de quebra de SLO para
  declarar saturação; estes números localizam o joelho, mas ainda não são a confirmação.
- O stack roda com uma réplica por serviço em um nó de 2 vCPU. A curva mede esse deploy,
  não a capacidade do desenho.
- O mix do k6 é o do harness: 85% Home, 5% Profile, 7% Post e 3% Like. Ele **não** inclui
  a hidratação de autores que o console faz depois da Home (uma chamada de perfil por
  post). A carga real do console é maior no serviço `social` e no gateway.
- `MEDIA=0`, então presign e upload não foram exercitados.
- O código com `author` embutido no item da timeline foi implementado localmente, mas
  ainda não foi deployado; os números acima são da imagem antiga. O mix do k6 também não
  faz a hidratação de autores que o console faz depois da Home.

## Saídas brutas

- [50 iters/s](raw/2026-10-28-cluster-steady-50-hel1-k6.txt)
- [100 iters/s](raw/2026-10-28-cluster-steady-100-hel1-k6.txt), resumo em
  [`2026-10-28-cluster-steady-100-hel1.md`](2026-10-28-cluster-steady-100-hel1.md)
- [200 iters/s](raw/2026-10-28-cluster-steady-200-hel1-k6.txt)
- [225 iters/s](raw/2026-10-28-cluster-steady-225-hel1-k6.txt)
- [250 iters/s](raw/2026-10-28-cluster-steady-250-hel1-k6.txt)
- [300 iters/s](raw/2026-10-28-cluster-steady-300-hel1-k6.txt)
- [300 rerun com 512Mi, OOM do k6](raw/2026-10-28-cluster-steady-300-hel1-run2-k6.txt)
- [300 rerun com 2Gi, colapso do stack](raw/2026-10-28-cluster-steady-300-hel1-run3-k6.txt)
- [400 iters/s](raw/2026-10-28-cluster-steady-400-hel1-k6.txt)

### Reset

O reset/seed gerou 20.000 usuários e 4.033.532 arestas.

- [50 iters/s pós-reset](reset/2026-10-28-cluster-reset-steady-50-k6.txt)
- [200 iters/s pós-reset, run 1](reset/2026-10-28-cluster-reset-steady-200-k6.txt)
- [200 iters/s pós-reset, run 2](reset/2026-10-28-cluster-reset-steady-200-run2-k6.txt)
