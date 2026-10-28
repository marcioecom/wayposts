# Resultados de carga

Esta pasta mantém somente a série no ambiente com o stack no mesmo nó (`hel1`) e o k6
isolado em `orbit-eu-pool-helsinki-k6`. Execuções com o k6 no mesmo nó do stack ou em
outra região não fazem parte desta pasta.

## Série considerada

| Data | Cenário | Taxa | Resultado | Arquivo |
|---|---|---|---|---|
| 2026-10-28 | steady | 50 iters/s por 3 min | todos os SLOs passaram | [série](2026-10-28-cluster-steady-series-hel1.md) |
| 2026-10-28 | steady | 100 iters/s por 3 min | todos os SLOs passaram | [`2026-10-28-cluster-steady-100-hel1.md`](2026-10-28-cluster-steady-100-hel1.md) |
| 2026-10-28 | steady | 200 iters/s por 3 min | todos os SLOs passaram (pré-reset) | [série](2026-10-28-cluster-steady-series-hel1.md) |
| 2026-10-28 | steady | 225 iters/s por 3 min | SLOs de latência quebrados | [série](2026-10-28-cluster-steady-series-hel1.md) |
| 2026-10-28 | steady | 250 iters/s por 3 min | SLOs de latência quebrados | [série](2026-10-28-cluster-steady-series-hel1.md) |
| 2026-10-28 | steady | 300 iters/s por 3 min | SLOs de latência quebrados | [série](2026-10-28-cluster-steady-series-hel1.md) |
| 2026-10-28 | steady | 400 iters/s por 3 min | SLOs quebrados; gerador limitado a 1.000 VUs | [série](2026-10-28-cluster-steady-series-hel1.md) |

A análise completa, incluindo fanout e comparação entre patamares, está em
[`2026-10-28-cluster-steady-series-hel1.md`](2026-10-28-cluster-steady-series-hel1.md).
O joelho fica entre 200 e 225 iters/s. O 300 foi repetido: o k6 de 512Mi sofreu OOM e o
de 2Gi derrubou pods do stack; os detalhes estão na série.

A tabela acima é a série pré-reset. Os runs pós-reset de 50 e 200 estão em `reset/`; o
200 pós-reset não passou o SLO, mesmo com 0% de erros.

## Estrutura

- `2026-10-28-cluster-steady-series-hel1.md`: análise completa da série.
- `2026-10-28-cluster-steady-100-hel1.md`: detalhe da execução de 100 iters/s.
- `raw/`: saídas brutas dos patamares de 50 a 400, incluindo os reruns do 300.
- `reset/`: execuções de 50 e 200 depois do reset/seed.
