import { Button } from '../../../../shared/components'

interface ProxyPoolHeaderProps {
  checkingAllIPHealth: boolean
  currentConnectorStatus: string
  hasURLImportSources: boolean
  onCheckAllIPHealth: () => void
  onOpenImport: () => void
  onOpenCoreDownload: () => void
  onOpenSettings: () => void
  onRefreshAllSources: () => void
  onTestAll: () => void
  refreshingAllSources: boolean
  testingAll: boolean
  totalCount: number
}

export function ProxyPoolHeader({
  checkingAllIPHealth,
  currentConnectorStatus,
  hasURLImportSources,
  onCheckAllIPHealth,
  onOpenImport,
  onOpenCoreDownload,
  onOpenSettings,
  onRefreshAllSources,
  onTestAll,
  refreshingAllSources,
  testingAll,
  totalCount,
}: ProxyPoolHeaderProps) {
  return (
    <div className="flex items-center justify-between">
      <div>
        <h1 className="text-xl font-semibold text-[var(--color-text-primary)]">代理池配置</h1>
      </div>
      <div className="flex items-center gap-2">
        <div className="flex items-center gap-2 rounded-xl border border-[var(--color-border-default)] bg-[var(--color-bg-primary)] px-2 py-1 shadow-sm">
          <span className="inline-flex items-center gap-1 whitespace-nowrap px-2 text-xs text-[var(--color-text-muted)]">
            内核状态：{currentConnectorStatus || '未知'}
          </span>
          <Button size="sm" variant="secondary" onClick={onOpenCoreDownload}>下载内核</Button>
        </div>
        <div className="flex items-center gap-2 rounded-xl border border-[var(--color-border-default)] bg-[var(--color-bg-primary)] px-2 py-1 shadow-sm">
          <Button
            size="sm"
            variant="secondary"
            onClick={onOpenSettings}
          >
            检测设置
          </Button>
          <Button
            size="sm"
            variant="secondary"
            onClick={onRefreshAllSources}
            loading={refreshingAllSources}
            disabled={!hasURLImportSources}
          >
            刷新订阅
          </Button>
          <Button
            size="sm"
            variant="secondary"
            onClick={onCheckAllIPHealth}
            loading={checkingAllIPHealth}
            disabled={totalCount === 0}
          >
            检测IP健康
          </Button>
          <Button
            size="sm"
            variant="secondary"
            onClick={onTestAll}
            loading={testingAll}
            disabled={totalCount === 0}
          >
            测试全部
          </Button>
        </div>
        <Button size="sm" onClick={onOpenImport}>导入代理</Button>
      </div>
    </div>
  )
}
