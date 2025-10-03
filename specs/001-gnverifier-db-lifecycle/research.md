# Research for GNverifier Database Lifecycle Management

## 1. Validation Rules for Lifecycle Phases

**Decision:**

*   **`create` phase:**
    *   Before execution: Check if the database is empty. If not, prompt the user for confirmation to drop all tables.
    *   After execution: Verify that all required tables and indexes have been created.
*   **`migrate` phase:**
    *   Before execution: Check the current schema version.
    *   After execution: Verify that the schema version has been updated to the latest version.
*   **`populate` phase:**
    *   Before execution: Validate the format of the input data sources (SFGA).
    *   After execution: Verify that the data has been imported correctly by checking the row counts of the tables.
*   **`restructure` phase:**
    *   Before execution: Check if the database is populated.
    *   After execution: Verify that the performance optimizations have been applied (e.g., by checking for the existence of materialized views).

**Rationale:**

These validation rules will ensure the integrity of the database at each stage of the lifecycle and prevent accidental data loss.

**Alternatives considered:**

*   No validation: This would be risky and could lead to data corruption.

## 2. Progress Indicators, Logging Level, and Output Format

**Decision:**

*   **Progress Indicators:** For long-running operations like `populate` and `restructure`, a progress bar will be displayed in the console.
*   **Logging Level:** The default logging level will be `info`. The user can change the logging level using a command-line flag (`--log-level`) or an environment variable (`GNDB_LOGGING_LEVEL`).
*   **Output Format:** The default output format will be `text`. The user can change the output format to `json` using a command-line flag (`--format`) or an environment variable (`GNDB_LOGGING_FORMAT`).

**Rationale:**

This provides a good balance between providing useful feedback to the user and not being too verbose by default. The user can customize the logging and output format to their needs.

**Alternatives considered:**

*   Only logging, no progress bar: This would make it difficult for the user to know the progress of long-running operations.
*   Only JSON output: This would be less user-friendly for interactive use.