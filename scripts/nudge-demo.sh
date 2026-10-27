#!/usr/bin/env bash
# Demonstra o nudge por WebSocket: abre uma conexão, publica um post de um
# autor seguido e mostra a mensagem que o gateway envia na varredura seguinte.
set -euo pipefail

BASE="${BASE:-http://localhost:8080}"
WS="${WS:-ws://localhost:8080/v1/ws}"
REDIS_CONTAINER="${REDIS_CONTAINER:-wayposts-redis-1}"
FOLLOWER="${FOLLOWER:-11}"
AUTHOR="${AUTHOR:-10}"

echo "== usuário $FOLLOWER segue o $AUTHOR =="
curl -s -o /dev/null -w "%{http_code}\n" -X POST "$BASE/v1/follow" \
  -H "Authorization: Bearer $FOLLOWER" -H 'Content-Type: application/json' \
  -d "{\"followee_id\":$AUTHOR}"

HEAD=$(docker exec "$REDIS_CONTAINER" redis-cli ZREVRANGE "feed:$FOLLOWER" 0 0 | tr -d '\r')
echo "== head atual de $FOLLOWER: ${HEAD:-<feed vazio>} =="

LOG=$(mktemp)
cleanup() {
  if [ -n "${PROBE:-}" ]; then
    kill "$PROBE" 2>/dev/null || true
    wait "$PROBE" 2>/dev/null || true
  fi
  rm -f "$LOG"
}
trap cleanup EXIT

# Compila o probe para poder encerrá-lo pelo PID, porque com `go run` o processo
# que escuta é filho do compilador e sobrevive ao kill.
go build -o bin/wsprobe ./cmd/wsprobe
./bin/wsprobe -url "$WS" -user "$FOLLOWER" -last-seen "${HEAD:-0}" -timeout 30s >"$LOG" 2>&1 &
PROBE=$!

# Espera o handshake para não perder o post na primeira varredura.
for _ in $(seq 1 40); do
  grep -q 'conectado' "$LOG" && break
  sleep 0.25
done

echo "== $AUTHOR publica =="
curl -s -X POST "$BASE/v1/posts" \
  -H "Authorization: Bearer $AUTHOR" -H 'Content-Type: application/json' \
  -d '{"body":"post anunciado por WebSocket"}' | python3 -m json.tool

echo "== aguardando o nudge (NUDGE_INTERVAL + folga) =="
for _ in $(seq 1 30); do
  grep -q 'nudge:' "$LOG" && break
  sleep 1
done

echo "== saída do probe =="
cat "$LOG"

echo "== métricas de nudge =="
curl -s "$BASE/metrics" | grep -E '^wayposts_ws_(nudges|connections)' || true
