import { AlertTriangle, DatabaseBackup, Download, Upload } from 'lucide-react'

import { Button, Modal } from '../../../shared/components'

type BackupMode = 'export' | 'import-merge' | 'import-reset' | 'none'

interface BrowserBackupModalProps {
  open: boolean
  runningCount: number
  selectedCount: number
  selectedExporting: boolean
  loadingMode: BackupMode
  onClose: () => void
  onExportSelected: () => void
  onExportFull: () => void
  onImportMerge: () => void
  onImportReset: () => void
}

export function BrowserBackupModal({
  open,
  runningCount,
  selectedCount,
  selectedExporting,
  loadingMode,
  onClose,
  onExportSelected,
  onExportFull,
  onImportMerge,
  onImportReset,
}: BrowserBackupModalProps) {
  const busy = loadingMode !== 'none' || selectedExporting

  return (
    <Modal
      open={open}
      onClose={() => {
        if (!busy) onClose()
      }}
      title="备份与导入"
      width="560px"
      closable={!busy}
      footer={(
        <>
          <Button variant="secondary" onClick={onClose} disabled={busy}>关闭</Button>
          <Button variant="secondary" onClick={onImportMerge} loading={loadingMode === 'import-merge'} disabled={busy && loadingMode !== 'import-merge'}>
            <Upload className="w-4 h-4" />合并导入
          </Button>
          <Button variant="danger" onClick={onImportReset} loading={loadingMode === 'import-reset'} disabled={busy && loadingMode !== 'import-reset'}>
            清空恢复
          </Button>
          <Button variant="secondary" onClick={onExportSelected} loading={selectedExporting} disabled={selectedCount === 0 || (busy && !selectedExporting)}>
            <Download className="w-4 h-4" />备份选中
          </Button>
          <Button onClick={onExportFull} loading={loadingMode === 'export'} disabled={busy && loadingMode !== 'export'}>
            <Download className="w-4 h-4" />全量备份
          </Button>
        </>
      )}
    >
      <div className="space-y-4 text-sm text-[var(--color-text-secondary)]">
        <div className="rounded-lg border border-[var(--color-border-default)] bg-[var(--color-bg-secondary)] p-3">
          <div className="flex items-center gap-2 font-medium text-[var(--color-text-primary)]">
            <DatabaseBackup className="w-4 h-4" />
            <span>全量备份范围</span>
          </div>
          <div className="mt-2 grid grid-cols-2 gap-x-4 gap-y-1 text-xs text-[var(--color-text-muted)]">
            <span>实例名称 / 分组 / 标签</span>
            <span>代理池 / 订阅 / 测速结果</span>
            <span>内核配置 / 内核文件</span>
            <span>应用书签 / 插件配置</span>
            <span>实例浏览器数据目录</span>
            <span>Cookie / LocalStorage / IndexedDB</span>
          </div>
        </div>

        <div className="rounded-lg border border-[var(--color-border-default)] p-3 text-xs leading-5 text-[var(--color-text-muted)]">
          <p><span className="font-medium text-[var(--color-text-secondary)]">备份选中：</span>导出当前选中的 {selectedCount} 个实例及浏览器用户数据，适合迁移少量实例。</p>
          <p>完整灾备请用全量备份，它会额外包含完整代理池、内核、数据库和应用级配置。</p>
        </div>

        {runningCount > 0 && (
          <div className="flex gap-2 rounded-lg border border-[var(--color-warning)]/40 bg-[var(--color-warning)]/10 p-3 text-xs text-[var(--color-warning)]">
            <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0" />
            <span>当前有 {runningCount} 个实例运行中。建议先停止实例再备份，否则 Cookie、数据库和缓存文件可能未完整落盘。</span>
          </div>
        )}

        <div className="rounded-lg border border-[var(--color-border-default)] p-3 text-xs leading-5 text-[var(--color-text-muted)]">
          <p>登录态会随浏览器用户数据一起打包，但 Windows 加密信息可能绑定当前系统用户。</p>
          <p>同机同用户恢复成功率最高；跨机器、重装系统或换 Windows 用户时，Cookie 和密码可能无法解密。</p>
        </div>

        <div className="grid grid-cols-1 gap-2 text-xs text-[var(--color-text-muted)]">
          <p><span className="font-medium text-[var(--color-text-secondary)]">全量备份：</span>导出配置、数据库、代理、内核、实例浏览器数据。</p>
          <p><span className="font-medium text-[var(--color-text-secondary)]">合并导入：</span>保留当前数据，按 ID、路径、URL 等规则跳过重复项。</p>
          <p><span className="font-medium text-[var(--color-text-secondary)]">清空恢复：</span>先初始化当前数据，再从备份包完整恢复。</p>
        </div>
      </div>
    </Modal>
  )
}
