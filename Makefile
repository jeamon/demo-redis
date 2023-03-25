# set shortened commit hash as tag for compose use.
export TAG.DEMO.REDIS=$(shell git rev-parse --short HEAD)

# Perform below commands
all: static-check test.all run

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
	go test -v -coverprofile=coverage.out ./... && go tool cover -func=coverage.out

## Obtain codebase testing coverage and view stats in browser.
test.cover.html:
	go test -v -coverprofile=coverage.out ./... && go tool cover -html=coverage.out

## Obtain codebase testing coverage and view stats on console and in browser.
test.cover.all: clean.test test.cover.console test.cover.html

## Run all tests and locally build the program
local.build: clean.all test.all
	CGO_ENABLED=0 go build -o bin/app.demo.redis -a -ldflags "-extldflags '-static' -X 'main.GitCommit=$(shell git rev-parse --short HEAD)' -X 'main.GitTag=$(shell git describe --tags --abbrev=0)' -X 'main.BuildTime=$(shell date -u '+%Y-%m-%d %I:%M:%S %p GMT')'" .

## Run lint and test-unit commands
local.run:
	go run -ldflags "-X 'main.GitCommit=$(shell git rev-parse --short HEAD)' -X 'main.GitTag=$(shell git describe --tags --abbrev=0)' -X 'main.BuildTime=$(shell date -u '+%Y-%m-%d %I:%M:%S %p GMT')'" .

## Execute tests then build contaners images
docker.build: test.all 
	docker-compose build

## Start the app service and required services
docker.run:
	docker-compose up --detach app.demo.redis

## Stop all running services (app and redis)
docker.stop:
	docker-compose stop

## Format the codebase
format:
	gofumpt -l -w .