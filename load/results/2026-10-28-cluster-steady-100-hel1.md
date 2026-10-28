# Cluster steady 100 iters/s no nó dedicado — 2026-10-28

Execução com o k6 em um worker dedicado em Helsinki (`orbit-eu-pool-helsinki-k6`),
separado do nó do stack (`orbit-eu-pool-helsinki-worker1`).

## Configuração

| Item | Valor |
|---|---|
| Contexto | `orbit-eu-master1` |
| Namespace | `wayposts` |
| Nó do k6 | `orbit-eu-pool-helsinki-k6` (`hel1`, dedicado, label `wayposts.io/loadgen=true`) |
| Nó do stack | `orbit-eu-pool-helsinki-worker1` (`hel1`) |
| Imagem do gateway | `ghcr.io/marcioecom/wayposts:41b7636b1be6d6f190fb74ec79413fc240d1c47c` |
| Seed | 20.000 usuários, 4.033.533 arestas de follow, 3 celebridades |
| `FANOUT_THRESHOLD` | 10.000 |
| Cenário | `steady.js` |
| Taxa | 100 iters/s |
| Duração | 3 minutos |
| `ENFORCE` | 1 |
| Job | `k6`, 00:14:56Z a 00:18:00Z (3m04s) |
| Saída bruta | [`2026-10-28-cluster-steady-100-hel1-k6.txt`](raw/2026-10-28-cluster-steady-100-hel1-k6.txt) |

## Resultado do k6

| Métrica | Valor |
|---|---|
| Iterações | 18.001 (100,00/s) |
| Requisições HTTP | 18.521 (102,89/s) |
| Checks | 18.521, 0 falhas (0,00%) |
| `http_req_failed` | 0,00% |
| Latência mínima / média / mediana | 1,42 ms / 7,60 ms / 5,59 ms |
| p90 / p95 / p99 total | 10,29 ms / 14,70 ms / 42,49 ms |
| p99 Home | 44,29 ms (alvo 150 ms) |
| p99 Profile | 28,39 ms (alvo 150 ms) |
| p99 Post | 36,69 ms (alvo 300 ms) |
| p99 Like | 22,18 ms (alvo 100 ms) |
| Presign / Upload | 0 s, não exercitados (`MEDIA=0`) |
| VUs ativas | máximo 7, com 50 pré-alocadas |
| Rede | 56 MB recebidos, 2,9 MB enviados |

Todos os thresholds passaram.

## Prometheus

Janela de 5 minutos em torno da execução, consultada no Prometheus do cluster:

| Consulta | Valor |
|---|---|
| Fanout lag p95 | 98,3 ms |
| Fanout lag p99 | 213,1 ms |
| Escritas em feeds (`wayposts_fanout_feed_writes_total`) | ~870/s |
| Mensagens de fanout `ok` / `retry` / `dlq` | 4,49/s / 0 / 0 |
| Cache de post `hit` / `miss` | 1.134,19/s / 0 |

## Comparação

| Execução | p99 total | p99 Home |
|---|---|---|
| 50 iters/s | 20,59 ms | 20,76 ms |
| 100 iters/s | 42,49 ms | 44,29 ms |

A comparação completa, incluindo fanout, está na
[`2026-10-28-cluster-steady-series-hel1.md`](2026-10-28-cluster-steady-series-hel1.md).

## Observações

- O nó dedicado em `hel1` manteve a CPU do gerador fora do nó do stack. O p99 total
  subiu de 20,6 ms (50 iters/s no nó dedicado) para 42,5 ms (100 iters/s), um
  crescimento pequeno para o dobro da taxa.
- O fanout lag ficou estável entre 50 e 100 iters/s; as escritas em feed e o cache de post
  acompanharam a taxa.
- O k6 foi agendado no nó novo pelo label `wayposts.io/loadgen=true`. O nó está no k3s
  pela rede privada (`10.0.0.5`); o Tailscale ainda não foi configurado nele.
- `MEDIA=0`, então presign e upload aparecem com 0 s e seus thresholds passam vazios.
