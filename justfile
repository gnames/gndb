# GNdb - Just task runner commands
# Install just: https://github.com/casey/just

# Default recipe - install gndb to ~/go/bin
default: install

# Run all tests (skip integration tests requiring database)
test:
    @echo "Cleaning up test database (gndb_test)..."
    @dropdb --if-exists gndb_test 2>/dev/null || true
    @createdb gndb_test 2>/dev/null || true
    go test -short -count=1 ./...

# Run all tests including integration tests (requires PostgreSQL)
test-all:
    go test -count=1 -p 1 ./...

# Run tests with coverage
test-coverage:
    go test -v -coverprofile=coverage.out ./...
    go tool cover -html=coverage.out -o coverage.html
    @echo "Coverage report generated: coverage.html"

# Run tests for a specific package
test-pkg pkg:
    go test -v ./{{pkg}}

# Run all tests with race detector
test-race:
    go test -count=1 -race ./...

# Build the gndb binary
build:
    @mkdir -p bin
    go build -o bin/gndb ./cmd/gndb
    @echo "✅ gndb built to bin/gndb"

# Install gndb to ~/go/bin
install:
    go install ./cmd/gndb
    @echo "✅ gndb installed to ~/go/bin/gndb"

# Clean build artifacts
clean:
    rm -rf bin coverage.out coverage.html
    go clean

# Format all Go code
fmt:
    go fmt ./...

# Run linter (requires golangci-lint)
lint:
    golangci-lint run

# Tidy dependencies
tidy:
    go mod tidy

# Verify project builds and all tests pass
verify: fmt tidy test build
    @echo "✅ Verification complete: code formatted, dependencies tidied, tests passing, build successful"
