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

gen-mocks: vet
	go run go.uber.org/mock/mockgen \
		-source=./internal/controllers/users.go \
		-destination=./internal/testutils/mocks/users.go

	go run go.uber.org/mock/mockgen \
		-source=./internal/controllers/distribution_lists.go \
		-destination=./internal/testutils/mocks/distribution_lists.go

	go run go.uber.org/mock/mockgen \
		-source=./internal/controllers/notifications.go \
		-destination=./internal/testutils/mocks/notifications.go

	go run go.uber.org/mock/mockgen \
		-source=./internal/controllers/templates.go \
		-destination=./internal/testutils/mocks/templates.go

.PHONY:gen-mocks

dependency-injection: vet
	cd ./internal/di && go run github.com/google/wire/cmd/wire
.PHONY:dependency-injection

deploy-dynamodb: vet
	go run ./cmd/deployments/dynamodb/main.go
.PHONY:deploy-dynamodb

deploy-postgres: vet
	go run ./cmd/deployments/postgres/main.go
.PHONY:deploy-postgres

deploy-sqs: vet
	go run ./cmd/deployments/sqs/main.go
.PHONY:deploy-sqs

deploy-rabbitmq: vet
	go run ./cmd/deployments/rabbitmq/main.go
.PHONY:deploy-rabbitmq