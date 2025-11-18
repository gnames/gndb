# Changelog

## Unreleased

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

[v0.0.3]: https://github.com/gnames/gndb/tree/v0.0.3
[v0.0.2]: https://github.com/gnames/gndb/tree/v0.0.2
[v0.0.1]: https://github.com/gnames/gndb/tree/v0.0.1
[#7]: https://github.com/gnames/gndb/issues/7
[#6]: https://github.com/gnames/gndb/issues/6
[#5]: https://github.com/gnames/gndb/issues/5
[#4]: https://github.com/gnames/gndb/issues/4
[#3]: https://github.com/gnames/gndb/issues/3
[#1]: https://github.com/gnames/gndb/issues/1
[changelog guidelines]: https://github.com/olivierlacan/keep-a-changelog
