# Research for Optimize Database Performance

## Decision: Follow `gnidump rebuild` Logic

**Rationale**: The user has specified that the logic for the `optimize` command should follow the `rebuild` command from the `gnidump` project. This approach is production-tested and directly addresses the requirements for optimizing the `gnverifier` database.

**Alternatives considered**: A new implementation from scratch was considered but rejected in favor of a proven, existing implementation.

## Key Implementation Steps from `gnidump rebuild`

The following steps have been identified from the user's instructions and analysis of the `gnidump` project:

1.  **Remove Orphaned `name_strings`**: Delete records from the `name_strings` table that are not referenced in the `name_string_indices` table. This is a data cleanup step.

2.  **Parse Names and Generate Canonical Forms**: For each name string, parse it to generate its canonical form, full canonical form, and stemmed canonical form. This is a CPU-intensive data processing step.

3.  **Extract and Link Words**: Extract words from the canonical forms and authorships. Link these words back to the `name_strings` table. This enables faster searching.

4.  **Create Materialized View**: Generate a materialized view to denormalize data for faster query performance. The exact structure of this view needs to be determined from the `gnidump` source code.
