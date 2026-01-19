.PHONY: all build build-all run test coverage fmt vet deps clean install config dev-setup docker-build docker-run docker-stop docker-clean help lint

# Build variables
BINARY_NAME=arbiter
VERSION?=dev
LDFLAGS=-ldflags "-X main.version=$(VERSION)"

all: build

# Build the application
build:
	@echo "Building $(BINARY_NAME)..."
	go build $(LDFLAGS) -o $(BINARY_NAME) ./cmd/arbiter

# Build for multiple platforms
build-all:
	@echo "Building for multiple platforms..."
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)-linux-amd64 ./cmd/arbiter
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)-darwin-amd64 ./cmd/arbiter
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BINARY_NAME)-darwin-arm64 ./cmd/arbiter
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)-windows-amd64.exe ./cmd/arbiter

# Run the application
run: build
	./$(BINARY_NAME) -config config.yaml

# Run tests
test:
	go test -v ./...

# Run tests with coverage
coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Format code
fmt:
	go fmt ./...

# Run go vet
vet:
	go vet ./...

# Install dependencies
deps:
	go mod download
	go mod tidy

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_NAME)-*
	rm -f coverage.out coverage.html
	rm -f *.db

# Run linters
lint: fmt vet

# Install the binary to $GOPATH/bin
install: build
	@echo "Installing $(BINARY_NAME) to $(GOPATH)/bin..."
	cp $(BINARY_NAME) $(GOPATH)/bin/

# Create example config if it doesn't exist
config:
	@if [ ! -f config.yaml ]; then \
		echo "Creating config.yaml from example..."; \
		cp config.yaml.example config.yaml; \
	else \
		echo "config.yaml already exists"; \
	fi

# Development setup
dev-setup: deps config
	@echo "Development environment setup complete"
	@echo "Run 'make run' to start the server"

# Docker build (for future use)
docker-build:
	docker build -t $(BINARY_NAME):$(VERSION) .

# Run application in Docker using docker-compose
docker-run:
	@echo "Starting arbiter in Docker..."
	@docker-compose up -d

# Stop Docker containers
docker-stop:
	@echo "Stopping Docker containers..."
	@docker-compose down

# Clean Docker resources
docker-clean: docker-stop
	@echo "Cleaning Docker resources..."
	@docker-compose down -v
	@docker rmi $(BINARY_NAME):$(VERSION) || true

help:
	@echo "Arbiter - Makefile Commands"
	@echo ""
	@echo "Development:"
	@echo "  make build        - Build the application"
	@echo "  make build-all    - Build for multiple platforms"
	@echo "  make run          - Build and run the application"
	@echo "  make test         - Run tests"
	@echo "  make coverage     - Run tests with coverage report"
	@echo "  make fmt          - Format code"
	@echo "  make vet          - Run go vet"
	@echo "  make lint         - Run linters"
	@echo "  make deps         - Download and tidy dependencies"
	@echo "  make clean        - Clean build artifacts"
	@echo "  make install      - Install binary to GOPATH/bin"
	@echo "  make config       - Create config.yaml from example"
	@echo "  make dev-setup    - Set up development environment"
	@echo ""
	@echo "Docker:"
	@echo "  make docker-build - Build Docker image"
	@echo "  make docker-run   - Run application in Docker"
	@echo "  make docker-stop  - Stop Docker containers"
	@echo "  make docker-clean - Clean Docker resources"
	@echo ""
	@echo "  make help         - Show this help message"
