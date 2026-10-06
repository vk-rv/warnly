.DEFAULT_GOAL := help
export PATH := $(CURDIR)/.tools/node/bin:$(CURDIR)/.tools/go/bin:$(PATH)
COMPOSE ?= docker compose

.PHONY: help install setup frontend-install frontend-build frontend-check frontend-test frontend-dev build run dev dev-down hooks align lint fmt test test-unit test-duckdb bench
help:
	@echo 'make install        Install Go/Node locally if needed and fetch dependencies (Linux/macOS)'
	@echo 'make dev            Build and run Go + Svelte + databases with Docker Compose'
	@echo 'make dev-down       Stop the development stack (keep data)'
	@echo 'make build          Build bin/warnly with embedded static frontend'
	@echo 'make run            Build and run locally using .env (databases must be available)'
	@echo 'make frontend-dev   Vite hot reload; proxy API to WARNLY_API_URL or localhost:8080'
	@echo 'make frontend-check / test-unit / test  Run frontend checks / unit / integration tests'

install:
	@sh scripts/bootstrap.sh
	npm ci --no-audit --no-fund
	go mod download
	@test -f .env || cp .env.sample .env

setup: install

frontend-install:
	npm ci --no-audit --no-fund

frontend-build: frontend-install
	npm run build

frontend-check: frontend-install
	npm run check

frontend-test: frontend-install
	npx playwright install chromium
	npm run test:e2e

frontend-dev: frontend-install
	npm run dev

build: frontend-build
	go build -o bin/warnly ./cmd/warnly

run: build
	@test -f .env || cp .env.sample .env
	./bin/warnly

dev:
	@test -f .env || cp .env.sample .env
	$(COMPOSE) up --build

dev-down:
	$(COMPOSE) down

hooks:
	pre-commit install

align:
	$$(go env GOPATH)/bin/fieldalignment -fix ./...

lint:
	$$(go env GOPATH)/bin/golangci-lint run

fmt:
	gofmt -s -w cmd internal migrations

# Database integration tests require Docker.
test: frontend-build
	INTEGRATION=1 go test -count=1 ./... -v

test-unit:
	go test ./...

test-duckdb:
	INTEGRATION=1 go test -count=1 -race -v ./internal/duckdb ./cmd/warnly

bench:
	drill --benchmark benchmark.yml
