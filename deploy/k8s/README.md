# Fase 0: orbit em 1:1000

Manifests para o smoke de 1:1000 no k3s do orbit. O objetivo aqui é validar deploy,
contrato, métricas e pipeline de dados, não tirar conclusão de capacidade. A curva de
saturação é a fase 1, em nós temporários.

## Deploy normal: GitHub Actions

O deploy dos microserviços roda em [`.github/workflows/ci.yml`](../../.github/workflows/ci.yml).
Em push para `main` ou `workflow_dispatch`, o workflow:

1. roda `gofmt`, `go vet`, `go test`, `go build` e o build do console em `web/`;
2. constrói a imagem `ghcr.io/<owner>/wayposts` e publica as tags do SHA e `dev`;
3. entra na tailnet, configura o kubeconfig e chama
   [`scripts/k8s-apply.sh`](../../scripts/k8s-apply.sh);
4. o script espera o Postgres, roda a migration, aplica o overlay com a imagem do SHA e
   espera o rollout dos cinco Deployments.

Segredos necessários no repositório:

| Secret | Uso |
|---|---|
| `TS_OAUTH_CLIENT_ID` | OAuth client do Tailscale com a tag `tag:ci` |
| `TS_OAUTH_CLIENT_SECRET` | Secret do mesmo OAuth client |
| `KUBECONFIG_BASE64` | kubeconfig do cluster, codificado em base64 |

O `GITHUB_TOKEN` é automático e cobre o push no GHCR. O pacote pode continuar privado.

O `ghcr-pull-secret` já existe no namespace `echo` e foi replicado para `wayposts`. O
workflow não cria esse secret: ele falha cedo se o secret não estiver no namespace. Em um
cluster novo, copie-o do Echo antes do primeiro deploy.

O workflow só faz deploy de `main` ou de um `workflow_dispatch`. Pull requests rodam
apenas a verificação, sem tocar no cluster.

## O que sobe

| Componente | Origem | Observação |
|---|---|---|
| `gateway`, `post`, `timeline`, `social`, `fanout` | imagem `ghcr.io/marcioecom/wayposts` | 1 réplica cada |
| Postgres | CloudNativePG já instalado | cluster próprio, 5Gi |
| Valkey | `valkey/valkey:8-alpine` | cache sem persistência |
| RabbitMQ | `rabbitmq:4-management-alpine` | 1 réplica, PVC 2Gi |
| Prometheus scrape | ServiceMonitor + PrometheusRule | label do kube-prometheus-stack |

Tudo é agendado no worker de Helsinki, `hel1`, que tinha ~1 vCPU e ~1,8Gi de requests
livres. O worker de Falkenstein está com ~89% de CPU requests e não comporta o stack. O
overlay troca a região em um patch; para outro nó, ajuste
`deploy/k8s/overlays/orbit/kustomization.yaml`.

Requests totais do stack: ~675m de CPU e ~950Mi de memória. Os limits deixam burst, mas
o nó não tem folga para réplicas extras.

## 1. Build manual (fallback)

O caminho normal é o workflow. Para construir e publicar manualmente:

```sh
# da raiz do repositório
docker build --platform linux/amd64 -t ghcr.io/marcioecom/wayposts:dev .
docker build --platform linux/amd64 -f deploy/k6/Dockerfile -t ghcr.io/marcioecom/wayposts-k6:dev .
```

### Se o push no GHCR funcionar

```sh
docker push ghcr.io/marcioecom/wayposts:dev
docker push ghcr.io/marcioecom/wayposts-k6:dev
```

### Se o push for negado, caminho usado no orbit

O token local não tinha `write:packages` e o pacote não é público. A imagem foi
importada direto no containerd do nó que roda o stack:

```sh
docker save ghcr.io/marcioecom/wayposts:dev ghcr.io/marcioecom/wayposts-k6:dev \
  | ssh root@orbit-eu-pool-helsinki-worker1 'k3s ctr images import -'
```

Depois de reconstruir com a mesma tag `dev`, importe de novo e reinicie os Deployments
que devem pegar a versão nova, já que `imagePullPolicy: IfNotPresent`:

