# Project commands (ADR 0010): a table of contents, no logic here.
# Each target calls go, go tool or npm in a line or two; anything longer
# belongs in a project tool (Go program or npm script).

VERSION ?= dev
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null)

.DEFAULT_GOAL := help
.PHONY: help tools dev lint test e2e vuln build ci

help: ## List the targets
	@grep -E '^[a-z0-9]+:.*## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*## "} {printf "  %-8s %s\n", $$1, $$2}'

tools: ## Install dependencies, Playwright browsers and their system libraries (idempotent)
	go mod download
	cd web && npm ci --ignore-scripts && npx playwright install --with-deps chromium webkit

dev: ## Rebuild the front end on change and serve on http://localhost:8080 (Ctrl-C stops both)
	npm --prefix web run watch & pid=$$!; trap 'kill $$pid 2>/dev/null' EXIT INT TERM; \
	go run ./cmd/tribe-menus serve -dev

lint: ## gofmt, go vet, staticcheck, tsc and webcheck
	@unformatted=$$(gofmt -l .); test -z "$$unformatted" || { echo "gofmt needed:"; echo "$$unformatted"; exit 1; }
	go vet ./...
	go tool staticcheck ./...
	cd web && npm run typecheck
	go run ./internal/tools/webcheck

test: ## Go and TypeScript unit tests
	go test ./...
	cd web && npm test

e2e: ## Playwright scenarios
	cd web && npm run e2e

vuln: ## Known vulnerabilities in Go and npm dependencies
	go tool govulncheck ./...
	cd web && npm audit

build: ## Front end, then static binary in bin/ (VERSION, COMMIT)
	cd web && npm run build
	CGO_ENABLED=0 go build -trimpath -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT)" -o bin/tribe-menus ./cmd/tribe-menus

ci: lint test e2e vuln build ## All checks required by the CI, in order
