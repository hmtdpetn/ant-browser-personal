import type { ReactNode } from 'react'

interface LaunchDocsLayoutProps {
  sidebar: ReactNode
  header: ReactNode
  content: ReactNode
  contextRail?: ReactNode
}

export function LaunchDocsLayout({
  sidebar,
  header,
  content,
  contextRail,
}: LaunchDocsLayoutProps) {
  const hasContextRail = Boolean(contextRail)

  return (
    <div className="-m-5 min-h-full bg-[var(--color-bg-subtle)]">
      <div className={hasContextRail
        ? 'xl:grid xl:min-h-full xl:grid-cols-[280px_minmax(0,1fr)_360px]'
        : 'xl:grid xl:min-h-full xl:grid-cols-[280px_minmax(0,1fr)]'}
      >
        <aside className="border-b border-[var(--color-border-default)] bg-[var(--color-bg-surface)] xl:border-b-0 xl:border-r">
          <div className="px-4 py-4 xl:sticky xl:top-0 xl:h-screen xl:overflow-y-auto xl:px-3 xl:py-6">
            {sidebar}
          </div>
        </aside>

        <main className="min-w-0">
          <div className={hasContextRail
            ? 'mx-auto max-w-4xl px-4 py-5 md:px-6 xl:max-w-none xl:px-8 xl:py-8'
            : 'mx-auto max-w-5xl px-4 py-5 md:px-6 xl:px-10 xl:py-8'}
          >
            <div className="space-y-5">
              {header}
              {content}
            </div>
          </div>
        </main>

        {hasContextRail && (
          <aside className="border-t border-[var(--color-border-default)] bg-[var(--color-bg-subtle)] xl:border-t-0 xl:border-l">
            <div className="space-y-4 px-4 py-5 md:px-6 xl:sticky xl:top-0 xl:h-screen xl:overflow-y-auto xl:px-5 xl:py-8">
              {contextRail}
            </div>
          </aside>
        )}
      </div>
    </div>
  )
}
