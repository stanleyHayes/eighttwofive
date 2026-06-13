.PHONY: dev build test test-short lint tidy docker

dev:
	go run ./cmd/server

build:
	CGO_ENABLED=0 go build -trimpath -o bin/server ./cmd/server

test:
	go test -race ./...

test-short:
	go test -race -short ./...

# Lower GOGC to keep memory usage under control when every linter is enabled.
lint:
	GOGC=20 golangci-lint run --concurrency=1

tidy:
	go mod tidy

docker:
	docker build -t eightfivetwo-api .
