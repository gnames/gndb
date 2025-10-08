# Code Preservation Guide

**Date**: 2025-10-08
**Purpose**: Document existing working code to preserve during architecture refactor

---

## ✅ Keep As-Is (100%)

### 1. Configuration Package
**Location**: `pkg/config/config.go`, `pkg/config/config_test.go`
**Reason**: Well-structured, follows pure/impure separation, has validation and defaults
**Status**: No changes needed

### 2. Configuration Loader
**Location**: `internal/io/config/loader.go`, `internal/io/config/loader_test.go`
**Reason**: Properly implements impure I/O, viper integration works
**Status**: No changes needed

### 3. GORM Models
**Location**: `pkg/schema/models.go`, `pkg/schema/gorm.go`
**Reason**: Table definitions and AutoMigrate logic are correct
**Action**: Wrap in SchemaManager interface, but preserve logic

---

## 🔄 Preserve & Relocate

### 4. DatabaseOperator - Connection Logic
**Source**: `internal/io/database/operator.go` lines 24-63
**Keep**:
```go
func (p *PgxOperator) Connect(ctx context.Context, cfg *config.DatabaseConfig) error {
    // DSN building
    dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s", ...)

    // Pool configuration
    poolConfig, err := pgxpool.ParseConfig(dsn)
    poolConfig.MaxConns = int32(cfg.MaxConnections)
    poolConfig.MinConns = int32(cfg.MinConnections)
    poolConfig.MaxConnLifetime = ...
    poolConfig.MaxConnIdleTime = ...

    // Create and verify pool
    pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
    err := pool.Ping(ctx)

    p.pool = pool
    return nil
}
```
**Action**: Keep in DatabaseOperator

---

### 5. DatabaseOperator - Core Methods
**Source**: `internal/io/database/operator.go`

**Keep in DatabaseOperator**:
- `Close()` (lines 65-71)
- `Pool()` (lines 314-317)
- `TableExists()` (lines 73-94)
- `DropAllTables()` (lines 96-137)

**Reason**: These are the 5 core methods per contract

---

### 6. SetCollation Logic
**Source**: `internal/io/database/operator.go` lines 283-312
**Current Location**: DatabaseOperator (❌ wrong)
**Move To**: SchemaManager (✅ correct - schema concern)

```go
func (p *PgxOperator) SetCollation(ctx context.Context) error {
    type columnDef struct {
        table, column string
        varchar       int
    }
    columns := []columnDef{
        {"name_strings", "name", 500},
        {"canonicals", "name", 255},
        // ... etc
    }

    qStr := `ALTER TABLE %s ALTER COLUMN %s TYPE VARCHAR(%d) COLLATE "C"`
    for _, col := range columns {
        q := fmt.Sprintf(qStr, col.table, col.column, col.varchar)
        if _, err := p.pool.Exec(ctx, q); err != nil {
            return fmt.Errorf("failed to set collation on %s.%s: %w", col.table, col.column, err)
        }
    }
    return nil
}
```

**Action**: Move to `internal/io/schema/manager.go`, call after GORM AutoMigrate

---

### 7. Optimization Methods
**Source**: `internal/io/database/operator.go`
**Current Location**: DatabaseOperator (❌ wrong)
**Move To**: Optimizer (✅ correct - optimization concern)

**Methods to move**:
- `VacuumAnalyze()` (lines 140-153)
- `CreateIndexConcurrently()` (lines 156-167)
- `RefreshMaterializedView()` (lines 170-186)
- `SetStatisticsTarget()` (lines 189-210)

**Action**: Move to `internal/io/optimize/optimizer.go`

---

## ❌ Delete (YAGNI - Violates Principle VIII)

### 8. Utility Methods
**Source**: `internal/io/database/operator.go`
**Delete**:
- `GetDatabaseSize()` (lines 213-227) - Used nowhere, "just in case" code
- `GetTableSize()` (lines 230-243) - Used nowhere
- `ListTables()` (lines 247-279) - Used once for display, inline it

**Reason**: Violate "No 'just in case' code" principle. Can be added later if truly needed.

---

## 📋 Refactor Tasks Summary

1. ✅ **Keep**: Config, loader, models (no changes)
2. 🔄 **Trim**: DatabaseOperator to 5 methods
3. 🔄 **Move**: SetCollation → SchemaManager
4. 🔄 **Move**: 4 optimization methods → Optimizer
5. ❌ **Delete**: 3 utility methods (GetDatabaseSize, GetTableSize, ListTables)
6. 🆕 **Create**: Interface layer in pkg/

---

## Code Statistics

**Before**: DatabaseOperator has 14 methods (318 lines)
**After**: DatabaseOperator has 5 methods (~150 lines)
**Preserved**: ~70% of working connection/query logic
**Relocated**: ~30% to proper components
**Deleted**: ~10% unnecessary utilities

**Result**: Clean architecture, preserved working code, Principle VIII compliant
