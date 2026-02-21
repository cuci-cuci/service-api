.PHONY: build run migrate-up migrate-down dev docker-up docker-down test lint

# Build the Go binary
build:
	go build -o bin/server ./cmd/server

# Run the server locally
run: build
	./bin/server

# Run goose migrations up
migrate-up:
	goose -dir migrations postgres "$(DATABASE_URL)" up

# Run goose migrations down
migrate-down:
	goose -dir migrations postgres "$(DATABASE_URL)" down

# Run with air for hot reload (install air: go install github.com/air-verse/air@latest)
dev:
	air -c .air.toml || go run ./cmd/server

# Start all services with Docker Compose
docker-up:
	docker compose up --build -d

# Stop all Docker Compose services
docker-down:
	docker compose down

# Run tests
test:
	go test ./... -v

# Run linter
lint:
	golangci-lint run ./...

# Tidy modules
tidy:
	go mod tidy
