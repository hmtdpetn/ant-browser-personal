import { Edit2, Settings } from 'lucide-react'
import { Button, Card } from '../../../../shared/components'
import type { BrowserSettings } from '../../types'

interface CoreSettingsCardProps {
  settings: BrowserSettings
  onEdit: () => void
}

const settingsValueClass = 'h-14 overflow-auto rounded-md bg-[var(--color-bg-subtle)] px-3 py-2 text-sm leading-5 text-[var(--color-text-primary)]'

export function CoreSettingsCard({ settings, onEdit }: CoreSettingsCardProps) {
  return (
    <Card>
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-2">
          <Settings className="w-5 h-5 text-[var(--color-text-muted)]" />
          <h3 className="text-base font-medium text-[var(--color-text-primary)]">全局设置</h3>
        </div>
        <Button size="sm" variant="ghost" onClick={onEdit}>
          <Edit2 className="w-4 h-4 mr-1" />
          编辑
        </Button>
      </div>
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <SettingsValue label="用户数据根目录" value={settings.userDataRoot || '-'} />
        <SettingsList label="默认指纹参数" values={settings.defaultFingerprintArgs} />
        <SettingsList label="默认启动参数" values={settings.defaultLaunchArgs} />
        <SettingsList label="默认启动页面" values={settings.defaultStartUrls} />
        <SettingsValue label="恢复上次标签页" value={settings.restoreLastSession ? '开启' : '关闭'} />
        <SettingsValue label="轻启动模式" value={settings.lightStartEnabled ? '开启' : '关闭'} />
        <SettingsValue label="启动就绪超时" value={`${settings.startReadyTimeoutMs} ms`} />
        <SettingsValue label="启动稳定窗口" value={`${settings.startStableWindowMs} ms`} />
      </div>
    </Card>
  )
}

function SettingsValue({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <p className="text-xs text-[var(--color-text-muted)] mb-1">{label}</p>
      <div className={`${settingsValueClass} break-all`}>
        {value}
      </div>
    </div>
  )
}

function SettingsList({ label, values }: { label: string; values: string[] }) {
  return (
    <div>
      <p className="text-xs text-[var(--color-text-muted)] mb-1">{label}</p>
      {values.length > 0 ? (
        <pre className={settingsValueClass}>
          {values.join('\n')}
        </pre>
      ) : (
        <div className={settingsValueClass}>
          -
        </div>
      )}
    </div>
  )
}