```sh
kubectl -n wayposts rollout restart deploy/gateway deploy/post deploy/timeline deploy/social deploy/fanout
```

Se um pod for agendado em outro nó sem a imagem, importe lá também, ou torne o pacote
público, ou crie um pull secret com acesso a ele.

Independente do caminho, o secret `ghcr-pull-secret` precisa existir no namespace,
porque o overlay o referencia em todos os Deployments e Jobs. A seção 2 mostra como
replicá-lo.

## 2. Apply manual (fallback)

Use o mesmo script do workflow. Ele espera o Postgres, roda a migration antes do rollout
e aplica o overlay com a imagem escolhida:

```sh
make k8s-apply
```

Ou com uma imagem e tag explícitas:

```sh
WAYPOSTS_IMAGE=ghcr.io/marcioecom/wayposts \
WAYPOSTS_TAG=dev \
./scripts/k8s-apply.sh
```

O script aplica o namespace, exige o `ghcr-pull-secret`, espera o Postgres, recria e
espera o Job de migration, aplica o overlay e espera o rollout dos cinco Deployments.

O `ghcr-pull-secret` já está no cluster. Para recriá-lo em um namespace novo, copie o
secret que funciona no Echo:

```sh
kubectl -n echo get secret ghcr-pull-secret -o json \
  | python3 -c 'import sys,json; d=json.load(sys.stdin); d["metadata"]={"name": d["metadata"]["name"]}; print(json.dumps(d))' \
  | kubectl -n wayposts apply -f -
```

O secret copiado funciona se o PAT tiver `read:packages` para o pacote `wayposts`. Se o
token for restrito ao pacote do Echo, gere outro com acesso ao pacote do Wayposts. Se o
token rotacionar, atualize os dois namespaces.

Alternativamente, crie com um PAT que tenha `read:packages`:

```sh
kubectl -n wayposts create secret docker-registry ghcr-pull-secret \
  --docker-server=ghcr.io \
  --docker-username=marcioecom \
  --docker-password="$GHCR_PULL_TOKEN" \
  --dry-run=client -o yaml | kubectl apply -f -
```

O overlay usa `${WAYPOSTS_IMAGE}` e `${WAYPOSTS_TAG}`. Não rode `kubectl apply -k`
diretamente, porque os placeholders chegariam sem substituição. O script faz o render
com `kubectl kustomize` antes do apply.

## 3. Seed

```sh
kubectl apply -f deploy/k8s/overlays/orbit/seed-job.yaml
kubectl -n wayposts wait --for=condition=complete job/wayposts-seed --timeout=20m
kubectl -n wayposts logs job/wayposts-seed
```

O seed é destrutivo, porque `-reset=true` é o default: trunca as tabelas e dá `FLUSHDB`
no Redis. Para outra escala, edite os `args` do Job. O teto de seguidores precisa ficar
abaixo de `FANOUT_THRESHOLD`, que está em 10k no ConfigMap.

Para rodar de novo, delete o Job antes com `kubectl delete job wayposts-seed`, ou o
apply não recria.

## 4. Smoke

De dentro do cluster, que é o caminho recomendado:

```sh
kubectl apply -f deploy/k8s/overlays/orbit/k6-job.yaml
kubectl -n wayposts wait --for=condition=complete job/k6 --timeout=10m
kubectl -n wayposts logs job/k6
kubectl -n wayposts delete job k6
```

Para iterar rápido pelo notebook, use port-forward. A latência inclui o Tailscale, então
não use esses números no relatório:

```sh
kubectl -n wayposts port-forward svc/gateway 8080:8080
BASE=http://localhost:8080 RATE=20 DURATION=30s ENFORCE=0 ./scripts/load.sh steady
```

## 5. Observabilidade

```sh
kubectl -n wayposts get servicemonitor,prometheusrule
kubectl -n wayposts port-forward svc/gateway 8080:8080 &
curl -s localhost:8080/metrics | grep wayposts_
```

No Prometheus do orbit, as séries aparecem com `namespace="wayposts"`. As duas regras em
`base/monitoring.yaml` cobrem 5xx acima de 0,1% e fanout lag p99 acima de 10s.

## 6. Mídia opcional

