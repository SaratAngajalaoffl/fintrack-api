.PHONY: run build test fmt lint tidy

run:
	go run ./cmd/api

build:
	go build -o bin/fintrack-api ./cmd/api

test:
	go test ./...

# Cross-package coverage (integration tests exercise handlers + repository via HTTP).
test-cover:
	go test ./... -covermode=atomic -coverpkg=./internal/...,./pkg/... -coverprofile=coverage.out
	go tool cover -func=coverage.out | tail -n 1

fmt:
	gofmt -s -w .

lint:
	golangci-lint run ./... 2>/dev/null || go vet ./...

tidy:
	go mod tidy
