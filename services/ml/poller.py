"""
Order Book Poller
-----------------
Polls the market engine's orderbook endpoint every 500ms to keep
the order book imbalance feature up to date.

Order book imbalance = (bid_qty - ask_qty) / (bid_qty + ask_qty)

This tells the model whether buyers or sellers are currently stronger.
A positive imbalance means more buying pressure — bullish signal.
A negative imbalance means more selling pressure — bearish signal.
"""

import asyncio
import os
from typing import TYPE_CHECKING

import httpx

if TYPE_CHECKING:
    from model import ModelRegistry


MARKET_ENGINE_URL = os.getenv("MARKET_ENGINE_URL", "http://localhost:8080")
POLL_INTERVAL     = 0.5   # seconds


async def start_poller(registry: "ModelRegistry") -> None:
    """Poll the order book for all known symbols every 500ms."""
    async with httpx.AsyncClient(timeout=3.0) as client:
        while True:
            try:
                # Discover symbols
                resp    = await client.get(f"{MARKET_ENGINE_URL}/api/v1/markets")
                symbols = resp.json().get("symbols", [])

                # Update order book state for each symbol
                for symbol in symbols:
                    try:
                        ob_resp = await client.get(
                            f"{MARKET_ENGINE_URL}/api/v1/markets/{symbol}/orderbook",
                            params={"depth": 5},
                        )
                        book = ob_resp.json()

                        bid_qty = sum(float(b["quantity"]) for b in book.get("bids", []))
                        ask_qty = sum(float(a["quantity"]) for a in book.get("asks", []))

                        state = registry.get_state(symbol)
                        state.update_orderbook(bid_qty, ask_qty)

                    except Exception as e:
                        pass   # market engine might not have this symbol yet

            except Exception as e:
                pass   # market engine might not be running yet

            await asyncio.sleep(POLL_INTERVAL)