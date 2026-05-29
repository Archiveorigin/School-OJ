SHELL := /bin/sh
COMPOSE ?= docker compose

.PHONY: up down restart logs ps test smoke sandbox-images api-test web-test worker-test fmt

up:
	$(COMPOSE) up -d --build

down:
	$(COMPOSE) down

restart:
	$(COMPOSE) down
	$(COMPOSE) up -d --build

logs:
	$(COMPOSE) logs -f --tail=200

ps:
	$(COMPOSE) ps

test: api-test worker-test web-test

api-test:
	cd apps/api && go test ./...

worker-test:
	cd apps/worker && go test ./...

web-test:
	cd apps/web && npm test -- --run

fmt:
	cd apps/api && gofmt -w .
	cd apps/worker && gofmt -w .
	cd apps/web && npm run format

smoke:
	./scripts/smoke.sh

sandbox-images:
	./scripts/pull_sandbox_images.sh
