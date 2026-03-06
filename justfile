# gndb project justfile

# Variables
app := "gndb"
org := "github.com/gnames/"
build_dir := "bin/"
release_dir := "/tmp/"
test_opts := "-count=1 -p 1 -shuffle=on -coverprofile=coverage.txt -covermode=atomic"

no_c := "CGO_ENABLED=0"
x86 := "GOARCH=amd64"
arm := "GOARCH=arm64"
linux := "GOOS=linux"
mac := "GOOS=darwin"
win := "GOOS=windows"

# Colors
green := `tput -Txterm setaf 2`
yellow := `tput -Txterm setaf 3`
white := `tput -Txterm setaf 7`
cyan := `tput -Txterm setaf 6`
reset := `tput -Txterm sgr0`

# Get version from git
version_full := `git describe --tags 2>/dev/null || echo "dev"`
version := `git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.1"`
date := `date -u '+%Y-%m-%d_%H:%M:%S%Z'`

# LD flags with version and build date
flags_ld := "-trimpath -ldflags '-X " + org + app + \
    "/pkg/gndb.Build=" + date + " -X " + org + app + \
    "/pkg/gndb.Version=" + version_full + "'"
flags_rel := "-trimpath -ldflags '-s -w -X " + org + app + \
    "/pkg/gndb.Build=" + date + "'"

# Default recipe
default: install

# Show this help
help:
    @echo ''
    @echo 'Usage:'
    @echo '  {{yellow}}just{{reset}} {{green}}<target>{{reset}}'
    @echo ''
    @echo 'Targets:'
    @just --list --unsorted

# Display current version
version:
    @echo {{version_full}}

# Clean up and sync dependencies
tidy:
    @go mod tidy
    @go mod verify

# Install tools
tools: tidy
    @go install tool
    @echo "✅ tools of the project are installed"

# Build binary
build:
    @mkdir -p {{build_dir}}
    {{no_c}} go build -o {{build_dir}}{{app}} {{flags_ld}}
    @echo "✅ {{app}} built to {{build_dir}}{{app}}"

# Build binary without debug info and with hardcoded version
buildrel:
    @mkdir -p {{build_dir}}
    {{no_c}} go build -o {{build_dir}}{{app}} {{flags_rel}}
    @echo "✅ {{app}} release binary built to {{build_dir}}{{app}}"

# Build and install binary
install:
    {{no_c}} go install {{flags_ld}}
    @echo "✅ {{app}} installed to ~/go/bin/{{app}}"

# Build and package binaries for a release
release: buildrel
    @echo "Building releases for Linux, Mac, Windows (Intel and Arm)"
    @mkdir -p {{release_dir}}

    {{no_c}} {{linux}} {{x86}} go build {{flags_rel}} -o {{release_dir}}{{app}}
    tar zcvf {{release_dir}}{{app}}-{{version}}-linux-amd64.tar.gz {{release_dir}}{{app}}
    rm {{release_dir}}{{app}}

    {{no_c}} {{linux}} {{arm}} go build {{flags_rel}} -o {{release_dir}}{{app}}
    tar zcvf {{release_dir}}{{app}}-{{version}}-linux-arm64.tar.gz {{release_dir}}{{app}}
    rm {{release_dir}}{{app}}

    {{no_c}} {{mac}} {{x86}} go build {{flags_rel}} -o {{release_dir}}{{app}}
    tar zcvf {{release_dir}}{{app}}-{{version}}-mac-amd64.tar.gz {{release_dir}}{{app}}
    rm {{release_dir}}{{app}}

    {{no_c}} {{mac}} {{arm}} go build {{flags_rel}} -o {{release_dir}}{{app}}
    tar zcvf {{release_dir}}{{app}}-{{version}}-mac-arm64.tar.gz {{release_dir}}{{app}}
    rm {{release_dir}}{{app}}

    {{no_c}} {{win}} {{x86}} go build {{flags_rel}} -o {{release_dir}}{{app}}.exe
    cd {{release_dir}} && zip -9 {{app}}-{{version}}-win-amd64.zip {{app}}.exe
    rm {{release_dir}}{{app}}.exe

    {{no_c}} {{win}} {{arm}} go build {{flags_rel}} -o {{release_dir}}{{app}}.exe
    cd {{release_dir}} && zip -9 {{app}}-{{version}}-win-arm64.zip {{app}}.exe
    rm {{release_dir}}{{app}}.exe

    @echo "✅ Release binaries created in {{release_dir}}"

# Clean all the files and binaries generated
clean:
    @rm -rf ./{{build_dir}}

# Lint the code
lint:
    golangci-lint run

# Run unit tests (skip integration tests)
test:
    go test -short {{test_opts}} ./...

# Run all tests including integration tests (requires PostgreSQL)
test-all:
    go test {{test_opts}} ./...

# Run tests for a specific package
test-pkg pkg:
    go test -v ./{{pkg}}

# Run tests with race detector
test-race:
    go test -count=1 -race ./...

# Run tests and export coverage
coverage:
    @go test -p 1 -cover -covermode=count -coverprofile=profile.cov ./...
    @go tool cover -func profile.cov
