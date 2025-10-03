# GNdb: GNverifier Database Lifecycle Management

## Project Overview

This project, `gndb`, is a command-line interface (CLI) tool written in Go for managing the lifecycle of a PostgreSQL database used by the `gnverifier` application. `gnverifier` is a tool for verifying and reconciling scientific names. `gndb` allows users to set up and maintain a local `gnverifier` instance with their own custom data sources, independent of the main `gnverifier` service.

The key functionalities of `gndb` include:

*   **Database Schema Management:** Creating and migrating the database schema.
*   **Data Population:** Populating the database with nomenclature data from user-provided sources.
*   **Database Optimization:** Applying performance-critical optimizations to enable fast name verification, vernacular name detection, and synonym resolution.

The project is designed to be used both for local installations and for the main `gnverifier` service, ensuring a consistent database lifecycle management process.

## Key Technologies

*   **Go:** The primary programming language.
*   **PostgreSQL:** The database system for storing `gnverifier` data.
*   **Cobra:** A library for creating powerful modern CLI applications in Go.
*   **Viper:** A complete configuration solution for Go applications.
*   **GORM:** The Go ORM library.
*   **pgx:** A PostgreSQL driver and toolkit for Go.

## Building and Running

The project uses a `justfile` for task automation. The following are the most common commands:

*   **Build the application:**
    ```bash
    just build
    ```
    This command compiles the project and creates a `gndb` binary in the root directory.

*   **Install the application:**
    ```bash
    just install
    ```
    This command installs the `gndb` binary to `~/go/bin`.

*   **Run tests:**
    ```bash
    just test
    ```
    This command runs the unit tests.

*   **Run all tests (including integration tests):**
    ```bash
    just test-all
    ```
    This command runs all tests, including integration tests that require a running PostgreSQL database.

*   **Format and lint the code:**
    ```bash
    just fmt
    just lint
    ```

*   **Verify the project:**
    ```bash
    just verify
    ```
    This command runs a sequence of formatting, tidying, testing, and building to ensure the project is in a good state.

## Development Conventions

*   **Code Style:** The project follows the standard Go formatting guidelines, enforced by `go fmt`.
*   **Linting:** `golangci-lint` is used for linting the codebase.
*   **Dependency Management:** Go modules are used for dependency management. The `go mod tidy` command is used to keep the `go.mod` and `go.sum` files up to date.
*   **Configuration:** Configuration is managed through a `gndb.yaml` file, environment variables (with the `GNDB_` prefix), and command-line flags. The order of precedence is: flags, environment variables, config file, and then defaults.
*   **Project Structure:** The project follows a standard Go project layout:
    *   `cmd`: Contains the main application entry point.
    *   `pkg`: Contains the public API of the application.
    *   `internal`: Contains the internal implementation details.
    *   `specs`: Contains the feature specifications.
    *   `migrations`: Contains the database migration files.
    *   `testdata`: Contains test data.
    *   `tests`: Contains the integration tests.

## Implementation Details

As part of the implementation planning workflow, the following artifacts have been generated in the `specs/001-gnverifier-db-lifecycle` directory:

*   `research.md`: Contains research on validation rules, progress indicators, logging levels, and output formats.
*   `data-model.md`: Defines the data model for the GNverifier database.
*   `contracts/`: Contains the Go interface contracts for the different components of the system.
*   `quickstart.md`: Provides a quickstart guide for using the `gndb` CLI.

### Policy

*   Always run `go mod tidy` before finalizing a coding task to avoid lint warnings.