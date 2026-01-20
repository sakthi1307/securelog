run:
	go run ./cmd/api

test:
	go test ./...

up:
	docker compose up -d

down:
	docker compose down

migrate:
	migrate -database $$DATABASE_URL -path migrations up
