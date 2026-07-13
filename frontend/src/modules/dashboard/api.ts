import type { DashboardStats } from './types'

const getBindings = async () => {
  try {
    return await import('../../wailsjs/go/main/App')
  } catch {
    return null
  }
}

export async function fetchDashboardStats(): Promise<DashboardStats> {
  const bindings: any = await getBindings()
  if (bindings?.GetDashboardStats) {
    try {
      const data = await bindings.GetDashboardStats()
      return {
        totalInstances: data?.totalInstances ?? 0,
        runningInstances: data?.runningInstances ?? 0,
        proxyCount: data?.proxyCount ?? 0,
        coreCount: data?.coreCount ?? 0,
        memUsedMB: data?.memUsedMB ?? 0,
        appVersion: data?.appVersion ?? 'unknown',
      }
    } catch (e) {
      console.error('fetchDashboardStats error:', e)
    }
  }
  return { totalInstances: 0, runningInstances: 0, proxyCount: 0, coreCount: 0, memUsedMB: 0, appVersion: 'unknown' }
}

export async function reloadConfig(): Promise<void> {
  const bindings: any = await getBindings()
  if (bindings?.ReloadConfig) {
    try {
      await bindings.ReloadConfig()
    } catch (e) {
      console.error('reloadConfig error:', e)
    }
  }
}
