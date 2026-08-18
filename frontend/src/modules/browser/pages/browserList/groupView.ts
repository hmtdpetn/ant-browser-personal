import type { BrowserGroupWithCount } from '../../types'

export const UNGROUPED_GROUP_ID = '__ungrouped__'

export function profileMatchesSelectedGroup(profileGroupId: string | undefined, selectedGroupId: string): boolean {
  if (!selectedGroupId) return true
  if (selectedGroupId === UNGROUPED_GROUP_ID) return !profileGroupId
  return profileGroupId === selectedGroupId
}

export function getDirectChildGroups(
  groups: BrowserGroupWithCount[],
  parentId: string
): BrowserGroupWithCount[] {
  return groups
    .filter(group => group.parentId === parentId)
    .sort((left, right) => left.sortOrder - right.sortOrder || left.groupName.localeCompare(right.groupName, 'zh-CN'))
}

export function getGroupPath(groups: BrowserGroupWithCount[], groupId: string): BrowserGroupWithCount[] {
  const groupById = new Map(groups.map(group => [group.groupId, group]))
  const path: BrowserGroupWithCount[] = []
  const visited = new Set<string>()
  let current = groupById.get(groupId)

  while (current && !visited.has(current.groupId)) {
    visited.add(current.groupId)
    path.unshift(current)
    current = current.parentId ? groupById.get(current.parentId) : undefined
  }

  return path
}
