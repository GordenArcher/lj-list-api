.PHONY: run build migrate-up migrate-down migrate-create

run:
	go run cmd/server/main.go

build:
	go build -o bin/lj-list-api cmd/server/main.go

migrate-up:
	migrate -path internal/database/migrations -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path internal/database/migrations -database "$(DATABASE_URL)" down 1

migrate-create:
	@read -p "Migration name: " name; \
	migrate create -ext sql -dir internal/database/migrations -seq $$name
