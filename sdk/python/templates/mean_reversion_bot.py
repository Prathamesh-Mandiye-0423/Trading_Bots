"""
Template Bot: Mean Reversion Strategy
=======================================
STRATEGY
--------
  Compute z-score: how many std devs is current price from rolling mean?
  BUY  when z < -threshold  (price unusually low  — expect recovery)
  SELL when z >  threshold  (price unusually high — expect pullback)

RUN
---
    pip install httpx
    BOT_ID=mr-bot API_URL=http://localhost:8080 API_KEY=dev python mean_reversion_bot.py
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
WINDOW            = 30
Z_THRESHOLD       = Decimal("1.5")
POSITION_SIZE     = "0.05000000"
STOP_LOSS_PCT     = Decimal("0.03")
PRECISION         = Decimal("0.00000001")

# ---- State ----
price_history: deque[Decimal]    = deque(maxlen=WINDOW)
position                         = "FLAT"
entry_price: Optional[Decimal]   = None
client: BotClient


def z_score(prices: deque[Decimal], current: Decimal) -> Optional[Decimal]:
    """
    z = (price - mean) / std_dev
    z > 0 → above average (overbought)
    z < 0 → below average (oversold)
    """
    if len(prices) < WINDOW:
        return None
    floats = [float(p) for p in prices]
    mean   = statistics.mean(floats)
    std    = statistics.stdev(floats)
    if std == 0:
        return None
    return Decimal(str((float(current) - mean) / std)).quantize(
        Decimal("0.00001"), ROUND_HALF_UP
    )


async def on_tick(ticker: Ticker) -> None:
    global position, entry_price

    bid = Decimal(ticker.bid_price)
    ask = Decimal(ticker.ask_price)
    if bid <= 0 or ask <= 0:
        return

    mid = ((bid + ask) / 2).quantize(PRECISION, ROUND_HALF_UP)
    price_history.append(mid)

    z = z_score(price_history, mid)
    if z is None:
        print(f"[{SYMBOL}] collecting data... {len(price_history)}/{WINDOW}")
        return

    print(f"[{SYMBOL}] price={mid} z={z} position={position}")

    # Stop loss
    if position != "FLAT" and entry_price is not None:
        if position == "LONG" and mid < entry_price * (1 - STOP_LOSS_PCT):
            print(f"  → STOP LOSS — exiting LONG at {mid}")
            try:
                await client.place_order(SYMBOL, Side.SELL, POSITION_SIZE,
                                         str(bid.quantize(PRECISION, ROUND_HALF_UP)))
            except Exception as e:
                print(f"  → exit failed: {e}")
            position    = "FLAT"
            entry_price = None
            return

    if position != "FLAT":
        return

    if z < -Z_THRESHOLD:
        price_str = str(ask.quantize(PRECISION, ROUND_HALF_UP))
        print(f"  → OVERSOLD (z={z}) — BUY at {price_str}")
        try:
            order       = await client.place_order(SYMBOL, Side.BUY, POSITION_SIZE, price_str)
            position    = "LONG"
            entry_price = Decimal(price_str)
            print(f"     order={order.id}")
        except Exception as e:
            print(f"     failed: {e}")

    elif z > Z_THRESHOLD:
        price_str = str(bid.quantize(PRECISION, ROUND_HALF_UP))
        print(f"  → OVERBOUGHT (z={z}) — SELL at {price_str}")
        try:
            order       = await client.place_order(SYMBOL, Side.SELL, POSITION_SIZE, price_str)
            position    = "SHORT"
            entry_price = Decimal(price_str)
            print(f"     order={order.id}")
        except Exception as e:
            print(f"     failed: {e}")


async def main():
    global client
    async with BotClient(
        bot_id  = os.getenv("BOT_ID",  "mr-bot"),
        api_url = os.getenv("API_URL", "http://localhost:8080"),
        api_key = os.getenv("API_KEY", "dev"),
    ) as client:
        print(f"Mean Reversion Bot | {SYMBOL} | window={WINDOW} | z_threshold={Z_THRESHOLD}")
        await client.subscribe(SYMBOL, on_tick)
        await client.run_forever()


if __name__ == "__main__":
    try:
        asyncio.run(main())
    except KeyboardInterrupt:
        print("\nBot stopped.")