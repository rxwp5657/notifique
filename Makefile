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

deploy-dynamodb: vet
	go run ./cmd/deployments/dynamodb/main.go
.PHONY:deploy-dynamodb
