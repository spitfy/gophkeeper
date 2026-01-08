# Client commands
client-build:
	go build -o bin/client ./cmd/client/main.go

client-run: client-build
	./bin/client

client-test:
	go test ./internal/app/client/...

client-lint:
	golangci-lint run ./internal/app/client/... ./cmd/client/...

client-clean:
	rm -rf bin/client
	rm -rf ~/.gophkeeper

# Development
dev-client:
	go run cmd/client/main.go

# Cross-compilation
client-linux:
	GOOS=linux GOARCH=amd64 go build -o bin/client-linux ./cmd/client/main.go

client-darwin:
	GOOS=darwin GOARCH=arm64 go build -o bin/client-macos ./cmd/client/main.go

client-windows:
	GOOS=windows GOARCH=amd64 go build -o bin/client-windows.exe ./cmd/client/main.go