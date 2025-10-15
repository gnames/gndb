# Data Model for Optimize Database Performance

This feature does not introduce new tables or change the existing table schemas. Instead, it focuses on adding a performance-enhancing materialized view.

## New Materialized Views

A new materialized view will be created to denormalize data for `gnverifier`. The exact schema of this view will be based on the implementation in `gnidump`.