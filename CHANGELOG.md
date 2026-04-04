# Changelog

## Unreleased

## [v0.1.3] - 2026-04-04 Sat

- Fix: contrain primary key in name_string_indices further, removing
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

[v0.1.3]: https://github.com/gnames/gndb/tree/v0.1.3
[v0.1.2]: https://github.com/gnames/gndb/tree/v0.1.2
[v0.1.1]: https://github.com/gnames/gndb/tree/v0.1.1
[v0.1.0]: https://github.com/gnames/gndb/tree/v0.1.0
[v0.0.3]: https://github.com/gnames/gndb/tree/v0.0.3
[v0.0.2]: https://github.com/gnames/gndb/tree/v0.0.2
[v0.0.1]: https://github.com/gnames/gndb/tree/v0.0.1
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
