default: help

SHELL:=/usr/bin/env bash

BINDIR := $(CURDIR)/bin
BINNAME ?= app.demo.redis
GOLANGCI_LINT_VERSION:=1.52.0
# Download the linter executable file from https://github.com/golangci/golangci-lint/releases/tag/v1.52.2

GIT_COMMIT = $(shell git rev-parse HEAD)
GIT_SHA    = $(shell git rev-parse --short HEAD)
GIT_TAG    = $(shell git describe --tags --abbrev=0 --exact-match 2>/dev/null)
GIT_DIRTY  = $(shell test -n "`git status --porcelain`" && echo "dirty" || echo "clean")
CURRENT_TIME = $(shell date -u '+%Y-%m-%d %I:%M:%S %p GMT')

LDFLAGS = -X 'main.GitCommit=${GIT_SHA}' \
				-X 'main.GitTag=${GIT_TAG}' \
				-X 'main.BuildTime=${CURRENT_TIME}'

EXTLDFLAGS = "-extldflags '-static' ${LDFLAGS}"

# Set shortened commit hash as tag to be used by docker-compose
# Using recursive assignment to have more uptodate tag value.
export LATEST_COMMIT_DEMO_REDIS = $(GIT_SHA)

.PHONY: help
help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: all 
all: lint test.all local.build ## Run linting and all tests and build the local binary.

.PHONY: format
format: ## Format the codebase
	gofumpt -l -w .

.PHONY: info
info: ## Display useful infos.
	@echo "Git Tag:           ${GIT_TAG}"
	@echo "Git Commit:        ${GIT_COMMIT}"
	@echo "Git Tree State:    ${GIT_DIRTY}"
	@echo "Current DateTime:  ${CURRENT_TIME}"

.PHONY: install-linter
install-linter: ## Install golangci-lint tool.
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v${GOLANGCI_LINT_VERSION}
	
.PHONY: lint
lint: ## Updates modules and execute linters.
	## use `make install-linter` to install linters if missing
	## or download the executable file from https://github.com/golangci/golangci-lint/releases/
	go mod tidy
	golangci-lint -v run --skip-dirs bin

.PHONY: clean.test
clean.test: ## Remove temporary files and cached tests results.
	go clean -testcache

.PHONY: clean.build
clean.build: ## Remove temporary and cached builds files
	go clean -cache

.PHONY: clean.all
clean.all: clean.test clean.build ## Remove temporary and cached builds files
	rm -rf ./bin

.PHONY: ci.test
ci.test: ## Run unit tests only and output coverage.
	go test -v -count=1 -coverprofile=coverage.out ./... && go tool cover -func=coverage.out

.PHONY: test.unit
test.unit: clean.test ## Remove cache and Run unit tests only.
	go test -v ./... -count=1

.PHONY: test.integration
test.integration: clean.test ## Remove cache and Run integration tests only.
	go test -v --tags=integration ./... -count=1

.PHONY: test.e2e
test.e2e: clean.test ## Remove cache and Run all e2e tests only.
	go test -v --tags=e2e ./... -count=1

.PHONY: test.all
test.all: test.unit test.integration test.e2e ## Run all tests (unit & integration & e2e)

.PHONY: coverage.console
coverage.console: clean.test ## Testing coverage and view stats in console.
	go test -v -coverprofile=coverage.out ./... && go tool cover -func=coverage.out

.PHONY: coverage.html
coverage.html: clean.test ## Testing coverage and view stats in browser.
	go test -v -coverprofile=coverage.out ./... && go tool cover -html=coverage.out

.PHONY: coverage.all ## Testing coverage and view stats both in console and browser.
coverage.all: clean.test coverage.console coverage.html

.PHONY: local.build
local.build: clean.all test.all clean.all ## Run all tests and locally build the program
	CGO_ENABLED=0 go build -o ${BINDIR}/${BINNAME} -a -ldflags ${EXTLDFLAGS} .

.PHONY: local.run
local.run: ## Run lint and test-unit commands
	go run -ldflags "${LDFLAGS}" .

.PHONY: docker.build
docker.build: test.all ## Execute tests then build contaners images
	docker-compose build

.PHONY: docker.up
docker.up: ## Start the app service and required services
	docker-compose up --detach app.demo.redis

.PHONY: docker.stop
docker.stop: ## Stop all running services (app and redis)
	docker-compose stop

.PHONY: docker.down
docker.down: ## Stop & Remove all services (app and redis) and network.
	docker-compose down

.PHONY: docker.clean
docker.clean: ## Stop & Remove all app containers (without the volume) and delete the images.
	@docker rm -f $(docker ps -aq --filter "name=app.demo.redis")
	@docker rmi -f $(docker images -aq --filter="reference=app.demo.redis:*")

.PHONY: swagger.generate
swagger.generate: ## Install swaggo/swag and generate openapi specs.
## use v1.8.12 due to https://github.com/swaggo/swag/issues/1568
	go install github.com/swaggo/swag/cmd/swag@v1.8.12
	swag init -g api.services.go
