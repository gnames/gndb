# GNdb - Just task runner commands
# Install just: https://github.com/casey/just

# Project variables
VERSION := `git describe --tags 2>/dev/null || echo "dev"`
VER := `git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.1"`
DATE := `date -u '+%Y-%m-%d_%H:%M:%S'`

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
    go test -count=1 -p 1 -timeout=10m ./...

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

# Build the gndb binary (development build with timestamp and git version)
build:
    @mkdir -p bin
    CGO_ENABLED=0 go build -ldflags "-w -s -X 'github.com/gnames/gndb/pkg.Build={{DATE}}' -X 'github.com/gnames/gndb/pkg.Version={{VERSION}}'" -o bin/gndb ./cmd/gndb
    @echo "✅ gndb built to bin/gndb (version: {{VERSION}}, build: {{DATE}})"

# Build release binary (uses version.go for Version, timestamp for Build)
build-release:
    @mkdir -p bin
    CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X 'github.com/gnames/gndb/pkg.Build={{DATE}}'" -o bin/gndb ./cmd/gndb
    @echo "✅ gndb release binary built to bin/gndb"

# Install gndb to ~/go/bin (development build with timestamp and git version)
install:
    CGO_ENABLED=0 go install -ldflags "-w -s -X 'github.com/gnames/gndb/pkg.Build={{DATE}}' -X 'github.com/gnames/gndb/pkg.Version={{VERSION}}'" ./cmd/gndb
    @echo "✅ gndb installed to ~/go/bin/gndb (version: {{VERSION}}, build: {{DATE}})"

# Build releases for multiple platforms
release:
    @echo "Building releases for Linux, Mac (Intel), Mac (ARM), Windows"
    @mkdir -p bin/releases
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-w -s -X 'github.com/gnames/gndb/pkg.Build={{DATE}}' -X 'github.com/gnames/gndb/pkg.Version={{VERSION}}'" -o bin/releases/gndb ./cmd/gndb
    tar zcvf bin/releases/gndb-{{VER}}-linux.tar.gz -C bin/releases gndb
    rm bin/releases/gndb
    CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags "-w -s -X 'github.com/gnames/gndb/pkg.Build={{DATE}}' -X 'github.com/gnames/gndb/pkg.Version={{VERSION}}'" -o bin/releases/gndb ./cmd/gndb
    tar zcvf bin/releases/gndb-{{VER}}-mac.tar.gz -C bin/releases gndb
    rm bin/releases/gndb
    CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags "-w -s -X 'github.com/gnames/gndb/pkg.Build={{DATE}}' -X 'github.com/gnames/gndb/pkg.Version={{VERSION}}'" -o bin/releases/gndb ./cmd/gndb
    tar zcvf bin/releases/gndb-{{VER}}-mac-arm64.tar.gz -C bin/releases gndb
    rm bin/releases/gndb
    CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags "-w -s -X 'github.com/gnames/gndb/pkg.Build={{DATE}}' -X 'github.com/gnames/gndb/pkg.Version={{VERSION}}'" -o bin/releases/gndb.exe ./cmd/gndb
    cd bin/releases && zip -9 gndb-{{VER}}-win-64.zip gndb.exe
    rm bin/releases/gndb.exe
    @echo "✅ Release binaries created in bin/releases/"

# Clean build artifacts
clean:
    rm -rf bin coverage.out coverage.html
    go clean
    @echo "✅ Cleaned build artifacts"

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
