.PHONY: run build test vet tidy docker-up docker-down

# Run in mock mode (no external keys required)
run:
	AI_MODE=mock BILLING_MODE=mock go run ./cmd/server

build:
	go build -o bin/server ./cmd/server

test:
	go test ./...

vet:
	go vet ./...

tidy:
	go mod tidy

# Start Postgres + app via Docker
docker-up:
	docker compose up --build

docker-down:
	docker compose down
