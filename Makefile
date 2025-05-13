.PHONY: all build test clean proto

# Default target
all: build

# Build the binary
build:
	@mkdir -p target
	go build -o target/schema2proto ./cmd/schema2proto

# Run tests
test:
	go test -v ./...

# Clean build artifacts
clean:
	rm -rf target/

# Generate proto files from schema
proto: build
	./target/schema2proto -schema schema.json -output proto/schema.proto

# Install dependencies
deps:
	go mod download
	go mod tidy

# Run linter
lint:
	golangci-lint run

# Help target
help:
	@echo "Available targets:"
	@echo "  all     - Default target, builds the binary"
	@echo "  build   - Build the binary into target/"
	@echo "  test    - Run tests"
	@echo "  clean   - Remove build artifacts"
	@echo "  proto   - Generate proto files from schema"
	@echo "  deps    - Download and tidy dependencies"
	@echo "  lint    - Run linter" 