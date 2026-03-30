from dataclasses import dataclass
from enum import Enum
from typing import Optional
from datetime import datetime


class Side(str, Enum):
    BUY="BUY"
    SELL="SELL"

class OrderType(str, Enum):
    MARKET="MARKET"
    LIMIT="LIMIT"

class OrderStatus(str, Enum):
    # PENDING="PENDING"
    OPEN="OPEN"
    FILLED="FILLED"
    SUSPENDED="SUSPENDED"
    # COMPLETED="COMPLETED"
    CANCELED="CANCELED"
    PARTIAL="PARTIAL"


@dataclass
class PriceLevel:
    price:str
    quantity: str
    orders: int

@dataclass
class OrderBook:
    symbol: str
    bids : list[PriceLevel]
    asks: list[PriceLevel]
    spread: str
    timestamp: str

@dataclass
class Order:
    id: str
    bot_id: str
    symbol: str
    side: Side
    type: OrderType
    price: str
    quantity: str
    remaining: str
    status: OrderStatus
    created_at: str
    updated_at: str


@dataclass
class Trade:
    id: str
    symbol: str
    buy_order_id: str
    sell_order_id: str
    buy_bot_id: str
    sell_bot_id: str
    price: str
    quantity: str
    notional: str
    executed_at: str


@dataclass
class Ticker:
    symbol: str
    bid_price: str
    ask_price: str
    last_price: str
    last_qty: str
    spread: str
    timestamp: str