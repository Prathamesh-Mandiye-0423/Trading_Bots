# AlgoTrade Platform

A high-performance algorithmic trading platform where AI bots analyze markets and execute trades autonomously. Built in Go with a fixed-point matching engine, Redpanda event streaming, and a multi-language bot SDK.

---

## Objective

AlgoTrade is a simulated financial exchange designed for algorithmic trading research. The platform handles all exchange infrastructure — order matching, price feeds, risk management, and event streaming — so bot authors can focus entirely on strategy.

The end goal is a production-grade system where:
- Users upload trading bots as ZIP files in Python, Go, or JavaScript
- Each bot runs in an isolated Docker sandbox with resource limits
- A real-time matching engine matches orders using price-time priority
- A supervisor agent monitors every bot for risk violations and kills rogue bots automatically
- An LLM-powered risk officer (via MCP) makes intelligent, context-aware intervention decisions

---

## Architecture

```
┌─────────────────────────────────────────────────┐
│              Bot (ZIP upload)                    │
│     Python / Go / Node.js + ML strategy         │
└─────────────────────┬───────────────────────────┘
                      │ REST API
┌─────────────────────▼───────────────────────────┐
│              Market Engine (Go)                  │
│   Limit order book · Matching engine · Feeds     │
└──────────┬──────────────────────────┬────────────┘
           │ publishes events         │ price feed
┌──────────▼──────────┐   ┌──────────▼────────────┐
│  Redpanda           │   │  Supervisor            │
│  trade.executed     │   │  Shadow ledger         │
│  order.updated      │   │  Risk rules            │
│  market.ticker      │   │  Kill switch           │
└─────────────────────┘   └────────────────────────┘
```

---

## What Is Built

### Market Engine
- **Limit order book** — price-time priority matching (FIFO within each price level)
- **Fixed-point arithmetic** — custom `decimal.Decimal` type backed by `int64` — zero floating-point rounding errors
- **Order types** — limit and market orders with partial fill support
- **Self-match prevention** — a bot cannot trade against itself
- **REST API** — submit orders, cancel orders, fetch order book depth snapshots
- **WebSocket feed** — live order book updates streamed to subscribers at 500ms intervals
- **Multi-symbol registry** — BTC-USD, ETH-USD, SOL-USD running as independent order books

### Event Bus
- **Redpanda** (Kafka-compatible) with three topics: `trade.executed`, `order.updated`, `market.ticker`
- **Publisher** — every trade the matching engine produces is immediately published
- **Consumer** — reusable base for all downstream services with offset tracking and consumer groups

### Supervisor (Risk Engine)
- **Shadow ledger** — per-bot state tracking using a sliding window order counter
- **Rate limit rule** — suspends bots submitting more than 10 orders/second
- **Burst rule** — suspends bots exceeding 30 orders in any 5-second window
- **Kill action** — cancels all open orders then stops the container
- **Pluggable rules** — add any `RuleFunc` to the chain without touching core logic

### Bot SDK (3 languages)
- **Python** — async `BotClient` using `httpx` and `asyncio`, `decimal.Decimal` price helpers
- **Node.js** — native `fetch` (Node 18+), `BigInt` price arithmetic, no external dependencies
- **Go** — typed request structs, `big.Rat` price helpers, `context`-aware subscription loop

### Strategy Templates
- `moving_average_bot.py` — dual MA crossover with stop loss, take profit, and ML swap point
- `mean_reversion_bot.py` — z-score based reversion with rolling mean and standard deviation
- `momentum_bot.py` — rate of change with volatility filter and time-based exit

---

## What Is Left to Build

| Component | Description | Priority |
|-----------|-------------|----------|
| **Sandbox manager** | Accept ZIP uploads, detect language, build Docker image, spawn isolated container with injected credentials and resource limits | High |
| **Dockerfile templates** | Per-language base images (Python 3.11-slim, Node 18-alpine, Go 1.23-alpine) with SDK pre-installed | High |
| **Database layer** | PostgreSQL for orders, trades, and bot metadata. TimescaleDB for OHLCV price history | High |
| **MCP server** | Expose supervisor tools (`get_bot_status`, `pause_trading`, `emergency_stop`) to an LLM agent acting as chief risk officer | Medium |
| **Additional risk rules** | Max drawdown, fat-finger protection, position size limits | Medium |
| **Dashboard** | React frontend — live order book depth chart, per-bot P&L, trade history, suspension log | Medium |
| **Authentication** | JWT-based bot API keys injected at container start time | Medium |
| **Backtesting engine** | Replay historical trade events against a strategy without live execution | Low |

---

## Tech Stack

| Layer | Technology | Why |
|-------|-----------|-----|
| Backend | Go 1.23 | Low-latency, goroutine concurrency, short GC pauses |
| HTTP framework | Fiber v3 | Fastest Go HTTP router, WebSocket support |
| Event streaming | Redpanda | Kafka-compatible, no ZooKeeper, simple dev setup |
| Kafka client | franz-go | Pure Go, fastest Kafka client available |
| Logging | zerolog | Zero-allocation structured logging |
| Bot isolation | Docker | Network isolation, resource quotas, read-only filesystem |
| Python SDK | httpx + asyncio | Async I/O, industry standard for market data |
| Node.js SDK | Native fetch | No dependencies, Node 18+ built-in |

---

## Prerequisites

- Go 1.23+
- Docker + Docker Compose
- Node.js 18+ (for Node.js bots)
- Python 3.10+ (for Python bots)

---

## Setup and Run

### 1. Clone the repository

```bash
git clone https://github.com/Prathamesh-Mandiye-0423/trading-platform.git
cd trading-platform
```

### 2. Install Go dependencies

```bash
go mod tidy
```

