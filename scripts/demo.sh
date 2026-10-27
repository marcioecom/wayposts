#!/usr/bin/env bash
set -euo pipefail

BASE="${BASE:-http://localhost:8080}"
jq() { python3 -m json.tool; }

echo "== usuário 11 segue o 10 (usuário comum) =="
curl -s -o /dev/null -w "%{http_code}\n" -X POST "$BASE/v1/follow" \
  -H "Authorization: Bearer 11" -H 'Content-Type: application/json' \
  -d '{"followee_id":10}'

echo "== usuário 12 segue o 10 (para ver o like na timeline) =="
curl -s -o /dev/null -w "%{http_code}\n" -X POST "$BASE/v1/follow" \
  -H "Authorization: Bearer 12" -H 'Content-Type: application/json' \
  -d '{"followee_id":10}'

echo "== usuário 5 segue a celebridade 1 (será lida no merge) =="
curl -s -o /dev/null -w "%{http_code}\n" -X POST "$BASE/v1/follow" \
  -H "Authorization: Bearer 5" -H 'Content-Type: application/json' \
  -d '{"followee_id":1}'

echo "== usuário 10 publica =="
POST10=$(curl -s -X POST "$BASE/v1/posts" \
  -H "Authorization: Bearer 10" -H 'Content-Type: application/json' \
  -d '{"body":"primeiro post do 10"}')
echo "$POST10" | jq
POST10_ID=$(echo "$POST10" | python3 -c 'import sys,json; print(json.load(sys.stdin)["id"])')

echo "== usuário 12 curte o post do 10 =="
curl -s -X POST "$BASE/v1/posts/$POST10_ID/likes" -H "Authorization: Bearer 12" | jq

echo "== celebridade 1 publica =="
POST1=$(curl -s -X POST "$BASE/v1/posts" \
  -H "Authorization: Bearer 1" -H 'Content-Type: application/json' \
  -d '{"body":"anúncio da marca"}')
echo "$POST1" | jq

echo "== aguardando fanout (2s) =="
sleep 2

echo "== timeline do 11 (push: post do 10, sem like) =="
curl -s "$BASE/v1/timeline/home" -H "Authorization: Bearer 11" | jq

echo "== timeline do 12 (push + like: like_count=1, viewer_liked=true) =="
curl -s "$BASE/v1/timeline/home" -H "Authorization: Bearer 12" | jq

echo "== usuário 12 descurte =="
curl -s -X DELETE "$BASE/v1/posts/$POST10_ID/likes" -H "Authorization: Bearer 12" | jq

echo "== timeline do 5 (merge: post da celebridade 1) =="
curl -s "$BASE/v1/timeline/home" -H "Authorization: Bearer 5" | jq

echo "== perfil da celebridade 1 =="
curl -s "$BASE/v1/timeline/profile/1" -H "Authorization: Bearer 5" | jq
