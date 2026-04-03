import { useQuery } from '@tanstack/react-query'
import { LineChart, Line, XAxis, YAxis, Tooltip, ResponsiveContainer, ReferenceLine } from 'recharts'
import { format } from 'date-fns'
import { Panel } from '../ui/Panel'
import { fetchMLSignal, fetchMLHistory } from '../../api/client'

interface Props { symbol: string }

export function MLPanel({ symbol }: Props) {
  const { data: signal } = useQuery({
    queryKey:        ['ml-signal', symbol],
    queryFn:         () => fetchMLSignal(symbol),
    refetchInterval: 2000,
  })

  const { data: history } = useQuery({
    queryKey:        ['ml-history', symbol],
    queryFn:         () => fetchMLHistory(symbol, 50),
    refetchInterval: 10000,
  })

  const chartData = (history ?? [])
    .slice()
    .reverse()
    .map(s => ({
      time:     format(new Date(s.RecordedAt), 'HH:mm'),
      accuracy: Math.round(s.Accuracy * 100),
      samples:  s.NSamples,
    }))

  const signalColor = {
    BUY:  'var(--green)',
    SELL: 'var(--red)',
    HOLD: 'var(--amber)',
  }[signal?.signal ?? 'HOLD']

  return (
    <Panel title="ML model" accent="blue">
      {/* Current signal */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 16, marginBottom: 16 }}>
        <div style={{ textAlign: 'center' }}>
          <div style={{ fontSize: 24, fontWeight: 700, color: signalColor,
            fontFamily: 'var(--font-mono)', letterSpacing: 2 }}>
            {signal?.warming_up ? 'WARM' : (signal?.signal ?? '—')}
          </div>
          <div style={{ fontSize: 10, color: 'var(--text-muted)', marginTop: 2 }}>
            {signal?.warming_up ? 'collecting data' : 'current signal'}
          </div>
        </div>

        <div style={{ flex: 1 }}>
          {signal && !signal.warming_up && (
            <>
              <ConfBar label="BUY"  value={signal.probabilities.BUY}  color="var(--green)" />
              <ConfBar label="HOLD" value={signal.probabilities.HOLD} color="var(--amber)" />
              <ConfBar label="SELL" value={signal.probabilities.SELL} color="var(--red)"   />
            </>
          )}
        </div>

        <div style={{ textAlign: 'right', color: 'var(--text-muted)', fontSize: 11 }}>
          <div>{signal?.model_samples ?? 0} samples</div>
          <div style={{ marginTop: 2 }}>
            {((signal?.confidence ?? 0) * 100).toFixed(0)}% confidence
          </div>
        </div>
      </div>

      {/* Accuracy over time chart */}
      {chartData.length > 1 ? (
        <>
          <div style={{ fontSize: 10, color: 'var(--text-muted)', marginBottom: 6 }}>
            Accuracy over time (last 50 snapshots)
          </div>
          <ResponsiveContainer width="100%" height={120}>
            <LineChart data={chartData}>
              <XAxis dataKey="time" tick={{ fill: 'var(--text-muted)', fontSize: 10 }}
                axisLine={false} tickLine={false} />
              <YAxis domain={[0, 100]} tick={{ fill: 'var(--text-muted)', fontSize: 10 }}
                axisLine={false} tickLine={false} unit="%" width={32} />
              <Tooltip
                contentStyle={{ background: 'var(--bg-elevated)', border: '1px solid var(--border)',
                  borderRadius: 6, fontSize: 11 }}
                labelStyle={{ color: 'var(--text-secondary)' }}
                formatter={(v: any) => [
                 `${Number(v ?? 0).toFixed(2)}%`, 
                'Accuracy'
                 ]}
                // Modified to any
              />
              <ReferenceLine y={50} stroke="var(--border-light)" strokeDasharray="4 2" />
              <Line type="monotone" dataKey="accuracy" stroke="var(--blue)"
                strokeWidth={2} dot={false} activeDot={{ r: 4 }} />
            </LineChart>
          </ResponsiveContainer>
        </>
      ) : (
        <p style={{ color: 'var(--text-muted)', fontSize: 11, textAlign: 'center', padding: 16 }}>
          Accuracy chart appears after first model snapshots
        </p>
      )}
    </Panel>
  )
}

function ConfBar({ label, value, color }: { label: string; value: number; color: string }) {
  return (
    <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 4 }}>
      <span style={{ width: 32, fontSize: 10, color: 'var(--text-muted)',
        fontFamily: 'var(--font-mono)' }}>{label}</span>
      <div style={{ flex: 1, height: 6, background: 'var(--bg-elevated)',
        borderRadius: 3, overflow: 'hidden' }}>
        <div style={{ width: `${value * 100}%`, height: '100%',
          background: color, transition: 'width 0.4s ease', borderRadius: 3 }} />
      </div>
      <span style={{ width: 36, fontSize: 10, textAlign: 'right',
        fontFamily: 'var(--font-mono)', color: 'var(--text-secondary)' }}>
        {(value * 100).toFixed(0)}%
      </span>
    </div>
  )
}