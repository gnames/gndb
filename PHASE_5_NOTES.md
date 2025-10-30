# Phase 5 Implementation Notes

## Current Status
- **Phases 1-4 COMPLETE** ✅
- **Next: Phase 5 - CLI Command & Integration**
- All code compiles, tests pass, lint clean
- Branch: `2-create`

## What's Been Done

### Phase 4 Completion (just finished)
- Copied and adapted 3 files from spec-kit:
  - `internal/iopopulate/vernaculars.go` (357 lines)
  - `internal/iopopulate/hierarchy.go` (390 lines)
  - `internal/iopopulate/indices.go` (727 lines)
- Fixed all references and wired everything up in `populator.go`
- Tests passing, lint clean

### Core Implementation Complete
All internal populate logic is now implemented:
- Source configuration (pkg/populate/sources.go)
- SFGA file handling (internal/iopopulate/sfga.go)
- Metadata import (internal/iopopulate/metadata.go)
- Name-string import (internal/iopopulate/names.go)
- Vernacular names (internal/iopopulate/vernaculars.go)
- Hierarchy building (internal/iopopulate/hierarchy.go)
- Name indices (internal/iopopulate/indices.go)
- Error handling (internal/iopopulate/errors.go)
- Main orchestrator (internal/iopopulate/populator.go)

## Phase 5 Tasks

### 1. Implement cmd/populate.go
Create new file with:
- Cobra command definition
- Flags (follow patterns from cmd/create.go and cmd/migrate.go):
  ```go
  --source-ids, -s      ([]int) - Filter specific data sources
  --release-version, -r (string) - SFGA release version
  --release-date, -d    (string) - SFGA release date
  --flat-classification (bool) - Use flat classification
  ```
- Command implementation:
  1. Create database operator (db.NewOperator)
  2. Create populator (iopopulate.NewPopulator)
  3. Call populator.Populate(ctx, cfg)
  4. Handle errors with gn.Error pattern
  5. Display progress with gn.Info

### 2. Wire into cmd/root.go
- Add populate command to root (like create/migrate)
- Pattern: `rootCmd.AddCommand(populateCmd)`

### 3. Testing Strategy
- Unit tests for cmd/populate_test.go
- Manual testing with testdata/sources.yaml
- Integration tests if time permits

### 4. Configuration Notes
Remember these config fields (from pkg/config/config.go):
```go
type Populate struct {
    SourceIDs               []int
    ReleaseVersion          string
    ReleaseDate             string
    WithFlatClassification  *bool
}
```

These are **runtime-only** fields (NOT in ToOptions, NOT in env vars):
- Set via CLI flags only
- Use Option functions: OptSourceIDs, OptReleaseVersion, etc.

### 5. Key Implementation Details

**Flag Processing Pattern** (from cmd/create.go):
```go
// Only apply flag if explicitly set
if cmd.Flags().Changed("flag-name") {
    opts = append(opts, config.OptFlagName(value))
}
```

**Error Handling Pattern**:
```go
if err != nil {
    gn.Error(err)
    os.Exit(1)
}
```

**Progress Display**:
- Use `gn.Info()` for user-facing messages
- Detailed logs go to file automatically

## Files to Reference

### Similar Commands
- `cmd/create.go` - Good example of database operations
- `cmd/migrate.go` - Similar structure

### Core Interfaces
- `pkg/lifecycle/lifecycle.go` - Populator interface
- `pkg/config/config.go` - Populate config struct
- `internal/iopopulate/populator.go` - Implementation to call

### Option Functions to Use
Check `pkg/config/options.go` for:
- `OptSourceIDs([]int)`
- `OptReleaseVersion(string)`
- `OptReleaseDate(string)`
- `OptWithFlatClassification(*bool)`

## Remaining Work After Phase 5

### Phase 6: Final Verification
- Test with real SFGA file (if available)
- End-to-end testing
- Performance verification
- Fix 80-column violations (noted in Phase 3/4 files from spec-kit)
- Final cleanup and commit

## Important Patterns to Follow

1. **80-column width** - User requirement
2. **gn.Error pattern** - All user-facing errors
3. **Private implementations** - lowercase struct names
4. **Full config passing** - Always pass `*config.Config`, not substructs
5. **User commits** - Don't auto-commit, let user review
6. **Flag precedence** - flags > env vars > config file > defaults

## Quick Start for Next Session

1. Check POPULATE_TASKS.md for full context
2. Implement cmd/populate.go (reference cmd/create.go)
3. Add to cmd/root.go
4. Run `just test` and `just lint`
5. Update POPULATE_TASKS.md when Phase 5 complete
