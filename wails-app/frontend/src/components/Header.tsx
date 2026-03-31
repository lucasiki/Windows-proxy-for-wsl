import type { ProxyState, WSLInfo } from '../hooks/useProxy'
import { Quit, WindowHide } from '../wailsjs/runtime/runtime'

interface Props {
  state: ProxyState
  wslInfo: WSLInfo | null
}

const statusConfig = {
  stopped: { label: 'PARADO', color: 'var(--red)' },
  running: { label: 'EXECUTANDO', color: 'var(--green)' },
  paused:  { label: 'PAUSADO', color: 'var(--yellow)' },
}

const btnStyle: React.CSSProperties = {
  background: 'transparent',
  color: 'var(--text-muted)',
  padding: '2px 8px',
  border: 'none',
  fontSize: 14,
  lineHeight: 1,
  borderRadius: 3,
  ['--wails-draggable' as string]: 'no-drag',
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
      ['--wails-draggable' as string]: 'drag'
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
      <div>
        <button style={btnStyle} onClick={WindowHide} title="Minimizar para tray">─</button>
        <button style={{ ...btnStyle, color: 'var(--red)' }} onClick={Quit} title="Fechar">✕</button>
      </div>
    </div>
  )
}
