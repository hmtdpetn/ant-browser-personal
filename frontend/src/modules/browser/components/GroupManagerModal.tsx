import { useMemo, useState } from 'react'
import { Check, Pencil, Plus, Trash2, X } from 'lucide-react'
import { Button, ConfirmModal, Input, Modal, Select, toast } from '../../../shared/components'
import type { BrowserGroupInput, BrowserGroupWithCount } from '../types'
import { createGroup, deleteGroup, updateGroup } from '../api'

interface GroupManagerModalProps {
  open: boolean
  onClose: () => void
  groups: BrowserGroupWithCount[]
  onChanged: () => void
}

interface FlatGroup extends BrowserGroupWithCount {
  level: number
}

function flattenGroups(groups: BrowserGroupWithCount[]): FlatGroup[] {
  const result: FlatGroup[] = []
  const addChildren = (parentId: string, level: number) => {
    groups
      .filter(group => group.parentId === parentId)
      .sort((a, b) => a.sortOrder - b.sortOrder)
      .forEach(group => {
        result.push({ ...group, level })
        addChildren(group.groupId, level + 1)
      })
  }
  addChildren('', 0)
  return result
}

export function GroupManagerModal({ open, onClose, groups, onChanged }: GroupManagerModalProps) {
  const [newName, setNewName] = useState('')
  const [newParentId, setNewParentId] = useState('')
  const [editingId, setEditingId] = useState<string | null>(null)
  const [editingName, setEditingName] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [pendingDelete, setPendingDelete] = useState<BrowserGroupWithCount | null>(null)

  const flatGroups = useMemo(() => flattenGroups(groups), [groups])
  const groupNameById = useMemo(() => {
    const map = new Map<string, string>()
    groups.forEach(group => map.set(group.groupId, group.groupName))
    return map
  }, [groups])

  const resetCreate = () => {
    setNewName('')
    setNewParentId('')
  }

  const handleCreate = async () => {
    const name = newName.trim()
    if (!name) {
      setError('分组名称不能为空')
      return
    }
    setBusy(true)
    setError('')
    try {
      const maxSort = groups.reduce((max, group) => Math.max(max, group.sortOrder), 0)
      const input: BrowserGroupInput = { groupName: name, parentId: newParentId, sortOrder: maxSort + 1 }
      await createGroup(input)
      resetCreate()
      onChanged()
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : '创建分组失败')
    } finally {
      setBusy(false)
    }
  }

  const startEdit = (group: BrowserGroupWithCount) => {
    setEditingId(group.groupId)
    setEditingName(group.groupName)
    setError('')
  }

  const cancelEdit = () => {
    setEditingId(null)
    setEditingName('')
  }

  const handleSaveEdit = async (group: BrowserGroupWithCount) => {
    const name = editingName.trim()
    if (!name) {
      setError('分组名称不能为空')
      return
    }
    setBusy(true)
    setError('')
    try {
      const input: BrowserGroupInput = { groupName: name, parentId: group.parentId, sortOrder: group.sortOrder }
      await updateGroup(group.groupId, input)
      cancelEdit()
      onChanged()
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : '更新分组失败')
    } finally {
      setBusy(false)
    }
  }

  const buildDeleteMessage = (group: BrowserGroupWithCount) => {
    const hasParent = group.parentId && groupNameById.has(group.parentId)
    const destination = hasParent ? `上级分组「${groupNameById.get(group.parentId)}」` : '未分组'
    if (group.instanceCount > 0) {
      return `该分组下有 ${group.instanceCount} 个实例。删除后，子分组和实例将移动到${destination}。确定删除「${group.groupName}」吗？`
    }
    return `确定删除分组「${group.groupName}」吗？`
  }

  const performDelete = async (group: BrowserGroupWithCount) => {
    setBusy(true)
    setError('')
    try {
      await deleteGroup(group.groupId)
      onChanged()
    } catch (caught) {
      const message = caught instanceof Error ? caught.message : '删除分组失败'
      setError(message)
      toast.error(message)
    } finally {
      setBusy(false)
    }
  }

  const parentOptions = useMemo(
    () => [
      { value: '', label: '根级分组' },
      ...flatGroups.map(group => ({ value: group.groupId, label: `${'　'.repeat(group.level)}${group.groupName}` })),
    ],
    [flatGroups]
  )

  return (
    <Modal
      open={open}
      onClose={onClose}
      title="分组管理"
      width="560px"
      footer={<Button variant="secondary" onClick={onClose}>关闭</Button>}
    >
      <div className="space-y-4">
        <div className="rounded-lg border border-[var(--color-border-default)] divide-y divide-[var(--color-border-muted)]">
          {flatGroups.length === 0 && (
            <div className="px-3 py-6 text-center text-sm text-[var(--color-text-muted)]">
              暂无分组，请在下方新建
            </div>
          )}
          {flatGroups.map(group => (
            <div key={group.groupId} className="flex items-center gap-2 px-3 py-2">
              {editingId === group.groupId ? (
                <>
                  <Input
                    value={editingName}
                    onChange={event => setEditingName(event.target.value)}
                    placeholder="分组名称"
                    className="flex-1"
                    autoFocus
                    style={{ marginLeft: `${group.level * 16}px` }}
                  />
                  <Button size="sm" variant="ghost" onClick={() => handleSaveEdit(group)} disabled={busy} title="保存">
                    <Check className="w-4 h-4 text-green-600" />
                  </Button>
                  <Button size="sm" variant="ghost" onClick={cancelEdit} disabled={busy} title="取消">
                    <X className="w-4 h-4" />
                  </Button>
                </>
              ) : (
                <>
                  <span
                    className="flex-1 truncate text-sm text-[var(--color-text-primary)]"
                    style={{ paddingLeft: `${group.level * 16}px` }}
                  >
                    {group.groupName}
                  </span>
                  <span className="text-xs text-[var(--color-text-muted)] whitespace-nowrap">{group.instanceCount} 实例</span>
                  <Button size="sm" variant="secondary" onClick={() => startEdit(group)} disabled={busy} title="重命名">
                    <Pencil className="w-4 h-4" />重命名
                  </Button>
                  <Button size="sm" variant="danger" onClick={() => setPendingDelete(group)} disabled={busy} title="删除">
                    <Trash2 className="w-4 h-4" />删除
                  </Button>
                </>
              )}
            </div>
          ))}
        </div>

        <div className="space-y-2 rounded-lg border border-[var(--color-border-default)] p-3">
          <div className="text-sm font-medium text-[var(--color-text-primary)]">新建分组</div>
          <div className="flex items-center gap-2 flex-wrap">
            <Input
              value={newName}
              onChange={event => setNewName(event.target.value)}
              onKeyDown={event => { if (event.key === 'Enter') void handleCreate() }}
              placeholder="分组名称"
              className="flex-1 min-w-[160px]"
            />
            {flatGroups.length > 0 && (
              <Select
                value={newParentId}
                onChange={event => setNewParentId(event.target.value)}
                options={parentOptions}
                style={{ width: '160px' }}
              />
            )}
            <Button size="sm" onClick={handleCreate} loading={busy}>
              <Plus className="w-4 h-4" />新建
            </Button>
          </div>
        </div>

        {error && <div className="text-sm text-[var(--color-error)]">{error}</div>}
      </div>

      <ConfirmModal
        open={pendingDelete !== null}
        onClose={() => setPendingDelete(null)}
        onConfirm={() => { const group = pendingDelete; if (group) void performDelete(group) }}
        title="删除分组"
        content={pendingDelete ? buildDeleteMessage(pendingDelete) : ''}
        confirmText="删除"
        danger
      />
    </Modal>
  )
}
