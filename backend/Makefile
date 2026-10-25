.PHONY: build test vet openapi run migrate-up migrate-down seed up down test-integration

export GOCACHE := $(CURDIR)/.cache/go-build
export GOMODCACHE := $(CURDIR)/.cache/go-mod
export GOPATH := $(CURDIR)/.cache/go
export TMPDIR := $(CURDIR)/.cache/tmp

build:
	mkdir -p tmp bin .cache/tmp
	go build -o bin/erp .

test:
	mkdir -p tmp .cache/tmp
	go test ./...

vet:
	mkdir -p tmp .cache/tmp
	go vet ./...

openapi:
	mkdir -p tmp .cache/tmp
	go test ./internal/router -run TestOpenAPIContractCompiles

run:
	mkdir -p tmp .cache/tmp
	go run .

migrate-up:
	mkdir -p tmp .cache/tmp
	go run . migrate up

migrate-down:
	mkdir -p tmp .cache/tmp
	go run . migrate down

seed:
	docker compose run --rm seed

up:
	docker compose up -d --build

down:
	docker compose down

test-integration:
	@trap 'docker compose -p mini-erp-assessment-tests -f docker-compose.test.yml down --volumes' EXIT; \
		docker compose -p mini-erp-assessment-tests -f docker-compose.test.yml up --build --abort-on-container-exit --exit-code-from test
