import { Button } from '../../../../shared/components'

interface ProxyPoolHeaderProps {
  checkingAllIPHealth: boolean
  hasURLImportSources: boolean
  onCheckAllIPHealth: () => void
  onOpenImport: () => void
  onOpenSettings: () => void
  onRefreshAllSources: () => void
  onTestAll: () => void
  refreshingAllSources: boolean
  testingAll: boolean
  totalCount: number
}

export function ProxyPoolHeader({
  checkingAllIPHealth,
  hasURLImportSources,
  onCheckAllIPHealth,
  onOpenImport,
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
        <p className="text-sm text-[var(--color-text-muted)] mt-1">管理代理配置，支持 Clash 订阅、HTTP、HTTPS、SOCKS5</p>
      </div>
      <div className="flex gap-2">
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
        <Button size="sm" onClick={onOpenImport}>导入代理</Button>
      </div>
    </div>
  )
}
