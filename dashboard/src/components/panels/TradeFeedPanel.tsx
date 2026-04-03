import { useEffect } from 'react'
import { useQuery } from '@tanstack/react-query'
import { formatDistanceToNow } from 'date-fns'
import { Panel } from '../ui/Panel'
import { fetchRecentTrades } from '../../api/client'
import { useDashboard } from '../../store'
import type { Trade } from '../../api/types'

function TradeRow({ trade }: { trade: Trade }) {
  const age = formatDistanceToNow(new Date(trade.ExecutedAt), { addSuffix: true })
  return (
    <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr auto',
      gap: 8, padding: '4px 2px', borderBottom: '1px solid var(--border)',
      fontFamily: 'var(--font-mono)', fontSize: 12, alignItems: 'center' }}>
      <span className="amber">${parseFloat(trade.Price).toFixed(2)}</span>
      <span>{parseFloat(trade.Quantity).toFixed(4)}</span>
      <span style={{ color: 'var(--text-secondary)', fontSize: 10 }}>
        {trade.BuyBotID.slice(0, 8)}
      </span>
      <span style={{ color: 'var(--text-muted)', fontSize: 10 }}>{age}</span>
    </div>
  )
}

export function TradeFeedPanel({ symbol }: { symbol: string }) {
  const liveTrades = useDashboard(s => s.liveTrades)
  const addTrade   = useDashboard(s => s.addTrade)

  const { data: historical } = useQuery({
    queryKey: ['trades', symbol],
    queryFn:  () => fetchRecentTrades(symbol, 30),
    refetchInterval: 5000,
  })

  useEffect(() => {
    if (historical) historical.forEach(addTrade)
  }, [historical])

  const trades = liveTrades.filter(t => t.Symbol === symbol)

  return (
    <Panel title="Trade feed" accent="amber">
      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr auto',
        gap: 8, padding: '0 2px 6px', color: 'var(--text-muted)', fontSize: 11 }}>
        <span>Price</span><span>Qty</span><span>Buyer</span><span>Time</span>
      </div>
      {trades.length === 0
        ? <p style={{ color: 'var(--text-muted)', textAlign: 'center', padding: 24 }}>
            No trades yet
          </p>
        : trades.slice(0, 40).map(t => <TradeRow key={t.ID} trade={t} />)
      }
    </Panel>
  )
}