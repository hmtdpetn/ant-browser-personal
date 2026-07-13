import { applyBrowserProfileCopyOptionsToArgs, createBrowserProfileCopyOptions } from '../copyOptions'
import { buildBrowserProfileCopyName } from '../copyName'
import type {
  BrowserProfile,
  BrowserProfileCopyOptions,
  BrowserProfileInput,
  BrowserProfilePackageExportResult,
  BrowserProfilePackageImportResult,
} from '../types'
import { getBindings, getMockProfiles, nowISOString, setMockProfiles } from './runtime'

export async function fetchBrowserProfiles(): Promise<BrowserProfile[]> {
  const bindings: any = await getBindings()
  if (bindings?.BrowserProfileList) {
    return (await bindings.BrowserProfileList()) || []
  }
  return getMockProfiles().filter((profile) => !profile.deletedAt)
}

export async function fetchBrowserProfileTrash(): Promise<BrowserProfile[]> {
  const bindings: any = await getBindings()
  if (bindings?.BrowserProfileTrashList) {
    return (await bindings.BrowserProfileTrashList()) || []
  }
  return getMockProfiles().filter((profile) => !!profile.deletedAt)
}

export async function fetchBrowserProfilesByTag(tag: string): Promise<BrowserProfile[]> {
  const bindings: any = await getBindings()
  if (bindings?.BrowserProfileListByTag) {
    return (await bindings.BrowserProfileListByTag(tag)) || []
  }
  return getMockProfiles().filter((profile) => profile.tags?.includes(tag))
}

export async function fetchAllTags(): Promise<string[]> {
  const bindings: any = await getBindings()
  if (bindings?.BrowserGetAllTags) {
    return (await bindings.BrowserGetAllTags()) || []
  }

  const tags = new Set<string>()
  getMockProfiles().forEach((profile) => profile.tags?.forEach((tag) => tags.add(tag)))
  return Array.from(tags).sort()
}

export async function exportBrowserProfilePackage(profileIds: string[]): Promise<BrowserProfilePackageExportResult> {
  const bindings: any = await getBindings()
  if (bindings?.BrowserProfilePackageExport) {
    return await bindings.BrowserProfilePackageExport(profileIds)
  }
  return {
    cancelled: true,
    zipPath: '',
    profileCount: 0,
    fileCount: 0,
    message: '当前环境不支持导出实例',
  }
}

export async function importBrowserProfilePackage(): Promise<BrowserProfilePackageImportResult> {
  const bindings: any = await getBindings()
  if (bindings?.BrowserProfilePackageImport) {
    return await bindings.BrowserProfilePackageImport()
  }
  return {
    cancelled: true,
    importedCount: 0,
    profileMappings: {},
    message: '当前环境不支持导入实例',
  }
}

export async function createBrowserProfile(input: BrowserProfileInput): Promise<BrowserProfile | null> {
  const bindings: any = await getBindings()
  if (bindings?.BrowserProfileCreate) {
    return (await bindings.BrowserProfileCreate(input)) || null
  }

  const profile: BrowserProfile = {
    profileId: `mock-${Date.now()}`,
    ...input,
    keywords: input.keywords || [],
    running: false,
    debugPort: 0,
    debugReady: false,
    pid: 0,
    runtimeWarning: '',
    lastError: '',
    createdAt: nowISOString(),
    updatedAt: nowISOString(),
  }
  setMockProfiles([profile, ...getMockProfiles()])
  return profile
}

export async function updateBrowserProfile(profileId: string, input: BrowserProfileInput): Promise<BrowserProfile | null> {
  const bindings: any = await getBindings()
  if (bindings?.BrowserProfileUpdate) {
    return (await bindings.BrowserProfileUpdate(profileId, input)) || null
  }

  const profiles = getMockProfiles()
  const index = profiles.findIndex((item) => item.profileId === profileId)
  if (index === -1) {
    return null
  }

  const nextProfiles = [...profiles]
  nextProfiles[index] = { ...nextProfiles[index], ...input, updatedAt: nowISOString() }
  setMockProfiles(nextProfiles)
  return nextProfiles[index]
}

export async function deleteBrowserProfile(profileId: string): Promise<boolean> {
  const bindings: any = await getBindings()
  if (bindings?.BrowserProfileDelete) {
    await bindings.BrowserProfileDelete(profileId)
    return true
  }

  const deletedAt = nowISOString()
  setMockProfiles(getMockProfiles().map((item) => (
    item.profileId === profileId ? { ...item, deletedAt, updatedAt: deletedAt, running: false } : item
  )))
  return true
}

export async function restoreBrowserProfile(profileId: string): Promise<BrowserProfile | null> {
  const bindings: any = await getBindings()
  if (bindings?.BrowserProfileRestore) {
    return (await bindings.BrowserProfileRestore(profileId)) || null
  }

  const updatedAt = nowISOString()
  let restored: BrowserProfile | null = null
  const nextProfiles = getMockProfiles().map((item) => {
    if (item.profileId !== profileId) return item
    restored = { ...item, deletedAt: '', updatedAt }
    return restored
  })
  setMockProfiles(nextProfiles)
  return restored
}

