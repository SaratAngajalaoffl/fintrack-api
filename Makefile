.PHONY: run build test fmt lint tidy

run:
	go run ./cmd/api

build:
	go build -o bin/fintrack-api ./cmd/api

test:
	go test ./...

fmt:
	gofmt -s -w .

lint:
	golangci-lint run ./... 2>/dev/null || go vet ./...

tidy:
	go mod tidy