### 3. Start Redpanda

```bash
docker compose up -d

# Wait for healthy status
docker compose ps
```

### 4. Create Redpanda topics

```bash
docker exec redpanda rpk topic create trade.executed --partitions 6 --replicas 1
docker exec redpanda rpk topic create order.updated  --partitions 6 --replicas 1
docker exec redpanda rpk topic create market.ticker  --partitions 6 --replicas 1

# Verify
docker exec redpanda rpk topic list
```

### 5. Configure environment

```bash
cp .env.example .env
# Edit .env if needed — defaults work for local development
```

`.env` defaults:
```
SERVER_PORT=8080
ENV=development
LOG_LEVEL=debug
```

### 6. Start the market engine

```bash
go run cmd/market-engine/main.go
```

You should see:
```
INF event publisher connected brokers=["localhost:19092"]
INF market registered symbol=BTC-USD
INF market registered symbol=ETH-USD
INF market registered symbol=SOL-USD
INF market engine listening port=8080
```

### 7. Verify with health check

```bash
curl http://localhost:8080/health
# {"status":"ok"}

curl http://localhost:8080/api/v1/markets
# {"symbols":["BTC-USD","ETH-USD","SOL-USD"]}
```

---

## Running Tests

```bash
# All tests
go test ./internal/... -v

# Specific packages
go test ./internal/matching/...  -v   # matching engine
go test ./internal/decimal/...   -v   # fixed-point arithmetic
go test ./internal/supervisor/... -v  # risk rules
```

---

## Bot Workflow

### How a bot interacts with the platform

```
1. Bot starts → reads BOT_ID and API_KEY from environment
2. Bot calls GET /api/v1/markets → discovers available symbols
3. Bot subscribes to a symbol → SDK polls order book every 500ms
4. on_tick() fires → bot runs its strategy logic
5. Strategy generates a signal → bot calls place_order()
6. Market engine matches the order → trade event published to Redpanda
7. Supervisor reads trade event → checks risk rules
8. If rules pass → trade confirmed, bot continues
9. If rules fail → all orders cancelled, bot suspended
```

### Running the Python example bot

```bash
cd sdk/python
pip install -r requirements.txt

BOT_ID=my-bot API_URL=http://localhost:8080 API_KEY=dev \
  python trading_bot/example_bot.py
```

### Running a strategy template

```bash
cd sdk/python
BOT_ID=ma-bot API_URL=http://localhost:8080 API_KEY=dev \
  python templates/moving_average_bot.py
```

### Running the Node.js example bot

```bash
cd sdk/node
BOT_ID=node-bot API_URL=http://localhost:8080 API_KEY=dev \
  node src/example_bot.js
```

### Placing orders manually (for testing)

```bash
# Place a resting sell
curl -X POST http://localhost:8080/api/v1/orders \
  -H "Content-Type: application/json" \
  -d '{
    "bot_id":   "bot-1",
    "symbol":   "BTC-USD",
    "side":     "SELL",
    "type":     "LIMIT",
    "price":    "50000.00",
    "quantity": "1.0"
  }'

# Place a matching buy — triggers a trade
curl -X POST http://localhost:8080/api/v1/orders \
  -H "Content-Type: application/json" \
  -d '{
    "bot_id":   "bot-2",
    "symbol":   "BTC-USD",
    "side":     "BUY",
    "type":     "LIMIT",
    "price":    "50000.00",
    "quantity": "1.0"
  }'

# Check the order book
curl "http://localhost:8080/api/v1/markets/BTC-USD/orderbook?depth=5"
```

### Watching live trade events

```bash
# Open a second terminal and watch Redpanda
docker exec redpanda rpk topic consume trade.executed
```

Every matched trade appears here as JSON in real time.

---

## API Reference

### Markets

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/v1/markets` | List all trading symbols |
| `GET` | `/api/v1/markets/:symbol/orderbook?depth=N` | Order book snapshot |

### Orders

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v1/orders` | Place a limit or market order |
| `DELETE` | `/api/v1/orders/:symbol/:id` | Cancel an open order |

### WebSocket

| Path | Description |
|------|-------------|
| `ws://localhost:8080/ws/:symbol` | Live order book (500ms push) |

### Order request body

```json
{
  "bot_id":   "my-bot",
  "symbol":   "BTC-USD",
  "side":     "BUY",
  "type":     "LIMIT",
  "price":    "50000.00",
  "quantity": "0.50000000"
}
```

> **Always send `price` and `quantity` as strings.** JSON floats are IEEE 754 and introduce rounding errors. The platform uses fixed-point arithmetic internally — string representation preserves this precision end-to-end.

---

## Project Structure

```
trading-platform/
├── cmd/market-engine/        Entry point
├── internal/
│   ├── api/                  REST + WebSocket handlers
│   ├── decimal/              Fixed-point arithmetic (no float64)
│   ├── events/               Redpanda publisher + consumer
│   ├── matching/             Matching engine + multi-symbol registry
│   ├── models/               Order, Trade, OrderBook domain types
│   ├── orderbook/            Limit order book data structure
│   └── supervisor/           Risk engine + shadow ledger + kill switch
├── sdk/
│   ├── python/               Async Python SDK + strategy templates
│   ├── node/                 Node.js SDK (native fetch)
│   └── go/                   Go SDK + example bot
├── docker-compose.yml        Redpanda + Redpanda Console
├── .env                      Local configuration
└── go.mod
```

---

## Redpanda Console

A web UI for inspecting topics, messages, and consumer groups is available at:

```
http://localhost:8090
```

---

## Contributing

This platform is under active development. The most impactful next contributions are the sandbox manager (bot isolation) and the database persistence layer. See the roadmap table above for the full list of open work.

---

## License

MIT