Com `MEDIA_BUCKET` vazio o post service aceita só texto. Para ligar no R2:

```sh
kubectl -n wayposts create secret generic wayposts-s3 \
  --from-literal=access-key=... --from-literal=secret-key=...

kubectl -n wayposts set env deployment/post \
  MEDIA_BUCKET=wayposts-media \
  S3_ENDPOINT=https://<account>.eu.r2.cloudflarestorage.com \
  S3_PATH_STYLE=false \
  MEDIA_PUBLIC_BASE=https://<domínio-público-do-bucket>
```

A URL pública precisa ser alcançável pelo cliente que for ler a timeline.

## 7. Limpeza e reset

```sh
# limpar feeds sem perder o grafo
kubectl -n wayposts exec deploy/valkey -- valkey-cli FLUSHDB

# derrubar tudo (os PVCs ficam)
kubectl delete -k deploy/k8s/overlays/orbit
kubectl -n wayposts delete pvc rabbitmq wayposts-postgres-1
```

## 8. Frontend (Cloudflare)

O console visual fica em `web/` e roda no Cloudflare Workers, fora do cluster. Ele
chama o gateway pelo túnel:

| Peça | Valor |
|---|---|
| Frontend | `https://wayposts.marcio.run` |
| API | `https://wayposts-api.marcio.run` |
| Rota do túnel | `http://gateway.wayposts.svc.cluster.local:8080` |
| Build variable | `VITE_API_URL=https://wayposts-api.marcio.run` |

No Workers Builds, use `web` como root directory, `npm ci && npm run build` como
build command e `npx wrangler deploy` como deploy command. O passo a passo do
dashboard, o DNS e o Pulumi opcional estão em [`web/README.md`](../../web/README.md).

O gateway libera o domínio do console em `CORS_ALLOWED_ORIGINS`, no
`base/config.yaml`. Depois de mudar o domínio, reaplique o overlay e reinicie o
Deployment do gateway.

## Resultado do smoke em 27/10/2026

Grafo: 20.000 usuários, 4.033.532 arestas, 3 celebridades com 12k seguidores. Seed em
~2,5 min.

| Cenário | Taxa | Iterações | Erros | p99 total | p99 home | p99 post | Fanout lag p95/p99 |
|---|---|---|---|---|---|---|---|
| steady | 50 iters/s por 3m | 9.001 | 0% | 34ms | 43ms | 18ms | 99ms / 218ms |
| ramp | 50 a 200 iters/s por 3,5m | 28.498 | 0% | 151ms | n/a | n/a | n/a |

Todos os SLOs do laboratório foram cumpridos no steady. No ramp, `ENFORCE=0` desliga os
thresholds e o resumo não abre o breakdown por tag, então o p99 agregado de 151ms é a
leitura disponível.

O worker de Helsinki ficou com 82% de CPU requests e 78% de requests de memória depois
do deploy. O uso real durante o smoke foi ~27% de CPU e ~75% de memória. Não há folga
para réplicas extras nesse nó.

## Limitações conhecidas

- **Redis não é durável.** Perder o Valkey perde os feeds materializados, e o desenho não
  reconstrói feed a partir do Postgres. Um restart exige rodar o seed de novo.
- **Credenciais de laboratório** em `base/secret.yaml`. Troque antes de expor qualquer
  coisa fora do cluster.
- **Sem NetworkPolicy.** Os serviços de dados são ClusterIP, mas qualquer pod do cluster
  alcança o Postgres do laboratório.
- **1 réplica de tudo.** Não há HA: um restart do fanout interrompe o consumo, e o
  gateway é ponto único para o smoke.
- **Outbox sem retenção.** O relay marca `published_at`, mas nada apaga as linhas
  publicadas. Em 20M DAU a tabela cresce para sempre; a fase 1 precisa de um job de
  limpeza, por exemplo apagar publicados com mais de 1h, em lotes.
- **Volumes mínimos de 10Gi.** O `hcloud-volumes` arredonda 5Gi e 2Gi para 10Gi; os dois
  PVCs ocupam 10Gi cada.
- O worker de Helsinki é compartilhado com outras cargas. Números de latência da fase 0
  servem para detectar erro de deploy, não para capacidade.
