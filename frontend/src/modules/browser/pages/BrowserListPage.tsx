import { useEffect, useMemo, useRef, useState } from 'react'
import { toast } from '../../../shared/components'
import { fetchDashboardStats, redeemCDKey, redeemGithubStar, reloadConfig } from '../../dashboard/api'
import type { BrowserCore, BrowserCoreInput, BrowserProfile, BrowserProxy, BrowserSettings, BrowserGroupWithCount } from '../types'
import { BrowserCoreEditorModal, BrowserListHeader, BrowserListSettingsModal, type BrowserViewMode } from '../components/BrowserListLayout'
import { BatchToolbar } from '../components/BrowserListWidgets'
import { BrowserProfilesPanel } from '../components/BrowserProfilesPanel'
import { EMPTY_FILTERS } from '../components/InstanceFilterBar'
import type { InstanceFilters } from '../components/InstanceFilterBar'
import { EventsOn, BrowserOpenURL } from '../../../wailsjs/runtime/runtime'
import { PROJECT_GITHUB_URL } from '../../../config/links'
import { resolveActionErrorMessage, resolveActionFeedback } from '../utils/actionErrors'
import { BrowserListDialogs } from './browserList/BrowserListDialogs'
import {
  copyBrowserProfile,
  deleteBrowserCore,
  deleteBrowserProfile,
  fetchBrowserCores,
  fetchBrowserProfiles,
  fetchBrowserProxies,
  fetchBrowserSettings,
  fetchGroups,
  restartBrowserInstance,
  saveBrowserCore,
  saveBrowserSettings,
  setDefaultBrowserCore,
  startBrowserInstance,
  startBrowserInstanceDirect,
  stopBrowserInstance,
  validateBrowserCorePath,
  validateProxyConfig,
} from '../api'

const resolveProfileStatus = (running: boolean, debugReady: boolean, starting: boolean, stopping: boolean) => {
  if (starting) {
    return { variant: 'info' as const, label: '启动中' }
  }
  if (stopping) {
    return { variant: 'default' as const, label: '停止中' }
  }
  if (running && !debugReady) {
    return { variant: 'info' as const, label: '运行中（待就绪）' }
  }
  if (running) {
    return { variant: 'success' as const, label: '运行中' }
  }
  return { variant: 'warning' as const, label: '已停止' }
}

