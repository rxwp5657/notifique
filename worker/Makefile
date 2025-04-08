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
	go test ./test/unit/... ./test/integration/...
.PHONY:test

dependency-injection: vet
	cd ./internal/di && go run github.com/google/wire/cmd/wire
.PHONY:dependency-injection

gen-mocks: vet
	go run go.uber.org/mock/mockgen \
		-source=./internal/worker/worker.go \
		-destination=./internal/testutils/mocks/worker.go \
		-package=mocks

.PHONY:gen-mocks
