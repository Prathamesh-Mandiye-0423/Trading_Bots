"""
ML Microservice — FastAPI
--------------------------
Exposes two endpoints:

  GET /api/v1/ml/:symbol/signal
      → current BUY/SELL/HOLD signal with confidence score
      → used by bots on every tick to get a model-driven signal

  GET /api/v1/ml/:symbol/stats
      → model health, accuracy, weight drift, signal distribution
      → used by the dashboard to observe how the model evolves

  GET /api/v1/ml/health
      → service health check

Run:
    pip install -r requirements.txt
    uvicorn main:app --host 0.0.0.0 --port 8001 --reload
"""

import asyncio
import os
import time
from contextlib import asynccontextmanager

import httpx
from dotenv import load_dotenv
from fastapi import FastAPI, HTTPException
from fastapi.middleware.cors import CORSMiddleware

from consumer import start_consumer
from model import ModelRegistry
from poller import start_poller

load_dotenv()

MARKET_ENGINE_URL = os.getenv("MARKET_ENGINE_URL", "http://localhost:8080")

# Global registry — shared across all requests
registry = ModelRegistry()


# ---- Startup / shutdown ----

@asynccontextmanager
async def lifespan(app: FastAPI):
    """Start background tasks when the service starts."""
    print("[ml-service] starting...")

    # Start Redpanda consumer in background
    consumer_task = asyncio.create_task(
        start_consumer(registry),
        name="trade-consumer"
    )

    # Start order book poller in background
    poller_task = asyncio.create_task(
        start_poller(registry),
        name="ob-poller"
    )

    print("[ml-service] ready")
    yield

    # Shutdown
    consumer_task.cancel()
    poller_task.cancel()
    await asyncio.gather(consumer_task, poller_task, return_exceptions=True)
    print("[ml-service] stopped")


# ---- App ----

app = FastAPI(
    title       = "AlgoTrade ML Service",
    description = "Online learning model for trading signal generation",
    version     = "0.1.0",
    lifespan    = lifespan,
)

app.add_middleware(
    CORSMiddleware,
    allow_origins = ["*"],
    allow_methods = ["*"],
    allow_headers = ["*"],
)


# ---- Endpoints ----

@app.get("/api/v1/ml/health")
async def health():
    return {
        "status":   "ok",
        "symbols":  registry.all_symbols(),
        "timestamp": time.time(),
    }


@app.get("/api/v1/ml/{symbol}/signal")
async def get_signal(symbol: str):
    """
    Get the current trading signal for a symbol.

    Bots call this on every tick instead of (or in addition to)
    computing their own signal. The model blends MA crossover,
    z-score, momentum, and order book imbalance into one prediction.

    Response:
        signal      — BUY / SELL / HOLD
        confidence  — model's confidence in this signal (0.0 - 1.0)
        probabilities — per-class probabilities
        features_used — the input features that drove this prediction
        model_samples — how many trades the model has trained on
        warming_up   — true if model hasn't seen enough data yet
    """
    symbol  = symbol.upper()
    state   = registry.get_state(symbol)
    model   = registry.get_model(symbol)

    # Fetch current order book to get fresh imbalance
    try:
        async with httpx.AsyncClient(timeout=2.0) as client:
            resp = await client.get(
                f"{MARKET_ENGINE_URL}/api/v1/markets/{symbol}/orderbook",
                params={"depth": 5}
            )
            book    = resp.json()
            bid_qty = sum(float(b["quantity"]) for b in book.get("bids", []))
            ask_qty = sum(float(a["quantity"]) for a in book.get("asks", []))
            state.update_orderbook(bid_qty, ask_qty)
    except Exception:
        pass   # use cached values if market engine is slow

    features = state.compute_features()

    if features is None:
        return {
            "symbol":      symbol,
            "warming_up":  True,
            "signal":      "HOLD",
            "confidence":  0.0,
            "message":     f"collecting data — {len(state.prices)}/{state.MIN_SAMPLES} samples",
            "model_samples": 0,
        }

    prediction = model.predict(features)

    if prediction is None:
        return {
            "symbol":      symbol,
            "warming_up":  True,
            "signal":      "HOLD",
            "confidence":  0.0,
            "message":     "model training — waiting for labelled samples",
            "model_samples": model.n_samples,
        }

    return {
        "symbol":      symbol,
        "warming_up":  False,
        **prediction,
    }


@app.get("/api/v1/ml/{symbol}/stats")
async def get_stats(symbol: str):
    """
    Get detailed model stats for a symbol.

    This is the endpoint to watch to see the model evolving over time:
      - n_samples grows as more trades happen
      - rolling_accuracy shows whether the model is improving
      - weight_snapshots shows how feature importance shifts
      - signal_distribution shows whether the model is balanced

    A healthy model:
      - Has balanced signal_distribution (not always predicting BUY)
      - Has rolling_accuracy > 0.5 (better than random)
      - Has weight_snapshots that change gradually (stable learning)
    """
    symbol = symbol.upper()

    if symbol not in registry.all_symbols():
        raise HTTPException(
            status_code = 404,
            detail      = f"No model found for {symbol}. "
                          f"Known symbols: {registry.all_symbols()}"
        )

    model = registry.get_model(symbol)
    stats = model.stats()

    # Add feature importance if model is fitted
    if model.is_fitted and hasattr(model.clf, "coef_"):
        feature_names = ["ma_cross", "z_score", "momentum", "ob_imbalance"]
        coef          = model.clf.coef_

        # coef_ shape is (n_classes, n_features)
        # Average absolute weight across classes = feature importance
        importance = {}
        for i, name in enumerate(feature_names):
            importance[name] = round(float(abs(coef[:, i]).mean()), 6)
        stats["feature_importance"] = importance

    return stats


@app.get("/api/v1/ml/all/stats")
async def get_all_stats():
    """Get stats for all symbols at once.""" 
    return {
        "models": registry.all_stats(),
        "total_symbols": len(registry.all_symbols()),
    }