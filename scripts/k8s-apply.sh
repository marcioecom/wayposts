#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."

# Same flow as .github/workflows/ci.yml: wait for Postgres, run migrations, then
# roll out the application. The image and tag come from the caller; the defaults
# are for a local build tagged dev.
export WAYPOSTS_IMAGE="${WAYPOSTS_IMAGE:-ghcr.io/marcioecom/wayposts}"
export WAYPOSTS_TAG="${WAYPOSTS_TAG:-dev}"

render() {
  sed \
    -e "s|\${WAYPOSTS_IMAGE}|${WAYPOSTS_IMAGE}|g" \
    -e "s|\${WAYPOSTS_TAG}|${WAYPOSTS_TAG}|g"
}

echo "== namespace"
kubectl apply -f deploy/k8s/base/namespace.yaml

if ! kubectl -n wayposts get secret ghcr-pull-secret >/dev/null 2>&1; then
  echo "missing ghcr-pull-secret in namespace wayposts" >&2
  echo "copy it from the echo namespace before applying" >&2
  exit 1
fi

echo "== waiting for Postgres"
kubectl -n wayposts wait \
  --for=condition=Ready \
  cluster/wayposts-postgres \
  --timeout=10m

echo "== migrations"
kubectl -n wayposts delete job wayposts-migrate --ignore-not-found
render < deploy/k8s/base/migrate-job.yaml | kubectl apply -n wayposts -f -
kubectl -n wayposts wait \
  --for=condition=complete \
  job/wayposts-migrate \
  --timeout=5m

echo "== application manifests"
kubectl kustomize deploy/k8s/overlays/orbit | render | kubectl apply -f -

echo "== rollout"
for deployment in gateway post timeline social fanout; do
  kubectl -n wayposts rollout status "deployment/${deployment}" --timeout=5m
done

kubectl -n wayposts get deploy,job,pods
