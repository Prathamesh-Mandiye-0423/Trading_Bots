import { useOrderBookWS } from '../../hooks/useOrderBookWS'
import { Panel } from '../ui/Panel'
import { LiveDot } from '../ui/Dot'

interface Props { symbol: string }

function Bar({ pct, side }: { pct: number; side: 'bid' | 'ask' }) {
  return (
    <div style={{
      position:   'absolute',
      top: 0, bottom: 0,
      right: side === 'bid' ? 0 : undefined,
      left:  side === 'ask' ? 0 : undefined,
      width:      `${pct}%`,
      background: side === 'bid' ? 'var(--green-muted)' : 'var(--red-muted)',
      transition: 'width 0.2s ease',
    }}/>
  )
}

export function OrderBookPanel({ symbol }: Props) {
  const { book, connected } = useOrderBookWS(symbol)

  const maxQty = Math.max(
    ...(book?.bids.map(b => parseFloat(b.quantity)) ?? [0]),
    ...(book?.asks.map(a => parseFloat(a.quantity)) ?? [0]),
    0.0001
  )

  return (
    <Panel
      title={`Order book · ${symbol}`}
      accent="green"
      badge={<LiveDot on={connected} />}
    >
      {/* Header */}
      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr',
        color: 'var(--text-muted)', fontSize: 11, marginBottom: 6,
        fontFamily: 'var(--font-mono)', padding: '0 2px' }}>
        <span>Price</span>
        <span style={{ textAlign: 'center' }}>Qty</span>
        <span style={{ textAlign: 'right' }}>Orders</span>
      </div>

      {/* Asks — reversed so lowest ask is nearest the spread */}
      {[...(book?.asks ?? [])].reverse().map((ask, i) => (
        <div key={i} style={{ position: 'relative', display: 'grid',
          gridTemplateColumns: '1fr 1fr 1fr', padding: '2px 2px',
          fontFamily: 'var(--font-mono)', fontSize: 12 }}>
          <Bar pct={(parseFloat(ask.quantity) / maxQty) * 100} side="ask" />
          <span className="red" style={{ zIndex: 1 }}>{parseFloat(ask.price).toFixed(2)}</span>
          <span style={{ textAlign: 'center', zIndex: 1 }}>{parseFloat(ask.quantity).toFixed(4)}</span>
          <span style={{ textAlign: 'right', color: 'var(--text-muted)', zIndex: 1 }}>{ask.orders}</span>
        </div>
      ))}

      {/* Spread */}
      <div style={{ textAlign: 'center', padding: '6px 0', fontSize: 11,
        color: 'var(--amber)', fontFamily: 'var(--font-mono)',
        borderTop: '1px solid var(--border)', borderBottom: '1px solid var(--border)',
        margin: '4px 0' }}>
        spread {book ? parseFloat(book.spread).toFixed(2) : '—'}
      </div>

      {/* Bids */}
      {(book?.bids ?? []).map((bid, i) => (
        <div key={i} style={{ position: 'relative', display: 'grid',
          gridTemplateColumns: '1fr 1fr 1fr', padding: '2px 2px',
          fontFamily: 'var(--font-mono)', fontSize: 12 }}>
          <Bar pct={(parseFloat(bid.quantity) / maxQty) * 100} side="bid" />
          <span className="green" style={{ zIndex: 1 }}>{parseFloat(bid.price).toFixed(2)}</span>
          <span style={{ textAlign: 'center', zIndex: 1 }}>{parseFloat(bid.quantity).toFixed(4)}</span>
          <span style={{ textAlign: 'right', color: 'var(--text-muted)', zIndex: 1 }}>{bid.orders}</span>
        </div>
      ))}
    </Panel>
  )
}