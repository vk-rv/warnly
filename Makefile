templates:
	$(shell go env GOPATH)/bin/templ generate

.PHONY: setup
setup: ## Setup the precommit hook
	@which pre-commit > /dev/null 2>&1 || (echo "pre-commit not installed see README." && false)
	@pre-commit install

lint:
	$(shell go env GOPATH)/bin/golangci-lint run

dev:
	@# Create .env from .env.sample if it doesn't exist
	@[ -f .env ] || cp .env.sample .env
	@# Use docker compose or docker-compose based on availability
	@if command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1; then \
		docker compose up; \
	elif command -v docker-compose >/dev/null 2>&1; then \
		docker-compose up; \
	else \
		echo "Error: Neither 'docker compose' nor 'docker-compose' is available"; \
		exit 1; \
	fi

.PHONY: fmt
fmt:
	gofmt -s -w .
	$(shell go env GOPATH)/bin/templ fmt .
	$(shell go env GOPATH)/bin/gofumpt -l -w .

.PHONY: tailwindcss
tailwindcss:
	npx @tailwindcss/cli -i internal/server/asset/static/main.css -o internal/server/asset/static/tailwind.css --minify
