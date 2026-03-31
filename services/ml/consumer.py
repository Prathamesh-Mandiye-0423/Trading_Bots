"""
Trade Consumer
--------------
Subscribes to trade.executed on Redpanda.
Every trade event:
  1. Extracts the price
  2. Updates the symbol's price history
  3. Computes features from current state
  4. Computes a retrospective label (did price go up or down?)
  5. Calls model.train() with the feature vector and label

This is the online learning loop — the model gets smarter with every trade.
"""

import asyncio
import json
import os
import time
from typing import TYPE_CHECKING

from kafka import KafkaConsumer
from kafka.errors import NoBrokersAvailable

if TYPE_CHECKING:
    from model import ModelRegistry


BROKERS      = os.getenv("REDPANDA_BROKERS", "localhost:19092")
TOPIC        = "trade.executed"
GROUP_ID     = "ml-service"
LABEL_HORIZON = 5   # ticks to wait before computing retrospective label


async def start_consumer(registry: "ModelRegistry") -> None:
    """
    Start consuming trade events in a background thread.
    Runs forever — reconnects on broker failure.
    """
    loop = asyncio.get_event_loop()
    await loop.run_in_executor(None, _consume_loop, registry)


def _consume_loop(registry: "ModelRegistry") -> None:
    """
    Blocking consume loop — runs in a thread pool executor
    so it doesn't block FastAPI's async event loop.
    """
    print(f"[consumer] connecting to {BROKERS}...")

    while True:
        try:
            consumer = KafkaConsumer(
                TOPIC,
                bootstrap_servers = BROKERS,
                group_id          = GROUP_ID,
                auto_offset_reset = "latest",   # start from now, not beginning
                value_deserializer= lambda v: json.loads(v.decode("utf-8")),
                consumer_timeout_ms = 1000,     # don't block forever between polls
            )
            print(f"[consumer] connected — listening on {TOPIC}")

            for message in consumer:
                try:
                    _process_trade(registry, message.value)
                except Exception as e:
                    print(f"[consumer] error processing message: {e}")

        except NoBrokersAvailable:
            print(f"[consumer] broker not available — retrying in 5s...")
            time.sleep(5)
        except Exception as e:
            print(f"[consumer] unexpected error: {e} — retrying in 5s...")
            time.sleep(5)


def _process_trade(registry: "ModelRegistry", trade: dict) -> None:
    """
    Process one trade event:
      1. Update price history for the symbol
      2. Compute features
      3. Compute retrospective label
      4. Train the model if we have both features and a label
    """
    symbol = trade.get("symbol")
    price  = trade.get("price")

    if not symbol or not price:
        return

    try:
        price_float = float(price)
    except (ValueError, TypeError):
        return

    state = registry.get_state(symbol)
    model = registry.get_model(symbol)

    # Step 1 — add new price to rolling window
    state.add_price(price_float)

    # Step 2 — compute features from current state
    features = state.compute_features()
    if features is None:
        # Not enough history yet — just collecting data
        return

    # Step 3 — compute retrospective label
    # This tells the model whether the signal N ticks ago was correct
    label = state.compute_label(horizon=LABEL_HORIZON)
    if label is None:
        return

    # Step 4 — train the model on this sample
    model.train(features, label)

    if model.n_samples % 10 == 0:
        print(
            f"[{symbol}] trained sample={model.n_samples} "
            f"accuracy={model.rolling_accuracy():.2%} "
            f"label={label}"
        )