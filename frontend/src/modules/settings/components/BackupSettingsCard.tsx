import type { RefObject } from 'react'
import { Download, RotateCcw, Upload } from 'lucide-react'

import { Button, Card, Modal, Progress } from '../../../shared/components'

import type { BackupExportLogItem, BackupExportProgress } from '../progress'

type BackupActionLoading = 'none' | 'init' | 'export' | 'import-reset' | 'import-merge'

interface BackupProgressPanelProps {
  progress: BackupExportProgress
  loadingLabel: string
  logs?: BackupExportLogItem[]
  logsRef?: RefObject<HTMLDivElement>
}

interface BackupSettingsCardProps {
  actionLoading: BackupActionLoading
  exportProgress: BackupExportProgress | null
  exportLogs: BackupExportLogItem[]
  exportLogsRef: RefObject<HTMLDivElement>
  onInitialize: () => void
  onExport: () => void
  onOpenImport: () => void
}

interface BackupImportModalProps {
  open: boolean
  actionLoading: BackupActionLoading
  importProgress: BackupExportProgress | null
  onClose: () => void
  onImport: (resetFirst: boolean) => void
}

function BackupProgressPanel({ progress, loadingLabel, logs = [], logsRef }: BackupProgressPanelProps) {
  return (
    <div className="rounded-md border border-[var(--color-border-default)] bg-[var(--color-bg-secondary)] px-3 py-2 space-y-2">
      <div className="flex items-center justify-between text-xs">
        <span className="text-[var(--color-text-secondary)]">{progress.message}</span>
        {progress.phase === 'error' && <span className="text-[var(--color-error)]">失败</span>}
        {progress.phase === 'done' && <span className="text-[var(--color-success)]">完成</span>}
        {progress.phase !== 'done' && progress.phase !== 'error' && (
          <span className="text-[var(--color-text-muted)]">{loadingLabel}</span>
        )}
      </div>
      {(progress.componentName || progress.componentId || logsRef) && (
        <div className="text-xs text-[var(--color-text-muted)]">
          当前组件：
          {' '}
          {progress.componentName || progress.componentId || '准备中'}
          {progress.entryIndex && progress.entryTotal
            ? `（${progress.entryIndex}/${progress.entryTotal}）`
            : ''}
        </div>
      )}
      <Progress
        percent={progress.progress}
        size="sm"
        status={progress.phase === 'error' ? 'error' : progress.phase === 'done' ? 'success' : 'normal'}
      />
      {logsRef && (
        <div className="rounded border border-[var(--color-border-muted)] bg-[var(--color-bg-primary)] px-2 py-2">
          <div className="flex items-center justify-between text-xs mb-1">
            <span className="text-[var(--color-text-secondary)]">导出日志</span>
            <span className="text-[var(--color-text-muted)]">{logs.length} 条</span>
          </div>
          <div ref={logsRef} className="max-h-36 overflow-y-auto pr-1 space-y-1">
            {logs.length === 0 && (
              <p className="text-xs text-[var(--color-text-muted)]">等待导出日志...</p>
            )}
            {logs.map(item => (
              <div key={item.id} className="text-xs leading-5 font-mono">
                <span className="text-[var(--color-text-muted)] mr-2">{item.time}</span>
                <span className={item.phase === 'error' ? 'text-[var(--color-error)]' : item.phase === 'done' ? 'text-[var(--color-success)]' : 'text-[var(--color-text-secondary)]'}>
                  {item.text}
                </span>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}

export function BackupSettingsCard({
  actionLoading,
  exportProgress,
  exportLogs,
  exportLogsRef,
  onInitialize,
  onExport,
  onOpenImport,
}: BackupSettingsCardProps) {
  return (
    <Card title="配置备份与恢复" subtitle="初始化、导出、加载全量配置与浏览器数据">
      <div className="space-y-3">
        <p className="text-xs text-[var(--color-text-muted)]">
          加载配置时可选择先初始化后全量恢复，或在现有数据上按规则判重合并。
        </p>
        <div className="flex flex-wrap gap-2">
          <Button
            variant="danger"
            size="sm"
            onClick={onInitialize}
            loading={actionLoading === 'init'}
          >
            <RotateCcw className="w-4 h-4" />
            初始化系统
          </Button>
          <Button
            variant="secondary"
            size="sm"
            onClick={onExport}
            loading={actionLoading === 'export'}
          >
            <Download className="w-4 h-4" />
            导出配置
          </Button>
          <Button size="sm" onClick={onOpenImport}>
            <Upload className="w-4 h-4" />
            加载配置
          </Button>
        </div>
        {exportProgress && (
          <BackupProgressPanel
            progress={exportProgress}
            loadingLabel="处理中"
            logs={exportLogs}
            logsRef={exportLogsRef}
          />
        )}
      </div>
    </Card>
  )
}

export function BackupImportModal({
  open,
  actionLoading,
  importProgress,
  onClose,
  onImport,
}: BackupImportModalProps) {
  const importRunning = actionLoading === 'import-reset' || actionLoading === 'import-merge'

  return (
    <Modal
      open={open}
      onClose={() => {
        if (actionLoading !== 'none') {
          return
        }
        onClose()
      }}
      title="加载配置"
      width="520px"
      closable={!importRunning}
      footer={(
        <>
          {!importRunning && (
            <Button variant="secondary" onClick={onClose}>
              取消
            </Button>
          )}
          <Button
            variant="danger"
            onClick={() => onImport(true)}
            loading={actionLoading === 'import-reset'}
            disabled={actionLoading !== 'none' && actionLoading !== 'import-reset'}
          >
            是，先初始化后加载
          </Button>
          <Button
            onClick={() => onImport(false)}
            loading={actionLoading === 'import-merge'}
            disabled={actionLoading !== 'none' && actionLoading !== 'import-merge'}
          >
            否，直接加载并判重
          </Button>
        </>
      )}
    >
      <div className="space-y-3 text-sm text-[var(--color-text-secondary)]">
        <p>是否先执行初始化再加载 ZIP 配置？</p>
        <p className="text-xs text-[var(--color-text-muted)]">
          选择“是”会先清空当前数据，再全量恢复；选择“否”会在现有数据上做判重合并。
        </p>
        {importProgress && (
          <BackupProgressPanel progress={importProgress} loadingLabel="加载中" />
        )}
        {importRunning && (
          <p className="text-xs text-[var(--color-warning)]">
            当前正在加载配置，弹窗不可关闭。若需中断，请直接关闭应用。
          </p>
        )}
      </div>
    </Modal>
  )
}
