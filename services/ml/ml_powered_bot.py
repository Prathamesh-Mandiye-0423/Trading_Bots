"""
ML-Powered Bot
--------------
Instead of computing its own signal, this bot queries the ML microservice
on every tick and trades based on the model's prediction.

This creates the feedback loop:
  model predicts → bot trades → trade updates model → model predicts again

You can observe this loop by watching GET /api/v1/ml/BTC-USD/stats
while this bot is running — you'll see n_samples grow and accuracy change.

RUN
---
    # Start the ML service first
    cd services/ml && uvicorn main:app --port 8001

    # Then run this bot
    pip install httpx
    BOT_ID=ml-bot API_URL=http://localhost:8080 API_KEY=dev \
      ML_URL=http://localhost:8001 python ml_powered_bot.py
"""

import asyncio
import os
from decimal import Decimal, ROUND_HALF_UP
from typing import Optional

import httpx

from trading_bot import BotClient, Side
from trading_bot.models import Ticker

# ---- Config ----
SYMBOL            = "BTC-USD"
ML_URL            = os.getenv("ML_URL", "http://localhost:8001")
POSITION_SIZE     = "0.05000000"
MIN_CONFIDENCE    = 0.60      # only trade if model is at least 60% confident
PRECISION         = Decimal("0.00000001")

# ---- State ----
position:       str              = "FLAT"
open_order_id:  Optional[str]   = None
entry_price:    Optional[Decimal]= None
client:         BotClient
http:           httpx.AsyncClient


async def get_signal(symbol: str) -> Optional[dict]:
    """Query the ML microservice for a trading signal."""
    try:
        resp = await http.get(f"{ML_URL}/api/v1/ml/{symbol}/signal", timeout=2.0)
        if resp.status_code == 200:
            return resp.json()
    except Exception as e:
        print(f"  [ml] signal fetch failed: {e}")
    return None


async def on_tick(ticker: Ticker) -> None:
    global position, open_order_id, entry_price

    bid = Decimal(ticker.bid_price)
    ask = Decimal(ticker.ask_price)
    if bid <= 0 or ask <= 0:
        return

    # Query the ML model for a signal
    signal_data = await get_signal(SYMBOL)

    if signal_data is None:
        return

    if signal_data.get("warming_up"):
        print(f"[{SYMBOL}] model warming up — {signal_data.get('message', '')}")
        return

    signal     = signal_data.get("signal", "HOLD")
    confidence = signal_data.get("confidence", 0.0)
    n_samples  = signal_data.get("model_samples", 0)

    print(
        f"[{SYMBOL}] bid={bid} ask={ask} "
        f"signal={signal} confidence={confidence:.2%} "
        f"samples={n_samples} position={position}"
    )

    # Skip low-confidence signals
    if confidence < MIN_CONFIDENCE:
        print(f"  → low confidence ({confidence:.2%} < {MIN_CONFIDENCE:.0%}) — holding")
        return

    # Already in a position — wait for exit signal
    if position != "FLAT":
        # Exit on opposite signal
        if position == "LONG" and signal == "SELL":
            print(f"  → EXIT LONG — model flipped to SELL")
            await _exit_position(bid)
        elif position == "SHORT" and signal == "BUY":
            print(f"  → EXIT SHORT — model flipped to BUY")
            await _exit_position(ask)
        return

    # Enter position based on model signal
    match signal:
        case "BUY":
            price_str = str(ask.quantize(PRECISION, ROUND_HALF_UP))
            print(f"  → MODEL BUY at {price_str} (confidence={confidence:.2%})")
            try:
                order         = await client.place_order(SYMBOL, Side.BUY, POSITION_SIZE, price_str)
                open_order_id = order.id
                entry_price   = Decimal(price_str)
                position      = "LONG"
                print(f"     order={order.id}")
            except Exception as e:
                print(f"     failed: {e}")

        case "SELL":
            price_str = str(bid.quantize(PRECISION, ROUND_HALF_UP))
            print(f"  → MODEL SELL at {price_str} (confidence={confidence:.2%})")
            try:
                order         = await client.place_order(SYMBOL, Side.SELL, POSITION_SIZE, price_str)
                open_order_id = order.id
                entry_price   = Decimal(price_str)
                position      = "SHORT"
                print(f"     order={order.id}")
            except Exception as e:
                print(f"     failed: {e}")

        case "HOLD":
            print(f"  → HOLD (model has no conviction)")


async def _exit_position(exit_price: Decimal) -> None:
    global position, open_order_id, entry_price

    if open_order_id:
        try:
            await client.cancel_order(SYMBOL, open_order_id)
        except Exception:
            pass
        open_order_id = None

    exit_side = Side.SELL if position == "LONG" else Side.BUY
    price_str = str(exit_price.quantize(PRECISION, ROUND_HALF_UP))
    try:
        await client.place_order(SYMBOL, exit_side, POSITION_SIZE, price_str)
        print(f"  → exited {position} at {price_str}")
    except Exception as e:
        print(f"  → exit failed: {e}")

    position    = "FLAT"
    entry_price = None


async def main():
    global client, http
    async with BotClient(
        bot_id  = os.getenv("BOT_ID",  "ml-bot"),
        api_url = os.getenv("API_URL", "http://localhost:8080"),
        api_key = os.getenv("API_KEY", "dev"),
    ) as client:
        async with httpx.AsyncClient() as http:
            print(f"ML-Powered Bot | {SYMBOL} | min_confidence={MIN_CONFIDENCE:.0%}")
            print(f"Querying ML service at {ML_URL}")
            print()
            await client.subscribe(SYMBOL, on_tick)
            await client.run_forever()


if __name__ == "__main__":
    try:
        asyncio.run(main())
    except KeyboardInterrupt:
        print("\nBot stopped.")