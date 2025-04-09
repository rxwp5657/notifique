FROM golang:1.23.4-alpine3.19 AS builder

ENV GIN_MODE=release

WORKDIR /app

COPY ./service/go.mod .

RUN go mod download 

COPY . .

WORKDIR /app/service

RUN go fmt ./... && \
    go vet ./... && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o notifique ./cmd/app/main.go

FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group

COPY --from=builder /app/service/notifique .

EXPOSE 8080

CMD [ "./notifique" ]
