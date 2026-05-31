import { useMemo } from 'react'
import { Select } from '../../../shared/components'
import type { BrowserGroup } from '../types'

interface GroupSelectorProps {
  groups: BrowserGroup[]
  value: string
  onChange: (groupId: string) => void
  placeholder?: string
  className?: string
}

interface FlatGroup extends BrowserGroup {
  level: number
}

// 将分组列表扁平化并计算层级
function flattenGroups(groups: BrowserGroup[]): FlatGroup[] {
  const result: FlatGroup[] = []
  const addChildren = (parentId: string, level: number) => {
    groups
      .filter(g => g.parentId === parentId)
      .sort((a, b) => a.sortOrder - b.sortOrder)
      .forEach(g => {
        result.push({ ...g, level })
        addChildren(g.groupId, level + 1)
      })
  }

  // 先添加根级分组
  addChildren('', 0)

  return result
}

export function GroupSelector({ groups, value, onChange, placeholder = '选择分组', className = '' }: GroupSelectorProps) {
  const flatGroups = useMemo(() => flattenGroups(groups), [groups])
  const options = useMemo(
    () => [
      { value: '', label: placeholder },
      ...flatGroups.map(g => ({
        value: g.groupId,
        label: `${'　'.repeat(g.level)}${g.groupName}`,
      })),
    ],
    [flatGroups, placeholder]
  )

  return (
    <Select
      className={className}
      value={value}
      onChange={e => onChange(e.target.value)}
      options={options}
    />
  )
}
