# GNdb Log Query Cheatsheet

This document provides useful `jq` queries for analyzing GNdb JSON logs.

## Log File Location

```bash
~/.local/share/gndb/logs/gndb.log
```

## Basic Queries

### View all logs in readable format
```bash
cat ~/.local/share/gndb/logs/gndb.log | jq '.'
```

### View logs with compact output
```bash
cat ~/.local/share/gndb/logs/gndb.log | jq -c '.'
```

### Get just timestamps and messages
```bash
cat ~/.local/share/gndb/logs/gndb.log | jq -r '"\(.time) - \(.msg)"'
```

### Extract specific fields
```bash
cat ~/.local/share/gndb/logs/gndb.log | jq '{time, level, msg}'
```

## Filter by Log Level

### Only errors
```bash
cat ~/.local/share/gndb/logs/gndb.log | jq 'select(.level == "ERROR")'
```

### Only warnings
```bash
cat ~/.local/share/gndb/logs/gndb.log | jq 'select(.level == "WARN")'
```

### Only info messages
```bash
cat ~/.local/share/gndb/logs/gndb.log | jq 'select(.level == "INFO")'
```

### Info and above (INFO, WARN, ERROR)
```bash
cat ~/.local/share/gndb/logs/gndb.log | jq 'select(.level == "INFO" or .level == "WARN" or .level == "ERROR")'
```

### Errors and warnings only
```bash
cat ~/.local/share/gndb/logs/gndb.log | jq 'select(.level == "ERROR" or .level == "WARN")'
```

## Search by Message Content

### All logs containing "config"
```bash
cat ~/.local/share/gndb/logs/gndb.log | jq 'select(.msg | contains("config"))'
```

### All logs containing "database"
```bash
cat ~/.local/share/gndb/logs/gndb.log | jq 'select(.msg | contains("database"))'
```

### All logs containing "error" or "failed" (case-insensitive)
```bash
cat ~/.local/share/gndb/logs/gndb.log | jq 'select(.msg | ascii_downcase | contains("error") or contains("failed"))'
```

### Specific message
```bash
cat ~/.local/share/gndb/logs/gndb.log | jq 'select(.msg == "Configuration loaded successfully")'
```

## Filter by Fields

### Logs with database host information
```bash
cat ~/.local/share/gndb/logs/gndb.log | jq 'select(.database_host)'
```

### Logs with error field
```bash
cat ~/.local/share/gndb/logs/gndb.log | jq 'select(.error)'
```

### Logs with config_file field
```bash
cat ~/.local/share/gndb/logs/gndb.log | jq 'select(.config_file)'
```

### Logs with specific field value
```bash
cat ~/.local/share/gndb/logs/gndb.log | jq 'select(.log_format == "json")'
```

## Time-Based Queries

### Filter logs after specific time
```bash
cat ~/.local/share/gndb/logs/gndb.log | jq 'select(.time > "2025-10-30T12:00:00Z")'
```

### Filter logs before specific time
```bash
cat ~/.local/share/gndb/logs/gndb.log | jq 'select(.time < "2025-10-30T13:00:00Z")'
```

### Filter logs within time range
```bash
cat ~/.local/share/gndb/logs/gndb.log | jq 'select(.time > "2025-10-30T12:00:00Z" and .time < "2025-10-30T13:00:00Z")'
```

## Aggregation and Statistics

### Count total log entries
```bash
cat ~/.local/share/gndb/logs/gndb.log | jq -s 'length'
```

### Count log entries by level
```bash
cat ~/.local/share/gndb/logs/gndb.log | jq -r '.level' | sort | uniq -c
```

### List all unique messages
```bash
cat ~/.local/share/gndb/logs/gndb.log | jq -r '.msg' | sort | uniq
```

### Count occurrences of each message
```bash
cat ~/.local/share/gndb/logs/gndb.log | jq -r '.msg' | sort | uniq -c | sort -rn
```

## Bootstrap and Configuration Queries

### View complete bootstrap sequence
```bash
cat ~/.local/share/gndb/logs/gndb.log | jq 'select(.msg | contains("Bootstrap") or contains("directory") or contains("Logger") or contains("Config"))'
```

