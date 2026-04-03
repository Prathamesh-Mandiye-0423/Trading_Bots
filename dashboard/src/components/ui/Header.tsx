import { useQuery } from '@tanstack/react-query'
import { fetchMarkets } from '../../api/client'
import { useDashboard } from '../../store'

export function Header() {
  const selected   = useDashboard(s => s.selectedSymbol)
  const setSymbol  = useDashboard(s => s.setSymbol)
  const { data: markets = ['BTC-USD', 'ETH-USD', 'SOL-USD'] } = useQuery({
    queryKey: ['markets'],
    queryFn:  fetchMarkets,
  })

  return (
    <header style={{
      display:        'flex',
      alignItems:     'center',
      justifyContent: 'space-between',
      padding:        '0 20px',
      height:         52,
      background:     'var(--bg-panel)',
      borderBottom:   '1px solid var(--border)',
      flexShrink:     0,
    }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
        <div style={{
          width: 28, height: 28,
          background: 'linear-gradient(135deg, var(--green), var(--blue))',
          borderRadius: 6,
        }}/>
        <span style={{ fontWeight: 700, fontSize: 15,
          letterSpacing: '-0.02em', color: 'var(--text-primary)' }}>
          AlgoTrade
        </span>
        <span style={{ color: 'var(--text-muted)', fontSize: 11, marginLeft: 4 }}>
          terminal
        </span>
      </div>

      {/* Symbol tabs */}
      <div style={{ display: 'flex', gap: 4 }}>
        {markets.map(sym => (
          <button key={sym} onClick={() => setSymbol(sym)} style={{
            padding:      '5px 14px',
            borderRadius: 'var(--radius-sm)',
            border:       `1px solid ${selected === sym ? 'var(--green)' : 'var(--border)'}`,
            background:   selected === sym ? 'var(--green-muted)' : 'transparent',
            color:        selected === sym ? 'var(--green)' : 'var(--text-secondary)',
            cursor:       'pointer',
            fontSize:     12,
            fontFamily:   'var(--font-mono)',
            fontWeight:   selected === sym ? 600 : 400,
            transition:   'all 0.15s',
          }}>
            {sym}
          </button>
        ))}
      </div>

      <div style={{ fontSize: 11, color: 'var(--text-muted)', fontFamily: 'var(--font-mono)' }}>
        {new Date().toLocaleTimeString()}
      </div>
    </header>
  )
}