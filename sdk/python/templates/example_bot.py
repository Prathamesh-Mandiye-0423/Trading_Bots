"""
Example bot — async, using the trading_bot SDK.

Run:
    pip install httpx
    BOT_ID=test-bot API_URL=http://localhost:8080 API_KEY=dev python example_bot.py
"""

import asyncio
import os

from trading_bot import BotClient, Side
from trading_bot.client import price_sub
from trading_bot.models import Ticker

SYMBOL        = "BTC-USD"
open_order_id = None
client: BotClient


async def on_tick(ticker: Ticker) -> None:
    global open_order_id

    print(f"[{ticker.symbol}] bid={ticker.bid_price} ask={ticker.ask_price} spread={ticker.spread}")

    # Only place one order at a time
    if ticker.ask_price == "0.00000000" or open_order_id is not None:
        return

    # Use price_sub — never float arithmetic
    price = price_sub(ticker.ask_price, "1.00")

    try:
        order = await client.place_order(SYMBOL, Side.BUY, "0.01", price)
        open_order_id = order.id
        print(f"  → placed buy at {price}, order={order.id}")
    except Exception as e:
        print(f"  → order failed: {e}")


async def main():
    global client
    async with BotClient(
        bot_id  = os.getenv("BOT_ID",  "test-bot"),
        api_url = os.getenv("API_URL", "http://localhost:8080"),
        api_key = os.getenv("API_KEY", "dev"),
    ) as client:
        await client.subscribe(SYMBOL, on_tick)
        print(f"Bot {client.bot_id} running on {SYMBOL}... (Ctrl+C to stop)")
        await client.run_forever()


if __name__ == "__main__":
    try:
        asyncio.run(main())
    except KeyboardInterrupt:
        print("\nBot stopped.")