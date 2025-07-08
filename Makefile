# Binary name
BINARY_NAME=NoveAPI

# Entry point
MAIN_PACKAGE=./cmd

# Default target
.DEFAULT_GOAL := help

# Standard Build
build:
	go build -o $(BINARY_NAME).exe $(MAIN_PACKAGE)

# Build with race detection
build-race:
	go build -race -o $(BINARY_NAME) $(MAIN_PACKAGE)

# Build with debugging flags
build-dev:
	go build -gcflags "all=-N -l" -o $(BINARY_NAME) $(MAIN_PACKAGE)

# Cross-compile for Linux (optional)
ifeq ($(OS),Linux)
		SET_GOOS := set GOOS=linux&& set GOARCH=amd64&&
else
		SET_GOOS := GOOS=linux GOARCH=amd64
endif

build-linux:
	$(SET_GOOS) go build -o $(BINARY_NAME)_linux $(MAIN_PACKAGE)

# Cross-compile for Windows (optional)
ifeq ($(OS),Windows_NT)
    SET_GOOS := set GOOS=windows&& set GOARCH=amd64&&
else
    SET_GOOS := GOOS=windows GOARCH=amd64
endif

build-win:
	$(SET_GOOS) go build -o ${BINARY_NAME}.exe ${MAIN_PACKAGE}

# Run the appa
run:
	go run $(MAIN_PACKAGE)

# Run tests
test:
	go test ./...

# Format code
format:
	go fmt ./...

# Static analysis
lint:
	go vet ./...

# Clean build artifacts
clean:
	go clean
	rm -f $(BINARY_NAME) $(BINARY_NAME).exe $(BINARY_NAME)_linux

release: clean build
	mkdir -p dist
	cp $(BINARY_NAME) dist/
	cp assets/noverna.toml dist/
	cd dist && zip $(BINARY_NAME)_release.zip $(BINARY_NAME) noverna.toml

release-linux: build-linux
	cd dist && zip $(BINARY_NAME)_linux.zip $(BINARY_NAME)_linux

release-win: build-win
	cd dist && zip $(BINARY_NAME)_win.zip $(BINARY_NAME).exe



# List available commands
help:
	@echo ""
	@echo "Available commands:"
	@echo "  make build         - Compiles the project"
	@echo "  make build-dev     - Debug build (no inlining, no optimization)"
	@echo "  make build-race    - Build with race detection"
	@echo "  make build-linux   - Cross-compile for Linux"
	@echo "  make build-win     - Cross-compile for Windows"
	@echo "  make run           - Runs the project"
	@echo "  make test          - Runs all tests"
	@echo "  make format        - Formats the code"
	@echo "  make lint          - Runs go vet"
	@echo "  make clean         - Removes binaries"
	@echo "  make release       - Creates a release ZIP with config"



# Mark phony targets
.PHONY: build build-dev build-race build-linux build-win run test format clean lint help
