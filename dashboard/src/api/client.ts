import axios from "axios";
import type { Trade, OrderBook, BotPnL, MLSnapshot, MLSignal} from './types';


// Check routing during testing and deployment
// Review vite.config.ts
const api= axios.create({baseURL:'/api/v1'})
const ml  = axios.create({baseURL:'http://localhost:8001/api/v1'})


export const fetchOrderBook = async (symbol:string):Promise<OrderBook>=>{
    const {data} = await api.get(`/markets/${symbol}/orderbook?depth=12`);
    return data;
}


export const fetchRecentTrades = async (symbol:string, limit=50):Promise<Trade[]>=>{
    const {data} = await api.get(`/trades/${symbol}?limit=${limit}`);
    return data.trades ??[];
}

export const fetchBotPnL = async (botId:string):Promise<BotPnL>=>{
    const {data} = await api.get(`/bots/${botId}/pnl`);
    return data;
}

export const fetchMarkets = async(): Promise<string[]>=>{
    const {data} = await api.get('/markets');
    return data.symbols ?? [];
}

export const fetchMLSignal = async(symbol:string): Promise<MLSignal>=>{
    const {data} = await ml.get(`/ml/${symbol}/signal`);
    return data;
}

export const fetchMLHistory = async(symbol:string, limit=100): Promise<MLSnapshot[]>=>{
    const {data} = await ml.get(`/ml/${symbol}/history?limit=${limit}`);
    return data.history ?? [];
}

