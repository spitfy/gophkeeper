.PHONY: build test migrate

build-client:
    cd cmd/client && go build -o ../../bin/client

build-server:
    cd cmd/server && go build -o ../../bin/server

migrate-up:
    go run cmd/migrate/main.go up

test:
    go test ./...