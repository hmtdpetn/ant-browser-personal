import { Copy, PlusSquare, Rocket, ShieldCheck, Trash2 } from 'lucide-react'
import { Badge, Button, Modal, toast } from '../../../shared/components'
import {
  automationDemoCreateProfile,
  automationDemoDeleteProfile,
  automationDemoHealthCheck,
  automationDemoLaunchProfile,
} from '../api'
import { AutomationEntryActions } from './AutomationEntryActions'
import { LaunchServerStatusBlock } from './LaunchServerStatusBlock'
import { useAutomationDemoSession } from '../hooks/useAutomationDemoSession'
import { useLaunchContext } from '../hooks/useLaunchContext'

interface AutomationToolboxModalProps {
  open: boolean
  onClose: () => void
}

async function copyToClipboard(text: string, successMessage: string) {
  try {
    await navigator.clipboard.writeText(text)
    toast.success(successMessage)
  } catch {
    toast.error('复制失败')
  }
}

function JsonPreview({ text }: { text: string }) {
  if (!text) {
    return (
      <div className="rounded-lg border border-[var(--color-border-muted)] bg-[var(--color-bg-secondary)] px-4 py-3 text-sm text-[var(--color-text-muted)]">
        还没有最近响应。先执行一次健康检查或 Demo 创建。
      </div>
    )
  }

  return (
    <pre className="max-h-[240px] overflow-auto rounded-lg border border-[var(--color-border-muted)] bg-[var(--color-bg-secondary)] p-3 text-xs leading-relaxed text-[var(--color-text-primary)]">
      <code>{text}</code>
    </pre>
  )
}

