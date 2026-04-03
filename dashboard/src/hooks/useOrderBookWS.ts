// Tasks for this file
// --Replace all URLs from proper env file references
// -- Update try catch block to handle errors
import { useEffect, useRef, useState } from "react";
import type { OrderBook } from "../api/types";

export function useOrderBookWS(symbol:string){
    const[book,setBook] = useState<OrderBook|null>(null);
    const [connected, setConn] = useState(false);
    const wsRef = useRef<WebSocket|null>(null);

    useEffect(()=>{
        if(!symbol) return;
        const url=`ws://localhost:8080/ws/${symbol}`;
        // Update env file link reference here
        // wsRef.current = new WebSocket(url);
        const ws = new WebSocket(url);
        wsRef.current = ws;

        ws.onopen = ()=>setConn(true);
        ws.onclose = ()=>setConn(false);
        ws.onerror=()=>setConn(false);
        ws.onmessage=(e)=>{
            try{
                setBook(JSON.parse(e.data));
            } catch{
                // Handle error
            }
        }
        return ()=>{
            ws.close();
        }
    },[symbol]);
    return {book,connected};
}