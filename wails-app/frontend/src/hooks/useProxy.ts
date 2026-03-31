import { useState, useEffect, useCallback, useRef } from 'react'
import { EventsOn, EventsOff } from '../wailsjs/runtime/runtime'
import {
  GetWSLInfo,
  GetMappings,
  GetState,
  AddMapping,
  RemoveMapping,
  Start,
  Pause,
  Stop,
  ClearLog,
} from '../wailsjs/go/main/App'

export type ProxyState = 'stopped' | 'running' | 'paused'

export interface PortMapping {
  id: string
  listenPort: number
  wslPort: number
}

export interface WSLInfo {
  ip: string
  mode: string
  targetIP: string
}

export interface LogEntry {
  id: number
  text: string
  isError: boolean
}

export interface ConnectionPayload {
  time: string
  sourceIP: string
  listenPort: number
  wslPort: number
}

const MAX_LOG = 500

let logSeq = 0

export function useProxy() {
  const [state, setState] = useState<ProxyState>('stopped')
  const [mappings, setMappings] = useState<PortMapping[]>([])
  const [wslInfo, setWslInfo] = useState<WSLInfo | null>(null)
  const [log, setLog] = useState<LogEntry[]>([])
  const [connCount, setConnCount] = useState(0)
  const [toast, setToast] = useState<string | null>(null)
  const toastTimer = useRef<ReturnType<typeof setTimeout> | null>(null)

  // Load initial state on mount
  useEffect(() => {
    GetWSLInfo().then((info) => { if (info) setWslInfo(info) })
    GetMappings().then((m) => setMappings(m ?? []))
    GetState().then((s) => setState((s ?? 'stopped') as ProxyState))
  }, [])

  // Subscribe to backend events
  useEffect(() => {
    const onState = (payload: { state: ProxyState; wslInfo?: WSLInfo }) => {
      setState(payload.state)
      if (payload.wslInfo) setWslInfo(payload.wslInfo)
    }

    const onConnection = (payload: ConnectionPayload) => {
      const entry: LogEntry = {
        id: logSeq++,
        text: `${payload.time}  ${payload.sourceIP.padEnd(18)}  :${payload.listenPort} → :${payload.wslPort}`,
        isError: false,
      }
      setLog((prev) => {
        const next = prev.length >= MAX_LOG ? prev.slice(1) : prev
        return [...next, entry]
      })
    }

    const onError = (payload: { message: string }) => {
      const entry: LogEntry = {
        id: logSeq++,
        text: `${new Date().toLocaleTimeString('pt-BR', { hour12: false })}  ${payload.message}`,
        isError: true,
      }
      setLog((prev) => [...prev, entry])
      showToast(payload.message)
    }

    const onConnCount = (payload: { count: number }) => {
      setConnCount(payload.count)
    }

    EventsOn('proxy:state', (...data: unknown[]) => onState(data[0] as Parameters<typeof onState>[0]))
    EventsOn('proxy:connection', (...data: unknown[]) => onConnection(data[0] as ConnectionPayload))
    EventsOn('proxy:error', (...data: unknown[]) => onError(data[0] as { message: string }))
    EventsOn('proxy:connCount', (...data: unknown[]) => onConnCount(data[0] as { count: number }))

    return () => {
      EventsOff('proxy:state')
      EventsOff('proxy:connection')
      EventsOff('proxy:error')
      EventsOff('proxy:connCount')
    }
  }, [])

  const showToast = useCallback((msg: string) => {
    setToast(msg)
    if (toastTimer.current) clearTimeout(toastTimer.current)
    toastTimer.current = setTimeout(() => setToast(null), 3500)
  }, [])

  const addMapping = useCallback(async (spec: string) => {
    try {
      await AddMapping(spec)
      const updated = await GetMappings()
      setMappings(updated)
    } catch (e: unknown) {
      showToast(String(e))
    }
  }, [showToast])

  const removeMapping = useCallback(async (id: string) => {
    try {
      await RemoveMapping(id)
      const updated = await GetMappings()
      setMappings(updated)
    } catch (e: unknown) {
      showToast(String(e))
    }
  }, [showToast])

  const startProxy = useCallback(async () => {
    try {
      await Start()
    } catch (e: unknown) {
      showToast(String(e))
    }
  }, [showToast])

  const pauseProxy = useCallback(() => Pause(), [])
  const stopProxy = useCallback(() => Stop(), [])

  const clearLog = useCallback(() => {
    setLog([])
    setConnCount(0)
    ClearLog()
  }, [])

  return {
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
  }
}
