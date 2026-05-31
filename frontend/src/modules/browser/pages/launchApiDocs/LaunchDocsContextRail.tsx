import type { LaunchServerInfo } from '../../api'
import type {
  AutomationDemoActionKey,
  AutomationDemoSession,
} from '../../demoSession'
import type { LaunchDocDemoConfig } from './catalog'
import { LaunchDemoPanel } from './panels/LaunchDemoPanel'
import { LaunchResponsePanel } from './panels/LaunchResponsePanel'
import { LaunchRuntimePanel } from './panels/LaunchRuntimePanel'
import { LaunchStatusPanel } from './panels/LaunchStatusPanel'

interface LaunchDocsContextRailProps {
  currentGroupLabel: string
  currentDocLabel: string
  launchBaseUrl: string
  launchServerReady: boolean
  apiAuth: LaunchServerInfo['apiAuth']
  docDemoConfig: LaunchDocDemoConfig | null
  demoSession: AutomationDemoSession
  demoBusyAction: AutomationDemoActionKey
  demoBusy: boolean
  demoResponseText: string
  onHealth: () => void
  onCreate: () => void
  onLaunch: () => void
  onDelete: () => void
  onCopyResponse: () => void
}

export function LaunchDocsContextRail({
  currentGroupLabel,
  currentDocLabel,
  launchBaseUrl,
  launchServerReady,
  apiAuth,
  docDemoConfig,
  demoSession,
  demoBusyAction,
  demoBusy,
  demoResponseText,
  onHealth,
  onCreate,
  onLaunch,
  onDelete,
  onCopyResponse,
}: LaunchDocsContextRailProps) {
  return (
    <>
      <LaunchStatusPanel
        currentGroupLabel={currentGroupLabel}
        currentDocLabel={currentDocLabel}
        launchBaseUrl={launchBaseUrl}
        launchServerReady={launchServerReady}
        apiAuth={apiAuth}
      />

      <LaunchRuntimePanel />

      {docDemoConfig && (
        <LaunchDemoPanel
          config={docDemoConfig}
          launchServerReady={launchServerReady}
          demoSession={demoSession}
          demoBusyAction={demoBusyAction}
          demoBusy={demoBusy}
          onHealth={onHealth}
          onCreate={onCreate}
          onLaunch={onLaunch}
          onDelete={onDelete}
        />
      )}

      <LaunchResponsePanel
        demoSession={demoSession}
        demoResponseText={demoResponseText}
        onCopyResponse={onCopyResponse}
      />
    </>
  )
}
