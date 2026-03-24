# Inferring Ranks for Unranked Breadcrumbs

This file covers the rank-inference strategy used during export when
`classification_ranks` is empty but `classification` names are present
(Case 3 of classification reconstruction — see `EXPORT_PLAN.md`).

The plan has two parts:

1. **Runtime inference** — how each breadcrumb name gets a rank during
   `gndb export`
2. **Dictionary generation** — how `rankdict.yaml` is produced from the
   gnames database

---

## Runtime Rank Inference

### The problem

PostgreSQL stores classifications as pipe-delimited strings:

```
classification:       "Animalia|Chordata|Mammalia|Carnivora|Felidae|Felis"
classification_ranks: "kingdom|phylum|class|order|family|genus"
classification_ids:   "1|5|6|7|8|9"
```

When `classification_ranks` is empty, we have names without ranks.
Rather than dropping the entire breadcrumb, we attempt rank inference for
each name using two complementary strategies.

### Strategy A — Suffix rules (family and below, code-aware)

Nomenclatural codes mandate specific suffixes for formal ranks:

| Rank | Botany (ICN) | Zoology (ICZN) |
|---|---|---|
| family | `-aceae` | `-idae` |
| subfamily | `-oideae` | `-inae` |
| tribe | `-eae` | `-ini` |
| subtribe | `-inae` | `-ina` |
| superfamily | — | `-oidea` |
| order | `-ales` | `-ales` (many groups) |
| suborder | `-ineae` | — |
| class | `-opsida` / `-mycetes` / `-phyceae` | — |
| phylum | `-phyta` / `-mycota` | — |

Key ambiguity: `-inae` means **subfamily** in zoology but **subtribe** in
botany. Resolve using this priority:

1. **`nsi.CodeID`** if set (1=ICZN → subfamily, 2=ICN → subtribe)
2. **Breadcrumb code vote** — scan every name in `classification` and
   collect its code from the dictionary or from unambiguous suffixes
   (`-aceae`/`-oideae`/`-eae` → icn; `-idae`/`-ini` → iczn). Tally
   the votes; the majority code applies to all ambiguous nodes in the
   same breadcrumb. Semihomonyms at higher ranks are rare enough that
   a single resolved node (kingdom, phylum, etc.) is normally decisive.
3. **Default to subfamily** if the breadcrumb yields no code signal at
   all (more common rank overall).

### Strategy B — Higher-taxon dictionary (order and above)

Suffix rules don't reliably cover order, class, phylum, kingdom in
zoology. Bacteria and viruses come from sources (NCBI, LPSN, ICTV) that
always include rank data — the missing-ranks problem only arises with
botanical and zoological sources. The dictionary therefore covers only
**Plantae** and **Animalia**. This keeps the scope tight: zoological
orders alone number under ~1,000; plant orders under ~100. Combined with
classes, phyla, and the two kingdoms, the total is well under 1,500
entries.

Store as a YAML data file embedded at build time:

```
internal/ioexport/data/rankdict.yaml
```

Each entry stores both **rank** and **nomenclatural code**, so a single
dictionary lookup resolves both questions at once — no separate
code-inference step needed for names found in the dictionary.

Structure:

```yaml
# Classical Linnaean orders and higher taxa: rank + nomenclatural code.
# Covers Animalia (iczn) and Plantae (icn) only.
# Sources: CoL (1), ITIS (3), WoRMS (9)
---
Animalia:       {rank: kingdom, code: iczn}
Plantae:        {rank: kingdom, code: icn}

Chordata:       {rank: phylum,  code: iczn}
Arthropoda:     {rank: phylum,  code: iczn}
Mollusca:       {rank: phylum,  code: iczn}
Tracheophyta:   {rank: phylum,  code: icn}
Bryophyta:      {rank: phylum,  code: icn}
# ... all ~60 phyla

Mammalia:       {rank: class, code: iczn}
Aves:           {rank: class, code: iczn}
Insecta:        {rank: class, code: iczn}
Magnoliopsida:  {rank: class, code: icn}
Liliopsida:     {rank: class, code: icn}
# ... all ~200 classes

Carnivora:      {rank: order, code: iczn}
Primates:       {rank: order, code: iczn}
Passeriformes:  {rank: order, code: iczn}
Rosales:        {rank: order, code: icn}
Poales:         {rank: order, code: icn}
# ... all ~800 orders
```

Load once at exporter startup with `go:embed` and build a
`map[string]RankInfo` for O(1) lookup:

```go
type RankInfo struct {
    Rank string
    Code string // "iczn" or "icn"
}
```

The file is the source of truth; updating it does not require code
changes.

### Inference pipeline per breadcrumb node

```go
func inferRank(name string, codeID int) string {
    // 1. Dictionary lookup (order and above — comprehensive)
    if rank, ok := higherRanks[name]; ok {
        return rank
    }
    // 2. Suffix rules (family and below)
    return inferRankBySuffix(name, codeID)
}
```

