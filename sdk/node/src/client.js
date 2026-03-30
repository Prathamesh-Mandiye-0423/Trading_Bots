/**
 * Trading Platform Bot SDK — Node.js
 * ------------------------------------
 * Requires Node 18+ (native fetch built-in, no extra dependencies).
 *
 * Usage:
 *   const { BotClient, Side, OrderType, priceSub } = require('./src/client')
 *
 *   const client = new BotClient({
 *     botId:  'my-bot',
 *     apiUrl: 'http://localhost:8080',
 *     apiKey: 'dev'
 *   })
 *
 *   client.subscribe('BTC-USD', async (ticker) => {
 *     const price = priceSub(ticker.askPrice, '1.00')
 *     await client.placeOrder('BTC-USD', Side.BUY, '0.01', price)
 *   })
 */

"use strict";

const { Side, OrderType, OrderStatus } = require("./models");

class BotClientError extends Error {
  constructor(message, status) {
    super(message);
    this.name   = "BotClientError";
    this.status = status;
  }
}

class BotClient {
  constructor({ botId, apiUrl, apiKey }) {
    this.botId  = botId;
    this.apiUrl = apiUrl.replace(/\/$/, "");
    this.apiKey = apiKey;

    this._subscriptions = {};  // symbol → [callbacks]
    this._pollIntervals = {};  // symbol → intervalId
    this._running       = true;
  }

  // ---- Order management ----

  async placeOrder(symbol, side, quantity, price = null, orderType = OrderType.LIMIT) {
    const payload = {
      bot_id:   this.botId,
      symbol,
      side,
      type:     orderType,
      quantity,             // always a string — never a JS number
    };
    if (orderType === OrderType.LIMIT) {
      if (!price) throw new BotClientError("price required for LIMIT orders");
      payload.price = price; // always a string
    }
    const resp = await this._post("/api/v1/orders", payload);
    return resp.order;
  }

  async cancelOrder(symbol, orderId) {
    return this._delete(`/api/v1/orders/${symbol}/${orderId}`);
  }

  async getOrderbook(symbol, depth = 10) {
    return this._get(`/api/v1/markets/${symbol}/orderbook?depth=${depth}`);
  }

  async getMarkets() {
    const resp = await this._get("/api/v1/markets");
    return resp.symbols;
  }

  // ---- Market data subscription ----

  subscribe(symbol, callback) {
    if (!this._subscriptions[symbol]) {
      this._subscriptions[symbol] = [];
    }
    this._subscriptions[symbol].push(callback);

    if (!this._pollIntervals[symbol]) {
      this._pollIntervals[symbol] = setInterval(async () => {
        if (!this._running) return;
        try {
          const book   = await this.getOrderbook(symbol, 1);
          const ticker = {
            symbol,
            bidPrice:  book.bids?.[0]?.price ?? "0.00000000",
            askPrice:  book.asks?.[0]?.price ?? "0.00000000",
            spread:    book.spread,
            timestamp: book.timestamp,
          };
          for (const cb of this._subscriptions[symbol] ?? []) {
            try {
              await Promise.resolve(cb(ticker));
            } catch (e) {
              console.error("[SDK] callback error:", e.message);
            }
          }
        } catch (e) {
          console.error(`[SDK] poll error for ${symbol}:`, e.message);
        }
      }, 500);
    }
  }

  unsubscribe(symbol) {
    clearInterval(this._pollIntervals[symbol]);
    delete this._pollIntervals[symbol];
    delete this._subscriptions[symbol];
  }

  stop() {
    this._running = false;
    Object.keys(this._pollIntervals).forEach(s => this.unsubscribe(s));
  }

  // ---- HTTP helpers — native fetch (Node 18+, no dependencies) ----

  async _request(method, path, body = null) {
    const response = await fetch(`${this.apiUrl}${path}`, {
      method,
      headers: {
        "Content-Type":  "application/json",
        "Authorization": `Bearer ${this.apiKey}`,
      },
      body: body ? JSON.stringify(body) : null,
    });
    const data = await response.json();
    if (!response.ok) {
      throw new BotClientError(
        data.error || response.statusText,
        response.status
      );
    }
    return data;
  }

  _get(path)        { return this._request("GET",    path);       }
  _post(path, body) { return this._request("POST",   path, body); }
  _delete(path)     { return this._request("DELETE", path);       }
}

// ============================================================
// Decimal price arithmetic helpers
// Use these instead of parseFloat() for any price math.
// Internally uses BigInt scaled to 8 decimal places.
// Zero float drift — same precision as the platform internals.
// ============================================================

const SCALE = 100_000_000n; // 10^8

function _toScaled(s) {
  const [intPart, fracPart = ""] = String(s).split(".");
  const frac = fracPart.padEnd(8, "0").slice(0, 8);
  return BigInt(intPart) * SCALE + BigInt(frac);
}

function _fromScaled(n) {
  const abs  = n < 0n ? -n : n;
  const sign = n < 0n ? "-" : "";
  const int  = abs / SCALE;
  const frac = String(abs % SCALE).padStart(8, "0");
  return `${sign}${int}.${frac}`;
}

/**
 * Add two price strings exactly.
 * priceAdd("50000.00", "1.00") → "50001.00000000"
 */
function priceAdd(a, b) { return _fromScaled(_toScaled(a) + _toScaled(b)); }

/**
 * Subtract two price strings exactly.
 * priceSub("50000.00", "1.00") → "49999.00000000"
 */
function priceSub(a, b) { return _fromScaled(_toScaled(a) - _toScaled(b)); }

/**
 * Multiply two price strings exactly.
 * priceMul("50000.00", "0.5") → "25000.00000000"
 */
function priceMul(a, b) { return _fromScaled(_toScaled(a) * _toScaled(b) / SCALE); }

module.exports = {
  BotClient,
  BotClientError,
  Side,
  OrderType,
  OrderStatus,
  priceAdd,
  priceSub,
  priceMul,
};