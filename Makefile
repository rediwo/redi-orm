# RediORM Makefile

# Version from git tags with fallback
VERSION := $(shell git describe --tags --exact-match 2>/dev/null || git describe --tags --always 2>/dev/null || echo "dev")

# Build flags with version injection
LDFLAGS := -X main.version=$(VERSION)

.PHONY: help test test-verbose test-short test-cover test-race test-integration test-benchmark test-sqlite test-mysql test-postgresql test-orm test-docker docker-up docker-down docker-wait clean fmt lint vet deps build install cli-build cli-install release-build

# Default target
help:
	@echo "Available targets:"
	@echo "  test          - Run all tests"
	@echo "  test-verbose  - Run tests with verbose output"
	@echo "  test-short    - Run tests in short mode"
	@echo "  test-cover    - Run tests with coverage"
	@echo "  test-race     - Run tests with race detection"
	@echo "  test-integration - Run integration tests only"
	@echo "  test-benchmark - Run benchmark tests"
	@echo "  test-sqlite   - Run SQLite tests only"
	@echo "  test-mysql    - Run MySQL tests only"
	@echo "  test-postgresql - Run PostgreSQL tests only"
	@echo "  test-orm      - Run ORM module tests only"
	@echo "  test-docker   - Run tests with Docker databases"
	@echo "  docker-up     - Start test databases"
	@echo "  docker-down   - Stop test databases"
	@echo "  docker-wait   - Wait for databases to be ready"
	@echo "  fmt           - Format code"
	@echo "  lint          - Run linter"
	@echo "  vet           - Run go vet"
	@echo "  deps          - Download and tidy module dependencies"
	@echo "  clean         - Clean build artifacts"
	@echo "  cli-build     - Build the CLI tool"
	@echo "  cli-install   - Install the CLI tool globally"
	@echo "  release-build - Build CLI with version injection for release"
	@echo "  version       - Show current version"

# Test targets
test:
	go test ./...

test-verbose:
	go test -v ./...

test-short:
	go test -short ./...

test-cover:
	go test -cover -short ./...

test-race:
	go test -race -short ./...

test-integration:
	go test -v -run Integration ./test

test-benchmark:
	go test -bench=. -benchmem ./test

test-sqlite:
	go test -v ./drivers/sqlite

test-mysql:
	go test -v ./drivers/mysql

test-postgresql:
	go test -v ./drivers/postgresql

test-orm:
	go test -v ./modules/orm/tests

test-docker: docker-up docker-wait
	@echo "Running tests with Docker databases..."
	go test -v ./drivers/mysql ./drivers/postgresql || true
	$(MAKE) docker-down

# Code quality targets
fmt:
	gofmt -w -r 'interface{} -> any' .
	go fmt ./...

lint:
	golangci-lint run

vet:
	go vet ./...

# Module management
deps:
	go mod download
	go mod tidy

# Clean targets
clean:
	rm -f *.db
	rm -f test_*.db
	rm -f ./redi-orm

# CLI targets
cli-build:
	go build -o redi-orm ./cmd/redi-orm

cli-install:
	go install ./cmd/redi-orm

# Release build with version injection
release-build:
	go build -o redi-orm -ldflags "$(LDFLAGS)" ./cmd/redi-orm

# Version information
version:
	@echo "Version: $(VERSION)"

# Development workflow
dev: fmt vet test

# CI workflow
ci: deps fmt vet test-race test-cover

# Docker targets
docker-up:
	docker-compose up -d
	@echo "Docker databases started"

docker-down:
	docker-compose down
	@echo "Docker databases stopped"

docker-wait:
	@echo "Waiting for databases to be ready..."
	@until docker exec redi-orm-mysql mysqladmin ping -h localhost --silent; do \
		echo "Waiting for MySQL..."; \
		sleep 2; \
	done
	@until docker exec redi-orm-postgresql pg_isready -U testuser -d testdb; do \
		echo "Waiting for PostgreSQL..."; \
		sleep 2; \
	done
	@echo "All databases are ready"

# All checks
all: clean deps fmt vet test-race test-cover