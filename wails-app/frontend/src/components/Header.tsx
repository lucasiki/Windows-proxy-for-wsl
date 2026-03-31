import type { ProxyState, WSLInfo } from '../hooks/useProxy'

interface Props {
  state: ProxyState
  wslInfo: WSLInfo | null
}

const statusConfig = {
  stopped: { label: 'PARADO', color: 'var(--red)' },
  running: { label: 'EXECUTANDO', color: 'var(--green)' },
  paused:  { label: 'PAUSADO', color: 'var(--yellow)' },
}

export function Header({ state, wslInfo }: Props) {
  const { label, color } = statusConfig[state]

  return (
    <div style={{
      background: 'var(--bg-secondary)',
      padding: '10px 16px',
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'space-between',
      borderBottom: '1px solid var(--border)',
      flexShrink: 0,
    }}>
      <span style={{ fontWeight: 700, fontSize: 14 }}>WSL Proxy</span>

      <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
        {wslInfo ? (
          <span style={{ color: 'var(--text-muted)', fontSize: 12 }}>
            WSL: {wslInfo.ip}{' '}
            <span style={{
              background: 'var(--bg-surface)',
              border: '1px solid var(--border)',
              borderRadius: 3,
              padding: '1px 5px',
              fontSize: 11,
            }}>
              {wslInfo.mode}
            </span>
          </span>
        ) : (
          <span style={{ color: 'var(--text-dim)', fontSize: 12 }}>WSL não detectado</span>
        )}

        <span style={{ color, fontWeight: 700, fontSize: 12 }}>
          ● {label}
        </span>
      </div>
    </div>
  )
}
