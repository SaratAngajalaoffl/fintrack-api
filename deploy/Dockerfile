FROM golang:1.24-alpine AS builder

RUN apk add --no-cache ca-certificates git

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/fintrack-api ./cmd/api

FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata wget

COPY --from=builder /out/fintrack-api /usr/local/bin/fintrack-api
COPY --from=builder /src/migrations /migrations

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/fintrack-api"]
