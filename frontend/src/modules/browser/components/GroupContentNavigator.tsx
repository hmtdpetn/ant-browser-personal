import { ArrowUp, ChevronRight, Folder, FolderOpen } from 'lucide-react'

import type { BrowserGroupWithCount } from '../types'
import { getDirectChildGroups, getGroupPath, UNGROUPED_GROUP_ID } from '../pages/browserList/groupView'

interface GroupContentNavigatorProps {
  groups: BrowserGroupWithCount[]
  selectedGroupId: string
  directProfileCount: number
  onSelectGroup: (groupId: string) => void
}

export function GroupContentNavigator({
  groups,
  selectedGroupId,
  directProfileCount,
  onSelectGroup,
}: GroupContentNavigatorProps) {
  if (!selectedGroupId || selectedGroupId === UNGROUPED_GROUP_ID) return null

  const currentGroup = groups.find(group => group.groupId === selectedGroupId)
  if (!currentGroup) return null

  const childGroups = getDirectChildGroups(groups, selectedGroupId)
  const groupPath = getGroupPath(groups, selectedGroupId)
  const pathLabel = groupPath.map(group => group.groupName).join(' / ')

  return (
    <section
      className="border-y border-[var(--color-border-muted)] bg-[var(--color-bg-secondary)]/55 px-3 py-2.5"
      aria-label="当前实例分组"
    >
      <div className="flex min-w-0 items-center gap-2">
        {currentGroup.parentId && (
          <button
            type="button"
            className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg text-[var(--color-text-muted)] transition-colors hover:bg-[var(--color-bg-muted)] hover:text-[var(--color-accent)]"
            onClick={() => onSelectGroup(currentGroup.parentId)}
            title="返回上级分组"
            aria-label="返回上级分组"
          >
            <ArrowUp className="h-4 w-4" />
          </button>
        )}
        <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-amber-500/10">
          <FolderOpen className="h-4 w-4 text-amber-500" />
        </span>
        <div className="min-w-0 flex-1" title={pathLabel}>
          <h2 className="truncate text-sm font-semibold text-[var(--color-text-primary)]">{currentGroup.groupName}</h2>
          <p className="truncate text-[11px] text-[var(--color-text-muted)]">{pathLabel}</p>
        </div>
        <div className="flex shrink-0 items-center gap-1.5 text-[11px] tabular-nums text-[var(--color-text-muted)]">
          <span className="rounded-full bg-[var(--color-bg-muted)] px-2 py-1">实例 {directProfileCount}</span>
          {childGroups.length > 0 && <span className="rounded-full bg-[var(--color-bg-muted)] px-2 py-1">子组 {childGroups.length}</span>}
        </div>
      </div>

      {childGroups.length > 0 && (
        <div className="mt-2 grid grid-cols-1 gap-1.5 sm:grid-cols-2 xl:grid-cols-3">
          {childGroups.map(group => (
            <button
              key={group.groupId}
              type="button"
              className="grid h-11 min-w-0 grid-cols-[auto_minmax(0,1fr)_auto] items-center gap-2 rounded-lg border border-[var(--color-border-default)] bg-[var(--color-bg-elevated)] px-2.5 text-left transition-colors hover:border-[var(--color-accent)]/45 hover:bg-[var(--color-accent)]/5"
              onClick={() => onSelectGroup(group.groupId)}
              title={`${group.groupName}（${group.instanceCount} 个直属实例）`}
            >
              <Folder className="h-4 w-4 shrink-0 text-amber-500" />
              <span className="min-w-0 truncate text-sm font-medium text-[var(--color-text-primary)]">{group.groupName}</span>
              <span className="flex shrink-0 items-center gap-1 text-[11px] tabular-nums text-[var(--color-text-muted)]">
                {group.instanceCount}
                <ChevronRight className="h-3.5 w-3.5" />
              </span>
            </button>
          ))}
        </div>
      )}
    </section>
  )
}
