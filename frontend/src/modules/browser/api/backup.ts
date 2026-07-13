import { getBindings } from './runtime'

export interface BrowserBackupActionResult {
  cancelled?: boolean
  message?: string
  zipPath?: string
  resetFirst?: boolean
  imported?: number
  skipped?: number
  conflicts?: number
  partial?: boolean
  componentTotal?: number
  componentSuccess?: number
  componentFailed?: number
  failedComponents?: Array<{
    componentId?: string
    componentName?: string
    error?: string
  }>
}

export async function exportFullBrowserBackup(): Promise<BrowserBackupActionResult> {
  const bindings: any = await getBindings()
  if (!bindings?.BackupExportPackage) {
    return { cancelled: false, message: '当前环境不支持全量备份' }
  }
  return (await bindings.BackupExportPackage()) || {}
}

export async function importFullBrowserBackup(resetFirst: boolean): Promise<BrowserBackupActionResult> {
  const bindings: any = await getBindings()
  if (!bindings?.BackupImportPackage) {
    return { cancelled: false, message: '当前环境不支持导入备份' }
  }
  return (await bindings.BackupImportPackage(resetFirst)) || {}
}
