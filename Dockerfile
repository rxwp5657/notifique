FROM golang:1.22.1-alpine3.19

ENV GIN_MODE=release

WORKDIR /app

COPY . .

RUN go get

RUN go build

EXPOSE 8080

ENTRYPOINT [ "./notifique" ]
