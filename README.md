# 韭菜農場

「大海撈針」選股 App：每天收盤後掃描全部上市、上櫃股票，找出符合策略條件的冷門股，並用回測驗證條件過去的勝率。

```
app/   Flutter（iOS / Android）
api/   Go + Gin 後端、排程、選股引擎、回測引擎
```

全部用 Docker 執行，本機不需要安裝 Go 或 Flutter。

## 啟動

```bash
cp .env.example .env            # optional
docker compose up -d --build    # TimescaleDB + Redis + API on :8080
curl localhost:8080/healthz
```

The API applies database migrations on startup.

## 資料

```bash
# Historical prices for the whole market (resumable; ~1 request / 3 s per exchange)
docker compose run --rm api worker backfill -years 5
docker compose run --rm api worker backfill -days 30          # quick test

docker compose run --rm api worker ingest-close     # today's closing prices
docker compose run --rm api worker ingest-evening   # valuations, institutional, revenue + screens
docker compose run --rm api worker screen -date 2026-10-30
```

Scheduled automatically while the API runs (Asia/Taipei, weekdays):

| Time      | Job                                                   |
|-----------|-------------------------------------------------------|
| every 10m | refresh quote cache                                   |
| 15:00     | store closing prices                                  |
| 20:00     | valuations, institutional trades, revenue → screens   |

## 開發

```bash
docker compose run --rm go go test ./...          # API tests
docker compose run --rm go go vet ./...
docker compose run --rm flutter flutter analyze   # app lint
docker compose run --rm flutter flutter test
docker compose run --rm flutter flutter build apk --dart-define=API_BASE_URL=http://<your-ip>:8080
```

iOS builds need macOS with Xcode (`flutter build ios`).

## API

| Method | Path                         | Notes                                   |
|--------|------------------------------|-----------------------------------------|
| GET    | `/api/v1/quotes`             | `?exchange=TWSE\|TPEx&q=&limit=`         |
| GET    | `/api/v1/quotes/:code`       |                                         |
| GET    | `/api/v1/stocks/:code/kline` | `?months=6`                             |
| GET    | `/api/v1/indicators`         | indicators usable in strategies         |
| GET    | `/api/v1/strategies`         |                                         |
| POST   | `/api/v1/strategies`         | `{"name", "definition"}`                |
| GET    | `/api/v1/screens/today`      | latest screening results                |
| POST   | `/api/v1/backtests`          | `{"strategy_id", "config"}` → job id    |
| GET    | `/api/v1/backtests/:id`      | status, progress, result                |
| *      | `/api/v1/watchlist`, `/api/v1/holdings` | header `X-User-ID` (dev only) |

## 策略格式

```json
{
  "name": "放量突破",
  "universe": {"common_stock_only": true, "min_price": 10, "min_avg_volume_20": 200000},
  "rule": {"op": "and", "rules": [
    {"label": "創 60 日新高", "left": {"indicator": "close"}, "cmp": ">", "right": {"indicator": "highest_close", "period": 60}},
    {"label": "量增 2 倍", "left": {"indicator": "volume_ratio", "period": 20}, "cmp": ">=", "right": {"value": 2}}
  ]}
}
```

Groups (`op: and|or`) nest arbitrarily. See `GET /api/v1/indicators` for the indicator list.

## 回測規則

- Signals from day *T*'s close enter at day *T+1*'s **open**.
- Exits: stop loss → trailing stop → take profit → max hold (conservative order on the same bar).
- Costs: 0.1425% fee (discount configurable, NT$20 minimum), 0.3% tax on sells.
- Locked limit-up days cannot buy; locked limit-down days cannot sell.
- Revenue is only visible the day **after** its announcement date.
- Delisted stocks stay in the data and are closed at their last price.

本 App 資訊僅供參考，非投資建議。
