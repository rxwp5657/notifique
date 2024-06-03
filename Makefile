.DEFAULT_GOAL := build

fmt:
	go fmt ./...
.PHONY:fmt

lint: fmt
	golint ./...
.PHONY:lint

vet: fmt
	go vet ./...
.PHONY:vet

build: vet
	go build .
.PHONY:build

run: vet
	go run ./cmd/app/main.go
.PHONY:run

test: vet
	go test ./test
.PHONY:test

deploy-dynamodb: vet
	go run ./cmd/deployments/dynamodb/main.go
.PHONY:deploy-dynamodb

deploy-postgres: vet
	go run ./cmd/deployments/postgres/main.go
.PHONY:deploy-postgres

deploy-sqs: vet
	go run ./cmd/deployments/sqs/main.go
.PHONY:deploy-sqs