export function BrowserListPage() {
  const [profiles, setProfiles] = useState<BrowserProfile[]>([])
  const [loading, setLoading] = useState(true)
  const [proxies, setProxies] = useState<BrowserProxy[]>([])
  const [groups, setGroups] = useState<BrowserGroupWithCount[]>([])

  // 视图模式
  const [viewMode, setViewMode] = useState<BrowserViewMode>(() => {
    return (localStorage.getItem('browser:viewMode') as BrowserViewMode) || 'table'
  })

  // 勾选状态
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set())
  const [batchLoading, setBatchLoading] = useState(false)

  // 筛选状态（从 localStorage 恢复）
  const [filters, setFilters] = useState<InstanceFilters>(() => {
    try {
      const saved = localStorage.getItem('browser:filters')
      if (saved) {
        const parsed = JSON.parse(saved)
        return { ...EMPTY_FILTERS, ...parsed, tags: new Set(parsed.tags || []) }
      }
    } catch { /* ignore */ }
    return EMPTY_FILTERS
  })
  const [headerCollapsed, setHeaderCollapsed] = useState(() => {
    return localStorage.getItem('browser:headerCollapsed') === 'true'
  })

  // 持久化筛选状态
  useEffect(() => {
    const serializable = { ...filters, tags: Array.from(filters.tags) }
    localStorage.setItem('browser:filters', JSON.stringify(serializable))
  }, [filters])

  useEffect(() => {
    localStorage.setItem('browser:viewMode', viewMode)
  }, [viewMode])

  useEffect(() => {
    localStorage.setItem('browser:headerCollapsed', String(headerCollapsed))
  }, [headerCollapsed])

  // 代理不支持弹窗
  const [proxyErrorModal, setProxyErrorModal] = useState(false)
  const [proxyErrorMsg, setProxyErrorMsg] = useState('')
  const [opError, setOpError] = useState('')
  const [pendingStartId, setPendingStartId] = useState<string | null>(null)
  const [startingIds, setStartingIds] = useState<Set<string>>(new Set())
  const [stoppingIds, setStoppingIds] = useState<Set<string>>(new Set())
  const profilesRef = useRef<BrowserProfile[]>([])
  const silentRefreshInFlightRef = useRef(false)

  // 关键字弹窗
  const [kwModal, setKwModal] = useState<{ open: boolean; profile: BrowserProfile | null }>({ open: false, profile: null })

  const openKwModal = (profile: BrowserProfile) => setKwModal({ open: true, profile })
  const closeKwModal = () => setKwModal({ open: false, profile: null })

  // 复制弹窗
  const [copyModal, setCopyModal] = useState<{ open: boolean; profile: BrowserProfile | null }>({ open: false, profile: null })
  const [copyName, setCopyName] = useState('')
  const [copying, setCopying] = useState(false)

  const openCopyModal = (profile: BrowserProfile) => {
    setCopyName(profile.profileName + ' (副本)')
    setCopyModal({ open: true, profile })
  }
  const closeCopyModal = () => {
    setCopyModal({ open: false, profile: null })
    setCopyName('')
  }

  // 基础配置弹窗
  const [settingsModalOpen, setSettingsModalOpen] = useState(false)
  const [settings, setSettings] = useState<BrowserSettings>({
    userDataRoot: 'data',
    defaultFingerprintArgs: [],
    defaultLaunchArgs: [],
    defaultStartUrls: [],
    restoreLastSession: false,
    startReadyTimeoutMs: 3000,
    startStableWindowMs: 1200,
  })
  const [fingerprintText, setFingerprintText] = useState('')
  const [launchText, setLaunchText] = useState('')
  const [startUrlsText, setStartUrlsText] = useState('')
  const [savingSettings, setSavingSettings] = useState(false)

  // 内核管理
  const [cores, setCores] = useState<BrowserCore[]>([])
  const [coreModalOpen, setCoreModalOpen] = useState(false)
  const [coreForm, setCoreForm] = useState<BrowserCoreInput>({ coreId: '', coreName: '', corePath: '', isDefault: false })
  const [coreValidation, setCoreValidation] = useState<{ valid: boolean; message: string } | null>(null)
  const [savingCore, setSavingCore] = useState(false)

  // 扩容管理
  const [expandModalOpen, setExpandModalOpen] = useState(false)
  const [cdKey, setCdKey] = useState('')
  const [redeeming, setRedeeming] = useState(false)
  const [maxProfileLimit, setMaxProfileLimit] = useState(9999)

  const updatePendingIds = (
    setter: React.Dispatch<React.SetStateAction<Set<string>>>,
    profileId: string,
    active: boolean
  ) => {
    setter(prev => {
      const next = new Set(prev)
      if (active) {
        next.add(profileId)
      } else {
        next.delete(profileId)
      }
      return next
    })
  }

  const replaceProfilesState = (items: BrowserProfile[]) => {
    profilesRef.current = items
    setProfiles(items)
  }

  const updateProfilesState = (updater: (items: BrowserProfile[]) => BrowserProfile[]) => {
    const next = updater(profilesRef.current)
    profilesRef.current = next
    setProfiles(next)
  }

  const mergeProfileState = (profile: BrowserProfile | null | undefined) => {
    if (!profile) return
    updateProfilesState(prev => prev.map(item => (
      item.profileId === profile.profileId ? { ...item, ...profile } : item
    )))
  }

  const syncProfiles = (items: BrowserProfile[], syncRuntimeState: boolean) => {
    if (syncRuntimeState) {
      const previousById = new Map(profilesRef.current.map(item => [item.profileId, item]))
      const newlyRunning = items.find(item => item.running && !previousById.get(item.profileId)?.running)
      if (newlyRunning) {
        updatePendingIds(setStartingIds, newlyRunning.profileId, false)
        updatePendingIds(setStoppingIds, newlyRunning.profileId, false)
      }
      items.forEach(item => {
        if (!item.running && previousById.get(item.profileId)?.running) {
          updatePendingIds(setStartingIds, item.profileId, false)
          updatePendingIds(setStoppingIds, item.profileId, false)
        }
      })
    }
    replaceProfilesState(items)
  }

  const loadProfiles = async ({ silent = false, syncRuntimeState = false }: { silent?: boolean; syncRuntimeState?: boolean } = {}) => {
    if (silent && silentRefreshInFlightRef.current) {
      return profilesRef.current
    }
    if (!silent) {
      setLoading(true)
    } else {
      silentRefreshInFlightRef.current = true
    }
    try {
      const items = await fetchBrowserProfiles()
      syncProfiles(items, syncRuntimeState)
      return items
    } finally {
      if (silent) {
        silentRefreshInFlightRef.current = false
      } else {
        setLoading(false)
      }
    }
  }

  const loadGroups = async () => {
    setGroups(await fetchGroups())
  }

  const loadSettings = async () => {
    const data = await fetchBrowserSettings()
    setSettings(data)
    setFingerprintText((data.defaultFingerprintArgs || []).join('\n'))
    setLaunchText((data.defaultLaunchArgs || []).join('\n'))
    setStartUrlsText((data.defaultStartUrls || []).join('\n'))
  }

  const loadCores = async () => {
    setCores(await fetchBrowserCores())
  }

  const loadQuota = async () => {
    try {
      await reloadConfig()
      const stats = await fetchDashboardStats()
      setMaxProfileLimit(stats.maxProfileLimit || 9999)
    } catch {
      // ignore
    }
  }

  useEffect(() => {
    void loadProfiles()
    loadGroups()
    loadQuota()
    fetchBrowserProxies().then(setProxies)
    fetchBrowserCores().then(setCores)

    // 监听浏览器实例生命周期事件，自动更新状态
    const offStarted = EventsOn('browser:instance:started', (payload: any) => {
      const profileId = typeof payload === 'string' ? payload : payload?.profileId
      if (profileId) {
        updatePendingIds(setStartingIds, profileId, false)
        updatePendingIds(setStoppingIds, profileId, false)
      }
      void loadProfiles({ silent: true, syncRuntimeState: true })
    })
    const offUpdated = EventsOn('browser:instance:updated', () => {
      void loadProfiles({ silent: true, syncRuntimeState: true })
    })
    const offStopped = EventsOn('browser:instance:stopped', (payload: any) => {
      const profileId = typeof payload === 'string' ? payload : payload?.profileId
      if (profileId) {
        updatePendingIds(setStartingIds, profileId, false)
        updatePendingIds(setStoppingIds, profileId, false)
      }
      void loadProfiles({ silent: true, syncRuntimeState: true })
    })
    const offCrashed = EventsOn('browser:instance:crashed', (payload: any) => {
      const profileId = typeof payload === 'string' ? payload : payload?.profileId
      if (profileId) {
        updatePendingIds(setStartingIds, profileId, false)
        updatePendingIds(setStoppingIds, profileId, false)
      }
      void loadProfiles({ silent: true, syncRuntimeState: true })
    })

    const timer = window.setInterval(() => {
      if (document.visibilityState !== 'visible') return
      void loadProfiles({ silent: true, syncRuntimeState: true })
    }, 2000)

    return () => {
      window.clearInterval(timer)
      offStarted?.()
      offUpdated?.()
      offStopped?.()
      offCrashed?.()
    }
  }, [])

  const runningCount = useMemo(() => profiles.filter(p => p.running).length, [profiles])
  const allTags = useMemo(() => {
    const set = new Set<string>()
    profiles.forEach(p => p.tags?.forEach(t => set.add(t)))
    return Array.from(set).sort()
  }, [profiles])

  const defaultCore = useMemo(() => {
    return cores.find(core => core.isDefault) || cores[0] || null
  }, [cores])

  const resolveProfileCore = (profile: BrowserProfile) => {
    const coreId = (profile.coreId || '').trim()
    if (coreId && !/^default$/i.test(coreId)) {
      return cores.find(core => core.coreId === coreId) || null
    }
    return defaultCore
  }

  const getProfileCoreLabel = (profile: BrowserProfile) => {
    const resolvedCore = resolveProfileCore(profile)
    if (resolvedCore) {
      return resolvedCore.coreName
    }

    const coreId = (profile.coreId || '').trim()
    if (!coreId || /^default$/i.test(coreId)) {
      return '使用默认内核'
    }
    return coreId
  }

  const isProfileStarting = (profileId: string) => startingIds.has(profileId)
  const isProfileStopping = (profileId: string) => stoppingIds.has(profileId)
  const isProfileBusy = (profileId: string) => isProfileStarting(profileId) || isProfileStopping(profileId)

  const getProfileStatus = (profile: BrowserProfile) => (
    resolveProfileStatus(profile.running, profile.debugReady, isProfileStarting(profile.profileId), isProfileStopping(profile.profileId))
  )

  const filteredProfiles = useMemo(() => {
    const naturalCompare = (a: string, b: string): number => {
      const re = /(\d+)|(\D+)/g
      const partsA = a.match(re) || []
      const partsB = b.match(re) || []
      for (let i = 0; i < Math.max(partsA.length, partsB.length); i++) {
        if (i >= partsA.length) return -1
        if (i >= partsB.length) return 1
        const pa = partsA[i], pb = partsB[i]
        const na = Number(pa), nb = Number(pb)
        if (!isNaN(na) && !isNaN(nb)) {
          if (na !== nb) return na - nb
        } else {
          const cmp = pa.localeCompare(pb, 'zh-CN')
          if (cmp !== 0) return cmp
        }
      }
      return 0
    }
    return profiles.filter(p => {
      // 分组筛选
      if (filters.groupId === '__ungrouped__' && p.groupId) return false
      if (filters.groupId && filters.groupId !== '__ungrouped__' && p.groupId !== filters.groupId) return false

      if (filters.keyword && !p.profileName.toLowerCase().includes(filters.keyword.toLowerCase())) return false
      if (filters.status === 'running' && !p.running) return false
      if (filters.status === 'stopped' && p.running) return false
      if (filters.proxyId === '__none__' && (p.proxyId || p.proxyConfig)) return false
      if (filters.proxyId && filters.proxyId !== '__none__' && p.proxyId !== filters.proxyId) return false
      if (filters.coreId) {
        const effectiveCore = resolveProfileCore(p)
        if (!effectiveCore || effectiveCore.coreId !== filters.coreId) return false
      }
      if (filters.tags.size > 0 && !p.tags?.some(t => filters.tags.has(t))) return false
      if (filters.kwSearch) {
        const q = filters.kwSearch.toLowerCase()
        const hit = p.keywords?.some(v => v.toLowerCase().includes(q))
        if (!hit) return false
      }
      return true
    }).sort((a, b) => naturalCompare(a.profileName, b.profileName))
  }, [profiles, filters, defaultCore, cores])

  const handleStart = async (profileId: string) => {
    const profile = profiles.find(p => p.profileId === profileId)
    updatePendingIds(setStartingIds, profileId, true)
    try {
      if (profile) {
        const result = await validateProxyConfig(profile.proxyConfig || '', profile.proxyId || '')
        if (!result.supported) {
          setProxyErrorMsg(result.errorMsg)
          setPendingStartId(profileId)
          setProxyErrorModal(true)
          return
        }
      }

      const startedProfile = await startBrowserInstance(profileId)
      mergeProfileState(startedProfile)
      if (startedProfile?.running && !startedProfile.debugReady && startedProfile.runtimeWarning) {
        toast.warning(startedProfile.runtimeWarning)
      } else {
        toast.success(`实例已启动${startedProfile?.profileName ? `：${startedProfile.profileName}` : ''}`)
      }
      await loadProfiles({ silent: true, syncRuntimeState: true })
    } catch (error: any) {
      const feedback = resolveActionFeedback(error, '实例启动失败')
      if (feedback.tone === 'warning') {
        toast.warning(feedback.message)
      } else {
        toast.error(feedback.message)
      }
      await loadProfiles({ silent: true, syncRuntimeState: true })
    } finally {
      updatePendingIds(setStartingIds, profileId, false)
    }
  }

  const handleStartDirect = async (profileId: string) => {
    updatePendingIds(setStartingIds, profileId, true)
    try {
      const startedProfile = await startBrowserInstanceDirect(profileId)
      mergeProfileState(startedProfile)
      setProxyErrorModal(false)
      setPendingStartId(null)
      if (startedProfile?.running && !startedProfile.debugReady && startedProfile.runtimeWarning) {
        toast.warning(startedProfile.runtimeWarning)
      } else {
        toast.success(`实例已直连启动${startedProfile?.profileName ? `：${startedProfile.profileName}` : ''}`)
      }
      await loadProfiles({ silent: true, syncRuntimeState: true })
    } catch (error: any) {
      setProxyErrorModal(false)
      setPendingStartId(null)
      const feedback = resolveActionFeedback(error, '实例直连启动失败')
      if (feedback.tone === 'warning') {
        toast.warning(feedback.message)
      } else {
        toast.error(feedback.message)
      }
      await loadProfiles({ silent: true, syncRuntimeState: true })
    } finally {
      updatePendingIds(setStartingIds, profileId, false)
    }
  }

  const handleStop = async (profileId: string) => {
    updatePendingIds(setStoppingIds, profileId, true)
    try {
      const stoppedProfile = await stopBrowserInstance(profileId)
      mergeProfileState(stoppedProfile)
      toast.success('实例已停止')
      await loadProfiles({ silent: true, syncRuntimeState: true })
    } catch (error: any) {
      toast.error(resolveActionErrorMessage(error, '实例停止失败'))
      await loadProfiles({ silent: true, syncRuntimeState: true })
    } finally {
      updatePendingIds(setStoppingIds, profileId, false)
    }
  }

  const handleRestart = async (profileId: string) => {
    updatePendingIds(setStoppingIds, profileId, true)
    try {
      const restartedProfile = await restartBrowserInstance(profileId)
      mergeProfileState(restartedProfile)
      toast.success(`实例已重启${restartedProfile?.profileName ? `：${restartedProfile.profileName}` : ''}`)
      await loadProfiles({ silent: true, syncRuntimeState: true })
    } catch (error: any) {
      const feedback = resolveActionFeedback(error, '实例重启失败')
      if (feedback.tone === 'warning') {
        toast.warning(feedback.message)
      } else {
        setOpError(feedback.message)
      }
      await loadProfiles({ silent: true, syncRuntimeState: true })
    } finally {
      updatePendingIds(setStoppingIds, profileId, false)
    }
  }

  const handleDelete = async (profileId: string) => {
    await deleteBrowserProfile(profileId)
    toast.success('配置已删除')
    loadProfiles()
  }

  // 批量操作
  const toggleSelect = (profileId: string) => {
    setSelectedIds(prev => {
      const next = new Set(prev)
      next.has(profileId) ? next.delete(profileId) : next.add(profileId)
      return next
    })
  }



  const handleSelectAll = () => {
    setSelectedIds(new Set(filteredProfiles.map(p => p.profileId)))
  }

  const handleDeselectAll = () => {
    setSelectedIds(new Set())
  }

  const handleBatchStart = async () => {
    const ids = Array.from(selectedIds)
    if (ids.length === 0) return
    setBatchLoading(true)
    let success = 0, pending = 0, failed = 0
    const pendingMessages: string[] = []
    const failureMessages: string[] = []
    for (const id of ids) {
      const profile = profiles.find(p => p.profileId === id)
      if (!profile || profile.running) continue
      updatePendingIds(setStartingIds, id, true)
      try {
        const startedProfile = await startBrowserInstance(id)
        mergeProfileState(startedProfile)
        success++
      } catch (error: any) {
        const feedback = resolveActionFeedback(error, '实例启动失败')
        if (feedback.pendingAttach) {
          pending++
          pendingMessages.push(`${profile.profileName}：${feedback.message}`)
        } else {
          failed++
          failureMessages.push(`${profile.profileName}：${feedback.message}`)
        }
      } finally {
        updatePendingIds(setStartingIds, id, false)
      }
    }
    setBatchLoading(false)
    const summary = [`成功 ${success}`]
    if (pending > 0) summary.push(`待接管 ${pending}`)
    if (failed > 0) summary.push(`失败 ${failed}`)
    toast.success(`批量启动完成：${summary.join('，')}`)
    if (pendingMessages.length > 0) {
      const preview = pendingMessages.slice(0, 3)
      const more = pendingMessages.length > preview.length ? `\n另有 ${pendingMessages.length - preview.length} 个实例已打开窗口，仍在后台接管。` : ''
      toast.warning(`以下实例已打开窗口，仍在后台接管：\n${preview.join('\n')}${more}`)
    }
    if (failureMessages.length > 0) {
      const preview = failureMessages.slice(0, 3)
      const more = failureMessages.length > preview.length ? `\n另有 ${failureMessages.length - preview.length} 个实例启动失败，请逐个检查。` : ''
      toast.error(`以下实例启动失败：\n${preview.join('\n')}${more}`)
    }
    loadProfiles()
  }

  const handleBatchStop = async () => {
    const ids = Array.from(selectedIds)
    if (ids.length === 0) return
    setBatchLoading(true)
    let success = 0, failed = 0
    for (const id of ids) {
      const profile = profiles.find(p => p.profileId === id)
      if (!profile || !profile.running) continue
      updatePendingIds(setStoppingIds, id, true)
      try {
        const stoppedProfile = await stopBrowserInstance(id)
        mergeProfileState(stoppedProfile)
        success++
      } catch {
        failed++
      } finally {
        updatePendingIds(setStoppingIds, id, false)
      }
    }
    setBatchLoading(false)
    toast.success(`批量停止完成：成功 ${success}${failed > 0 ? `，失败 ${failed}` : ''}`)
    loadProfiles()
  }

  const handleBatchDelete = async () => {
    const ids = Array.from(selectedIds)
    if (ids.length === 0) return
    if (!confirm(`确定删除选中的 ${ids.length} 个实例？`)) return
    setBatchLoading(true)
    for (const id of ids) {
      await deleteBrowserProfile(id)
    }
    setBatchLoading(false)
    setSelectedIds(new Set())
    toast.success(`已删除 ${ids.length} 个实例`)
    loadProfiles()
  }

  const handleCopy = async (profileId: string) => {
    if (!copyModal.profile) return
    setCopying(true)
    try {
      await copyBrowserProfile(profileId, copyName)
      toast.success('实例已复制')
      closeCopyModal()
      loadProfiles()
    } catch (error: any) {
      closeCopyModal()
      setOpError(typeof error === 'string' ? error : error?.message || '复制失败')
    } finally {
      setCopying(false)
    }
  }

  const handleOpenSettings = async () => {
    await Promise.all([loadSettings(), loadCores()])
    setSettingsModalOpen(true)
  }

  const handleSaveSettings = async () => {
    setSavingSettings(true)
    try {
      await saveBrowserSettings({
        ...settings,
        defaultFingerprintArgs: fingerprintText.split('\n').map(s => s.trim()).filter(Boolean),
        defaultLaunchArgs: launchText.split('\n').map(s => s.trim()).filter(Boolean),
        defaultStartUrls: startUrlsText.split('\n').map(s => s.trim()).filter(Boolean),
      })
      toast.success('配置已保存')
      setSettingsModalOpen(false)
    } catch (error: any) {
      toast.error(error?.message || '保存失败')
    } finally {
      setSavingSettings(false)
    }
  }

  // 内核管理
  const handleOpenCoreModal = (core?: BrowserCore) => {
    setCoreForm(core ? { ...core } : { coreId: '', coreName: '', corePath: '', isDefault: false })
    setCoreValidation(null)
    setCoreModalOpen(true)
  }

  const handleValidateCorePath = async () => {
    if (!coreForm.corePath.trim()) {
      setCoreValidation({ valid: false, message: '请输入路径' })
      return
    }
    const result = await validateBrowserCorePath(coreForm.corePath)
    setCoreValidation(result)
  }

  const handleSaveCore = async () => {
    if (!coreForm.coreName.trim()) {
      toast.error('请输入内核名称')
      return
    }
    if (!coreForm.corePath.trim()) {
      toast.error('请输入内核路径')
      return
    }
    setSavingCore(true)
    try {
      await saveBrowserCore(coreForm)
      toast.success('内核已保存')
      setCoreModalOpen(false)
      loadCores()
    } catch (error: any) {
      toast.error(error?.message || '保存失败')
    } finally {
      setSavingCore(false)
    }
  }

  const handleDeleteCore = async (coreId: string) => {
    if (cores.length <= 1) {
      toast.error('至少保留一个内核')
      return
    }
    await deleteBrowserCore(coreId)
    toast.success('内核已删除')
    loadCores()
  }

  const handleSetDefaultCore = async (coreId: string) => {
    await setDefaultBrowserCore(coreId)
    toast.success('已设为默认')
    loadCores()
  }

  const handleRedeem = async () => {
    if (!cdKey.trim()) return
    setRedeeming(true)
    const result = await redeemCDKey(cdKey.trim())
    setRedeeming(false)
    if (result.success) {
      toast.success('兑换成功！此名额已到账')
      setCdKey('')
      loadQuota()
    } else {
      toast.error(result.message || '兑换失败')
    }
  }

  const handleClaimStarGift = async () => {
    setRedeeming(true)
    const starRes = await redeemGithubStar()
    setRedeeming(false)
    if (starRes.success) {
      toast.success('感谢您的支持！已额外赠送 50 个永久额度！')
      setCdKey('')
      loadQuota()
    } else {
      toast.error(starRes.message || '领取失败')
    }
  }

  const handleOpenGithubStarGift = async () => {
    BrowserOpenURL(PROJECT_GITHUB_URL)
    await handleClaimStarGift()
  }


  return (
    <div className="overflow-auto p-5 space-y-5 animate-fade-in h-full">
      <BrowserListHeader
        profileCount={profiles.length}
        filteredProfileCount={filteredProfiles.length}
        runningCount={runningCount}
        headerCollapsed={headerCollapsed}
        viewMode={viewMode}
        proxies={proxies}
        cores={cores}
        groups={groups}
        allTags={allTags}
        filters={filters}
        onFiltersChange={setFilters}
        onToggleHeaderCollapsed={() => setHeaderCollapsed((prev) => !prev)}
        onRefresh={() => { void loadProfiles() }}
        onOpenSettings={handleOpenSettings}
        onOpenExpandModal={() => {
          setCdKey('')
          setExpandModalOpen(true)
          loadQuota()
        }}
        onViewModeChange={setViewMode}
      />

      {/* 批量操作工具栏 */}
      <BatchToolbar
        selectedCount={selectedIds.size}
        totalCount={filteredProfiles.length}
        onSelectAll={handleSelectAll}
        onDeselectAll={handleDeselectAll}
        onBatchStart={handleBatchStart}
        onBatchStop={handleBatchStop}
        onBatchDelete={handleBatchDelete}
        batchLoading={batchLoading}
      />

      <BrowserProfilesPanel
        loading={loading}
        viewMode={viewMode}
        profiles={filteredProfiles}
        proxies={proxies}
        selectedIds={selectedIds}
        resolveProfileCore={resolveProfileCore}
        getProfileCoreLabel={getProfileCoreLabel}
        getProfileStatus={getProfileStatus}
        isProfileStarting={isProfileStarting}
        isProfileStopping={isProfileStopping}
        isProfileBusy={isProfileBusy}
        onToggleSelect={toggleSelect}
        onSelectAll={handleSelectAll}
        onDeselectAll={handleDeselectAll}
        onRefreshProfiles={() => { void loadProfiles() }}
        onStart={(profileId) => { void handleStart(profileId) }}
        onStop={(profileId) => { void handleStop(profileId) }}
        onRestart={(profileId) => { void handleRestart(profileId) }}
        onOpenKeywords={openKwModal}
        onOpenCopy={openCopyModal}
        onDelete={(profileId) => { void handleDelete(profileId) }}
      />

      <BrowserListSettingsModal
        open={settingsModalOpen}
        settings={settings}
        fingerprintText={fingerprintText}
        launchText={launchText}
        startUrlsText={startUrlsText}
        savingSettings={savingSettings}
        cores={cores}
        onClose={() => setSettingsModalOpen(false)}
        onSave={handleSaveSettings}
        onSettingsChange={(patch) => setSettings((prev) => ({ ...prev, ...patch }))}
        onFingerprintTextChange={setFingerprintText}
        onLaunchTextChange={setLaunchText}
        onStartUrlsTextChange={setStartUrlsText}
        onAddCore={() => handleOpenCoreModal()}
        onEditCore={handleOpenCoreModal}
        onDeleteCore={handleDeleteCore}
        onSetDefaultCore={handleSetDefaultCore}
      />

      <BrowserCoreEditorModal
        open={coreModalOpen}
        coreForm={coreForm}
        coreValidation={coreValidation}
        savingCore={savingCore}
        onClose={() => setCoreModalOpen(false)}
        onSave={handleSaveCore}
        onValidate={handleValidateCorePath}
        onCoreFormChange={(patch) => {
          setCoreForm((prev) => ({ ...prev, ...patch }))
          if (Object.prototype.hasOwnProperty.call(patch, 'corePath')) {
            setCoreValidation(null)
          }
        }}
      />

      <BrowserListDialogs
        proxyErrorModal={proxyErrorModal}
        pendingStartId={pendingStartId}
        proxyErrorMsg={proxyErrorMsg}
        onCloseProxyError={() => {
          setProxyErrorModal(false)
          setPendingStartId(null)
        }}
        onStartDirect={() => {
          if (pendingStartId) {
            void handleStartDirect(pendingStartId)
          }
        }}
        startingDirect={pendingStartId ? startingIds.has(pendingStartId) : false}
        kwModal={kwModal}
        onCloseKeywords={closeKwModal}
        onKeywordsSaved={(keywords) => {
          updateProfilesState(prev => prev.map(p =>
            p.profileId === kwModal.profile!.profileId ? { ...p, keywords } : p
          ))
        }}
        expandModalOpen={expandModalOpen}
        onCloseExpand={() => setExpandModalOpen(false)}
        profilesCount={profiles.length}
        maxProfileLimit={maxProfileLimit}
        cdKey={cdKey}
        onCdKeyChange={setCdKey}
        onRedeem={handleRedeem}
        redeeming={redeeming}
        onOpenGithubStarGift={handleOpenGithubStarGift}
        copyModal={copyModal}
        copyName={copyName}
        onCopyNameChange={setCopyName}
        onCloseCopy={closeCopyModal}
        onConfirmCopy={() => copyModal.profile && handleCopy(copyModal.profile.profileId)}
        copying={copying}
        opError={opError}
        onCloseOpError={() => setOpError('')}
      />
    </div>
  )
}
