import { Input, Select } from '../../../shared/components'
import { DEFAULT_CLASH_SUBSCRIPTION_USER_AGENTS } from '../api/proxies'

export const CUSTOM_SUBSCRIPTION_USER_AGENT = '__custom__'

interface SubscriptionUserAgentFieldsProps {
  presetValue: string
  customValue: string
  fallbackEnabled: boolean
  disabled?: boolean
  onPresetChange: (value: string) => void
  onCustomChange: (value: string) => void
  onFallbackEnabledChange: (enabled: boolean) => void
}

export function resolveSubscriptionUserAgent(presetValue: string, customValue: string): string {
  return presetValue === CUSTOM_SUBSCRIPTION_USER_AGENT ? customValue.trim() : presetValue.trim()
}

export function SubscriptionUserAgentFields({
  presetValue,
  customValue,
  fallbackEnabled,
  disabled = false,
  onPresetChange,
  onCustomChange,
  onFallbackEnabledChange,
}: SubscriptionUserAgentFieldsProps) {
  return (
    <div className="rounded-xl border border-[var(--color-border)] bg-[var(--color-bg-secondary)]/55 p-3 space-y-3">
      <div className="flex items-start justify-between gap-3">
        <div>
          <div className="text-sm font-medium text-[var(--color-text-primary)]">订阅 User-Agent</div>
          <p className="mt-0.5 text-xs text-[var(--color-text-muted)]">
            可选择 Ant / FlClash 内置值，也可以输入自定义标识。
          </p>
        </div>
        <span className="shrink-0 rounded-full bg-[var(--color-primary)]/10 px-2 py-1 text-[11px] font-medium text-[var(--color-primary)]">
          可切换
        </span>
      </div>
      <Select
        value={presetValue}
        disabled={disabled}
        onChange={(event) => onPresetChange(event.target.value)}
        options={[
          ...DEFAULT_CLASH_SUBSCRIPTION_USER_AGENTS.map((item) => ({
            value: item.userAgent,
            label: item.label,
          })),
          { value: CUSTOM_SUBSCRIPTION_USER_AGENT, label: '自定义 User-Agent…' },
        ]}
      />
      {presetValue === CUSTOM_SUBSCRIPTION_USER_AGENT && (
        <Input
          value={customValue}
          disabled={disabled}
          maxLength={512}
          onChange={(event) => onCustomChange(event.target.value)}
          placeholder="例如：clash-verge/v2.4.2"
        />
      )}
      <label className="flex cursor-pointer items-start gap-2.5 rounded-lg border border-transparent px-1 py-1 text-sm text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)]">
        <input
          type="checkbox"
          checked={fallbackEnabled}
          disabled={disabled}
          onChange={(event) => onFallbackEnabledChange(event.target.checked)}
          className="mt-0.5 h-4 w-4 rounded border-[var(--color-border)] accent-[var(--color-primary)]"
        />
        <span>
          当前 UA 失败后自动尝试其他内置 UA
          <span className="mt-0.5 block text-xs text-[var(--color-text-muted)]">关闭后只使用当前选择，不会自动切换。</span>
        </span>
      </label>
    </div>
  )
}
