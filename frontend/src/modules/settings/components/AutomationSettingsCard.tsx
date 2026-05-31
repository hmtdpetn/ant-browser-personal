import { Badge, Button, Card, FormItem, Input, Progress, Select, Switch } from '../../../shared/components'

import type { AutomationNodeSource, AutomationRuntimeCheck, AutomationState, AutomationSystemNodeProbe } from '../api'
import type { AutomationRuntimeProgress } from '../progress'

type AutomationBusyState = 'none' | 'toggle' | 'probe' | 'runtime' | 'package' | 'install' | 'check'
type AutomationStatusVariant = 'default' | 'success' | 'error' | 'warning' | 'info'

const AUTOMATION_NODE_SOURCE_OPTIONS: Array<{ value: AutomationNodeSource; label: string }> = [
  { value: 'auto', label: 'auto · 优先系统 Node，失败回退内建' },
  { value: 'system', label: 'system · 强制系统 Node，不可用则报错' },
  { value: 'bundled', label: 'bundled · 总是使用内建 Node' },
]

interface AutomationSettingsCardProps {
  automationState: AutomationState
  automationProgress: AutomationRuntimeProgress | null
  automationBusy: AutomationBusyState
  automationCheck: AutomationRuntimeCheck | null
  automationProbe: AutomationSystemNodeProbe | null
  automationNodeSourceDraft: AutomationNodeSource
  automationSystemNodePathDraft: string
  automationRuntimeDirty: boolean
  onEnabledChange: (enabled: boolean) => void
  onHeadlessChange: (headlessDefault: boolean) => void
  onNodeSourceDraftChange: (value: AutomationNodeSource) => void
  onSystemNodePathDraftChange: (value: string) => void
  onTypeScriptBuildChange: (allowTypeScriptBuild: boolean) => void
  onProbeSystemNode: () => void
  onSaveRuntimeSettings: () => void
  onInstall: () => void
  onSelfCheck: () => void
}

function resolveAutomationStatus(state: AutomationState): {
  enabled: boolean
  ready: boolean
  installing: boolean
  statusLabel: string
  statusVariant: AutomationStatusVariant
  nodeSource: string
  nodeSourceLabel: string
  systemNodePath: string
  systemNodeLabel: string
} {
  const enabled = state.settings.enabled
  const ready = state.status.ready
  const installing = state.status.installing
  const statusLabel = installing
    ? '准备中'
    : ready
      ? '已就绪'
      : state.status.installed
        ? '已安装'
        : state.status.lastError
          ? '异常'
          : '未安装'
  const statusVariant = installing
    ? 'warning'
    : ready
      ? 'success'
      : state.status.lastError
        ? 'error'
        : 'default'
  const nodeSource = state.status.nodeSource || state.settings.nodeSource || 'auto'
  const nodeSourceLabel = nodeSource === 'system'
    ? 'system（系统 Node）'
    : nodeSource === 'bundled'
      ? 'bundled（内建 Node）'
      : 'auto（自动选择）'
  const systemNodePath = state.status.systemNodePath || state.settings.systemNodePath
  const systemNodeLabel = state.status.systemNodeDetected
    ? '已检测到'
    : systemNodePath
      ? '已配置，待验证'
      : '未检测到'

  return {
    enabled,
    ready,
    installing,
    statusLabel,
    statusVariant,
    nodeSource,
    nodeSourceLabel,
    systemNodePath,
    systemNodeLabel,
  }
}

