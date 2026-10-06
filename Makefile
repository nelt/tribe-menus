# Project commands (ADR 0010): a table of contents, no logic here.
# Each target calls go, go tool or npm in a line or two; anything longer
# belongs in a project tool (Go program or npm script).

VERSION ?= dev
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null)

.DEFAULT_GOAL := help
.PHONY: help tools dev seed generate lint test acceptance e2e vuln build dist ci

help: ## List the targets
	@grep -E '^[a-z0-9]+:.*## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*## "} {printf "  %-10s %s\n", $$1, $$2}'

tools: ## Install dependencies, Playwright browsers and their system libraries (idempotent)
	go mod download
	cd web && npm ci --ignore-scripts && npx playwright install --with-deps chromium webkit

dev: ## Rebuild the front end on change and serve on http://localhost:8080 (Ctrl-C stops both)
	go build -o bin/tribe-menus-dev ./cmd/tribe-menus
	npm --prefix web run watch & pid=$$!; trap 'kill $$pid 2>/dev/null' EXIT INT TERM; bin/tribe-menus-dev serve -dev

seed: ## Create the demonstration tribe in data/ (http://localhost:8080/tribes/demo/)
	go run ./cmd/tribe-menus admin seed

generate: ## Generate the data access code from the SQL queries (sqlc)
	go tool sqlc generate

lint: ## gofmt, go vet, staticcheck, generated code up to date, tsc, webcheck and actionlint
	@unformatted=$$(gofmt -l .); test -z "$$unformatted" || { echo "gofmt needed:"; echo "$$unformatted"; exit 1; }
	go vet ./...
	go tool staticcheck ./...
	go tool sqlc diff
	cd web && npm run typecheck
	go run ./internal/tools/webcheck
	go tool actionlint -shellcheck= -pyflakes=

test: ## Go and TypeScript unit tests
	go test ./...
	cd web && npm test

acceptance: ## Gherkin scenarios against the API (godog), compared with acceptance/pending.txt
	go test ./acceptance -count=1 -acceptance

e2e: ## Playwright scenarios
	cd web && npm run e2e

vuln: ## Known vulnerabilities in Go and npm dependencies
	go tool govulncheck ./...
	cd web && npm audit

build: ## Front end, then static binary for linux/amd64 in bin/ (VERSION, COMMIT)
	cd web && npm run build
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT)" -o bin/tribe-menus ./cmd/tribe-menus

dist: build ## Archive in dist/: binary, site/, deploy/ and LICENSE, with its SHA-256 (VERSION)
	go run ./internal/tools/dist -version $(VERSION)

ci: lint test acceptance e2e vuln build dist ## All checks required by the CI, in order
