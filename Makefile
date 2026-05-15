.PHONY: up down test backup logs

up:
	docker-compose up -d

down:
	docker-compose down

test:
	go test -v ./...

backup:
	bash scripts/db_backup.sh

logs:
	docker-compose logs -f dpp-backend
