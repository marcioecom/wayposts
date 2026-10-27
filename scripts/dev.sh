#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."

export DATABASE_URL="${DATABASE_URL:-postgres://wayposts:wayposts@localhost:5432/wayposts?sslmode=disable}"
export REDIS_ADDR="${REDIS_ADDR:-localhost:6379}"
export RABBIT_URL="${RABBIT_URL:-amqp://guest:guest@localhost:5672/}"
export FANOUT_THRESHOLD="${FANOUT_THRESHOLD:-500}"
export FANOUT_SHARDS="${FANOUT_SHARDS:-16}"

export SOCIAL_ADDR="${SOCIAL_ADDR:-:8083}"
export POST_ADDR="${POST_ADDR:-:8081}"
export TIMELINE_ADDR="${TIMELINE_ADDR:-:8082}"
export GATEWAY_ADDR="${GATEWAY_ADDR:-:8080}"
export POST_PUBLIC_URL="${POST_PUBLIC_URL:-http://localhost:8081}"
export SOCIAL_PUBLIC_URL="${SOCIAL_PUBLIC_URL:-http://localhost:8083}"
export TIMELINE_PUBLIC_URL="${TIMELINE_PUBLIC_URL:-http://localhost:8082}"
export NODE_ID="${NODE_ID:-1}"
export NUDGE_INTERVAL="${NUDGE_INTERVAL:-5s}"

# MinIO local faz o papel do R2. Sem MEDIA_BUCKET o serviço de post sobe com
# upload desligado e aceita apenas texto.
export MEDIA_BUCKET="${MEDIA_BUCKET-wayposts-media}"
export S3_ENDPOINT="${S3_ENDPOINT:-http://localhost:9000}"
export S3_REGION="${S3_REGION:-us-east-1}"
export S3_ACCESS_KEY="${S3_ACCESS_KEY:-wayposts}"
export S3_SECRET_KEY="${S3_SECRET_KEY:-wayposts123}"
export S3_PATH_STYLE="${S3_PATH_STYLE:-true}"
export MEDIA_MAX_BYTES="${MEDIA_MAX_BYTES:-10485760}"
export MEDIA_PRESIGN_TTL="${MEDIA_PRESIGN_TTL:-10m}"
export MEDIA_PUBLIC_BASE="${MEDIA_PUBLIC_BASE:-http://localhost:9000/wayposts-media}"

# Uma porta ocupada costuma ser um binário antigo. O novo morreria no bind e o
# teste falaria com o processo velho.
busy=0
for port in 8080 8081 8082 8083 8084; do
  if lsof -tiTCP:"$port" -sTCP:LISTEN >/dev/null 2>&1; then
    echo "porta $port já está em uso:" >&2
    lsof -nP -iTCP:"$port" -sTCP:LISTEN >&2 || true
    busy=1
  fi
done
if ((busy)); then
  echo "rode ./scripts/stop.sh e tente de novo" >&2
  exit 1
fi

pids=()
cleanup() {
  trap - EXIT INT TERM
  if ((${#pids[@]})); then
    kill "${pids[@]}" 2>/dev/null || true
    # Um serviço pode demorar no shutdown; depois de 2s, força o kill.
    for _ in {1..20}; do
      alive=0
      for pid in "${pids[@]}"; do
        if kill -0 "$pid" 2>/dev/null; then
          alive=1
          break
        fi
      done
      ((alive)) || break
      sleep 0.1
    done
    kill -9 "${pids[@]}" 2>/dev/null || true
    wait 2>/dev/null || true
  fi
}
trap cleanup EXIT INT TERM

go build -o bin/social ./cmd/social
go build -o bin/post ./cmd/post
go build -o bin/timeline ./cmd/timeline
go build -o bin/fanout ./cmd/fanout
go build -o bin/gateway ./cmd/gateway

# Process substitution mantém $! apontando para o binário (com pipe, $! seria o
# sed e o binário sobreviveria ao kill).
./bin/social > >(sed 's/^/[social] /') 2>&1 &
pids+=($!)
./bin/post > >(sed 's/^/[post] /') 2>&1 &
pids+=($!)
./bin/timeline > >(sed 's/^/[timeline] /') 2>&1 &
pids+=($!)
./bin/fanout > >(sed 's/^/[fanout] /') 2>&1 &
pids+=($!)
./bin/gateway > >(sed 's/^/[gateway] /') 2>&1 &
pids+=($!)

echo "gateway em http://localhost:8080"
wait
