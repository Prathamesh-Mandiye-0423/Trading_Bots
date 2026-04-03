export function LiveDot({ on }: { on: boolean }) {
  return (
    <span style={{
      display:      'inline-block',
      width:        7, height: 7,
      borderRadius: '50%',
      background:   on ? 'var(--green)' : 'var(--text-muted)',
      boxShadow:    on ? '0 0 6px var(--green)' : 'none',
      marginRight:  6,
      transition:   'all 0.3s',
    }}/>
  )
}