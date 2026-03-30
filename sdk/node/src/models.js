"use strict";

const Side = Object.freeze({
  BUY:  "BUY",
  SELL: "SELL",
});

const OrderType = Object.freeze({
  LIMIT:  "LIMIT",
  MARKET: "MARKET",
});

const OrderStatus = Object.freeze({
  OPEN:      "OPEN",
  FILLED:    "FILLED",
  PARTIAL:   "PARTIAL",
  CANCELLED: "CANCELLED",
  SUSPENDED: "SUSPENDED",
});

module.exports = { Side, OrderType, OrderStatus };