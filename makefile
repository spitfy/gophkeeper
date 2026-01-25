COVERAGE_FILE = coverage.out

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

lint:
	golangci-lint run ./...

fmt:
	golangci-lint run --fix ./...

test-coverage:
	go test -coverprofile=$(COVERAGE_FILE) ./internal/...
	go tool cover -func=$(COVERAGE_FILE)

# Детальный HTML отчет о покрытии
test-coverage-html:
	go test -coverprofile=$(COVERAGE_FILE) ./internal/...
	go tool cover -html=$(COVERAGE_FILE)

# Показать только общий процент покрытия
test-coverage-total:
	@go test -coverprofile=$(COVERAGE_FILE) ./... > /dev/null 2>&1
	@go tool cover -func=$(COVERAGE_FILE) | grep total | awk '{print $$3}'