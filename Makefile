.PHONY: help build run test tidy lint clean up down logs restart ps

BINARY := bin/server
PKG    := ./...

help:
	@echo "Vibemeet — available targets:"
	@echo "  build     Build the server binary into $(BINARY)"
	@echo "  run       Run the server locally (requires Postgres + Redis + LiveKit)"
	@echo "  test      Run unit tests"
	@echo "  tidy      Sync Go module dependencies"
	@echo "  lint      Run go vet"
	@echo "  clean     Remove build artifacts"
	@echo "  up        Start the full stack (docker compose, dev)"
	@echo "  down      Stop the dev stack"
	@echo "  logs      Tail logs from all services"
	@echo "  restart   Recreate containers with rebuild"
	@echo "  ps        Show container status"

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
