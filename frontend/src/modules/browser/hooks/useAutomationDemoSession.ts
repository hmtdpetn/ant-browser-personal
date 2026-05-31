import { useEffect, useRef, useState } from 'react'
import { toast } from '../../../shared/components'
import { fetchBrowserProfiles, type AutomationDemoResult } from '../api'
import {
  applyAutomationDemoResult,
  buildAutomationDemoErrorResult,
  loadAutomationDemoSession,
  reconcileAutomationDemoSession,
  saveAutomationDemoSession,
  type AutomationDemoActionKey,
} from '../demoSession'
import { DEFAULT_LAUNCH_BASE_URL } from '../launchContext'

interface UseAutomationDemoSessionOptions {
  enabled?: boolean
  baseUrl?: string
}

interface RunAutomationDemoActionOptions {
  actionKey: AutomationDemoActionKey
  actionLabel: string
  runner: () => Promise<AutomationDemoResult>
  successMessage: string
  failureMessage: string
}

export function useAutomationDemoSession({
  enabled = true,
  baseUrl = DEFAULT_LAUNCH_BASE_URL,
}: UseAutomationDemoSessionOptions = {}) {
  const mountedRef = useRef(true)
  const [demoSession, setDemoSession] = useState(() => loadAutomationDemoSession())
  const [demoBusyAction, setDemoBusyAction] = useState<AutomationDemoActionKey>('')

  const reloadDemoSession = () => {
    const nextSession = loadAutomationDemoSession()
    if (mountedRef.current) {
      setDemoSession(nextSession)
    }
    return nextSession
  }

  const refreshDemoProfiles = async (showError = false) => {
    try {
      const profiles = await fetchBrowserProfiles()
      if (mountedRef.current) {
        setDemoSession((current) => reconcileAutomationDemoSession(current, profiles))
      }
      return profiles
    } catch (error: unknown) {
      if (showError) {
        const message = error instanceof Error ? error.message : '演示实例状态刷新失败'
        toast.error(message)
      }
      return null
    }
  }

  const runDemoAction = async ({
    actionKey,
    actionLabel,
    runner,
    successMessage,
    failureMessage,
  }: RunAutomationDemoActionOptions) => {
    if (!mountedRef.current) {
      return null
    }

    setDemoBusyAction(actionKey)
    try {
      const result = await runner()
      if (mountedRef.current) {
        setDemoSession((current) => applyAutomationDemoResult(current, actionLabel, result))
      }

      if (result.ok) {
        toast.success(successMessage)
      } else {
        toast.error(result.error || failureMessage)
      }
      return result
    } catch (error: unknown) {
      const message = error instanceof Error ? error.message : failureMessage
      if (mountedRef.current) {
        setDemoSession((current) => ({
          ...current,
          lastAction: actionLabel,
          lastResult: buildAutomationDemoErrorResult(message, baseUrl),
        }))
      }
      toast.error(message)
      return null
    } finally {
      if (mountedRef.current) {
        setDemoBusyAction('')
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
      return
    }

    void refreshDemoProfiles(false)
  }, [enabled])

  useEffect(() => {
    saveAutomationDemoSession(demoSession)
  }, [demoSession])

  return {
    demoSession,
    setDemoSession,
    reloadDemoSession,
    demoBusyAction,
    demoBusy: demoBusyAction !== '',
    demoResponseText: demoSession.lastResult ? JSON.stringify(demoSession.lastResult.response || {}, null, 2) : '',
    refreshDemoProfiles,
    runDemoAction,
  }
}
