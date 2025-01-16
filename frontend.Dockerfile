FROM golang:1.23-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o /usr/bin/railpack cmd/cli/main.go

FROM alpine

COPY --from=builder /usr/bin/railpack /usr/bin/railpack
ENTRYPOINT ["/usr/bin/railpack", "frontend"]
