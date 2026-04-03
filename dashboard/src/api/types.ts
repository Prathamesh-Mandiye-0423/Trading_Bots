export interface Trade{
    ID: string;
    Symbol: string;
    BuyBotID: string;
    SellBotID: string;
    Price: string;
    Quantity: string;
    Notional: string;
    ExecutedAt: string;
}

export interface PriceLevel{
    price: string;
    quantity: string;
    orders: number;
}

export interface OrderBook{
    symbol: string;
    bids: PriceLevel[];
    asks: PriceLevel[];
    spread: string;
    timestamp: string;
}

export interface BotPnL {
    BotID: string;
    TradeCount: number;
    TotalVolume: string;
    AvgPrice: string;
}


export interface MLSnapshot{
    ID: string;
    Symbol: string;
    NSamples: number;
    Accuracy: number;
    RecordedAt: string;
}

export interface MLSignal{
    symbol: string;
    signal: 'BUY' | 'SELL' | 'HOLD';
    confidence: number;
    warming_up: boolean;
    model_samples: number;
    probabilities:{BUY:number;SELL:number;HOLD:number};
    features_used: Record<string,number>;
}

export interface SupervisorViolation{
    bot_id: string;
    rule: string;
    reason: string;
    timestamp: string;
}
