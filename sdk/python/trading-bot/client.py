"""
Trading Platform Bot SDK — Python (async)
------------------------------------------
Uses asyncio + httpx for non-blocking I/O.
Industry standard for high-concurrency market data consumption.
 
Usage:
    import asyncio
    from trading_bot import BotClient, Side, OrderType
 
    async def main():
        async with BotClient(bot_id="my-bot", api_url="...", api_key="...") as client:
            await client.subscribe("BTC-USD", on_tick)
            await client.run_forever()
 
    asyncio.run(main())
"""
 

import asyncio
import os
from decimal import Decimal, ROUND_HALF_UP

from typing import Callable, Awaitable, Optional
import httpx

from .models import Order, OrderBook, Ticker, PriceLevel, Side, OrderType, OrderStatus


class BotClientError(Exception):
    pass

TickerCallback = Callable[[Ticker], Awaitable[None]]

class BotClient:
        """
    Async bot client. Always use as an async context manager:
 
        async with BotClient(...) as client:
            await client.subscribe("BTC-USD", on_tick)
            await client.run_forever()
    """
    def __init__(self, bot_id:str, api_url: str, api_key: str):
        self.bot_id= bot_id
        self.api_url=api_url.rstrip("/")
        self.api_key=api_key

        self._http: Optional[httpx.AsyncClient] = None
        self._subscriptions: dict[str, list[TickerCallback]]={}
        self._poll_tasks: dict[str, asyncio.Task]= {}
        self._running = False

    async def __aenter__(self)->"BotClient":
        await self.start()
        return self

    async def __aexit__(self,*_)->None:
        await self.stop()

    async def start(self)->None:
        self._http = httpx.AsyncClient(
        headers={"Authorization": f"Bearer {self.api_key}"},
        timeout=5.0,
        )
        self._running = True

    async def stop(self)->None:
        self._running = False
        if self._http:
            await self._http.aclose()

    async def run_forever(self)->None:
        "Block until stop() is called or KeyboardInterrupt"
        try:
            while self._running:
                await asyncio.sleep(1)
        except asyncio.CancelledError:
            pass
        
    #Order Management
    async def place_order(
        self,
        symbol: str,
        side: Side,
        quantity: str,
        price: Optional[str]=None,
        order_type: OrderType = OrderType.LIMIT,
        )->Order:
           """
        Place a limit or market order.
 
        IMPORTANT: price and quantity must be strings, never floats.
        Use the price_add / price_sub helpers for arithmetic:
 
            from trading_bot.client import price_sub
            price = price_sub(ticker.ask_price, "1.00")
            await client.place_order("BTC-USD", Side.BUY, "0.1", price)
            """
        payload: dict ={
            "bot_id": self.bot_id,
            "symbol": symbol,
            "side": side.value,            
            # "price": price,
            "type": order_type.value,
            "quantity": quantity,
        }

        match order_type:
            case OrderType.LIMIT:
                if price is None:
                    raise BotClientError("Price is required for LIMIT orders")
                payload["price"]= price
            case OrderType.MARKET:
                pass
        resp = await self.post("api/v1/orders", payload)
        return self._parse_order(resp["order"])
    async def cancel_order(self, symbol:str, order_id: str)->Order:
        "Cancel a Resting Order by ID"
        resp = await self._delete(f"api/v1/orders/{symbol}/{order_id}")
        return self._parse_order(resp)

    async def get_order_book(self, symbol:str, depth: int =10)->OrderBook:
        "Fetch current Order book snapshot"
        resp= await self._get(f"/api/v1/markets/{symbol}/order_book?depth={depth}")
        return OrderBook{
            symbol =resp["symbol"],
            bids= [PriceLevel(**b) for b in resp["bids"]],
            asks= [PriceLevel(**a) for a in resp["asks"]],
            spread= resp["spread"]
            timestamp= resp["timestamp"],
        }
    async def get_markets(self)->list[str]:
        "Return all active trading symbols"
        resp= await.self._get("api/v1//markets")
        return resp["symbols"]

    # Subscription

    async def subscribe(self, symbol:str, callback: TickerCallback)->None:
         """
        Subscribe to live ticker updates for a symbol.
        callback is an async function called every ~500ms.
        Non-blocking — spawns a background task.
 
        Example:
            async def on_tick(ticker):
                print(ticker.bid_price)
 
            await client.subscribe("BTC-USD", on_tick)
        """
        if symbol not in self._subscriptions:
            self._subscriptions[symbol] = []
        self._subscriptions[symbol].append(callback)

        if symbol not in self._poll_tasks:
            task= asyncio.create_task(
                self._poll_loop(symbol),
                name=f"poll-{symbol}"
            )
            self._poll_tasks[symbol] = task

    async def unsubscribe(self, sybmol: str)->None:
        self._subscriptions(syumbol,None)
        if task :=self._poll_tasks.pop(symbol,None):
            task.cancel()
            await asyncio.gather(task, return_exceptions= True)

            
