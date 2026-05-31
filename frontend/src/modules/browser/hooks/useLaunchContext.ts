import { useEffect, useRef, useState } from 'react'
import { toast } from '../../../shared/components'
import { fetchLaunchServerInfo, type LaunchServerInfo } from '../api'
import { DEFAULT_API_AUTH, DEFAULT_LAUNCH_BASE_URL } from '../launchContext'

interface UseLaunchContextOptions {
  enabled?: boolean
}

export function useLaunchContext({ enabled = true }: UseLaunchContextOptions = {}) {
  const mountedRef = useRef(true)
  const [launchBaseUrl, setLaunchBaseUrl] = useState(DEFAULT_LAUNCH_BASE_URL)
  const [launchServerReady, setLaunchServerReady] = useState(false)
  const [apiAuth, setApiAuth] = useState<LaunchServerInfo['apiAuth']>(DEFAULT_API_AUTH)
  const [launchContextLoading, setLaunchContextLoading] = useState(enabled)

  const applyLaunchInfo = (info: LaunchServerInfo) => {
    if (!mountedRef.current) {
      return
    }

    setLaunchBaseUrl(info.baseUrl || DEFAULT_LAUNCH_BASE_URL)
    setLaunchServerReady(info.ready)
    setApiAuth(info.apiAuth)
  }

  const refreshLaunchContext = async (showError = false) => {
    if (!mountedRef.current) {
      return null
    }

    setLaunchContextLoading(true)
    try {
      const info = await fetchLaunchServerInfo()
      applyLaunchInfo(info)
      return info
    } catch (error: unknown) {
      if (showError) {
        const message = error instanceof Error ? error.message : 'Launch API 状态刷新失败'
        toast.error(message)
      }
      return null
    } finally {
      if (mountedRef.current) {
        setLaunchContextLoading(false)
      }
    }
  }

  useEffect(() => {
    return () => {
      mountedRef.current = false
    }
  }, [])

  useEffect(() => {
    if (!enabled) {
      setLaunchContextLoading(false)
      return
    }

    void refreshLaunchContext(false)
  }, [enabled])

  return {
    launchBaseUrl,
    launchServerReady,
    apiAuth,
    launchContextLoading,
    refreshLaunchContext,
  }
}
