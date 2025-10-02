# GNdb - Just task runner commands
# Install just: https://github.com/casey/just

# Default recipe - show available commands
default:
    @just --list

# Run all tests with verbose output
test:
    go test -v ./...

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
    go test -v -race ./...

# Build the gndb binary
build:
    go build -o gndb ./cmd/gndb

# Clean build artifacts
clean:
    rm -f gndb coverage.out coverage.html
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
