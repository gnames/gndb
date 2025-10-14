# Development Tools

This directory contains development and testing tools for the gndb project.

These tools are **not** part of the production CLI and are intended for:
- Generating test data
- Development workflows
- One-off tasks during development

## Available Tools

### subset-sfga

Creates smaller SFGA test files from full data sources.

**Purpose**: Extract ~20k-40k representative records from large SFGA sources to create compact test files that preserve edge cases and hierarchy consistency.

**Usage**:
```bash
cd dev/subset-sfga
go run . --source-id 1 --output ../../testdata/0001-subset.sqlite --size 30000
```

See [subset-sfga/README.md](subset-sfga/README.md) for detailed documentation.

---

## Adding New Tools

To add a new development tool:

1. Create a new directory: `dev/my-tool/`
2. Add a `main.go` file with `package main`
3. Document usage in `dev/my-tool/README.md`
4. Update this README with a brief description

Keep tools simple - start with a single file, refactor only if complexity grows.
