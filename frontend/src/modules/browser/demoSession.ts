import type { AutomationDemoResult } from './api'
import type { BrowserProfile } from './types'

export interface AutomationDemoSession {
  profileId: string
  profileName: string
  launchCode: string
  cdpUrl: string
  debugPort: number
  lastAction: string
  lastResult: AutomationDemoResult | null
}

export type AutomationDemoActionKey = '' | 'health' | 'create' | 'launch' | 'delete'

const AUTOMATION_DEMO_SESSION_KEY = 'automation_demo_session_v1'

export const EMPTY_AUTOMATION_DEMO_SESSION: AutomationDemoSession = {
  profileId: '',
  profileName: '',
  launchCode: '',
  cdpUrl: '',
  debugPort: 0,
  lastAction: '',
  lastResult: null,
}

export function buildAutomationDemoErrorResult(message: string, baseUrl: string): AutomationDemoResult {
  return {
    ok: false,
    status: 0,
    method: 'LOCAL',
    path: '',
    baseUrl,
    requestedAt: new Date().toISOString(),
    error: message,
    requestedCode: '',
    profileId: '',
    profileName: '',
    launchCode: '',
    cdpUrl: '',
    debugPort: 0,
    created: false,
    launched: false,
    deleted: false,
    stoppedBeforeDelete: false,
    stopError: '',
    authHeader: '',
    response: { ok: false, error: message },
  }
}

export function applyAutomationDemoResult(
  current: AutomationDemoSession,
  actionLabel: string,
  result: AutomationDemoResult
): AutomationDemoSession {
  const next: AutomationDemoSession = {
    ...current,
    lastAction: actionLabel,
    lastResult: result,
  }

  if (result.deleted && result.ok) {
    return {
      ...next,
      profileId: '',
      profileName: '',
      launchCode: '',
      cdpUrl: '',
      debugPort: 0,
    }
  }

  if (result.profileId) {
    next.profileId = result.profileId
  }
  if (result.profileName) {
    next.profileName = result.profileName
  }
  if (result.launchCode) {
    next.launchCode = result.launchCode
  }
  if (result.cdpUrl) {
    next.cdpUrl = result.cdpUrl
  }
  if (result.debugPort > 0) {
    next.debugPort = result.debugPort
  }

  return next
}

function normalizeLaunchCode(value?: string): string {
  return String(value || '').trim().toUpperCase()
}

export function reconcileAutomationDemoSession(
  current: AutomationDemoSession,
  profiles: BrowserProfile[]
): AutomationDemoSession {
  const profileId = String(current.profileId || '').trim()
  const launchCode = normalizeLaunchCode(current.launchCode)
  if (!profileId && !launchCode) {
    return current
  }

  const matched = profiles.find((item) => item.profileId === profileId)
    || profiles.find((item) => normalizeLaunchCode(item.launchCode) === launchCode)

  if (!matched) {
    return {
      ...current,
      profileId: '',
      profileName: '',
      launchCode: '',
      cdpUrl: '',
      debugPort: 0,
    }
  }

  const next: AutomationDemoSession = {
    ...current,
    profileId: matched.profileId,
    profileName: matched.profileName || current.profileName,
    launchCode: matched.launchCode || current.launchCode,
    debugPort: matched.running && matched.debugPort > 0 ? matched.debugPort : 0,
  }

  if (!matched.running || !matched.debugReady) {
    next.cdpUrl = ''
  }

  return next
}

export function loadAutomationDemoSession(): AutomationDemoSession {
  if (typeof window === 'undefined' || !window.localStorage) {
    return EMPTY_AUTOMATION_DEMO_SESSION
  }

  try {
    const raw = window.localStorage.getItem(AUTOMATION_DEMO_SESSION_KEY)
    if (!raw) {
      return EMPTY_AUTOMATION_DEMO_SESSION
    }

    const parsed = JSON.parse(raw)
    return {
      ...EMPTY_AUTOMATION_DEMO_SESSION,
      ...(parsed || {}),
      debugPort: Number(parsed?.debugPort) || 0,
      lastResult: parsed?.lastResult && typeof parsed.lastResult === 'object'
        ? parsed.lastResult as AutomationDemoResult
        : null,
    }
  } catch {
    return EMPTY_AUTOMATION_DEMO_SESSION
  }
}

export function saveAutomationDemoSession(session: AutomationDemoSession) {
  if (typeof window === 'undefined' || !window.localStorage) {
    return
  }

  try {
    window.localStorage.setItem(AUTOMATION_DEMO_SESSION_KEY, JSON.stringify(session))
  } catch {
    // Ignore storage failures and keep the page usable.
  }
}
