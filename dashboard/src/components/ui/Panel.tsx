import {clsx} from 'clsx'
import type React from 'react';

interface PanelProps{
    title: string;
    children: React.ReactNode;
    className?: string;
    badge?: React.ReactNode;
    accent?: 'green' | 'amber' | 'red' | 'blue';
}

const accentMap={
    green: 'var(--green)',
    amber: 'var(--amber)',
    red:   'var(--red)',
    blue:  'var(--blue)',
}

export function Panel({ title, children, className, badge, accent = 'green' }: PanelProps) {
  return (
    <div className={clsx('panel', className)} style={{
      background:   'var(--bg-panel)',
      border:       `1px solid var(--border)`,
      borderRadius: 'var(--radius-lg)',
      display:      'flex',
      flexDirection:'column',
      overflow:     'hidden',
    }}>
      <div style={{
        display:        'flex',
        alignItems:     'center',
        justifyContent: 'space-between',
        padding:        '10px 14px',
        borderBottom:   `1px solid var(--border)`,
        borderLeft:     `3px solid ${accentMap[accent]}`,
      }}>
        <span style={{ fontWeight: 600, fontSize: 11, letterSpacing: '0.08em',
          textTransform: 'uppercase', color: 'var(--text-secondary)' }}>
          {title}
        </span>
        {badge}
      </div>
      <div style={{ flex: 1, overflow: 'auto', padding: 14 }}>
        {children}
      </div>
    </div>
  )
}