### Get configuration summary
```bash
cat ~/.local/share/gndb/logs/gndb.log | jq 'select(.msg == "Configuration loaded successfully")'
```

### View all configuration-related logs
```bash
cat ~/.local/share/gndb/logs/gndb.log | jq 'select(.msg | contains("onfiguration"))'
```

### View environment variable binding
```bash
cat ~/.local/share/gndb/logs/gndb.log | jq 'select(.msg | contains("environment variable"))'
```

## Error Analysis

### Pretty print errors with context
```bash
cat ~/.local/share/gndb/logs/gndb.log | jq 'select(.level == "ERROR") | {time, msg, error}'
```

### List all error messages
```bash
cat ~/.local/share/gndb/logs/gndb.log | jq -r 'select(.level == "ERROR") | .msg' | sort | uniq
```

### Get full error details
```bash
cat ~/.local/share/gndb/logs/gndb.log | jq 'select(.level == "ERROR")'
```

### Errors with specific context field
```bash
cat ~/.local/share/gndb/logs/gndb.log | jq 'select(.level == "ERROR" and .config_path)'
```

## Advanced Queries

### Create custom report
```bash
cat ~/.local/share/gndb/logs/gndb.log | jq -r '"\(.level | .[0:1]) [\(.time | split("T")[1] | split(".")[0])] \(.msg)"'
```

### Extract database configuration from logs
```bash
cat ~/.local/share/gndb/logs/gndb.log | jq 'select(.database_host) | {database_host, database_port, database_name, batch_size}'
```

### Get log format settings from logs
```bash
cat ~/.local/share/gndb/logs/gndb.log | jq 'select(.log_format) | {log_format, log_level, log_destination}'
```

### Group logs by hour
```bash
cat ~/.local/share/gndb/logs/gndb.log | jq -r '.time | split("T")[1] | split(":")[0]' | sort | uniq -c
```

## Useful Aliases

Add these to your `.bashrc` or `.zshrc` for quick access:

```bash
# View GNdb logs
alias gndb-logs='cat ~/.local/share/gndb/logs/gndb.log | jq "."'

# View GNdb errors
alias gndb-errors='cat ~/.local/share/gndb/logs/gndb.log | jq "select(.level == \"ERROR\")"'

# View GNdb log summary
alias gndb-summary='cat ~/.local/share/gndb/logs/gndb.log | jq -r ".level" | sort | uniq -c'

# Tail GNdb logs in real-time
alias gndb-tail='tail -f ~/.local/share/gndb/logs/gndb.log | jq "."'

# Search GNdb logs
gndb-search() {
    cat ~/.local/share/gndb/logs/gndb.log | jq "select(.msg | contains(\"$1\"))"
}
```

## Real-Time Monitoring

### Tail logs with jq formatting
```bash
tail -f ~/.local/share/gndb/logs/gndb.log | jq '.'
```

### Tail only errors
```bash
tail -f ~/.local/share/gndb/logs/gndb.log | jq 'select(.level == "ERROR")'
```

### Tail with compact output
```bash
tail -f ~/.local/share/gndb/logs/gndb.log | jq -c '.'
```

## Tips

- Use `-r` flag with jq for raw output (no quotes around strings)
- Use `-c` flag with jq for compact JSON output (one line per entry)
- Use `-s` flag with jq to slurp all entries into an array for aggregation
- Pipe through `less -R` for colored pagination: `jq '.' | less -R`
- Combine with `grep` for simple text filtering before jq processing
- Use `jq --help` for more options and syntax

## Common Troubleshooting Patterns

### Find when application started
```bash
cat ~/.local/share/gndb/logs/gndb.log | jq 'select(.msg == "Bootstrap process started")'
```

### Check logger configuration
```bash
cat ~/.local/share/gndb/logs/gndb.log | jq 'select(.msg | contains("Logger") and .format)'
```

### Verify directories were created
```bash
cat ~/.local/share/gndb/logs/gndb.log | jq 'select(.msg | contains("directories ensured"))'
```

### Check configuration loading sequence
```bash
cat ~/.local/share/gndb/logs/gndb.log | jq 'select(.msg | contains("Configuration")) | {time, msg, config_file}'
```
