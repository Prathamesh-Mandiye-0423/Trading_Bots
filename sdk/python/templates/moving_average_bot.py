"""
Template Bot: Moving Average Crossover Strategy
================================================
STRATEGY
--------
  BUY  when short MA crosses above long MA  (upward momentum)
  SELL when short MA crosses below long MA  (downward momentum)

HOW TO MODIFY
-------------
  1. Tune SHORT_WINDOW / LONG_WINDOW
  2. Replace signal() with your ML model output
  3. Add features to compute_features() for richer signals

RUN
---
    pip install httpx
    BOT_ID=ma-bot API_URL=http://localhost:8080 API_KEY=dev python moving_average_bot.py
"""

import asyncio
import os
import statistics
from collections import deque
from decimal import Decimal, ROUND_HALF_UP
from typing import Optional

from trading_bot import BotClient, Side
from trading_bot.models import Ticker

# ---- Config ----
SYMBOL            = "BTC-USD"
SHORT_WINDOW      = 5
LONG_WINDOW       = 20
MAX_POSITION_SIZE = "0.10000000"
STOP_LOSS_PCT     = Decimal("0.02")   # 2%
TAKE_PROFIT_PCT   = Decimal("0.04")   # 4%
PRECISION         = Decimal("0.00000001")

# ---- State ----
price_history: deque[Decimal]    = deque(maxlen=LONG_WINDOW)
open_order_id: Optional[str]     = None
entry_price:   Optional[Decimal] = None
position                         = "FLAT"
client: BotClient


# ---- Feature engineering ----

def moving_average(prices: deque[Decimal], window: int) -> Optional[Decimal]:
    if len(prices) < window:
        return None
    total = sum(list(prices)[-window:], Decimal("0"))
    return (total / Decimal(window)).quantize(PRECISION, ROUND_HALF_UP)


def compute_features(prices: deque[Decimal]) -> dict:
    short_ma = moving_average(prices, SHORT_WINDOW)
    long_ma  = moving_average(prices, LONG_WINDOW)
    features = {
        "short_ma": short_ma,
        "long_ma":  long_ma,
        "spread":   (short_ma - long_ma) if short_ma and long_ma else None,
    }
    if len(prices) >= 3:
        recent = list(prices)[-3:]
        features["momentum"] = recent[-1] - recent[0]
    if len(prices) >= 5:
        features["volatility"] = Decimal(
            str(statistics.stdev([float(p) for p in list(prices)[-5:]]))
        )
    return features


# ---- Signal — swap this for your ML model ----

def signal(features: dict) -> Optional[str]:
    """
    Replace this with your model:

        pred = model.predict([[float(features["short_ma"]), ...]])[0]
        return "BUY" if pred == 1 else "SELL" if pred == -1 else None
    """
    short_ma = features.get("short_ma")
    long_ma  = features.get("long_ma")
    if short_ma is None or long_ma is None:
        return None
    if short_ma > long_ma:
        return "BUY"
    if short_ma < long_ma:
        return "SELL"
    return None


# ---- Risk management ----

def should_stop_loss(current: Decimal) -> bool:
    if entry_price is None or position == "FLAT":
        return False
    if position == "LONG":
        return current < entry_price * (1 - STOP_LOSS_PCT)
    return current > entry_price * (1 + STOP_LOSS_PCT)


def should_take_profit(current: Decimal) -> bool:
    if entry_price is None or position == "FLAT":
        return False
    if position == "LONG":
        return current > entry_price * (1 + TAKE_PROFIT_PCT)
    return current < entry_price * (1 - TAKE_PROFIT_PCT)


# ---- Tick handler ----

async def on_tick(ticker: Ticker) -> None:
    global open_order_id, entry_price, position

    bid = Decimal(ticker.bid_price)
    ask = Decimal(ticker.ask_price)
    if bid <= 0 or ask <= 0:
        return

    mid = ((bid + ask) / 2).quantize(PRECISION, ROUND_HALF_UP)
    price_history.append(mid)

    features = compute_features(price_history)
    print(
        f"[{ticker.symbol}] mid={mid} "
        f"short={features.get('short_ma') or 'N/A'} "
        f"long={features.get('long_ma') or 'N/A'} "
        f"pos={position} pts={len(price_history)}/{LONG_WINDOW}"
    )

    # Exit checks
    if position != "FLAT":
        if should_stop_loss(mid):
            print(f"  → STOP LOSS at {mid}")
            await _exit_position(mid)
            return
        if should_take_profit(mid):
            print(f"  → TAKE PROFIT at {mid}")
            await _exit_position(mid)
            return
        return

    if open_order_id is not None:
        return

    # Entry signals
    match signal(features):
        case "BUY":
            price_str = str(ask.quantize(PRECISION, ROUND_HALF_UP))
            try:
                order = await client.place_order(SYMBOL, Side.BUY, MAX_POSITION_SIZE, price_str)
                open_order_id = order.id
                entry_price   = Decimal(price_str)
                position      = "LONG"
                print(f"  → BUY at {price_str} order={order.id}")
            except Exception as e:
                print(f"  → BUY failed: {e}")

        case "SELL":
            price_str = str(bid.quantize(PRECISION, ROUND_HALF_UP))
            try:
                order = await client.place_order(SYMBOL, Side.SELL, MAX_POSITION_SIZE, price_str)
                open_order_id = order.id
                entry_price   = Decimal(price_str)
                position      = "SHORT"
                print(f"  → SELL at {price_str} order={order.id}")
            except Exception as e:
                print(f"  → SELL failed: {e}")


async def _exit_position(current: Decimal) -> None:
    global open_order_id, entry_price, position
    if open_order_id:
        try:
            await client.cancel_order(SYMBOL, open_order_id)
        except Exception:
            pass
        open_order_id = None
    exit_side = Side.SELL if position == "LONG" else Side.BUY
    price_str = str(current.quantize(PRECISION, ROUND_HALF_UP))
    try:
        await client.place_order(SYMBOL, exit_side, MAX_POSITION_SIZE, price_str)
        print(f"  → exited {position} at {price_str}")
    except Exception as e:
        print(f"  → exit failed: {e}")
    entry_price = None
    position    = "FLAT"


# ---- Entry point ----

async def main():
    global client
    async with BotClient(
        bot_id  = os.getenv("BOT_ID",  "ma-bot"),
        api_url = os.getenv("API_URL", "http://localhost:8080"),
        api_key = os.getenv("API_KEY", "dev"),
    ) as client:
        print(f"Moving Average Bot | {SYMBOL} | short={SHORT_WINDOW} long={LONG_WINDOW}")
        print(f"Collecting {LONG_WINDOW} ticks before first signal...\n")
        await client.subscribe(SYMBOL, on_tick)
        await client.run_forever()


if __name__ == "__main__":
    try:
        asyncio.run(main())
    except KeyboardInterrupt:
        print("\nBot stopped.")