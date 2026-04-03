import { formatDistanceToNow } from 'date-fns'
import { Panel } from '../ui/Panel'
import { useDashboard } from '../../store'

export function SupervisorPanel() {
  const violations = useDashboard(s => s.violations)

  return (
    <Panel title="Supervisor log" accent="red">
      {violations.length === 0 ? (
        <div style={{ textAlign: 'center', padding: 24,
          color: 'var(--text-muted)', fontSize: 12 }}>
          <span style={{ color: 'var(--green)', marginRight: 6 }}>✓</span>
          No violations detected
        </div>
      ) : violations.map((v, i) => (
        <div key={i} style={{
          padding: '8px 10px',
          marginBottom: 6,
          background: 'var(--red-muted)',
          border: '1px solid var(--red)',
          borderRadius: 'var(--radius-sm)',
          fontSize: 11,
        }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 4 }}>
            <span style={{ color: 'var(--red)', fontWeight: 600,
              fontFamily: 'var(--font-mono)' }}>
              {v.rule}
            </span>
            <span style={{ color: 'var(--text-muted)', fontSize: 10 }}>
              {formatDistanceToNow(new Date(v.timestamp), { addSuffix: true })}
            </span>
          </div>
          <div style={{ color: 'var(--text-secondary)' }}>
            <span style={{ color: 'var(--amber)' }}>{v.bot_id}</span>
            {' — '}{v.reason}
          </div>
        </div>
      ))}
    </Panel>
  )
}