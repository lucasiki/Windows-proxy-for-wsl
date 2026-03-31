import type { ProxyState } from '../hooks/useProxy'

interface Props {
  state: ProxyState
  onStart: () => void
  onPause: () => void
  onStop: () => void
}

export function Controls({ state, onStart, onPause, onStop }: Props) {
  const isStopped = state === 'stopped'
  const isRunning = state === 'running'
  const isPaused  = state === 'paused'

  const startLabel = isPaused ? '▶ Retomar' : '▶ Iniciar'

  return (
    <div style={{
      background: 'var(--bg-surface)',
      padding: '8px 12px',
      display: 'flex',
      gap: 8,
      borderTop: '1px solid var(--border)',
      borderRadius: '0 0 6px 6px',
    }}>
      <button
        onClick={isStopped || isPaused ? onStart : onPause}
        style={{
          background: isRunning ? 'var(--yellow)' : 'var(--accent)',
          color: '#fff',
        }}
      >
        {isRunning ? '⏸ Pausar' : startLabel}
      </button>

      <button
        onClick={onStop}
        disabled={isStopped}
        style={{
          background: 'transparent',
          color: 'var(--red)',
          border: '1px solid var(--red)',
        }}
      >
        ■ Parar
      </button>
    </div>
  )
}
