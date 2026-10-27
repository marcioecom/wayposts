DATABASE_URL ?= postgres://wayposts:wayposts@localhost:5432/wayposts?sslmode=disable
REDIS_ADDR ?= localhost:6379
RABBIT_URL ?= amqp://guest:guest@localhost:5672/
USERS ?= 2000
FOLLOWS ?= 100
CELEBS ?= 3
CELEB_FOLLOWERS ?= 600
MAX_FOLLOWERS ?= 400
THRESHOLD ?= 500
IMAGE ?= ghcr.io/marcioecom/wayposts:dev
K6_IMAGE ?= ghcr.io/marcioecom/wayposts-k6:dev

export DATABASE_URL
export REDIS_ADDR
export RABBIT_URL
export FANOUT_THRESHOLD := $(THRESHOLD)

.PHONY: up down reset migrate seed build test test-integration vet run demo nudge media image image-k6 k8s-diff k8s-apply k8s-smoke load load-ramp load-spike load-soak load-celeb load-cold flush-posts

up:
	docker compose up -d --wait

down:
	docker compose down

reset:
	docker compose down -v
	docker compose up -d --wait

migrate:
	go run ./cmd/migrate

seed:
	go run ./cmd/seed -users $(USERS) -follows $(FOLLOWS) -celebs $(CELEBS) -celeb-followers $(CELEB_FOLLOWERS) -max-followers $(MAX_FOLLOWERS)

build:
	go build ./...

test:
	go test ./...

test-integration:
	TEST_DATABASE_URL="$(DATABASE_URL)" go test ./internal/social/ -run TestFollowerCacheAgainstPostgres -v

vet:
	go vet ./...

run:
	./scripts/dev.sh

demo:
	./scripts/demo.sh

nudge:
	./scripts/nudge-demo.sh

media:
	./scripts/media-demo.sh

image:
	docker build --platform linux/amd64 -t $(IMAGE) .
	docker push $(IMAGE)

image-k6:
	docker build --platform linux/amd64 -f deploy/k6/Dockerfile -t $(K6_IMAGE) .
	docker push $(K6_IMAGE)

k8s-diff:
	kubectl diff -k deploy/k8s/overlays/orbit || true

k8s-apply:
	kubectl apply -k deploy/k8s/overlays/orbit

k8s-smoke:
	kubectl -n wayposts delete job k6 --ignore-not-found
	kubectl apply -f deploy/k8s/overlays/orbit/k6-job.yaml
	kubectl -n wayposts wait --for=condition=complete job/k6 --timeout=15m
	kubectl -n wayposts logs job/k6 | tail -45

load:
	./scripts/load.sh steady

load-ramp:
	./scripts/load.sh ramp

load-spike:
	./scripts/load.sh spike

load-soak:
	./scripts/load.sh soak

load-celeb:
	./scripts/load.sh celebrity

load-cold:
	./scripts/load.sh cold-cache

flush-posts:
	docker exec wayposts-redis-1 redis-cli EVAL "local ks=redis.call('keys','post:*'); for i=1,#ks do redis.call('del',ks[i]) end; return #ks" 0
