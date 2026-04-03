import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { Header }          from './components/ui/Header'
import { OrderBookPanel }  from './components/panels/OrderBookPanel'
import { TradeFeedPanel }  from './components/panels/TradeFeedPanel'
import { MLPanel }         from './components/panels/MLPanel'
import { BotTablePanel }   from './components/panels/BotTablePanel'
import { SupervisorPanel } from './components/panels/SupervisorPanel'
import { useDashboard }    from './store'

const queryClient = new QueryClient({
  defaultOptions: { queries: { retry: 1, staleTime: 2000 } }
})

function Dashboard() {
  const symbol = useDashboard(s => s.selectedSymbol)

  return (
    <div style={{ height: '100vh', display: 'flex', flexDirection: 'column' }}>
      <Header />

      <main style={{
        flex:     1,
        overflow: 'hidden',
        padding:  12,
        display:  'grid',
        gap:      12,
        gridTemplateColumns: '320px 1fr 1fr',
        gridTemplateRows:    '1fr 1fr',
      }}>
        {/* Column 1 — Order book (full height) */}
        <div style={{ gridRow: '1 / 3' }}>
          <OrderBookPanel symbol={symbol} />
        </div>

        {/* Column 2 row 1 — Trade feed */}
        <TradeFeedPanel symbol={symbol} />

        {/* Column 3 row 1 — ML model */}
        <MLPanel symbol={symbol} />

        {/* Column 2 row 2 — Bot table */}
        <BotTablePanel />

        {/* Column 3 row 2 — Supervisor log */}
        <SupervisorPanel />
      </main>
    </div>
  )
}

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <Dashboard />
    </QueryClientProvider>
  )
}