import { useMemo } from 'react'
import { Select } from '../../../shared/components'
import type { BrowserProxyGroupWithCount } from '../types'

interface ProxyGroupSelectProps {
  groups: BrowserProxyGroupWithCount[]
  value: string
  onChange: (groupId: string) => void
  includeAll?: boolean
  allLabel?: string
  className?: string
  style?: React.CSSProperties
}

export function buildProxyGroupPathMap(groups: BrowserProxyGroupWithCount[]): Map<string, string> {
  const byId = new Map(groups.map(group => [group.groupId, group]))
  const cache = new Map<string, string>()
  const resolve = (groupId: string, stack = new Set<string>()): string => {
    const cached = cache.get(groupId)
    if (cached) return cached
    const group = byId.get(groupId)
    if (!group) return ''
    if (stack.has(groupId)) return group.groupName
    const nextStack = new Set(stack).add(groupId)
    const parentPath = group.parentId ? resolve(group.parentId, nextStack) : ''
    const path = parentPath ? `${parentPath} / ${group.groupName}` : group.groupName
    cache.set(groupId, path)
    return path
  }
  groups.forEach(group => resolve(group.groupId))
  return cache
}

export function ProxyGroupSelect({
  groups,
  value,
  onChange,
  includeAll = false,
  allLabel = '全部代理',
  className,
  style,
}: ProxyGroupSelectProps) {
  const options = useMemo(() => {
    const pathMap = buildProxyGroupPathMap(groups)
    const sorted = [...groups].sort((a, b) =>
      (pathMap.get(a.groupId) || a.groupName).localeCompare(pathMap.get(b.groupId) || b.groupName, 'zh-CN'),
    )
    return [
      ...(includeAll ? [{ value: '__all__', label: allLabel }] : []),
      { value: '', label: '未分组' },
      ...sorted.map(group => ({
        value: group.groupId,
        label: pathMap.get(group.groupId) || group.groupName,
      })),
    ]
  }, [allLabel, groups, includeAll])

  return (
    <Select
      value={value}
      onChange={event => onChange(event.target.value)}
      options={options}
      className={className}
      style={style}
    />
  )
}
