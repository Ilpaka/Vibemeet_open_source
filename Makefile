.PHONY: help build run test tidy lint clean up down logs restart ps \
        migrate-up migrate-down migrate-redo migrate-status migrate-version

BINARY := bin/server
PKG    := ./...

help:
	@echo "Vibemeet - available targets:"
	@echo ""
	@echo "Build & run:"
	@echo "  build              Build the server binary into $(BINARY)"
	@echo "  run                Run the server locally (Postgres + Redis + LiveKit required)"
	@echo "  test               Run unit tests"
	@echo "  tidy               Sync Go module dependencies"
	@echo "  lint               Run go vet"
	@echo "  clean              Remove build artifacts"
	@echo ""
	@echo "Database migrations (goose, embedded in the binary):"
	@echo "  migrate-up         Apply all pending migrations"
	@echo "  migrate-down       Roll back the most recent migration"
	@echo "  migrate-redo       Roll back the latest migration and re-apply it"
	@echo "  migrate-status     Show applied/pending state of every migration"
	@echo "  migrate-version    Print the current schema version"
	@echo ""
	@echo "Docker Compose (dev stack):"
	@echo "  up                 Start the full stack"
	@echo "  down               Stop the stack"
	@echo "  logs               Tail logs from all services"
	@echo "  restart            Recreate containers with rebuild"
	@echo "  ps                 Show container status"

build:
	@mkdir -p bin
	go build -o $(BINARY) ./cmd/server

run:
	go run ./cmd/server

test:
	go test $(PKG)

tidy:
	go mod tidy

lint:
	go vet $(PKG)

clean:
	rm -rf bin/
	go clean

# --- Migrations -------------------------------------------------------------
# Reads DATABASE_DSN (or the same env the server uses) for the connection.

migrate-up:
	go run ./cmd/server migrate up

migrate-down:
	go run ./cmd/server migrate down

migrate-redo:
	go run ./cmd/server migrate redo

migrate-status:
	go run ./cmd/server migrate status

migrate-version:
	go run ./cmd/server migrate version

# --- Docker Compose ---------------------------------------------------------

up:
	docker compose --env-file .env up -d --build

down:
	docker compose --env-file .env down

logs:
	docker compose --env-file .env logs -f

restart:
	docker compose --env-file .env up -d --build --force-recreate

ps:
	docker compose --env-file .env ps
