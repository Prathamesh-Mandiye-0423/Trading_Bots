import {create} from 'zustand';
import type { Trade, SupervisorViolation} from '../api/types';

interface DashboardStore {
    selectedSymbol: string;
    setSymbol: (s:string)=>void;

    liveTrades: Trade[];
    addTrade: (t:Trade)=>void;

    violations: SupervisorViolation[];
    addViolation: (v:SupervisorViolation)=>void;

    connectedBots: Set<string>;
    registerBot: (id:string)=>void;
    suspendBot: (id:string)=>void;
    suspendedBots: Set<string>;
}


export const useDashboard = create<DashboardStore>((set) => ({
    selectedSymbol: 'BTC-USD',
    setSymbol: (s)=>set({selectedSymbol: s}),

    liveTrades: [],
    addTrade: (t)=>set((state) => (
        {
            liveTrades: [t,...state.liveTrades].slice(0, 200)
        }
    )),

    violations:[],
    addViolation:(v)=>set((state) => ({
        violations: [v,...state.violations].slice(0, 100)
    })),
    connectedBots: new Set(),
    suspendedBots: new Set(),
    registerBot: (id) =>set((state)=>({
        connectedBots: new Set([...state.connectedBots,id])
    })),
    suspendBot: (id)=>set((state)=>({
        suspendedBots: new Set([...state.suspendedBots,id])
    })),
}))