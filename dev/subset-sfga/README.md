# subset-sfga

Development tool to create smaller SFGA test files from full data sources.

## Purpose

Large SFGA files (100MB-10GB+) are too big for the `testdata/` directory, but we need representative test data that includes edge cases found in production sources.

This tool:
- Extracts 20k-40k records from a full SFGA source
- Preserves edge cases (empty names, special chars, deep hierarchies, orphans)
- Maintains hierarchy consistency (all parent nodes included)
- Outputs a valid SFGA SQLite file ready for testing

## Usage

### Basic Usage

```bash
# From dev/subset-sfga directory
go run . <source> <output>
```

**Arguments:**
- `source`: SFGA source URL or local file path
- `output`: Path for output subset SFGA file

**Configuration:**
- Target size: 30,000 name_string records (hardcoded constant)
- Edge cases: Minimum 50 records per category
- Hierarchy: All parent chains preserved

## Examples

### Download and subset from URL
```bash
go run . "http://opendata.globalnames.org/sfga/latest/0001.sqlite.zip" ../../testdata/0001-col-subset.sqlite
```

### Subset from local file
```bash
go run . ~/Downloads/0206.sqlite ../../testdata/0206-ruhoff.sqlite
```

### Subset ITIS (local copy)
```bash
go run . /data/sfga/0003.sqlite ../../testdata/0003-itis-subset.sqlite
```

## Implementation Status

⚠️ **Currently a stub** - Full implementation pending.

See `main.go` for implementation plan:
1. Open/fetch source SFGA
2. Analyze edge cases
3. Build intelligent sample set
4. Create output database with selected records
5. Validate hierarchy integrity

## Edge Cases to Preserve

The tool should identify and include:

1. **Empty/Nil Names**: Records with empty `gn__scientific_name_string`
2. **Special Characters**: Unicode, combining diacritics, unusual punctuation
3. **Deep Hierarchy**: Taxa with 10+ parent levels
4. **Orphans**: Taxon records with missing parent_id references
5. **Rich Vernaculars**: Records with many vernacular names (5+ languages)
6. **Long Names**: Scientific names >100 characters
7. **Parsing Edge Cases**: Names that challenge gnparser

## Output Validation

After creating a subset, validate it:

```bash
# Test with gndb populate
cd ../..
gndb create --force
gndb populate --sources-yaml <(echo "data_sources:
  - id: 9999
    parent: ./testdata
    title_short: Test Subset") --sources 9999

# Verify tables populated
psql -d gnames_test -c "SELECT COUNT(*) FROM name_strings;"
```

## Future Enhancements

When implementing, consider:
- [ ] Configurable edge case rules (JSON/YAML config)
- [ ] Multiple sampling strategies (random, stratified, cluster)
- [ ] Dry-run mode to preview sample before extraction
- [ ] Summary report of included edge cases
- [ ] Parallel processing for large sources

## Notes

- Keep subset size 20k-40k for fast test execution
- Ensure hierarchy completeness (include all parent chains)
- Output file should be <5MB for easy git tracking
- Consider compressing output (gzip) if still too large
