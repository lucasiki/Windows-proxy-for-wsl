import { useState, KeyboardEvent } from 'react'
import type { PortMapping, ProxyState } from '../hooks/useProxy'

interface Props {
  mappings: PortMapping[]
  state: ProxyState
  onAdd: (spec: string) => void
  onRemove: (id: string) => void
}

export function PortList({ mappings, state, onAdd, onRemove }: Props) {
  const [input, setInput] = useState('')
  const locked = state !== 'stopped'

  const handleAdd = () => {
    const spec = input.trim()
    if (!spec) return
    onAdd(spec)
    setInput('')
  }

  const handleKey = (e: KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter') handleAdd()
  }

  return (
    <div style={{
      background: 'var(--bg-secondary)',
      borderRadius: 6,
      overflow: 'hidden',
      border: '1px solid var(--border)',
    }}>
      {/* Section title */}
      <div style={{
        background: 'var(--bg-surface)',
        padding: '6px 12px',
        borderBottom: '1px solid var(--border)',
      }}>
        <span style={{ fontWeight: 700, fontSize: 11, color: 'var(--text-muted)', letterSpacing: 1 }}>
          MAPEAMENTO DE PORTAS
        </span>
      </div>

      {/* Input row */}
      <div style={{ padding: '10px 12px', display: 'flex', gap: 8 }}>
        <input
          value={input}
          onChange={(e) => setInput(e.target.value)}
          onKeyDown={handleKey}
          placeholder="ex: 3000  ou  3000:3001  (Windows:WSL)"
          disabled={locked}
        />
        <button
          onClick={handleAdd}
          disabled={locked}
          style={{
            background: 'var(--accent)',
            color: '#fff',
            whiteSpace: 'nowrap',
            flexShrink: 0,
          }}
        >
          + Adicionar
        </button>
      </div>

      {/* Mapping list */}
      <div style={{ maxHeight: 140, overflowY: 'auto', padding: '0 12px 8px' }}>
        {mappings.length === 0 ? (
          <div style={{
            textAlign: 'center',
            color: 'var(--text-dim)',
            padding: '16px 0',
            fontSize: 12,
          }}>
            Nenhuma porta configurada — adicione uma acima
          </div>
        ) : (
          mappings.map((m) => (
            <div
              key={m.id}
              style={{
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'space-between',
                padding: '5px 0',
                borderBottom: '1px solid var(--border)',
              }}
            >
              <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
                <span style={{ fontFamily: 'monospace', fontSize: 13 }}>
                  :{m.listenPort} → :{m.wslPort}
                </span>
                <span style={{ color: 'var(--text-muted)', fontSize: 11 }}>Windows → WSL</span>
              </div>
              <button
                onClick={() => onRemove(m.id)}
                disabled={locked}
                style={{
                  background: 'transparent',
                  color: 'var(--red)',
                  padding: '2px 8px',
                  border: '1px solid var(--red)',
                  fontSize: 12,
                }}
              >
                ✕
              </button>
            </div>
          ))
        )}
      </div>
    </div>
  )
}
