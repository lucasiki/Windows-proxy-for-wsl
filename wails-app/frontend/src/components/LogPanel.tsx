import { useEffect, useRef } from 'react'
import type { LogEntry } from '../hooks/useProxy'

interface Props {
  entries: LogEntry[]
  connCount: number
  onClear: () => void
}

export function LogPanel({ entries, connCount, onClear }: Props) {
  const bottomRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [entries.length])

  const countLabel =
    connCount === 0 ? '' : connCount === 1 ? '1 conexão' : `${connCount} conexões`

  return (
    <div style={{
      background: 'var(--bg-secondary)',
      border: '1px solid var(--border)',
      borderRadius: 6,
      display: 'flex',
      flexDirection: 'column',
      flex: 1,
      overflow: 'hidden',
    }}>
      {/* Title row */}
      <div style={{
        background: 'var(--bg-surface)',
        padding: '6px 12px',
        borderBottom: '1px solid var(--border)',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
        flexShrink: 0,
      }}>
        <span style={{ fontWeight: 700, fontSize: 11, color: 'var(--text-muted)', letterSpacing: 1 }}>
          LOG DE CONEXÕES
        </span>
        <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
          {countLabel && (
            <span style={{ color: 'var(--text-muted)', fontSize: 12 }}>{countLabel}</span>
          )}
          <button
            onClick={onClear}
            style={{
              background: 'transparent',
              color: 'var(--text-muted)',
              border: '1px solid var(--border)',
              padding: '2px 8px',
              fontSize: 11,
            }}
          >
            Limpar
          </button>
        </div>
      </div>

      {/* Log entries */}
      <div style={{ flex: 1, overflowY: 'auto', padding: '6px 12px' }}>
        {entries.length === 0 ? (
          <div style={{
            color: 'var(--text-dim)',
            fontSize: 12,
            textAlign: 'center',
            marginTop: 20,
          }}>
            Nenhuma conexão registrada
          </div>
        ) : (
          entries.map((e) => (
            <div
              key={e.id}
              style={{
                fontFamily: 'monospace',
                fontSize: 12,
                color: e.isError ? 'var(--red)' : 'var(--text-primary)',
                padding: '2px 0',
                borderBottom: '1px solid var(--border)',
                opacity: e.isError ? 1 : 0.9,
              }}
            >
              {e.text}
            </div>
          ))
        )}
        <div ref={bottomRef} />
      </div>
    </div>
  )
}
