BIN := bin/server

.PHONY: build run vet migrate-up migrate-down docker-up docker-down

build:
	go build -o $(BIN) ./cmd/server

run: build
	./$(BIN)

vet:
	go vet ./...

migrate-up:
	goose -dir migrations postgres "$$DATABASE_URL" up

migrate-down:
	goose -dir migrations postgres "$$DATABASE_URL" down

docker-up:
	docker compose up -d

docker-down:
	docker compose down
