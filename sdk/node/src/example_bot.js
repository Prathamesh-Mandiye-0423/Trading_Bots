/**
 * Example bot — Node.js
 *
 * Run:
 *   BOT_ID=node-bot API_URL=http://localhost:8080 API_KEY=dev node example_bot.js
 */

"use strict";

const { BotClient, Side, priceSub } = require("./client");

const client = new BotClient({
  botId:  process.env.BOT_ID  || "node-bot",
  apiUrl: process.env.API_URL || "http://localhost:8080",
  apiKey: process.env.API_KEY || "dev",
});

const SYMBOL      = "BTC-USD";
let openOrderId   = null;

client.subscribe(SYMBOL, async (ticker) => {
  console.log(`[${SYMBOL}] bid=${ticker.bidPrice} ask=${ticker.askPrice} spread=${ticker.spread}`);

  if (ticker.askPrice === "0.00000000" || openOrderId) return;

  // Use priceSub — never parseFloat()
  const price = priceSub(ticker.askPrice, "1.00");
  try {
    const order  = await client.placeOrder(SYMBOL, Side.BUY, "0.01", price);
    openOrderId  = order.id;
    console.log(`  → placed buy at ${price}, order=${order.id}`);
  } catch (e) {
    console.error("  → order failed:", e.message);
  }
});

console.log(`Bot ${client.botId} running on ${SYMBOL}... (Ctrl+C to stop)`);
process.on("SIGINT", () => { client.stop(); process.exit(0); });