export function AutomationSettingsCard({
  automationState,
  automationProgress,
  automationBusy,
  automationCheck,
  automationProbe,
  automationNodeSourceDraft,
  automationSystemNodePathDraft,
  automationRuntimeDirty,
  onEnabledChange,
  onHeadlessChange,
  onNodeSourceDraftChange,
  onSystemNodePathDraftChange,
  onTypeScriptBuildChange,
  onProbeSystemNode,
  onSaveRuntimeSettings,
  onInstall,
  onSelfCheck,
}: AutomationSettingsCardProps) {
  const {
    enabled,
    ready,
    installing,
    statusLabel,
    statusVariant,
    nodeSource,
    nodeSourceLabel,
    systemNodePath,
    systemNodeLabel,
  } = resolveAutomationStatus(automationState)

  return (
    <Card title="自动化支持" subtitle="首次启用时优先检测系统 Node，仅在需要时回退下载内建 Node，并准备私有 playwright-core">
      <div className="space-y-5">
        <div className="flex items-start justify-between gap-4">
          <div>
            <div className="flex items-center gap-2 flex-wrap">
              <p className="text-sm font-medium text-[var(--color-text-primary)]">启用自动化支持</p>
              <Badge variant={statusVariant} size="sm" dot>{statusLabel}</Badge>
            </div>
            <p className="text-xs text-[var(--color-text-muted)] mt-1">
              开启后应用会自动准备本地 automation runtime；关闭时不会卸载，后续再次启用可直接复用。
            </p>
          </div>
          <Switch
            checked={enabled}
            onChange={onEnabledChange}
            disabled={automationBusy === 'toggle'}
          />
        </div>

        <div className="h-px bg-[var(--color-border-muted)]" />

        <div className="flex items-start justify-between gap-4">
          <div>
            <p className="text-sm font-medium text-[var(--color-text-primary)]">默认无头模式</p>
            <p className="text-xs text-[var(--color-text-muted)] mt-1">
              作为后续自动化任务的默认启动策略，首版先只保存配置，不直接改实例启动参数。
            </p>
          </div>
          <Switch
            checked={automationState.settings.headlessDefault}
            onChange={onHeadlessChange}
            disabled={automationBusy === 'toggle'}
          />
        </div>

        <div className="h-px bg-[var(--color-border-muted)]" />

        <div className="grid grid-cols-1 lg:grid-cols-[minmax(0,220px)_minmax(0,1fr)] gap-4">
          <FormItem label="Node 来源策略">
            <Select
              value={automationNodeSourceDraft}
              onChange={event => onNodeSourceDraftChange(event.target.value as AutomationNodeSource)}
              disabled={automationBusy !== 'none'}
              options={AUTOMATION_NODE_SOURCE_OPTIONS}
            />
          </FormItem>
          <FormItem label="系统 Node 路径" hint="留空则走 PATH">
            <Input
              value={automationSystemNodePathDraft}
              onChange={event => onSystemNodePathDraftChange(event.target.value)}
              placeholder="例如 C:\\Program Files\\nodejs\\node.exe"
              disabled={automationBusy !== 'none' || automationNodeSourceDraft === 'bundled'}
            />
          </FormItem>
        </div>

        <div className="h-px bg-[var(--color-border-muted)]" />

        <div className="flex items-start justify-between gap-4">
          <div>
            <p className="text-sm font-medium text-[var(--color-text-primary)]">允许导入 TypeScript 脚本（实验）</p>
            <p className="text-xs text-[var(--color-text-muted)] mt-1">
              仅支持单入口、本地相对依赖，并会在导入时构建为 CommonJS；不支持外部 npm 依赖。
            </p>
          </div>
          <Switch
            checked={automationState.settings.allowTypeScriptBuild}
            onChange={onTypeScriptBuildChange}
            disabled={automationBusy !== 'none'}
          />
        </div>

        <div className="flex items-center justify-between gap-4 rounded-md border border-[var(--color-border-default)] bg-[var(--color-bg-secondary)] px-3 py-3">
          <p className="text-xs text-[var(--color-text-muted)]">
            `auto` 适合大多数环境；`system` 用于强制复用本机 Node；`bundled` 会忽略系统 Node，始终使用应用内建 runtime。
          </p>
          <div className="flex flex-wrap gap-2">
            <Button
              size="sm"
              variant="secondary"
              onClick={onProbeSystemNode}
              loading={automationBusy === 'probe'}
              disabled={automationBusy !== 'none' || automationNodeSourceDraft === 'bundled'}
            >
              检测系统 Node
            </Button>
            <Button
              size="sm"
              variant="secondary"
              onClick={onSaveRuntimeSettings}
              loading={automationBusy === 'runtime' && automationRuntimeDirty}
              disabled={!automationRuntimeDirty || automationBusy !== 'none'}
            >
              保存运行时策略
            </Button>
          </div>
        </div>

        {automationProbe && (
          <div className="rounded-md border border-[var(--color-border-default)] bg-[var(--color-bg-secondary)] px-3 py-2 text-xs text-[var(--color-text-secondary)] break-all">
            系统 Node 检测：<code>{automationProbe.version}</code> · <code>{automationProbe.path}</code>
          </div>
        )}

        <div className="rounded-md border border-[var(--color-border-default)] bg-[var(--color-bg-secondary)] px-3 py-3 space-y-2 text-xs">
          <div className="flex flex-wrap items-center gap-x-4 gap-y-1 text-[var(--color-text-secondary)]">
            <span>安装策略：<code>{automationState.settings.installPolicy}</code></span>
            <span>Runtime：<code>{automationState.settings.runtimeVersion}</code></span>
          </div>
          <div className="flex flex-wrap items-center gap-x-4 gap-y-1 text-[var(--color-text-secondary)]">
            <span>Node 来源：<code>{nodeSourceLabel}</code></span>
            <span>Node：<code>{automationState.status.nodeVersion || automationState.settings.nodeVersion}</code></span>
            <span>playwright-core：<code>{automationState.status.playwrightVersion || automationState.settings.playwrightVersion}</code></span>
            <span>TS 导入构建：<code>{automationState.settings.allowTypeScriptBuild ? 'enabled' : 'disabled'}</code></span>
          </div>
          <div className="flex flex-wrap items-center gap-x-4 gap-y-1 text-[var(--color-text-secondary)]">
            <span>系统 Node：<code>{systemNodeLabel}</code></span>
          </div>
          {automationState.status.nodeResolution && (
            <div className="text-[var(--color-text-muted)] break-all">
              解析说明：{automationState.status.nodeResolution}
            </div>
          )}
          {automationState.status.runtimeDir && (
            <div className="text-[var(--color-text-muted)] break-all">
              运行时目录：<code>{automationState.status.runtimeDir}</code>
            </div>
          )}
          {automationState.status.nodePath && (
            <div className="text-[var(--color-text-muted)] break-all">
              Node 路径：<code>{automationState.status.nodePath}</code>
            </div>
          )}
          {systemNodePath && (
            <div className="text-[var(--color-text-muted)] break-all">
              系统 Node 路径：<code>{systemNodePath}</code>
            </div>
          )}
          {automationState.status.systemNodeError && (
            <div className="text-[var(--color-warning)] break-all">
              系统 Node 异常：{automationState.status.systemNodeError}
            </div>
          )}
          {automationState.status.lastError && (
            <div className="text-[var(--color-error)] break-all">
              最近错误：{automationState.status.lastError}
            </div>
          )}
        </div>

        {automationProgress && (
          <div className="rounded-md border border-[var(--color-border-default)] bg-[var(--color-bg-secondary)] px-3 py-3 space-y-2">
            <div className="flex items-center justify-between text-xs">
              <span className="text-[var(--color-text-secondary)]">{automationProgress.message}</span>
              <span className="text-[var(--color-text-muted)]">
                {automationProgress.component ? `${automationProgress.component} · ` : ''}
                {automationProgress.phase}
              </span>
            </div>
            <Progress
              percent={automationProgress.progress}
              size="sm"
              status={automationProgress.phase === 'error' ? 'error' : automationProgress.phase === 'done' ? 'success' : 'normal'}
            />
          </div>
        )}

        {automationCheck && (
          <div className="rounded-md border border-[var(--color-border-default)] bg-[var(--color-bg-secondary)] px-3 py-2 text-xs text-[var(--color-text-secondary)]">
            最近自检：<code>{automationCheck.nodeSource || nodeSource}</code> / Node <code>{automationCheck.nodeVersion}</code> / playwright-core <code>{automationCheck.playwrightVersion}</code>
          </div>
        )}

        <div className="flex flex-wrap gap-2">
          <Button
            size="sm"
            variant="secondary"
            onClick={onInstall}
            loading={automationBusy === 'install'}
            disabled={installing}
          >
            {automationState.status.installed ? '修复/重装运行时' : '立即准备运行时'}
          </Button>
          <Button
            size="sm"
            onClick={onSelfCheck}
            loading={automationBusy === 'check'}
            disabled={!ready}
          >
            运行自检
          </Button>
        </div>
      </div>
    </Card>
  )
}
