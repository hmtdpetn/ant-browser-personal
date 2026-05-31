import type { LaunchServerInfo } from './api'

export const DEFAULT_LAUNCH_BASE_URL = 'http://127.0.0.1:19876'

export const DEFAULT_API_AUTH: LaunchServerInfo['apiAuth'] = {
  requested: false,
  configured: false,
  enabled: false,
  header: 'X-Ant-Api-Key',
}
