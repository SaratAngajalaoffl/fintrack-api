.PHONY: run build test fmt lint tidy swagger

run:
	go run ./cmd/api

build:
	go build -o bin/fintrack-api ./cmd/api

# Regenerate OpenAPI 2.0 spec and docs package (swag) after changing handler comments or API metadata in cmd/api/main.go.
swagger:
	go run github.com/swaggo/swag/cmd/swag@v1.16.4 init -g main.go -d ./cmd/api,./internal/handler -o ./docs --parseInternal

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