export async function permanentlyDeleteBrowserProfile(profileId: string): Promise<boolean> {
  const bindings: any = await getBindings()
  if (bindings?.BrowserProfilePermanentlyDelete) {
    await bindings.BrowserProfilePermanentlyDelete(profileId)
    return true
  }

  setMockProfiles(getMockProfiles().filter((item) => item.profileId !== profileId))
  return true
}

export async function cleanupBrowserProfileTrash(): Promise<boolean> {
  const bindings: any = await getBindings()
  if (bindings?.BrowserProfileTrashCleanup) {
    await bindings.BrowserProfileTrashCleanup()
    return true
  }

  const expiredBefore = Date.now() - 3 * 24 * 60 * 60 * 1000
  setMockProfiles(getMockProfiles().filter((item) => {
    if (!item.deletedAt) return true
    const deletedAt = new Date(item.deletedAt).getTime()
    return Number.isNaN(deletedAt) || deletedAt > expiredBefore
  }))
  return true
}

export async function copyBrowserProfile(
  profileId: string,
  newName: string,
  options: BrowserProfileCopyOptions = createBrowserProfileCopyOptions(),
): Promise<BrowserProfile | null> {
  const bindings: any = await getBindings()
  if (bindings?.BrowserProfileCopyWithOptions) {
    return (await bindings.BrowserProfileCopyWithOptions(profileId, newName, options)) || null
  }
  if (bindings?.BrowserProfileCopyWithMode) {
    return (await bindings.BrowserProfileCopyWithMode(profileId, newName, options.mode)) || null
  }
  if (bindings?.BrowserProfileCopy) {
    return (await bindings.BrowserProfileCopy(profileId, newName)) || null
  }

  const source = getMockProfiles().find((profile) => profile.profileId === profileId)
  if (!source) {
    return null
  }

  const timestamp = Date.now()
  const launchCode = `MOCK_${timestamp.toString(36).toUpperCase().slice(-6).padStart(6, '0')}`
  const copy: BrowserProfile = {
    ...source,
    profileId: `mock-${timestamp}`,
    profileName: newName.trim() || buildBrowserProfileCopyName(source.profileName),
    userDataDir: `mock-${timestamp}`,
    fingerprintArgs: applyBrowserProfileCopyOptionsToArgs(
      source.fingerprintArgs || [],
      [],
      options,
    ),
    launchCode,
    running: false,
    debugReady: false,
    runtimeWarning: '',
    createdAt: nowISOString(),
    updatedAt: nowISOString(),
  }
  setMockProfiles([copy, ...getMockProfiles()])
  return copy
}

export async function setProfileKeywords(profileId: string, keywords: string[]): Promise<BrowserProfile | null> {
  const bindings: any = await getBindings()
  if (bindings?.BrowserProfileSetKeywords) {
    return (await bindings.BrowserProfileSetKeywords(profileId, keywords)) || null
  }

  const nextProfiles = getMockProfiles().map((profile) =>
    profile.profileId === profileId ? { ...profile, keywords, updatedAt: nowISOString() } : profile,
  )
  setMockProfiles(nextProfiles)
  return nextProfiles.find((profile) => profile.profileId === profileId) || null
}

export async function getBrowserProfileCode(profileId: string): Promise<string> {
  const bindings: any = await getBindings()
  if (bindings?.BrowserProfileGetCode) {
    return (await bindings.BrowserProfileGetCode(profileId)) || ''
  }
  return ''
}

export async function regenerateBrowserProfileCode(profileId: string): Promise<string> {
  const bindings: any = await getBindings()
  if (bindings?.BrowserProfileRegenerateCode) {
    return (await bindings.BrowserProfileRegenerateCode(profileId)) || ''
  }
  return ''
}

export async function setBrowserProfileCode(profileId: string, code: string): Promise<string> {
  const bindings: any = await getBindings()
  if (bindings?.BrowserProfileSetCode) {
    return (await bindings.BrowserProfileSetCode(profileId, code)) || ''
  }
  return code.trim().toUpperCase()
}

export async function batchSetProfileTags(profileIds: string[], tags: string[], replace: boolean): Promise<boolean> {
  const bindings: any = await getBindings()
  if (bindings?.BrowserProfileBatchSetTags) {
    await bindings.BrowserProfileBatchSetTags(profileIds, tags, replace)
    return true
  }
  return true
}

export async function batchRemoveProfileTags(profileIds: string[], tags: string[]): Promise<boolean> {
  const bindings: any = await getBindings()
  if (bindings?.BrowserProfileBatchRemoveTags) {
    await bindings.BrowserProfileBatchRemoveTags(profileIds, tags)
    return true
  }
  return true
}

export async function renameBrowserTag(oldName: string, newName: string): Promise<boolean> {
  const bindings: any = await getBindings()
  if (bindings?.BrowserRenameTag) {
    await bindings.BrowserRenameTag(oldName, newName)
    return true
  }
  return true
}
