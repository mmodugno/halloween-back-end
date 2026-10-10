run: build
	@./bin/api

build:
	@go build -o bin/api ./cmd

test:
	@go test -v ./...
