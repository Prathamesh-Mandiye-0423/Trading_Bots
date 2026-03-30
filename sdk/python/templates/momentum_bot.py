"""
Template Bot: Momentum Strategy
=================================
STRATEGY
--------
  Rate of Change (ROC): % price change over N ticks
  BUY  when ROC >  threshold  (strong upward momentum)
  SELL when ROC < -threshold  (strong downward momentum)
  Exit after EXIT_AFTER ticks (time-based exit)

RUN
---
    pip install httpx
    BOT_ID=mom-bot API_URL=http://localhost:8080 API_KEY=dev python momentum_bot.py
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
SYMBOL          = "BTC-USD"
ROC_WINDOW      = 10
ROC_THRESHOLD   = Decimal("0.001")   # 0.1% move = signal
MAX_VOLATILITY  = Decimal("500.0")   # skip when market too choppy
POSITION_SIZE   = "0.05000000"
EXIT_AFTER      = 20                 # ticks before forced exit
PRECISION       = Decimal("0.00000001")

# ---- State ----
price_history: deque[Decimal]  = deque(maxlen=max(ROC_WINDOW, 20))
position                       = "FLAT"
ticks_in_pos                   = 0
client: BotClient


def rate_of_change(prices: deque[Decimal], window: int) -> Optional[Decimal]:
    """ROC = (current - N_ticks_ago) / N_ticks_ago"""
    if len(prices) < window:
        return None
    current  = list(prices)[-1]
    previous = list(prices)[-window]
    if previous == 0:
        return None
    return ((current - previous) / previous).quantize(
        Decimal("0.00001"), ROUND_HALF_UP
    )


def volatility(prices: deque[Decimal], window: int = 10) -> Optional[Decimal]:
    if len(prices) < window:
        return None
    return Decimal(str(statistics.stdev([float(p) for p in list(prices)[-window:]])))


async def on_tick(ticker: Ticker) -> None:
    global position, ticks_in_pos

    bid = Decimal(ticker.bid_price)
    ask = Decimal(ticker.ask_price)
    if bid <= 0 or ask <= 0:
        return

    mid = ((bid + ask) / 2).quantize(PRECISION, ROUND_HALF_UP)
    price_history.append(mid)

    roc = rate_of_change(price_history, ROC_WINDOW)
    vol = volatility(price_history)

    if roc is None:
        print(f"[{SYMBOL}] warming up... {len(price_history)}/{ROC_WINDOW}")
        return

    print(f"[{SYMBOL}] price={mid} roc={roc} vol={vol or 'N/A'} pos={position}")

    # Time-based exit
    if position != "FLAT":
        ticks_in_pos += 1
        if ticks_in_pos >= EXIT_AFTER:
            print(f"  → TIME EXIT after {ticks_in_pos} ticks")
            exit_side  = Side.SELL if position == "LONG" else Side.BUY
            exit_price = str(bid.quantize(PRECISION, ROUND_HALF_UP)) if position == "LONG" \
                    else str(ask.quantize(PRECISION, ROUND_HALF_UP))
            try:
                await client.place_order(SYMBOL, exit_side, POSITION_SIZE, exit_price)
                print(f"  → exited at {exit_price}")
            except Exception as e:
                print(f"  → exit failed: {e}")
            position     = "FLAT"
            ticks_in_pos = 0
        return

    # Skip noisy markets
    if vol is not None and vol > MAX_VOLATILITY:
        print(f"  → HIGH VOLATILITY ({vol}) — skipping")
        return

    if roc > ROC_THRESHOLD:
        price_str = str(ask.quantize(PRECISION, ROUND_HALF_UP))
        print(f"  → UPWARD MOMENTUM ({roc}) — BUY at {price_str}")
        try:
            order        = await client.place_order(SYMBOL, Side.BUY, POSITION_SIZE, price_str)
            position     = "LONG"
            ticks_in_pos = 0
            print(f"     order={order.id}")
        except Exception as e:
            print(f"     failed: {e}")

    elif roc < -ROC_THRESHOLD:
        price_str = str(bid.quantize(PRECISION, ROUND_HALF_UP))
        print(f"  → DOWNWARD MOMENTUM ({roc}) — SELL at {price_str}")
        try:
            order        = await client.place_order(SYMBOL, Side.SELL, POSITION_SIZE, price_str)
            position     = "SHORT"
            ticks_in_pos = 0
            print(f"     order={order.id}")
        except Exception as e:
            print(f"     failed: {e}")


async def main():
    global client
    async with BotClient(
        bot_id  = os.getenv("BOT_ID",  "mom-bot"),
        api_url = os.getenv("API_URL", "http://localhost:8080"),
        api_key = os.getenv("API_KEY", "dev"),
    ) as client:
        print(f"Momentum Bot | {SYMBOL} | roc_window={ROC_WINDOW} | threshold={ROC_THRESHOLD}")
        await client.subscribe(SYMBOL, on_tick)
        await client.run_forever()


if __name__ == "__main__":
    try:
        asyncio.run(main())
    except KeyboardInterrupt:
        print("\nBot stopped.")