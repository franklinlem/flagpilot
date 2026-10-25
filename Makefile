SHELL := /bin/bash
ROOT := $(CURDIR)
export PATH := $(ROOT)/.tools/go/bin:$(ROOT)/.tools/bin:$(PATH)
COMPOSE := docker compose --env-file .env -f deploy/compose.dev.yml

.PHONY: setup install dev-db dev-api dev-web dev-containers down migrate test test-go test-integration lint build generate ci
setup:
	python3 scripts/setup.py
install:
	npm ci
	cd apps/api && go mod download
dev-db:
	$(COMPOSE) up -d --wait db
	$(COMPOSE) run --rm migrate
dev-api:
	python3 scripts/run-local.py go -C apps/api run ./cmd/server
dev-web:
	python3 scripts/run-local.py npm run dev
dev-containers:
	$(COMPOSE) --profile full up -d --build --wait
down:
	$(COMPOSE) --profile full down
migrate:
	$(COMPOSE) run --rm migrate
test: test-go
	npm test
test-go:
	cd apps/api && go test -race ./...
test-integration:
	cd apps/api && go test -race -tags=integration -count=1 ./tests/integration
lint:
	cd apps/api && test -z "$$(gofmt -l .)" && go vet ./...
	npm run lint
	npm run typecheck
build:
	cd apps/api && go build ./...
	npm run build
	npm run check:package --workspace @flagpilot/sdk-js
generate:
	cd apps/api && sqlc generate
	npm run generate:api
ci: lint test test-integration build

.PHONY: admin bootstrap reset-password
admin:
	mkdir -p .tools/bin
	cd apps/api && go build -o "$(ROOT)/.tools/bin/flagpilot-admin" ./cmd/admin
bootstrap: admin
	python3 scripts/run-local.py .tools/bin/flagpilot-admin bootstrap $(ARGS)
reset-password: admin
	python3 scripts/run-local.py .tools/bin/flagpilot-admin reset-password $(ARGS)

.PHONY: create-user
create-user: admin
	python3 scripts/run-local.py .tools/bin/flagpilot-admin create-user $(ARGS)

.PHONY: recovery-rehearsal
recovery-rehearsal:
	python3 scripts/recovery.py $(ARGS)

.PHONY: load-rehearsal
load-rehearsal:
	python3 scripts/load.py $(ARGS)

.PHONY: release-artifacts rollback-rehearsal
release-artifacts:
	python3 scripts/release-artifacts.py $(ARGS)
rollback-rehearsal:
	python3 scripts/rollback.py $(ARGS)
