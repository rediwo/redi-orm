# ReORM Makefile

.PHONY: help build test test-verbose test-short test-cover test-race test-integration test-benchmark test-docker test-mysql test-postgresql test-sqlite docker-up docker-down clean fmt lint vet mod-tidy mod-download example run-example

# Default target
help:
	@echo "Available targets:"
	@echo "  build         - Build the project"
	@echo "  test          - Run all tests"
	@echo "  test-verbose  - Run tests with verbose output"
	@echo "  test-short    - Run tests in short mode"
	@echo "  test-cover    - Run tests with coverage"
	@echo "  test-race     - Run tests with race detection"
	@echo "  test-integration - Run integration tests only"
	@echo "  test-benchmark - Run benchmark tests"
	@echo "  test-docker   - Run tests with Docker databases"
	@echo "  test-mysql    - Run MySQL tests only"
	@echo "  test-postgresql - Run PostgreSQL tests only"
	@echo "  test-sqlite   - Run SQLite tests only"
	@echo "  docker-up     - Start Docker databases"
	@echo "  docker-down   - Stop Docker databases"
	@echo "  fmt           - Format code"
	@echo "  lint          - Run linter"
	@echo "  vet           - Run go vet"
	@echo "  mod-tidy      - Tidy module dependencies"
	@echo "  mod-download  - Download module dependencies"
	@echo "  clean         - Clean build artifacts"

# Build targets
build:
	go build ./...

# Test targets
test:
	go test ./...

test-verbose:
	go test -v ./...

test-short:
	go test -short ./...

test-cover:
	go test -cover ./...
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

test-race:
	go test -race ./...

test-integration:
	go test -v -run Integration ./...

test-benchmark:
	go test -bench=. -benchmem ./test

test-docker:
	./scripts/test-docker.sh

test-mysql:
	go test -v ./test -run TestMySQL

test-postgresql:
	go test -v ./test -run TestPostgreSQL

test-sqlite:
	go test -v ./test -run TestSQLite

# Docker database management
docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

# Code quality targets
fmt:
	go fmt ./...

lint:
	golangci-lint run

vet:
	go vet ./...

# Module management
mod-tidy:
	go mod tidy

mod-download:
	go mod download

# Clean targets
clean:
	rm -rf bin/
	rm -f coverage.out coverage.html
	rm -f *.db
	rm -f test_*.db

# Development workflow
dev: fmt vet test

# CI workflow
ci: mod-download fmt vet test-race test-cover

# All checks
all: clean mod-tidy fmt vet test-race test-cover example