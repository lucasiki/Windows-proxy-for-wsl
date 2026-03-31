import { useProxy } from './hooks/useProxy'
import { Header } from './components/Header'
import { PortList } from './components/PortList'
import { Controls } from './components/Controls'
import { LogPanel } from './components/LogPanel'
import { Toast } from './components/Toast'

export default function App() {
  const {
    state,
    mappings,
    wslInfo,
    log,
    connCount,
    toast,
    addMapping,
    removeMapping,
    startProxy,
    pauseProxy,
    stopProxy,
    clearLog,
  } = useProxy()

  return (
    <div style={{
      display: 'flex',
      flexDirection: 'column',
      height: '100vh',
      background: 'var(--bg-primary)',
    }}>
      <Header state={state} wslInfo={wslInfo} />

      <div style={{
        display: 'flex',
        flexDirection: 'column',
        flex: 1,
        padding: 14,
        gap: 12,
        overflow: 'hidden',
      }}>
        {/* Port panel: fixed height, controls attached at bottom */}
        <div style={{
          background: 'var(--bg-secondary)',
          borderRadius: 6,
          border: '1px solid var(--border)',
          overflow: 'hidden',
          flexShrink: 0,
        }}>
          <PortList
            mappings={mappings}
            state={state}
            onAdd={addMapping}
            onRemove={removeMapping}
          />
          <Controls
            state={state}
            onStart={startProxy}
            onPause={pauseProxy}
            onStop={stopProxy}
          />
        </div>

        {/* Log panel: grows to fill remaining space */}
        <LogPanel entries={log} connCount={connCount} onClear={clearLog} />
      </div>

      <Toast message={toast} />
    </div>
  )
}
