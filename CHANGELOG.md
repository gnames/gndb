# Changelog

## Unreleased

- Add: `abbr_genera` table built in the abbreviation step of
  `gndb optimize`. It has the size of every genus of `abbr_lookups`
  (distinct epithet stems) and its classification, one row per kingdom
  for homonyms. Classifications come from Catalogue of Life XR, then
  Catalogue of Life, then the largest data source with genus ranks.
  Needs `gndb migrate` on existing databases.
- Add: `known` flag in `abbr_genera`: the genus is in IRMNG (data
  source 181) or has a classification. Misspelled genera are usually
  unknown. Needs `gndb migrate`.
- Fix: abbreviation lookup skips names with an abbreviated genus
  (`C. elegans`) or a broken character in the genus, which cannot be
  candidates for abbreviations found in text.
- Fix: only `GNDB_*` environment variables configure gndb. Generic
  variables like `DATABASE_HOST`, `DATABASE_PASSWORD` or `LOG_LEVEL`,
  set for other programs, were used as well.

## [v0.1.5] - 2026-10-09 Fri

- Add [#14]: `abbr_lookups` table built by `gndb optimize` (new step 6 of 7)
  to find candidate names for abbreviations like `C. elegans`. Keys
  are built with gnparser's `stemmer.AbbrKey`, shared with gnames.
- Fix: issues with `gndb export` subcommand, add the rest of missing
  datasets to `sources.yaml`.
- Add: abbreviation lookup considers only canonical forms with
  cardinality > 1, because uninomials cannot be abbreviated this way.
- Add: Mammal Diversity Database (MDD) with ID 216.
- Add: dinoflagellates datasets to `sources.yaml`.
- Remove: name linkage bar.
- Change: rename version id to `sf__id`.
- Update: Go dependencies.

## [v0.1.4] - 2026-04-07 Tue

- Add [#13]: subcommand `gndb delete` to remove datasets from database.
- Add: datasets 93-110.

## [v0.1.3] - 2026-04-04 Sat

- Fix: constrain primary key in name_string_indices further, removing
  possibility to have duplicates. Two datasets that are outdated are now
  removed: Union and Freebase (they had duplicates and were not worth to
  be fixed.
- Add [#10]: export subcommand to convert dataset from gnames database to
  SFGA format. So far it does work, however some of the datasets have duplicated
  record_id values and this problem will need to be resolved. Also we do
  not have a composite primary key for name_string_indices table, this needs
  to be fixed as well.
- Add [#12]: improve migration using Atlas.

## [v0.1.2] - 2026-03-26 Thu

- Add: Calculate IsOutlinkReady accoring to validation of outlink data.
- Add: World Spider Catalogue with ID 215
- Add [#11]: automatic update of 'official' datasets configuration by
  always copying sources.yaml file to configuration directory.
- Add [#9]: prefer_flat_classification settiing in sources.yaml.
- Add: allow name.col__link to cary outlink information.
- Add: Zenodo button in README.md, CITATION.cff.

## [v0.1.1] - 2026-03-06 Fri

- Add: .zenodo.json to integrate the repo with Zenodo.
- Add: update modules
- Add [#8]: Improve README.md documentation (installation, quick start,
  commands, configuration, data sources with file formats and naming
  conventions).

## [v0.1.0] - 2026-02-20 Fri

- Add: ICTV viruses to sources.yaml
- Add: Plazi source to sources.yaml
- Add: Fauna and flora of Brazil to sources.yaml
- Add: Timespan reports for populate and optimize stages
- Fix: Memory usage during word parsing
- Perf: Stream word parsing for lower memory footprint
- Update: Go module dependencies
- Update: CoL XR title in sources.yaml

## [v0.0.3] - 2025-11-18 Tue

- Add [#5]: Database optimization (word extraction, verification views,
  vernacular normalization, vacuum analyze)
- Add [#6]: Check SFGA data source version during population
- Add: Comparison tool
- Add: CoL XR to sources.yaml
- Fix [#7]: Migrate by dropping materialized view
- Refactor: Clean Architecture alignment, sources I/O isolation

## [v0.0.2] - 2025-10-31 Thu

- Add [#3]: Database schema migration
- Add [#4]: Data population from SFGA files
- Fix: Remove broken runtime.Caller from error functions

## [v0.0.1] - 2025-10-30 Thu

- Add [#1]: configuration framework
- Add: logging

## Footnotes

This document follows [changelog guidelines]

[v0.1.5]: https://github.com/gnames/gndb/tree/v0.1.5
[v0.1.4]: https://github.com/gnames/gndb/tree/v0.1.4
[v0.1.3]: https://github.com/gnames/gndb/tree/v0.1.3
[v0.1.2]: https://github.com/gnames/gndb/tree/v0.1.2
[v0.1.1]: https://github.com/gnames/gndb/tree/v0.1.1
[v0.1.0]: https://github.com/gnames/gndb/tree/v0.1.0
[v0.0.3]: https://github.com/gnames/gndb/tree/v0.0.3
[v0.0.2]: https://github.com/gnames/gndb/tree/v0.0.2
[v0.0.1]: https://github.com/gnames/gndb/tree/v0.0.1
[#14]: https://github.com/gnames/gndb/issues/14
[#13]: https://github.com/gnames/gndb/issues/13
[#12]: https://github.com/gnames/gndb/issues/12
[#11]: https://github.com/gnames/gndb/issues/11
[#10]: https://github.com/gnames/gndb/issues/10
[#9]: https://github.com/gnames/gndb/issues/9
[#8]: https://github.com/gnames/gndb/issues/8
[#7]: https://github.com/gnames/gndb/issues/7
[#6]: https://github.com/gnames/gndb/issues/6
[#5]: https://github.com/gnames/gndb/issues/5
[#4]: https://github.com/gnames/gndb/issues/4
[#3]: https://github.com/gnames/gndb/issues/3
[#1]: https://github.com/gnames/gndb/issues/1
[changelog guidelines]: https://github.com/olivierlacan/keep-a-changelog
