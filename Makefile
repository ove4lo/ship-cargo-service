.PHONY: run test lint

run:
	go run ./cmd/api

test:
	go test ./... -v -race

lint:
	golangci-lint run ./...

migrate-up:
	migrate -path migrations -database "postgres://cargo:cargo_secret@localhost:5432/ship_cargo?sslmode=disable" up

migrate-down:
	migrate -path migrations -database "postgres://cargo:cargo_secret@localhost:5432/ship_cargo?sslmode=disable" down