### Handling unresolvable nodes

If a name's rank cannot be inferred by either strategy, skip that node
from the classification breadcrumb — do not emit a guess. The remaining
nodes whose ranks are known are still written. A breadcrumb with gaps
(e.g. kingdom and family known, phylum and class unknown) is better than
no breadcrumb at all.

---

## Generating `rankdict.yaml`

The dictionary is generated from the gnames PostgreSQL database itself —
the authoritative higher taxonomy is already present from imported sources
(CoL, ITIS, NCBI, WoRMS). This means:

- No external downloads needed
- Same sources we already trust for verification
- Regenerate whenever those sources are refreshed

### Source priority

Only plant and animal sources are needed. Resolve conflicts by source
authority:

| Priority | Data source ID | Source | Covers |
|---|---|---|---|
| 1 | 1 | Catalogue of Life | both |
| 2 | 3 | ITIS | Animalia + Plantae |
| 3 | 9 | WoRMS | Animalia (marine) |
| 4 | others | any other curated source | — |

NCBI (ID 4) is excluded — it carries ranks consistently and its higher
taxonomy can differ from classical Linnaean treatment.

### SQL query

The query anchors on the Plantae and Animalia subtrees by joining through
the classification breadcrumb: any taxon whose `classification` starts
with `Animalia` or `Plantae` is in scope.

```sql
SELECT DISTINCT ON (c.name)
    c.name          AS taxon_name,
    lower(nsi.rank) AS rank,
    CASE
        WHEN nsi.classification LIKE 'Plantae%'
          OR c.name = 'Plantae'  THEN 'icn'
        WHEN nsi.classification LIKE 'Animalia%'
          OR c.name = 'Animalia' THEN 'iczn'
        ELSE 'unknown'
    END             AS code
FROM name_string_indices nsi
JOIN name_strings ns ON ns.id = nsi.name_string_id
JOIN canonical    c  ON c.id  = ns.canonical_id
WHERE nsi.data_source_id IN (1, 3, 9)
  AND lower(nsi.rank) IN (
      'kingdom', 'subkingdom',
      'phylum',  'subphylum',
      'class',   'subclass',
      'order',   'suborder'
  )
  AND (
      nsi.classification LIKE 'Animalia%'
   OR nsi.classification LIKE 'Plantae%'
   OR c.name IN ('Animalia', 'Plantae')
  )
  AND nsi.taxonomic_status NOT IN ('synonym', 'ambiguous synonym', 'misapplied')
  AND c.name != ''
  AND c.name NOT LIKE '% %'     -- higher taxa are uninomials
ORDER BY c.name,
    CASE nsi.data_source_id
        WHEN 1 THEN 1
        WHEN 3 THEN 2
        WHEN 9 THEN 3
        ELSE        4
    END
;
```

`DISTINCT ON (c.name) ... ORDER BY c.name, priority` picks one rank per
name, preferring the highest-priority source.

### `gndb gen-rankdict` command

Add a hidden maintenance subcommand that runs the query and writes
`rankdict.yaml`:

```
gndb gen-rankdict [--output path/to/rankdict.yaml]
```

- Hidden from help (`cmd.Hidden = true`) — not part of the user-facing API
- Output path defaults to `internal/ioexport/data/rankdict.yaml` (the
  embedded file)
- Writes YAML grouped by rank for readability:

```yaml
# Generated by gndb gen-rankdict on 2025-08-25
# Sources: CoL (1), ITIS (3), WoRMS (9)
# Scope: Animalia (iczn) + Plantae (icn) only
# Total entries: ~1100
---
# kingdoms (2)
Animalia: {rank: kingdom, code: iczn}
Plantae:  {rank: kingdom, code: icn}

# phyla (~60)
Acanthocephala: {rank: phylum, code: iczn}
Annelida:       {rank: phylum, code: iczn}
Arthropoda:     {rank: phylum, code: iczn}
Bryophyta:      {rank: phylum, code: icn}
Chordata:       {rank: phylum, code: iczn}
Tracheophyta:   {rank: phylum, code: icn}
# ...

# classes (~200)
Actinopterygii: {rank: class, code: iczn}
Aves:           {rank: class, code: iczn}
Insecta:        {rank: class, code: iczn}
Magnoliopsida:  {rank: class, code: icn}
Mammalia:       {rank: class, code: iczn}
# ...

# orders (~800)
Accipitriformes: {rank: order, code: iczn}
Agaricales:      {rank: order, code: icn}
Carnivora:       {rank: order, code: iczn}
Poales:          {rank: order, code: icn}
Rosales:         {rank: order, code: icn}
# ...
```

### When to regenerate

Regenerate `rankdict.yaml` after importing a new version of CoL or ITIS,
then commit the updated file. The embedded dictionary in the binary stays
current with the data it was built from.
