"""
Online ML Model
----------------
Uses scikit-learn's SGDClassifier with partial_fit() for true online learning.
 
One model instance per trading symbol. Each model:
  - Accepts one new training sample at a time (no full retrain)
  - Predicts BUY / SELL / HOLD with a confidence score
  - Tracks rolling accuracy over the last 100 predictions
  - Records its weight history so you can see how it changes over time
 
WHY SGDClassifier?
  Most ML models require all training data in memory to retrain.
  SGDClassifier.partial_fit() updates weights from a single sample
  using stochastic gradient descent — the same algorithm that trains
  neural networks, just applied to logistic regression here.
 
  Weight update formula (simplified):
    w = w - learning_rate * gradient(loss(w, x, y))
 
  Each call to partial_fit() nudges the weights slightly toward
  correctly classifying the new sample. Over thousands of trades,
  the weights converge to a useful decision boundary.
"""

import time
from collections import deque
from dataclasses import dataclass, field
from typing import Optional
 
import numpy as np
from sklearn.linear_model import SGDClassifier
from sklearn.preprocessing import StandardScaler
 
from features import Features, SymbolState
 
 
CLASSES = np.array([-1, 0, 1])   # SELL, HOLD, BUY
SIGNAL_MAP = {1: "BUY", -1: "SELL", 0: "HOLD"}
 
 
@dataclass
class PredictionRecord:
    """Records a single prediction for accuracy tracking."""
    timestamp:  float
    signal:     str
    confidence: float
    features:   dict
    label:      Optional[int] = None   # filled in retrospectively
 
 
