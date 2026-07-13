import { useState } from 'react'
import { toast } from '../../../../shared/components'
import type { BrowserProxy } from '../../types'
import { BUILTIN_PROXY_IDS, type ProxyDisplayInfo } from './helpers'

interface UseProxySelectionOptions {
  proxies: BrowserProxy[]
  filteredList: ProxyDisplayInfo[]
  saveProxies: (list: BrowserProxy[]) => Promise<void>
}

export function useProxySelection({ proxies, filteredList, saveProxies }: UseProxySelectionOptions) {
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set())
  const [batchDeleteConfirmOpen, setBatchDeleteConfirmOpen] = useState(false)

  const allFilteredSelected = filteredList.length > 0 && filteredList.every(p => selectedIds.has(p.proxyId))
  const someFilteredSelected = filteredList.some(p => selectedIds.has(p.proxyId))
  const selectedCount = selectedIds.size

  const handleToggleAll = () => {
    if (allFilteredSelected) {
      setSelectedIds(prev => {
        const next = new Set(prev)
        filteredList.forEach(p => next.delete(p.proxyId))
        return next
      })
    } else {
      setSelectedIds(prev => {
        const next = new Set(prev)
        filteredList.filter(p => !BUILTIN_PROXY_IDS.has(p.proxyId)).forEach(p => next.add(p.proxyId))
        return next
      })
    }
  }

  const handleToggleOne = (proxyId: string) => {
    if (BUILTIN_PROXY_IDS.has(proxyId)) return
    setSelectedIds(prev => {
      const next = new Set(prev)
      next.has(proxyId) ? next.delete(proxyId) : next.add(proxyId)
      return next
    })
  }

  const handleBatchDeleteConfirm = async () => {
    try {
      const newProxies = proxies.filter(p => !selectedIds.has(p.proxyId))
      await saveProxies(newProxies)
      toast.success(`已删除 ${selectedIds.size} 个代理`)
      setSelectedIds(new Set())
    } catch (error: any) {
      toast.error(error?.message || '删除失败')
    }
  }

  const removeSelectedId = (proxyId: string) => {
    setSelectedIds(prev => {
      const next = new Set(prev)
      next.delete(proxyId)
      return next
    })
  }

  return {
    selectedIds,
    selectedCount,
    allFilteredSelected,
    someFilteredSelected,
    batchDeleteConfirmOpen,
    setBatchDeleteConfirmOpen,
    handleToggleAll,
    handleToggleOne,
    handleBatchDeleteConfirm,
    removeSelectedId,
  }
}
