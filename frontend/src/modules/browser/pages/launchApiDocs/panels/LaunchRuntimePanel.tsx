import { AutomationRuntimeSnapshot } from '../../../components/AutomationRuntimeSnapshot'

export function LaunchRuntimePanel() {
  return (
    <AutomationRuntimeSnapshot
      title="运行时快照"
      subtitle="运行时状态固定留在右侧，与正文和 Demo 调试解耦。"
      className="bg-[var(--color-bg-elevated)] shadow-[var(--shadow-sm)]"
      showSettingsAction={false}
    />
  )
}
