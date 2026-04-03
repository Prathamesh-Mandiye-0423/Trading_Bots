import { useQuery } from '@tanstack/react-query'
import { Panel } from '../ui/Panel'
import { fetchBotPnL } from '../../api/client'
import { useDashboard } from '../../store'

const KNOWN_BOTS = ['bot-1', 'bot-2', 'bot-3', 'bot-4', 'ml-bot', 'ma-bot']

function BotRow({ botID }: { botID: string }) {
  const suspended = useDashboard(s => s.suspendedBots.has(botID))

  const { data } = useQuery({
    queryKey:        ['pnl', botID],
    queryFn:         () => fetchBotPnL(botID),
    refetchInterval: 5000,
    retry:           false,
  })

  const status = suspended ? 'SUSPENDED' : data?.TradeCount ? 'ACTIVE' : 'IDLE'
  const statusColor = {
    ACTIVE:    'var(--green)',
    SUSPENDED: 'var(--red)',
    IDLE:      'var(--text-muted)',
  }[status]

  return (
    <div style={{ display: 'grid',
      gridTemplateColumns: '1.2fr 80px 80px 80px 70px',
      gap: 8, padding: '6px 4px',
      borderBottom: '1px solid var(--border)',
      fontFamily: 'var(--font-mono)', fontSize: 12, alignItems: 'center' }}>
      <span style={{ color: 'var(--text-primary)' }}>{botID}</span>
      <span style={{ color: statusColor, fontSize: 10, fontWeight: 600 }}>{status}</span>
      <span style={{ textAlign: 'right' }}>{data?.TradeCount ?? 0}</span>
      <span style={{ textAlign: 'right', color: 'var(--text-secondary)' }}>
        {data?.TotalVolume ? parseFloat(data.TotalVolume).toFixed(3) : '—'}
      </span>
      <span style={{ textAlign: 'right', color: 'var(--amber)' }}>
        {data?.AvgPrice ? `$${parseFloat(data.AvgPrice).toFixed(0)}` : '—'}
      </span>
    </div>
  )
}

export function BotTablePanel() {
  return (
    <Panel title="Bot performance" accent="amber">
      <div style={{ display: 'grid',
        gridTemplateColumns: '1.2fr 80px 80px 80px 70px',
        gap: 8, padding: '0 4px 8px',
        color: 'var(--text-muted)', fontSize: 10 }}>
        <span>Bot</span>
        <span>Status</span>
        <span style={{ textAlign: 'right' }}>Trades</span>
        <span style={{ textAlign: 'right' }}>Volume</span>
        <span style={{ textAlign: 'right' }}>Avg px</span>
      </div>
      {KNOWN_BOTS.map(id => <BotRow key={id} botID={id} />)}
    </Panel>
  )
}