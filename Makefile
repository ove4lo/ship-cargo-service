.PHONY: run test lint

run:
	go test ./... -v -race

test:
	go test ./... -v -race

lint:
	golangci-lint run ./...