import type { BrowserProxyGroup, BrowserProxyGroupInput, BrowserProxyGroupWithCount } from '../types'
import { getBindings } from './runtime'

export async function fetchProxyGroups(): Promise<BrowserProxyGroupWithCount[]> {
  const bindings: any = await getBindings()
  if (bindings?.BrowserProxyGroupList) {
    return (await bindings.BrowserProxyGroupList()) || []
  }
  return []
}

export async function createProxyGroup(input: BrowserProxyGroupInput): Promise<BrowserProxyGroup | null> {
  const bindings: any = await getBindings()
  if (bindings?.BrowserProxyGroupCreate) {
    return (await bindings.BrowserProxyGroupCreate(input)) || null
  }
  return null
}

export async function updateProxyGroup(groupId: string, input: BrowserProxyGroupInput): Promise<BrowserProxyGroup | null> {
  const bindings: any = await getBindings()
  if (bindings?.BrowserProxyGroupUpdate) {
    return (await bindings.BrowserProxyGroupUpdate(groupId, input)) || null
  }
  return null
}

export async function deleteProxyGroup(groupId: string): Promise<boolean> {
  const bindings: any = await getBindings()
  if (bindings?.BrowserProxyGroupDelete) {
    await bindings.BrowserProxyGroupDelete(groupId)
    return true
  }
  return false
}

export async function moveProxiesToGroup(proxyIds: string[], groupId: string): Promise<boolean> {
  const bindings: any = await getBindings()
  if (bindings?.BrowserProxyMoveToGroup) {
    await bindings.BrowserProxyMoveToGroup(proxyIds, groupId)
    return true
  }
  return false
}
