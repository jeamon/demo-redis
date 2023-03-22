# Perform below commands
all: static-check test run

static-check:
	golangci-lint --tests=false run

## Clean package and fix linters warnings
lint:
	go mod tidy
	golangci-lint --fix --tests=false --timeout=2m30s run

## Remove temporary files and cached tests results
clean.test:
	go clean -testcache

## Remove temporary and cached builds files
clean.build:
	go clean -cache

## Remove temporary and cached builds files
clean.all: clean.test clean.build
	rm -rf ./bin

## Run all tests (unit and integration and e2e) after cache cleanup
test.all: clean.test
	go test -v ./... -count=1

## Obtain codebase testing coverage and view stats in console.
test.cover.console:
	go test -coverprofile=coverage.out ./... && go tool cover -func=coverage.out

## Obtain codebase testing coverage and view stats in browser.
test.cover.html:
	go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out

## Obtain codebase testing coverage and view stats on console and in browser.
test.cover.all: clean.test test.cover.console test.cover.html

## Run all tests and locally build the program
local.build: clean.all test.all
	go build -o bin/demo-redis -a -ldflags "-extldflags '-static' -X 'main.GitCommit=$(git rev-parse --short HEAD)' -X 'main.GitTag=$(git describe --tags --abbrev=0)' -X 'main.BuildTime=$(date -u '+%Y-%m-%d %I:%M:%S %p GMT')'" .

## Run lint and test-unit commands
local.run:
	go run -ldflags "-X 'main.GitCommit=$(git rev-parse --short HEAD)' -X 'main.GitTag=$(git describe --tags --abbrev=0)' -X 'main.BuildTime=$(date -u '+%Y-%m-%d %I:%M:%S %p GMT')'" .

## Execute tests then build and run the app container
docker.run: test.all
	docker-compose build && docker-compose up app

## Format the codebase
format:
	gofumpt -l -w .