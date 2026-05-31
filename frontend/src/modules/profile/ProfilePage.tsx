import { Card } from '../../shared/components'

export function ProfilePage() {
  return (
    <div className="mx-auto max-w-3xl space-y-4 animate-fade-in">
      <Card title="Local Personal Edition" padding="lg">
        <div className="space-y-3 text-sm leading-7 text-[var(--color-text-secondary)]">
          <p>
            This local build keeps the core browser profile, proxy pool, core management, tag, and automation features.
          </p>
          <p>
            Author profiles, external community links, and project promotion content are hidden in this local UI.
          </p>
        </div>
      </Card>
    </div>
  )
}
