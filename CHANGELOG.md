# Changelog

## Unreleased

## [v0.0.1] - 2025-10-15 Wed

- Add: Parse version and date from SFGA filenames.
- Add: JobsNumber config and move logs to stderr.
- Add: Improve SFGA file selection and config validation.
- Add: Implement flexible outlink ID generation.
- Add: Handle flat classification in SFGA sources.
- Add: subset-sfga tool for test data generation.
- Add: Finalize architecture and quickstart documentation.
- Add: Implement data population workflow with 5-phase process.
- Add: Implement data source metadata update.
- Add: Implement vernacular names processing.
- Add: Implement name indices processing with integration tests.
- Add: Concurrent hierarchy building for taxonomy trees.
- Add: Name import integration tests.
- Add: SFGA fetching and refactor sources config.
- Add: Source filtering and cache management.
- Add: migrate command with integration tests.
- Add: populate stub and enhance create command safety.
- Add: Structured logging with slog.
- Add: sources.yaml configuration with comprehensive tests.
- Add: Environment variable overrides for all config fields.
- Add: CLI root and create command.
- Add: DatabaseOperator with pgxpool connection pooling.
- Add: GORM AutoMigrate for schema creation.
- Add: Configuration loader with viper.
- Add: migrate, populate, and optimize commands.
- Add: .envrc.example for direnv integration.
- Add: justfile for common development tasks.
- Add: Schema models with DDL generation.
- Fix: Database test configuration to use config.Load() system.
- Fix: Environment variables now work with empty config file.
- Refactor: Shorten internal package paths and rename pkg/database.
- Refactor: Standardize import aliases.
- Refactor: Decompose DatabaseOperator into lifecycle components.
- Refactor: Simplify schema and add collation "C" support.
- Refactor: Consolidate templates in pkg/templates for easy access.
- Refactor: Rename template files from .yaml.example to .yaml.
- Refactor: Align schema with gnidump for gnverifier compatibility.
- Refactor: Change default database name from 'gndb' to 'gnames'.

## Footnotes

This document follows [changelog guidelines]

[v0.0.1]: https://github.com/gnames/gndb/tree/v0.0.1
[changelog guidelines]: https://github.com/olivierlacan/keep-a-changelog
