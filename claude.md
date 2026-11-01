# 韭菜農場 — notes for Claude

Hidden-gem Taiwan stock screener + backtester. Flutter app (`app/`) + Go/Gin API (`api/`).

## Rules
- Everything runs in Docker; Go and Flutter are NOT installed locally. Use
  `docker compose run --rm go ...` / `docker compose run --rm flutter ...` (see README).
- Code comments in English. User-facing UI strings in Traditional Chinese.
- Do not write files with Windows PowerShell 5.1 `Set-Content`/`Get-Content -Raw` round-trips:
  they corrupt UTF-8 (mojibake + BOM). Use the Write/Edit tools.
- Taiwan colors: red = up, green = down, with ▲▼.

## Architecture (api/)
- `internal/market`   exchange fetchers (TWSE/TPEx OpenAPI + web endpoints), throttled client, quote service
- `internal/store`    PostgreSQL/TimescaleDB (goose migrations embedded), Redis cache, dataset loader
- `internal/screener` JSON strategy → point-in-time evaluation (bars sliced at asOf)
- `internal/backtest` day-by-day simulator, metrics, background job manager
- `internal/jobs`     scheduler, daily ingest, resumable whole-market backfill
- `internal/api`      Gin handlers
- `cmd/server`, `cmd/worker`

## Non-negotiable backtest rules
- Only data available on/before the evaluation day; revenue visible the day AFTER announcement.
- Enter at next day's open; costs and limit-up/down locks applied; delisted stocks retained.
