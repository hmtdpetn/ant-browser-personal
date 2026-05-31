import { ArrowLeft } from 'lucide-react'
import { Button } from '../../../../shared/components'

interface LaunchDocsHeaderProps {
  activeGroupLabel: string
  activeDocLabel: string
  onBack: () => void
  onJumpTutorial: () => void
  onJumpCoreIntro: () => void
  onJumpProxyIntro: () => void
  onJumpApiOverview: () => void
}

export function LaunchDocsHeader({
  activeGroupLabel,
  activeDocLabel,
  onBack,
  onJumpTutorial,
  onJumpCoreIntro,
  onJumpProxyIntro,
  onJumpApiOverview,
}: LaunchDocsHeaderProps) {
  const quickLinks = [
    { label: '使用教程', onClick: onJumpTutorial },
    { label: '内核介绍', onClick: onJumpCoreIntro },
    { label: '代理介绍', onClick: onJumpProxyIntro },
    { label: '接口总览', onClick: onJumpApiOverview },
  ]

  return (
    <section className="rounded-2xl border border-[var(--color-border-default)] bg-[var(--color-bg-elevated)] px-4 py-4 shadow-[var(--shadow-sm)]">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="min-w-0">
          <div className="text-[11px] font-semibold uppercase tracking-[0.18em] text-[var(--color-text-muted)]">
            文档中心
          </div>
          <div className="mt-1 flex flex-wrap items-center gap-2 text-sm">
            <span className="text-[var(--color-text-secondary)]">{activeGroupLabel}</span>
            <span className="text-[var(--color-text-muted)]">/</span>
            <span className="font-medium text-[var(--color-text-primary)]">{activeDocLabel}</span>
          </div>
        </div>

        <div className="flex flex-wrap gap-2">
          <Button size="sm" variant="secondary" onClick={onBack}>
            <ArrowLeft className="h-4 w-4" />
            打开实例列表
          </Button>
        </div>
      </div>

      <div className="mt-3 flex flex-wrap items-center gap-2 border-t border-[var(--color-border-muted)] pt-3">
        <span className="text-xs font-medium text-[var(--color-text-muted)]">快捷入口</span>
        {quickLinks.map((link) => (
          <button
            key={link.label}
            onClick={link.onClick}
            className="rounded-full border border-[var(--color-border-default)] bg-[var(--color-bg-surface)] px-3 py-1.5 text-xs text-[var(--color-text-secondary)] transition-colors hover:border-[var(--color-border-strong)] hover:text-[var(--color-text-primary)]"
          >
            {link.label}
          </button>
        ))}
      </div>
    </section>
  )
}
