"""
Feature Engineering
--------------------
Computes the feature vector the ML model uses for prediction and training.
 
Each feature captures a different aspect of market structure:
 
  ma_cross        — trend direction (short MA minus long MA, normalised)
  z_score         — mean reversion signal (how far price is from its average)
  momentum        — rate of price change over recent ticks
  ob_imbalance    — order book pressure (bid depth vs ask depth)
 
All features are normalised to roughly [-1, 1] so the SGD optimiser
treats them equally regardless of their raw scale.
"""

import statistics
from collections import deque
from dataclasses import dataclass, field
from typing import Optional

@dataclass
class Features:
    ma_cross: float # (short_ma - long_ma) / long_ma
    z_score: float  # (price - mean) / std
    momentum: float  # (price_now - price_N_ago) / price_N_ago
    ob_imbalance: float  # (bid_qty - ask_qty) / (bid_qty + ask_qty)

    def to_list(self)->list[float]:
        "Return features as a flat list for sklearn"
        return [self.ma_cross, self.z_score, self.momentum, self.ob_imbalance]

    def to_dict(self)->dict:
        return {
            "ma_cross": round(self.ma_cross,6),
            "z_score": round(self.z_score,6),
            "momentum": round(self.momentum,6),
            "ob_imbalance": round(self.ob_imbalance,6)
        }

#Per symbol state
@dataclass
class SymbolState:
    """
    Maintains rolling price history and order book snapshot for one symbol.
    Updated on every trade event and orderbook poll.
    """
    symbol: str
    prices: deque=field(default_factory=lambda: deque(maxlen=50))
    bid_qty: float=0.0 
    ask_qty: float=0.0

    SHORT_WINDOW= 5 
    LONG_WINDOW= 20
    MOM_WINDOW = 10
    MIN_SAMPLES = 21 # need at least LONG_WINDOW + 1 prices before we compute
    def add_price(self,price: float)->None:
        self.prices.append(price)

    def update_orderbook(self,bid_qty: float, ask_qty: float)->None:
        self.bid_qty = bid_qty
        self.ask_qty = ask_qty

    def compute_features(self)->Optional[Features]:
        """
        Compute the full feature vector from current state.
        Returns None if we don't have enough price history yet.
        """
        if len(self.prices) < self.MIN_SAMPLES:
            return None
 
        prices = list(self.prices)
        current = prices[-1]
 
        # ---- MA crossover ----
        # Positive = short MA above long MA = uptrend
        # Negative = short MA below long MA = downtrend
        short_ma = statistics.mean(prices[-self.SHORT_WINDOW:])
        long_ma  = statistics.mean(prices[-self.LONG_WINDOW:])
        ma_cross = (short_ma - long_ma) / long_ma if long_ma != 0 else 0.0
 
        # ---- Z-score ----
        # How many std devs is current price from its rolling mean?
        # Positive = above average (overbought signal)
        # Negative = below average (oversold signal)
        mean = statistics.mean(prices[-self.LONG_WINDOW:])
        std  = statistics.stdev(prices[-self.LONG_WINDOW:])
        z_score = (current - mean) / std if std != 0 else 0.0
 
        # ---- Momentum ----
        # Rate of change over MOM_WINDOW ticks
        # Positive = price accelerating upward
        # Negative = price accelerating downward
        past = prices[-self.MOM_WINDOW]
        momentum = (current - past) / past if past != 0 else 0.0
 
        # ---- Order book imbalance ----
        # Positive = more bid pressure (buyers stronger)
        # Negative = more ask pressure (sellers stronger)
        # Range is always [-1, 1]
        total = self.bid_qty + self.ask_qty
        ob_imbalance = (self.bid_qty - self.ask_qty) / total if total > 0 else 0.0
 
        return Features(
            ma_cross     = float(ma_cross),
            z_score      = float(max(-3.0, min(3.0, z_score))),   # clip to [-3, 3]
            momentum     = float(momentum),
            ob_imbalance = float(ob_imbalance),
        )

    def compute_label(self, horizon: int = 5)->Optional[int]:
        """
        Compute the training label by looking at future price movement.
 
        We can only label a data point AFTER `horizon` more prices arrive.
        The label answers: did price go up or down from this point?
 
            1  = price went UP   by more than threshold → BUY was correct
           -1  = price went DOWN by more than threshold → SELL was correct
            0  = price moved less than threshold        → HOLD
 
        This is called a RETROSPECTIVE label — we use hindsight to train.
        In a live system this means there's always a `horizon`-tick delay
        before a data point can be used for training, which is realistic.
        """
        THRESHOLD = 0.0005   # 0.05% move = meaningful signal
 
        if len(self.prices) < horizon + self.MIN_SAMPLES:
            return None
 
        prices  = list(self.prices)
        past    = prices[-(horizon + 1)]
        current = prices[-1]
 
        if past == 0:
            return None
 
        ret = (current - past) / past
 
        if ret >  THRESHOLD:
            return 1    # price went up → BUY signal was correct
        if ret < -THRESHOLD:
            return -1   # price went down → SELL signal was correct
        return 0        # noise → HOLD
 