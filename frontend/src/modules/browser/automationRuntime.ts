import type { AutomationState } from '../settings/api'

export function getAutomationRuntimeBadgeVariant(state: AutomationState): 'default' | 'success' | 'error' | 'warning' {
  if (state.status.installing) return 'warning'
  if (state.status.ready) return 'success'
  if (state.status.lastError) return 'error'
  return 'default'
}

export function getAutomationRuntimeBadgeText(state: AutomationState): string {
  if (!state.settings.enabled) return '未启用'
  if (state.status.installing) return '准备中'
  if (state.status.ready) return '已就绪'
  if (state.status.installed) return '已安装'
  if (state.status.lastError) return '异常'
  return '待准备'
}

export function getAutomationNodeSource(state: AutomationState): string {
  return String(state.status.nodeSource || state.settings.nodeSource || 'auto').trim() || 'auto'
}

export function getAutomationNodeSourceLabel(nodeSource: string): string {
  switch (nodeSource) {
    case 'system':
      return '系统 Node'
    case 'bundled':
      return '内置 Node'
    default:
      return '自动选择'
  }
}

export function getAutomationNodeVersion(state: AutomationState): string {
  return state.status.nodeVersion || state.settings.nodeVersion || '-'
}

export function getAutomationPlaywrightVersion(state: AutomationState): string {
  return state.status.playwrightVersion || state.settings.playwrightVersion || '-'
}

export function getAutomationSystemNodePath(state: AutomationState): string {
  return state.status.systemNodePath || state.settings.systemNodePath || ''
}
