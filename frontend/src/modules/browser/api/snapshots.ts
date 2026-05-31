import type { SnapshotInfo } from '../types'
import { getBindings, nowISOString } from './runtime'

export async function listSnapshots(profileId: string): Promise<SnapshotInfo[]> {
  const bindings: any = await getBindings()
  if (bindings?.BrowserSnapshotList) {
    return (await bindings.BrowserSnapshotList(profileId)) || []
  }
  return []
}

export async function createSnapshot(profileId: string, name: string): Promise<SnapshotInfo | null> {
  const bindings: any = await getBindings()
  if (bindings?.BrowserSnapshotCreate) {
    return (await bindings.BrowserSnapshotCreate(profileId, name)) || null
  }
  return {
    snapshotId: `snap-${Date.now()}`,
    profileId,
    name,
    sizeMB: 12.5,
    createdAt: nowISOString(),
  }
}

export async function restoreSnapshot(profileId: string, snapshotId: string): Promise<boolean> {
  const bindings: any = await getBindings()
  if (bindings?.BrowserSnapshotRestore) {
    await bindings.BrowserSnapshotRestore(profileId, snapshotId)
    return true
  }
  return true
}

export async function deleteSnapshot(profileId: string, snapshotId: string): Promise<boolean> {
  const bindings: any = await getBindings()
  if (bindings?.BrowserSnapshotDelete) {
    await bindings.BrowserSnapshotDelete(profileId, snapshotId)
    return true
  }
  return true
}