@dataclass
class SymbolModel:
    """
    Online learning model for a single trading symbol.
 
    The scaler normalises features before feeding them to the classifier.
    We use a warm_start=False SGD so each partial_fit() is a true
    incremental update, not a full refit.
    """
    symbol:    str
    n_samples: int = 0
 
    # Sklearn components
    clf: SGDClassifier = field(default_factory=lambda: SGDClassifier(
        loss          = "log_loss",    # gives us probability scores
        learning_rate = "adaptive",    # reduces LR as it gets more confident
        eta0          = 0.01,          # initial learning rate
        penalty       = "l2",          # regularisation — prevents overfitting
        alpha         = 0.0001,        # regularisation strength
        warm_start    = False,
        random_state  = 42,
    ))
    scaler: StandardScaler = field(default_factory=StandardScaler)
 
    # Rolling accuracy tracking
    recent_predictions: deque = field(default_factory=lambda: deque(maxlen=100))
    pending_labels:     deque = field(default_factory=lambda: deque(maxlen=200))
 
    # Weight history — one snapshot per 50 samples
    weight_snapshots: list = field(default_factory=list)
 
    # Signal distribution counters
    signal_counts: dict = field(default_factory=lambda: {"BUY": 0, "SELL": 0, "HOLD": 0})
 
    # Timestamps
    created_at:     float = field(default_factory=time.time)
    last_updated:   float = field(default_factory=time.time)
    is_fitted:      bool  = False
 
    def train(self, features: Features, label: int) -> None:
        """
        Update the model with one new training sample.
        This is the core online learning step.
 
        features — computed from current market state
        label    — 1 (BUY), -1 (SELL), 0 (HOLD) from retrospective labelling
        """
        X = np.array([features.to_list()])
 
        # Update the scaler incrementally
        # StandardScaler tracks running mean and variance
        self.scaler.partial_fit(X)
        X_scaled = self.scaler.transform(X)
 
        # Update the classifier with one sample
        # partial_fit() nudges weights toward correctly classifying this sample
        self.clf.partial_fit(X_scaled, np.array([label]), classes=CLASSES)
 
        self.n_samples   += 1
        self.last_updated = time.time()
        self.is_fitted    = True
 
        # Snapshot weights every 50 samples so we can observe drift
        if self.n_samples % 50 == 0:
            self._snapshot_weights()
 
    def predict(self, features: Features) -> Optional[dict]:
        """
        Generate a signal prediction from current market features.
 
        Returns None if the model hasn't been trained yet.
        Returns a dict with signal, confidence, and feature breakdown.
        """
        if not self.is_fitted:
            return None
 
        X = np.array([features.to_list()])
        X_scaled = self.scaler.transform(X)
 
        # Get class probabilities: [P(SELL), P(HOLD), P(BUY)]
        proba     = self.clf.predict_proba(X_scaled)[0]
        label_idx = int(np.argmax(proba))
        label     = CLASSES[label_idx]
        confidence = float(proba[label_idx])
        signal    = SIGNAL_MAP[label]
 
        self.signal_counts[signal] += 1
 
        record = PredictionRecord(
            timestamp  = time.time(),
            signal     = signal,
            confidence = confidence,
            features   = features.to_dict(),
        )
        self.recent_predictions.append(record)
        self.pending_labels.append(record)
 
        return {
            "signal":       signal,
            "confidence":   round(confidence, 4),
            "probabilities": {
                "BUY":  round(float(proba[2]), 4),
                "HOLD": round(float(proba[1]), 4),
                "SELL": round(float(proba[0]), 4),
            },
            "features_used": features.to_dict(),
            "model_samples": self.n_samples,
        }
 
    def resolve_labels(self, current_price: float, horizon_prices: list[float]) -> None:
        """
        Retrospectively label pending predictions using actual price outcomes.
        Called every time new price data arrives.
 
        This closes the feedback loop — the model sees whether its past
        predictions were actually correct.
        """
        THRESHOLD = 0.0005
 
        resolved = []
        for record in list(self.pending_labels):
            age = time.time() - record.timestamp
            if age < 2.5:   # wait ~5 ticks (500ms each) before resolving
                continue
 
            # Compute actual return since prediction
            ret = (current_price - float(
                list(self.recent_predictions)[0].features.get("ma_cross", 0)
                or current_price
            ))
 
            if abs(ret) > THRESHOLD:
                label = 1 if ret > 0 else -1
            else:
                label = 0
 
            record.label = label
            resolved.append(record)
 
        for r in resolved:
            if r in self.pending_labels:
                self.pending_labels.remove(r)
 
    def rolling_accuracy(self) -> float:
        """
        Compute accuracy over the last 100 resolved predictions.
        Returns 0.0 if no predictions have been resolved yet.
        """
        resolved = [r for r in self.recent_predictions if r.label is not None]
        if not resolved:
            return 0.0
 
        correct = sum(
            1 for r in resolved
            if SIGNAL_MAP.get(r.label, "HOLD") == r.signal
        )
        return round(correct / len(resolved), 4)
 
    def _snapshot_weights(self) -> None:
        """Save current model weights for drift visualisation."""
        if hasattr(self.clf, "coef_"):
            self.weight_snapshots.append({
                "sample":   self.n_samples,
                "weights":  self.clf.coef_.tolist(),
                "timestamp": time.time(),
            })
            # Keep last 20 snapshots
            if len(self.weight_snapshots) > 20:
                self.weight_snapshots.pop(0)
 
    def stats(self) -> dict:
        """Return full model stats for the /stats endpoint."""
        return {
            "symbol":              self.symbol,
            "n_samples":           self.n_samples,
            "is_fitted":           self.is_fitted,
            "rolling_accuracy":    self.rolling_accuracy(),
            "signal_distribution": self.signal_counts,
            "weight_snapshots":    self.weight_snapshots[-5:],  # last 5
            "last_updated":        self.last_updated,
            "created_at":          self.created_at,
            "pending_labels":      len(self.pending_labels),
        }
 
 
class ModelRegistry:
    """
    Manages one SymbolModel per trading symbol.
    Thread-safe via a simple dict — FastAPI runs async so GIL protects us.
    """
 
    def __init__(self):
        self._models:  dict[str, SymbolModel]  = {}
        self._states:  dict[str, SymbolState]  = {}
 
    def get_model(self, symbol: str) -> SymbolModel:
        if symbol not in self._models:
            self._models[symbol] = SymbolModel(symbol=symbol)
        return self._models[symbol]
 
    def get_state(self, symbol: str) -> SymbolState:
        if symbol not in self._states:
            self._states[symbol] = SymbolState(symbol=symbol)
        return self._states[symbol]
 
    def all_symbols(self) -> list[str]:
        return list(self._models.keys())
 
    def all_stats(self) -> list[dict]:
        return [m.stats() for m in self._models.values()]
 