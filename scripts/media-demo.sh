#!/usr/bin/env bash
# Exercita o caminho de mídia: presign -> PUT direto no bucket -> validação por
# HEAD na criação do post. Usa MinIO no lugar do R2.
set -euo pipefail
cd "$(dirname "$0")/.."

BASE="${BASE:-http://localhost:8080}"
PUBLIC_BASE="${MEDIA_PUBLIC_BASE:-http://localhost:9000/wayposts-media}"
AUTHOR="${AUTHOR:-10}"
TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT

jq() { python3 -m json.tool; }

# PNG 1x1 válido.
printf 'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8DwHwAFAAH/q842iQAAAABJRU5ErkJggg==' \
  | base64 --decode >"$TMP/pixel.png"
SIZE=$(wc -c <"$TMP/pixel.png" | tr -d ' ')

echo "== pedindo presign de 1 imagem ($SIZE bytes) =="
PRESIGN=$(curl -s -X POST "$BASE/v1/media/presign" \
  -H "Authorization: Bearer $AUTHOR" -H 'Content-Type: application/json' \
  -d "{\"items\":[{\"content_type\":\"image/png\",\"size\":$SIZE}]}")
echo "$PRESIGN" | jq

KEY=$(echo "$PRESIGN" | python3 -c 'import sys,json; print(json.load(sys.stdin)["items"][0]["key"])')
URL=$(echo "$PRESIGN" | python3 -c 'import sys,json; print(json.load(sys.stdin)["items"][0]["url"])')

echo "== upload direto no bucket =="
curl -s -o /dev/null -w "PUT %{http_code}\n" -X PUT "$URL" \
  -H 'Content-Type: image/png' --data-binary "@$TMP/pixel.png"

echo "== objeto acessível na URL pública =="
curl -s -o /dev/null -w "GET %{http_code} (%{size_download} bytes)\n" "$PUBLIC_BASE/$KEY"

echo "== criando post com a mídia =="
POST=$(curl -s -X POST "$BASE/v1/posts" \
  -H "Authorization: Bearer $AUTHOR" -H 'Content-Type: application/json' \
  -d "{\"body\":\"post com imagem\",\"media\":[\"$KEY\"]}")
echo "$POST" | jq
POST_ID=$(echo "$POST" | python3 -c 'import sys,json; print(json.load(sys.stdin)["id"])')

echo "== post relido traz a URL pública da mídia =="
curl -s "$BASE/v1/posts/$POST_ID" -H "Authorization: Bearer $AUTHOR" | jq '.media'

echo "== timeline hidrata a mesma URL =="
curl -s -o /dev/null -X POST "$BASE/v1/follow" \
  -H "Authorization: Bearer 11" -H 'Content-Type: application/json' \
  -d "{\"followee_id\":$AUTHOR}"
sleep 2
curl -s "$BASE/v1/timeline/home" -H "Authorization: Bearer 11" \
  | python3 -c "import sys,json; items=json.load(sys.stdin)['items']; print([p['media'] for p in items if p['id']==$POST_ID])"

echo "== chave de outro autor é rejeitada =="
curl -s -X POST "$BASE/v1/posts" \
  -H "Authorization: Bearer $AUTHOR" -H 'Content-Type: application/json' \
  -d "{\"body\":\"roubando mídia\",\"media\":[\"media/99/123.png\"]}" | jq

echo "== chave inexistente é rejeitada =="
curl -s -X POST "$BASE/v1/posts" \
  -H "Authorization: Bearer $AUTHOR" -H 'Content-Type: application/json' \
  -d "{\"body\":\"mídia fantasma\",\"media\":[\"media/$AUTHOR/000.png\"]}" | jq

echo "== tipo não suportado é rejeitado no presign =="
curl -s -X POST "$BASE/v1/media/presign" \
  -H "Authorization: Bearer $AUTHOR" -H 'Content-Type: application/json' \
  -d '{"items":[{"content_type":"application/pdf","size":100}]}' | jq

echo "== métricas de mídia =="
curl -s "http://localhost:8081/metrics" | grep -E '^wayposts_media_' || true
