import type { ReactNode } from 'react'
import { Badge } from '../../../shared/components'
import type { LaunchServerInfo } from '../api'

interface LaunchServerStatusBlockProps {
  launchBaseUrl: string
  launchServerReady: boolean
  apiAuth: LaunchServerInfo['apiAuth']
  children?: ReactNode
}

export function LaunchServerStatusBlock({
  launchBaseUrl,
  launchServerReady,
  apiAuth,
  children,
}: LaunchServerStatusBlockProps) {
  return (
    <div className="space-y-3">
      <div className="flex flex-wrap items-center gap-2">
        <Badge variant={launchServerReady ? 'success' : 'warning'} size="sm" dot>
          Launch API {launchServerReady ? '已就绪' : '待就绪'}
        </Badge>
        <Badge variant="default" size="sm">
          Base URL {launchBaseUrl}
        </Badge>
        {apiAuth.enabled && (
          <Badge variant="warning" size="sm">
            API 认证已启用 · {apiAuth.header}
          </Badge>
        )}
      </div>

      <div className="text-sm leading-relaxed text-[var(--color-text-secondary)]">
        <p>
          当前 Launch 地址：<code>{launchBaseUrl}</code>
          {!launchServerReady ? '（服务启动后会自动刷新）' : ''}
        </p>
        <p className="mt-2">
          {apiAuth.enabled
            ? <>当前 API 认证已启用，请为所有 <code>/api/*</code> 请求追加 <code>{apiAuth.header}: &lt;your-api-key&gt;</code>。</>
            : apiAuth.requested && !apiAuth.configured
              ? <>当前配置要求启用 API 认证，但 <code>api_key</code> 为空，认证尚未生效。</>
              : <>当前 API 认证未启用；如需开启，可在 <code>config.yaml</code> 的 <code>launch_server.auth</code> 下配置。</>}
        </p>
      </div>

      {children}
    </div>
  )
}
