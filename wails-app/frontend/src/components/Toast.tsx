interface Props {
  message: string | null
}

export function Toast({ message }: Props) {
  if (!message) return null

  return (
    <div style={{
      position: 'fixed',
      bottom: 16,
      right: 16,
      background: 'var(--red)',
      color: '#fff',
      padding: '10px 16px',
      borderRadius: 6,
      maxWidth: 320,
      fontSize: 13,
      boxShadow: '0 4px 12px rgba(0,0,0,0.4)',
      zIndex: 1000,
      wordBreak: 'break-word',
    }}>
      {message}
    </div>
  )
}
