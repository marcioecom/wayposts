#!/usr/bin/env bash
# Roda um cenário k6 em Docker. k6 não precisa estar instalado na máquina.
#
#   ./scripts/load.sh steady
#   RATE=200 DURATION=5m ./scripts/load.sh steady
#   START_RATE=20 MAX_RATE=1500 ./scripts/load.sh ramp
#
# De dentro do container, "localhost" é o container; por isso o default usa
# host.docker.internal. Para MEDIA=1, suba o S3_ENDPOINT como
# http://host.docker.internal:9000 (a URL assinada carrega o host).
set -euo pipefail
cd "$(dirname "$0")/.."

SCENARIO="${1:-steady}"
if [ ! -f "load/$SCENARIO.js" ]; then
  echo "cenário não encontrado: load/$SCENARIO.js" >&2
  exit 1
fi
shift || true

IMAGE="${K6_IMAGE:-grafana/k6:latest}"
BASE="${BASE:-http://host.docker.internal:8080}"

args=(-e "BASE=$BASE")
for name in USERS CELEBS TIMELINE_LIMIT ENFORCE MEDIA \
  RATE DURATION START_RATE MAX_RATE RAMP_DURATION WARMUP_DURATION HOLD_DURATION \
  BASE_RATE SPIKE_FACTOR SPIKE_DURATION SOAK \
  PREALLOC_VUS MAX_VUS \
  HOME_PCT PROFILE_PCT POST_PCT CELEB_WRITE_PCT; do
  if [ -n "${!name:-}" ]; then
    args+=(-e "$name=${!name}")
  fi
done

exec docker run --rm -i \
  -v "$PWD/load:/load:ro" \
  "${args[@]}" \
  "$IMAGE" run "/load/$SCENARIO.js" "$@"
