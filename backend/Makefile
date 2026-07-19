APP_NAME=Gmail-Service
MAIN=cmd/main.go

## Start RabbitMQ only (connect to it without running the email service)
up-rabbitmq:
	docker-compose up -d rabbitmq

## Start RabbitMQ + Postgres (full infra, no app)
up-infra:
	docker-compose up -d rabbitmq postgres

## Run app in dev mode (start infra first: make up-rabbitmq or make up-infra)
## Run from backend dir so migrations path is correct
dev:
	go run $(MAIN)

## Run migrations only (requires DB_* in env or .env sourced; run from backend dir)
## Install CLI: go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
migrate-up:
	migrate -path migrations -database "postgres://$${DB_USER}:$${DB_PASSWORD}@$${DB_HOST}:$${DB_PORT}/$${DB_NAME}?sslmode=disable" up

migrate-down:
	migrate -path migrations -database "postgres://$${DB_USER}:$${DB_PASSWORD}@$${DB_HOST}:$${DB_PORT}/$${DB_NAME}?sslmode=disable" down

## Build binary
build:
	go build -o bin/$(APP_NAME) $(MAIN)

## Run built binary
start:
	./bin/$(APP_NAME)

## Run tests
test:
	go test ./... -v

## Run tests with coverage
test-cover:
	go test ./... -cover

## Format code
fmt:
	go fmt ./...

## Lint (requires golangci-lint)
lint:
	golangci-lint run

## Clean build files
clean:
	rm -rf bin

## Docker: build backend image
docker-build:
	docker compose build backend

## Docker: run full stack (postgres + rabbitmq + backend)
docker-up:
	docker compose up

## Docker: run full stack and follow logs
docker-up-logs:
	docker compose up

## Docker: stop all services
docker-down:
	docker compose down
