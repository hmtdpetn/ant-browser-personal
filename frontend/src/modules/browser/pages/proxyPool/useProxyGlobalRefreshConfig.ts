import { useEffect, useMemo, useState } from 'react'
import { normalizeRefreshIntervalM } from './helpers'
import { readGlobalRefreshConfig, writeGlobalRefreshConfig } from './storage'

export function useProxyGlobalRefreshConfig() {
  const [globalAutoRefreshEnabled, setGlobalAutoRefreshEnabled] = useState(false)
  const [globalRefreshIntervalM, setGlobalRefreshIntervalM] = useState('60')
  const globalRefreshInterval = useMemo(() => {
    const interval = normalizeRefreshIntervalM(Number(globalRefreshIntervalM || 0))
    return interval > 0 ? interval : 60
  }, [globalRefreshIntervalM])

  useEffect(() => {
    const cfg = readGlobalRefreshConfig()
    setGlobalAutoRefreshEnabled(cfg.enabled)
    setGlobalRefreshIntervalM(String(cfg.intervalM))
  }, [])

  useEffect(() => {
    writeGlobalRefreshConfig(globalAutoRefreshEnabled, globalRefreshInterval)
  }, [globalAutoRefreshEnabled, globalRefreshInterval])

  return {
    globalAutoRefreshEnabled,
    setGlobalAutoRefreshEnabled,
    globalRefreshInterval,
    globalRefreshIntervalM,
    setGlobalRefreshIntervalM,
  }
}
