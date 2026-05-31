import type { ProxyCheckSettings } from '../types'
import { getBindings } from './runtime'

export function createDefaultProxyCheckSettings(): ProxyCheckSettings {
  return {
    bridgeStartTimeoutMs: 15000,
    speedTargetId: '',
    ipHealthTargetId: '',
    targets: [],
  }
}

export async function fetchProxyCheckSettings(): Promise<ProxyCheckSettings> {
  const bindings: any = await getBindings()
  if (bindings?.GetProxyCheckSettings) {
    return (await bindings.GetProxyCheckSettings()) || createDefaultProxyCheckSettings()
  }
  return createDefaultProxyCheckSettings()
}

export async function saveProxyCheckSettings(settings: ProxyCheckSettings): Promise<boolean> {
  const bindings: any = await getBindings()
  if (bindings?.SaveProxyCheckSettings) {
    await bindings.SaveProxyCheckSettings(settings)
    return true
  }
  return true
}
