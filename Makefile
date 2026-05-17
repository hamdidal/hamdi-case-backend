.PHONY: help up down build rebuild restart ps test lint tidy backup logs health

help:
	@echo "Available commands:"
	@echo "  make up       - Start all services"
	@echo "  make down     - Stop all services"
	@echo "  make build    - Build Docker images"
	@echo "  make rebuild  - Build Docker images without cache"
	@echo "  make restart  - Restart all services"
	@echo "  make ps       - Show running containers"
	@echo "  make test     - Run test suite"
	@echo "  make lint     - Run go vet"
	@echo "  make tidy     - Run go mod tidy"
	@echo "  make backup   - Run database backup"
	@echo "  make logs     - Stream backend logs"
	@echo "  make health   - Check API health"

up:
	docker compose up -d

down:
	docker compose down

build:
	docker compose build

rebuild:
	docker compose build --no-cache

restart:
	docker compose restart

ps:
	docker compose ps

test:
	go test -v ./...

lint:
	go vet ./...

tidy:
	go mod tidy

backup:
	bash scripts/db_backup.sh

logs:
	docker compose logs -f dpp-backend

health:
	curl -s http://localhost:8080/health | python3 -m json.tool
