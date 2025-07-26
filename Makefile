# RediORM Makefile

# Version from git tags with fallback
VERSION := $(shell git describe --tags --exact-match 2>/dev/null || git describe --tags --always 2>/dev/null || echo "dev")

# Build flags with version injection
LDFLAGS := -X main.version=$(VERSION)

.PHONY: help test test-verbose test-short test-cover test-race test-integration test-benchmark test-sqlite test-mysql test-postgresql test-orm test-docker docker-up docker-down docker-wait clean fmt lint vet deps build install build-orm build-mcp install-orm install-mcp cli-build cli-install release-build

# Default target
help:
	@echo "Available targets:"
	@echo "  test          - Run all tests"
	@echo "  test-verbose  - Run tests with verbose output"
	@echo "  test-short    - Run tests in short mode"
	@echo "  test-cover    - Run tests with coverage"
	@echo "  test-sqlite   - Run SQLite tests only"
	@echo "  test-mysql    - Run MySQL tests only"
	@echo "  test-postgresql - Run PostgreSQL tests only"
	@echo "  test-docker   - Run tests with Docker databases"
	@echo "  docker-up     - Start test databases"
	@echo "  docker-down   - Stop test databases"
	@echo "  docker-wait   - Wait for databases to be ready"
	@echo "  fmt           - Format code"
	@echo "  lint          - Run linter"
	@echo "  vet           - Run go vet"
	@echo "  deps          - Download and tidy module dependencies"
	@echo "  clean         - Clean build artifacts"
	@echo "  build-orm     - Build the redi-orm CLI tool"
	@echo "  build-mcp     - Build the redi-mcp CLI tool"
	@echo "  install-orm   - Install the redi-orm CLI tool globally"
	@echo "  install-mcp   - Install the redi-mcp CLI tool globally"
	@echo "  cli-build     - Build both CLI tools (deprecated, use build-orm/build-mcp)"
	@echo "  cli-install   - Install both CLI tools (deprecated, use install-orm/install-mcp)"
	@echo "  release-build - Build CLI with version injection for release"
	@echo "  version       - Show current version"

# Test targets
test:
	go test -p 1 -count=1 ./...

test-verbose:
	go test -p 1 -count=1 -v ./...

test-short:
	go test -p 1 -count=1 -short ./...

test-cover:
	go test -p 1 -count=1 -cover -short ./...

test-sqlite:
	go test -p 1 -count=1 -v ./drivers/sqlite

test-mysql:
	go test -p 1 -count=1 -v ./drivers/mysql

test-postgresql:
	go test -p 1 -count=1 -v ./drivers/postgresql

test-mongodb:
	go test -p 1 -count=1 -v ./drivers/mongodb

test-docker: docker-up docker-wait
	@echo "Running tests with Docker databases..."
	go test -p 1 -v ./drivers/mysql ./drivers/postgresql ./drivers/mongodb || true
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
	rm -f ./redi-mcp

# Individual CLI build targets
build-orm:
	go build -o redi-orm ./cmd/redi-orm

build-mcp:
	go build -o redi-mcp ./cmd/redi-mcp

# Individual CLI install targets
install-orm:
	go install ./cmd/redi-orm

install-mcp:
	go install ./cmd/redi-mcp

# Legacy CLI targets (for backward compatibility)
cli-build: build-orm build-mcp

cli-install: install-orm install-mcp

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
	docker compose up -d
	@echo "Docker databases started"

docker-down:
	docker compose down
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
	@until docker exec redi-orm-mongodb mongosh --eval "db.adminCommand('ping').ok" --quiet 2>/dev/null; do \
		echo "Waiting for MongoDB..."; \
		sleep 2; \
	done
	@echo "All databases are ready"

# All checks
all: clean deps fmt vet test-race test-cover