export function AutomationToolboxModal({ open, onClose }: AutomationToolboxModalProps) {
  const { launchBaseUrl, apiAuth, launchServerReady } = useLaunchContext({ enabled: open })
  const {
    demoSession,
    demoBusyAction,
    demoBusy,
    demoResponseText,
    runDemoAction,
  } = useAutomationDemoSession({ enabled: open, baseUrl: launchBaseUrl })

  return (
    <Modal open={open} onClose={onClose} title="自动化工具箱" width="1100px">
      <div className="space-y-5">
        <section className="rounded-xl border border-[var(--color-border-default)] bg-[var(--color-bg-surface)] p-4">
          <div>
            <h3 className="text-sm font-semibold text-[var(--color-text-primary)]">自动化入口</h3>
            <p className="mt-1 text-xs text-[var(--color-text-muted)]">
              主页面只保留脚本管理，Smoke、文档和运行时入口统一从这里进入。
            </p>
          </div>
          <div className="mt-4">
            <LaunchServerStatusBlock
              launchBaseUrl={launchBaseUrl}
              launchServerReady={launchServerReady}
              apiAuth={apiAuth}
            >
              <AutomationEntryActions
                onBeforeNavigate={onClose}
              />
            </LaunchServerStatusBlock>
          </div>
        </section>

        <div className="grid grid-cols-1 gap-4 xl:grid-cols-[0.92fr_1.08fr]">
          <div className="space-y-4">
            <section className="rounded-xl border border-[var(--color-border-default)] bg-[var(--color-bg-surface)] p-4">
              <div>
                <h3 className="text-sm font-semibold text-[var(--color-text-primary)]">Demo 调试</h3>
                <p className="mt-1 text-xs text-[var(--color-text-muted)]">保留真实请求链路，方便核对 LaunchServer、实例创建和 CDP 返回。</p>
              </div>

              <div className="mt-4 flex flex-wrap gap-2">
                <Button
                  size="sm"
                  onClick={() => void runDemoAction({
                    actionKey: 'health',
                    actionLabel: '健康检查',
                    runner: () => automationDemoHealthCheck(),
                    successMessage: '健康检查已完成',
                    failureMessage: '健康检查失败',
                  })}
                  loading={demoBusyAction === 'health'}
                  disabled={demoBusy}
                >
                  <ShieldCheck className="h-4 w-4" />
                  健康检查
                </Button>
                <Button
                  size="sm"
                  onClick={() => void runDemoAction({
                    actionKey: 'create',
                    actionLabel: '创建演示实例',
                    runner: () => automationDemoCreateProfile(),
                    successMessage: '演示实例已创建',
                    failureMessage: '演示实例创建失败',
                  })}
                  loading={demoBusyAction === 'create'}
                  disabled={demoBusy}
                >
                  <PlusSquare className="h-4 w-4" />
                  创建 Demo
                </Button>
                <Button
                  size="sm"
                  onClick={() => void runDemoAction({
                    actionKey: 'launch',
                    actionLabel: '按 Code 唤起',
                    runner: () => automationDemoLaunchProfile(demoSession.launchCode),
                    successMessage: '演示实例已唤起',
                    failureMessage: '演示实例唤起失败',
                  })}
                  loading={demoBusyAction === 'launch'}
                  disabled={demoBusy || !demoSession.launchCode}
                >
                  <Rocket className="h-4 w-4" />
                  按 Code 唤起
                </Button>
                <Button
                  size="sm"
                  variant="danger"
                  onClick={() => void runDemoAction({
                    actionKey: 'delete',
                    actionLabel: '清理演示实例',
                    runner: () => automationDemoDeleteProfile(demoSession.profileId),
                    successMessage: '演示实例已清理',
                    failureMessage: '演示实例清理失败',
                  })}
                  loading={demoBusyAction === 'delete'}
                  disabled={demoBusy || !demoSession.profileId}
                >
                  <Trash2 className="h-4 w-4" />
                  清理 Demo
                </Button>
              </div>

              <div className="mt-4 rounded-lg border border-[var(--color-border-muted)] bg-[var(--color-bg-secondary)] px-4 py-3 text-sm text-[var(--color-text-secondary)]">
                <div>最近动作：<code>{demoSession.lastAction || '-'}</code></div>
                <div>Profile ID：<code>{demoSession.profileId || '-'}</code></div>
                <div>Launch Code：<code>{demoSession.launchCode || '-'}</code></div>
                <div className="break-all">CDP URL：<code>{demoSession.cdpUrl || '-'}</code></div>
              </div>

              <div className="mt-3 flex flex-wrap gap-2">
                <Button
                  size="sm"
                  variant="secondary"
                  onClick={() => void copyToClipboard(demoSession.launchCode, 'Launch Code 已复制')}
                  disabled={!demoSession.launchCode}
                >
                  <Copy className="h-3.5 w-3.5" />
                  复制 Launch Code
                </Button>
                <Button
                  size="sm"
                  variant="secondary"
                  onClick={() => void copyToClipboard(demoSession.cdpUrl, 'CDP URL 已复制')}
                  disabled={!demoSession.cdpUrl}
                >
                  <Copy className="h-3.5 w-3.5" />
                  复制 CDP URL
                </Button>
              </div>
            </section>
          </div>

          <section className="rounded-xl border border-[var(--color-border-default)] bg-[var(--color-bg-surface)] p-4">
            <div className="flex items-center justify-between gap-3">
              <div>
                <h3 className="text-sm font-semibold text-[var(--color-text-primary)]">最近响应</h3>
                <p className="mt-1 text-xs text-[var(--color-text-muted)]">调试响应保留在工具箱，主页面不再混入接口演示内容。</p>
              </div>
              {demoResponseText && (
                <Button size="sm" variant="secondary" onClick={() => void copyToClipboard(demoResponseText, '响应 JSON 已复制')}>
                  <Copy className="h-3.5 w-3.5" />
                  复制 JSON
                </Button>
              )}
            </div>

            <div className="mt-4 flex flex-wrap items-center gap-2">
              <Badge variant={demoSession.lastResult?.ok ? 'success' : demoSession.lastResult ? 'error' : 'default'} size="sm" dot>
                {demoSession.lastResult ? (demoSession.lastResult.ok ? '请求成功' : '请求失败') : '暂无请求'}
              </Badge>
              {demoSession.lastResult && (
                <Badge variant="default" size="sm">
                  HTTP {demoSession.lastResult.status || '-'}
                </Badge>
              )}
              {demoSession.lastResult?.method && demoSession.lastResult?.path && (
                <Badge variant="default" size="sm">
                  {demoSession.lastResult.method} {demoSession.lastResult.path}
                </Badge>
              )}
            </div>

            {demoSession.lastResult?.error && (
              <p className="mt-3 break-all text-sm text-[var(--color-error)]">{demoSession.lastResult.error}</p>
            )}

            <div className="mt-4">
              <JsonPreview text={demoResponseText} />
            </div>
          </section>
        </div>
      </div>
    </Modal>
  )
}
