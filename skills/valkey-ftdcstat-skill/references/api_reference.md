# valkey-ftdcstat usage and local API reference

## CLI usage

Build:

```bash
go build -o valkey-ftdcstat ./cmd/valkey-ftdcstat
```

General form:

```bash
valkey-ftdcstat <path-to-diagnostic-data-directory> [--view VIEW] [--interval N] [--avg DURATION] [--top N] [--device DEVICE] [--from ISO_TIME] [--to ISO_TIME] [--json] [--web] [--listen ADDR] [--verbose]
```

Input must be a `valkey-ftdc` diagnostic directory containing `metadata.*.json` and `metrics.*.vkftdc` files.

## command recipes

```bash
valkey-ftdcstat diagnostic.data
valkey-ftdcstat diagnostic.data --view summary | less -S
valkey-ftdcstat diagnostic.data --view summary --avg 5m
valkey-ftdcstat diagnostic.data --from "2026-06-20T12:00:00" --to "2026-06-20T18:00:00"
valkey-ftdcstat diagnostic.data --view commandstats --top 5
valkey-ftdcstat diagnostic.data --view host --device sda --verbose
valkey-ftdcstat diagnostic.data --view latency --json
valkey-ftdcstat diagnostic.data --web --view summary --avg 5m
valkey-ftdcstat diagnostic.data --web --listen 127.0.0.1:8080
```

## local web mode

`--web` starts a local HTTP server and **still prints** the terminal report. Default bind: `127.0.0.1:0` (random port printed in output).

Endpoints:

```text
GET /              -> embedded index.html
GET /app.js        -> embedded javascript
GET /style.css     -> embedded css
GET /api/metadata  -> view, sections, headerText, warnings, rowCount
GET /api/data      -> derived rows grouped by section
```

`/api/metadata` includes `headerText` mirroring the CLI header. `/api/data` rows use section keys matching chart groups (`server`, `commands`, `memory`, …).

Use `--avg` or `--from`/`--to` on large captures for responsive browser rendering.

## JSON mode

`--json` emits the full `derive.Report` structure: metadata, columns, rows, optional `commands` capture summary, header, and latency notes.

Rows contain a `time` field and a `values` map keyed by column name. Missing numerics are omitted or null depending on serialization.

`--json` cannot be combined with `--web`.

## regenerating golden tests

After intentional output changes:

```bash
go test ./cmd/valkey-ftdcstat -run TestGolden -v   # verify
# regenerate fixtures manually:
go build -o valkey-ftdcstat ./cmd/valkey-ftdcstat
for v in summary server latency commandstats host; do
  extra=""; [ "$v" = commandstats ] && extra="--top 3"
  ./valkey-ftdcstat testfixtures/diagnostic.data --view $v --interval 60 $extra > testfixtures/outputs/${v}.golden
done
```
