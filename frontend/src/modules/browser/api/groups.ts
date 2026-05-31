import type { BrowserGroup, BrowserGroupInput, BrowserGroupWithCount } from '../types'
import { getBindings } from './runtime'

export async function fetchGroups(): Promise<BrowserGroupWithCount[]> {
  const bindings: any = await getBindings()
  if (bindings?.ListGroups) {
    return (await bindings.ListGroups()) || []
  }
  return []
}

export async function createGroup(input: BrowserGroupInput): Promise<BrowserGroup | null> {
  const bindings: any = await getBindings()
  if (bindings?.CreateGroup) {
    return (await bindings.CreateGroup(input)) || null
  }
  return null
}

export async function updateGroup(groupId: string, input: BrowserGroupInput): Promise<BrowserGroup | null> {
  const bindings: any = await getBindings()
  if (bindings?.UpdateGroup) {
    return (await bindings.UpdateGroup(groupId, input)) || null
  }
  return null
}

export async function deleteGroup(groupId: string): Promise<boolean> {
  const bindings: any = await getBindings()
  if (bindings?.DeleteGroup) {
    await bindings.DeleteGroup(groupId)
    return true
  }
  return false
}

export async function moveInstancesToGroup(profileIds: string[], groupId: string): Promise<boolean> {
  const bindings: any = await getBindings()
  if (bindings?.MoveInstancesToGroup) {
    await bindings.MoveInstancesToGroup(profileIds, groupId)
    return true
  }
  return false
}
