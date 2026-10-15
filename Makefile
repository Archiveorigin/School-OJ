SHELL := /bin/sh
COMPOSE ?= docker compose

.PHONY: up down restart logs ps test smoke sandbox-images api-test web-test worker-test fmt migrate backup restore cleanup legacy-report

up:
	$(COMPOSE) build api worker web
	$(COMPOSE) up -d --wait postgres redis minio
	$(COMPOSE) run --rm --no-deps api --migrate-only
	$(COMPOSE) up -d --wait api worker web

down:
	$(COMPOSE) down

restart:
	$(COMPOSE) restart

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
	./scripts/warm_sandbox_images.sh

migrate:
	$(COMPOSE) run --rm --no-deps api --migrate-only

backup:
	./scripts/backup.sh

restore:
	./scripts/restore.sh "$(BACKUP)"

cleanup:
	./scripts/cleanup_docker.sh

legacy-report:
	./scripts/legacy_api_report.sh
