---
name: valkey-ftdcstat-metrics
description: interpret valkey-ftdcstat output from valkey-ftdc diagnostic.data captures, including terminal tables, json output, and local web api data. use when analyzing valkey-ftdcstat metrics, explaining summary/server/memory/clients/host/network/latency/commandstats views, diagnosing valkey performance symptoms, mapping columns to source metrics and formulas, advising which valkey-ftdcstat command to run, or updating docs/tests that change valkey-ftdcstat metric semantics.
---

# valkey-ftdcstat metrics

## overview

Use this skill to read, explain, and troubleshoot `valkey-ftdcstat` output from Valkey FTDC `diagnostic.data` captures. Prefer the bundled reference over re-reading project source when interpreting metric columns, formulas, output formatting, command flags, or common performance symptoms.

## workflow

1. Determine what the user provided:
   - pasted terminal table output
   - `--json` output
   - local web API data from `/api/metadata` or `/api/data`
   - a request for which `valkey-ftdcstat` command to run
   - a code/docs/test change that affects metric semantics
2. Identify the selected view: `summary`, `server`, `memory`, `clients`, `cpu`, `persistence`, `replication`, `commandstats`, `host`, `network`, or `latency`.
3. Load `references/metric-reference.md` when column meanings, formulas, display semantics, or diagnostic hints are needed.
4. For web/API behavior, load `references/api_reference.md`.
5. Separate observed values from interpretation. State confidence and missing context when diagnosis depends on workload, deployment, or topology.
6. For code changes that modify columns, formulas, flags, JSON shape, or output semantics, update the metric reference, golden tests, and relevant unit tests together.

## how to use valkey-ftdcstat

Build from the repository root:

```bash
go build -o valkey-ftdcstat ./cmd/valkey-ftdcstat
```

General form:

```bash
valkey-ftdcstat <path-to-diagnostic-data-directory> [--view VIEW] [--interval N] [--avg DURATION] [--top N] [--device DEVICE] [--from ISO_TIME] [--to ISO_TIME] [--json] [--web] [--listen ADDR] [--verbose]
```

Common command recipes:

```bash
# broad triage
valkey-ftdcstat diagnostic.data --view summary | less -S

# fewer rows on long captures
valkey-ftdcstat diagnostic.data --view summary --avg 5m

# time window; from inclusive, to exclusive
valkey-ftdcstat diagnostic.data --from "2026-06-20T12:00:00" --to "2026-06-20T18:00:00"

# command mix over time
valkey-ftdcstat diagnostic.data --view commandstats --top 5

# host disk focus
valkey-ftdcstat diagnostic.data --view host --device sda

# latency events and fallbacks
valkey-ftdcstat diagnostic.data --view latency

# local charts; still prints terminal output
valkey-ftdcstat diagnostic.data --web --view summary --avg 5m
```

Flag rules:

- `--view summary` is the default.
- `--interval N` controls display spacing; it does not aggregate samples.
- `--avg DURATION` averages derived rows into fixed UTC buckets (`1m`–`15m`); cannot combine with `--interval`.
- `--top N` limits command columns in `summary` and `commandstats` (default 8; `--top 0` = all active commands).
- `--json` cannot combine with `--web`.
- `--verbose` expands only `memory`, `clients`, `replication`, `host`, and `network`.
- `--device` is only valid with `--view host`.

## interpreting pasted output

- Read header first: `metricsRange`, `serverInfo`, process/restart markers.
- `0` = present and zero; `-` = missing or rate suppressed after gap/restart.
- Command columns look like `get/s`, `info/s` (not `ops/s`).
- After `gap …: rate baseline reset`, rate columns are blank for that row.
- Do not over-diagnose from one spike; prefer sustained patterns.

## planned features

See `docs/backlog.md` in the repo. **Phase 8** will add `--view slowlog`: deduplicated slow operations ranked slowest-first with a repeat counter.

## bundled references

- `references/metric-reference.md` — views, flags, columns, formulas, formatting, diagnostic hints.
- `references/api_reference.md` — local web mode and JSON/API behavior.
