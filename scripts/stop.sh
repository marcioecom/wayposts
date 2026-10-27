#!/usr/bin/env bash
# Encerra os binários do laboratório que estiverem escutando nas portas do dev.
set -uo pipefail

for port in 8080 8081 8082 8083 8084; do
  pids=$(lsof -tiTCP:"$port" -sTCP:LISTEN 2>/dev/null || true)
  if [ -n "$pids" ]; then
    echo "encerrando porta $port: $pids"
    # shellcheck disable=SC2086
    kill -9 $pids 2>/dev/null || true
  fi
done

pkill -9 -f 'bash ./scripts/dev.sh' 2>/dev/null || true
pkill -9 -f 'bin/wsprobe' 2>/dev/null || true

echo "portas liberadas"
