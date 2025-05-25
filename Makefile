.PHONY: build run clean test deps lint

# Build the application
build:
	go build -o ulb-bot cmd/ulb-bot/main.go

# Run the application
run:
	go run cmd/ulb-bot/main.go

# Clean build artifacts
clean:
	rm -f ulb-bot

# Run tests
test:
	go test -v ./...

# Download dependencies
deps:
	go mod download
	go mod tidy

# Run linter (requires golangci-lint)
lint:
	golangci-lint run

# Development run with live reload (requires air)
dev:
	air

# Build for production
build-prod:
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o ulb-bot cmd/ulb-bot/main.go