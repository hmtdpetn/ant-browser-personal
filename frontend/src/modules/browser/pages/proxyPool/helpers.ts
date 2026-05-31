import yaml from 'js-yaml'

import type { BrowserProxy } from '../../types'

export const BUILTIN_PROXY_IDS = new Set(['__direct__'])

const BUILTIN_PROXIES: BrowserProxy[] = [
  { proxyId: '__direct__', proxyName: '直连（不走代理）', proxyConfig: 'direct://' },
]

export interface ClashProxy {
  name: string
  type: string
  server: string
  port: number
  [key: string]: unknown
}

export type ProxyImportMode = 'clash' | 'direct' | 'chain'

export interface DirectImportForm {
  proxyName: string
  protocol: 'http' | 'https' | 'socks5'
  server: string
  port: string
  username: string
  password: string
}

export interface ChainHopForm {
  protocol: 'http' | 'socks5'
  server: string
  port: string
  username: string
  password: string
}

export interface ChainImportForm {
  proxyName: string
  localPort: string
  first: ChainHopForm
  second: ChainHopForm
}

interface ChainSocks5HopConfig {
  protocol: 'http' | 'socks5'
  server: string
  port: number
  username?: string
  password?: string
}

interface ChainSocks5Config {
  localPort?: number
  first: ChainSocks5HopConfig
  second: ChainSocks5HopConfig
}

const CHAIN_SOCKS5_PREFIX = 'chain+socks5://'

export const CHAIN_QUICK_IMPORT_TEMPLATE = `{
  "name": "",
  "group": "",
  "localPort": "",
  "first": {
    "protocol": "http",
    "server": "",
    "port": "",
    "username": "",
    "password": ""
  },
  "second": {
    "protocol": "http",
    "server": "",
    "port": "",
    "username": "",
    "password": ""
  }
}`

export const DIRECT_QUICK_IMPORT_TEMPLATE = `{
  "name": "",
  "group": "",
  "protocol": "http",
  "server": "",
  "port": "",
  "username": "",
  "password": ""
}`

export const DIRECT_PROXY_PROTOCOL_OPTIONS = [
  { value: 'http', label: 'HTTP' },
  { value: 'https', label: 'HTTPS' },
  { value: 'socks5', label: 'SOCKS5' },
] as const

export const INITIAL_DIRECT_IMPORT_FORM: DirectImportForm = {
  proxyName: '',
  protocol: 'http',
  server: '',
  port: '',
  username: '',
  password: '',
}

export const INITIAL_CHAIN_IMPORT_FORM: ChainImportForm = {
  proxyName: '',
  localPort: '',
  first: {
    protocol: 'http',
    server: '',
    port: '',
    username: '',
    password: '',
  },
  second: {
    protocol: 'http',
    server: '',
    port: '',
    username: '',
    password: '',
  },
}

function parseChainSocks5Config(proxyConfig: string): ChainSocks5Config | null {
  const cfg = proxyConfig.trim()
  if (!cfg.toLowerCase().startsWith(CHAIN_SOCKS5_PREFIX)) {
    return null
  }

  const encoded = cfg.slice(CHAIN_SOCKS5_PREFIX.length)
  if (!encoded) {
    return null
  }

  const normalizeHop = (raw: unknown): ChainSocks5HopConfig | null => {
    if (!raw || typeof raw !== 'object') return null
    const hop = raw as Record<string, unknown>
    const protocol = String(hop.protocol || '').trim().toLowerCase()
    if (protocol && protocol !== 'socks5' && protocol !== 'http') return null

    const server = String(hop.server || '').trim()
    if (!server) return null

    const portVal = Number(hop.port || 0)
    if (!Number.isInteger(portVal) || portVal < 1 || portVal > 65535) return null

    const username = String(hop.username || '').trim()
    const password = hop.password === undefined || hop.password === null ? '' : String(hop.password)
    if (password && !username) return null

    return {
      protocol: protocol === 'http' ? 'http' : 'socks5',
      server,
      port: portVal,
      username: username || undefined,
      password: password || undefined,
    }
  }

  try {
    const decoded = decodeURIComponent(encoded)
    const parsed = JSON.parse(decoded) as Record<string, unknown>
    const first = normalizeHop(parsed.first)
    const second = normalizeHop(parsed.second)
    if (!first || !second) return null

    const localPortRaw = parsed.localPort
    const localPortNum = localPortRaw === undefined || localPortRaw === null || localPortRaw === ''
      ? 0
      : Number(localPortRaw)
    if (!Number.isInteger(localPortNum) || localPortNum < 0 || localPortNum > 65535) return null

    return {
      first,
      second,
      localPort: localPortNum > 0 ? localPortNum : undefined,
    }
  } catch {
    return null
  }
}

export function toChainImportForm(proxyName: string, proxyConfig: string): ChainImportForm | null {
  const cfg = parseChainSocks5Config(proxyConfig)
  if (!cfg) {
    return null
  }

  return {
    proxyName,
    localPort: cfg.localPort ? String(cfg.localPort) : '',
    first: {
      protocol: cfg.first.protocol,
      server: cfg.first.server,
      port: String(cfg.first.port),
      username: cfg.first.username || '',
      password: cfg.first.password || '',
    },
    second: {
      protocol: cfg.second.protocol,
      server: cfg.second.server,
      port: String(cfg.second.port),
      username: cfg.second.username || '',
      password: cfg.second.password || '',
    },
  }
}

export interface ImportCandidate {
  proxyName: string
  proxyConfig: string
  groupName?: string
}

export interface ProxyDisplayInfo {
  proxyId: string
  proxyName: string
  proxyConfig: string
  groupName: string
  sourceId: string
  sourceUrl: string
  sourceAutoRefresh: boolean
  sourceRefreshIntervalM: number
  sourceLastRefreshAt: string
  type: string
  server: string
  port: number
  latencyMs?: number
}

export interface URLImportSourceMeta {
  sourceId: string
  sourceUrl: string
  sourceNamePrefix: string
  sourceGroupName: string
  sourceDnsServers: string
  sourceAutoRefresh: boolean
  sourceRefreshIntervalM: number
  sourceLastRefreshAt: string
}

export function ensureBuiltinProxies(proxies: BrowserProxy[]): BrowserProxy[] {
  const result = [...proxies]
  for (const builtin of BUILTIN_PROXIES) {
    if (!result.find((proxy) => proxy.proxyId === builtin.proxyId)) {
      result.unshift(builtin)
    }
  }
  return result
}

export function parseProxyInfo(proxyConfig: string): { type: string; server: string; port: number } {
  const cfg = proxyConfig.trim()
  if (cfg === 'direct://') return { type: 'direct', server: '-', port: 0 }

  const chain = parseChainSocks5Config(cfg)
  if (chain) {
    return { type: 'chain-socks5', server: '127.0.0.1', port: chain.localPort || 0 }
  }

  const urlMatch = cfg.match(/^([a-zA-Z0-9+\-]+):\/\//)
  if (urlMatch) {
    const scheme = urlMatch[1].toLowerCase()
    try {
      const parsed = new URL(cfg)
      return { type: scheme, server: parsed.hostname, port: parseInt(parsed.port, 10) || 0 }
    } catch {
      return { type: scheme, server: '-', port: 0 }
    }
  }

  try {
    const parsed = yaml.load(cfg) as ClashProxy[] | ClashProxy
    const proxy = Array.isArray(parsed) ? parsed[0] : parsed
    return { type: proxy?.type || '-', server: proxy?.server || '-', port: proxy?.port || 0 }
  } catch {
    return { type: '-', server: '-', port: 0 }
  }
}

export function toDisplayList(proxies: BrowserProxy[]): ProxyDisplayInfo[] {
  return proxies.map((proxy) => {
    const info = parseProxyInfo(proxy.proxyConfig)
    return {
      proxyId: proxy.proxyId,
      proxyName: proxy.proxyName,
      proxyConfig: proxy.proxyConfig,
      groupName: proxy.groupName || '',
      sourceId: proxy.sourceId || '',
      sourceUrl: proxy.sourceUrl || '',
      sourceAutoRefresh: !!proxy.sourceAutoRefresh,
      sourceRefreshIntervalM: Math.max(0, Number(proxy.sourceRefreshIntervalM || 0)),
      sourceLastRefreshAt: proxy.sourceLastRefreshAt || '',
      ...info,
    }
  })
}

function proxyToYaml(proxy: ClashProxy): string {
  return yaml.dump([proxy], { flowLevel: -1, lineWidth: -1 }).trim()
}

function quoteYamlScalar(value: string): string {
  const trimmed = value.trim()
  if (!trimmed) return "''"
  return `'${trimmed.replace(/'/g, "''")}'`
}

function normalizeImportedProxyArray(payload: unknown): ClashProxy[] | null {
  const asArray = (input: unknown): ClashProxy[] => {
    if (!Array.isArray(input)) return []
    return input.filter((item): item is ClashProxy => !!item && typeof item === 'object')
  }

  if (Array.isArray(payload)) {
    return asArray(payload)
  }
  if (!payload || typeof payload !== 'object') {
    return null
  }

  const record = payload as Record<string, unknown>
  if (Array.isArray(record.proxies)) {
    return asArray(record.proxies)
  }
  if (Array.isArray(record.proxy)) {
    return asArray(record.proxy)
  }
  if (Array.isArray(record.Proxy)) {
    return asArray(record.Proxy)
  }
  return null
}

function normalizeLooseClashImportText(raw: string): string {
  const normalizedNewline = raw.replace(/\uFEFF/g, '').replace(/\r\n/g, '\n').trim()
  if (!normalizedNewline) return normalizedNewline

  const fixedLines = normalizedNewline.split('\n').map((line) => {
    const match = line.match(/^(\s*)-\s*([^,{][^,]*?)\s*,\s*(type\s*:.*)$/i)
    if (!match) return line
    const indent = match[1] || ''
    const name = match[2] || ''
    const tail = match[3] || ''
    return `${indent}- { name: ${quoteYamlScalar(name)}, ${tail.trim()} }`
  })

  const hasProxiesRoot = fixedLines.some((line) => /^\s*proxies\s*:/.test(line))
  if (hasProxiesRoot) {
    return fixedLines.join('\n')
  }

  const looksLikeProxyList = fixedLines.some((line) => /^\s*-\s*/.test(line))
  if (!looksLikeProxyList) {
    return fixedLines.join('\n')
  }

  const indented = fixedLines.map((line) => (line.trim() ? `  ${line}` : line))
  return `proxies:\n${indented.join('\n')}`
}

export function parseClashImportText(raw: string): ClashProxy[] {
  const input = raw.trim()
  if (!input) {
    throw new Error('请输入 YAML 内容')
  }

  const attempts = [input]
  const normalized = normalizeLooseClashImportText(input)
  if (normalized && normalized !== input) {
    attempts.push(normalized)
  }

  let lastError: unknown = null
  for (const text of attempts) {
    try {
      const parsed = yaml.load(text)
      const proxies = normalizeImportedProxyArray(parsed)
      if (proxies) {
        return proxies
      }
    } catch (error) {
      lastError = error
    }
  }

  if (lastError && typeof lastError === 'object' && lastError !== null && 'message' in lastError) {
    throw new Error(String((lastError as { message?: string }).message || '解析失败'))
  }
  throw new Error('无效的 YAML 格式，需要包含 proxies 数组')
}

function normalizeDirectProxyConfig(raw: string): string {
  const trimmed = raw.trim()
  if (!trimmed) return ''
  if (/^socket:\/\//i.test(trimmed)) {
    return trimmed.replace(/^socket:\/\//i, 'socks5://')
  }
  if (/^socks:\/\//i.test(trimmed)) {
    return trimmed.replace(/^socks:\/\//i, 'socks5://')
  }
  return trimmed
}

function resolveDirectProxyName(
  rawName: string,
  scheme: string,
  server: string,
  port: number,
  index: number,
  prefix: string,
): string {
  const name = rawName.trim()
  const fallbackName = server
    ? `${scheme.toUpperCase()}-${server}${port > 0 ? `:${port}` : ''}`
    : `导入代理 ${index + 1}`
  const finalName = name || fallbackName
  return prefix ? `${prefix}-${finalName}` : finalName
}

function formatDirectProxyHost(raw: string): string {
  const host = raw.trim()
  if (!host) return ''
  if (host.startsWith('[') && host.endsWith(']')) {
    return host
  }
  return host.includes(':') ? `[${host}]` : host
}

function normalizeDirectProtocol(raw: unknown): DirectImportForm['protocol'] {
  const protocol = String(raw || '').trim().toLowerCase()
  if (protocol === 'http' || protocol === 'https' || protocol === 'socks5') {
    return protocol
  }
  if (protocol === 'socks' || protocol === 'socket') {
    return 'socks5'
  }
  throw new Error('protocol 仅支持 http / https / socks5')
}

function parseDirectProxyURL(raw: string): DirectImportForm {
  const normalized = normalizeDirectProxyConfig(raw)
  if (!normalized) {
    throw new Error('请输入标准代理地址')
  }
  if (!/^[a-zA-Z][a-zA-Z0-9+.-]*:\/\//.test(normalized)) {
    throw new Error('单行文本需要包含协议头，需要包含协议头')
  }

  let parsedURL: URL
  try {
    parsedURL = new URL(normalized)
  } catch {
    throw new Error('单行代理文本格式无效')
  }

  const protocol = normalizeDirectProtocol(parsedURL.protocol.replace(/:$/, ''))
  const server = parsedURL.hostname.replace(/^\[(.*)\]$/, '$1').trim()
  if (!server) {
    throw new Error('代理地址缺少主机名')
  }

  const port = Number(parsedURL.port)
  if (!Number.isInteger(port) || port < 1 || port > 65535) {
    throw new Error('代理地址缺少有效端口')
  }

  return {
    proxyName: '',
    protocol,
    server,
    port: String(port),
    username: parsedURL.username ? decodeURIComponent(parsedURL.username) : '',
    password: parsedURL.password ? decodeURIComponent(parsedURL.password) : '',
  }
}

export function buildDirectImportCandidate(form: DirectImportForm): ImportCandidate {
  const serverInput = form.server.trim()
  if (!serverInput) {
    throw new Error('请输入代理地址')
  }
  if (/^[a-zA-Z][a-zA-Z0-9+.-]*:\/\//.test(serverInput)) {
    throw new Error('代理地址只需要填写主机名或 IP，不需要协议头')
  }

  const portInput = form.port.trim()
  if (!portInput) {
    throw new Error('请输入代理端口')
  }
  if (!/^\d+$/.test(portInput)) {
    throw new Error('代理端口必须为数字')
  }

  const port = Number(portInput)
  if (port < 1 || port > 65535) {
    throw new Error('代理端口必须在 1-65535 之间')
  }

  const username = form.username.trim()
  const password = form.password
  if (password && !username) {
    throw new Error('填写密码时请同时填写账号')
  }

  const auth = username
    ? `${encodeURIComponent(username)}${password ? `:${encodeURIComponent(password)}` : ''}@`
    : ''
  const rawConfig = `${form.protocol}://${auth}${formatDirectProxyHost(serverInput)}:${port}`

  let parsedURL: URL
  try {
    parsedURL = new URL(rawConfig)
  } catch {
    throw new Error('请输入有效的代理地址')
  }

  if (!parsedURL.hostname) {
    throw new Error('请输入有效的代理地址')
  }

  const normalizedConfig = normalizeDirectProxyConfig(parsedURL.toString()).replace(/\/$/, '')
  const normalizedServer = parsedURL.hostname.replace(/^\[(.*)\]$/, '$1')

  return {
    proxyName: resolveDirectProxyName(form.proxyName, form.protocol, normalizedServer, port, 0, ''),
    proxyConfig: normalizedConfig,
  }
}

interface ParsedDirectImportItem {
  form: DirectImportForm
  groupName: string
}

function parseDirectImportObject(payload: Record<string, unknown>, fallbackGroupName: string): ParsedDirectImportItem {
  const proxyName = String(payload.name ?? payload.proxyName ?? '').trim()
  const groupName = String(payload.group ?? payload.groupName ?? fallbackGroupName).trim()
  const proxyURL = String(payload.url ?? payload.proxyUrl ?? payload.proxy ?? payload.proxyConfig ?? '').trim()
  if (proxyURL) {
    const parsedForm = parseDirectProxyURL(proxyURL)
    return {
      form: {
        ...parsedForm,
        proxyName: proxyName || parsedForm.proxyName,
      },
      groupName,
    }
  }

  const protocol = normalizeDirectProtocol(payload.protocol ?? payload.scheme)
  const server = String(payload.server ?? payload.host ?? '').trim()
  if (!server) {
    throw new Error('JSON 缺少 server')
  }

  const portValue = Number(payload.port)
  if (!Number.isInteger(portValue) || portValue < 1 || portValue > 65535) {
    throw new Error('JSON 缺少有效 port')
  }

  const username = String(payload.username ?? payload.user ?? '').trim()
  const password = payload.password === undefined || payload.password === null ? '' : String(payload.password)
  if (password && !username) {
    throw new Error('填写 password 时请同时填写 username')
  }

  return {
    form: {
      proxyName,
      protocol,
      server,
      port: String(portValue),
      username,
      password,
    },
    groupName,
  }
}

function parseDirectImportItems(raw: string): { items: ParsedDirectImportItem[]; defaultGroupName: string } {
  const text = raw.trim()
  if (!text) {
    throw new Error('请输入 HTTP / SOCKS5 文本')
  }

  if (text.startsWith('{') || text.startsWith('[')) {
    let payload: unknown
    try {
      payload = JSON.parse(text)
    } catch {
      throw new Error('JSON 格式无效')
    }

    let defaultGroupName = ''
    let sources: unknown[] = []
    if (Array.isArray(payload)) {
      sources = payload
    } else if (payload && typeof payload === 'object') {
      const record = payload as Record<string, unknown>
      defaultGroupName = String(record.group ?? record.groupName ?? '').trim()
      if (Array.isArray(record.proxies)) {
        sources = record.proxies
      } else if (Array.isArray(record.items)) {
        sources = record.items
      } else if (Array.isArray(record.list)) {
        sources = record.list
      } else {
        sources = [record]
      }
    } else {
      throw new Error('JSON 根节点必须是对象或数组')
    }

    const items = sources.map((item, index) => {
      if (typeof item === 'string') {
        return {
          form: parseDirectProxyURL(item),
          groupName: defaultGroupName,
        }
      }
      if (!item || typeof item !== 'object' || Array.isArray(item)) {
        throw new Error(`第 ${index + 1} 项格式无效`)
      }
      return parseDirectImportObject(item as Record<string, unknown>, defaultGroupName)
    })

    if (items.length === 0) {
      throw new Error('JSON 未解析到可导入代理')
    }
    return { items, defaultGroupName }
  }

  const lines = text
    .split(/\r?\n/)
    .map((line) => line.trim())
    .filter((line) => line && !line.startsWith('#') && !line.startsWith('//'))
  if (lines.length === 0) {
    throw new Error('请输入标准代理地址')
  }

  return {
    items: lines.map((line) => ({
      form: parseDirectProxyURL(line),
      groupName: '',
    })),
    defaultGroupName: '',
  }
}

export function parseDirectImportText(raw: string): { form: DirectImportForm; groupName: string } {
  const { items } = parseDirectImportItems(raw)
  if (items.length !== 1) {
    throw new Error('检测到多条代理，请直接点击解析进行批量导入')
  }
  return items[0]
}

export function buildDirectImportCandidatesFromText(raw: string): { candidates: ImportCandidate[]; defaultGroupName: string } {
  const { items, defaultGroupName } = parseDirectImportItems(raw)
  return {
    candidates: items.map((item) => ({
      ...buildDirectImportCandidate(item.form),
      groupName: item.groupName,
    })),
    defaultGroupName,
  }
}

export function buildChainImportCandidate(form: ChainImportForm): ImportCandidate {
  const parseHop = (label: string, hop: ChainHopForm): ChainSocks5HopConfig => {
    const protocol = hop.protocol === 'socks5' ? 'socks5' : 'http'
    const server = hop.server.trim()
    if (!server) {
      throw new Error(`请输入${label}代理地址`)
    }
    if (/^[a-zA-Z][a-zA-Z0-9+.-]*:\/\//.test(server)) {
      throw new Error(`${label}代理地址只需要填写主机名或 IP，不需要协议头`)
    }

    const portInput = hop.port.trim()
    if (!portInput) {
      throw new Error(`请输入${label}代理端口`)
    }
    if (!/^\d+$/.test(portInput)) {
      throw new Error(`${label}代理端口必须为数字`)
    }

    const port = Number(portInput)
    if (!Number.isInteger(port) || port < 1 || port > 65535) {
      throw new Error(`${label}代理端口必须在 1-65535 之间`)
    }

    const username = hop.username.trim()
    const password = hop.password
    if (password && !username) {
      throw new Error(`${label}填写密码时请同时填写账号`)
    }

    return {
      protocol,
      server,
      port,
      username: username || undefined,
      password: password || undefined,
    }
  }

  const localPortInput = form.localPort.trim()
  if (localPortInput && !/^\d+$/.test(localPortInput)) {
    throw new Error('本地监听端口必须为数字')
  }
  const localPort = localPortInput ? Number(localPortInput) : 0
  if (localPortInput && (!Number.isInteger(localPort) || localPort < 1 || localPort > 65535)) {
    throw new Error('本地监听端口必须在 1-65535 之间')
  }

  const payload: ChainSocks5Config = {
    first: parseHop('第一层', form.first),
    second: parseHop('第二层', form.second),
    localPort: localPort > 0 ? localPort : undefined,
  }

  return {
    proxyName: form.proxyName.trim() || `链式代理-${payload.first.server}-${payload.second.server}`,
    proxyConfig: `${CHAIN_SOCKS5_PREFIX}${encodeURIComponent(JSON.stringify(payload))}`,
  }
}

function parseOptionalChainPort(raw: unknown, label: string): number | undefined {
  if (raw === undefined || raw === null || String(raw).trim() === '') {
    return undefined
  }
  const value = Number(raw)
  if (!Number.isInteger(value) || value < 1 || value > 65535) {
    throw new Error(`${label}必须在 1-65535 之间`)
  }
  return value
}

function parseChainQuickImportHop(raw: unknown, label: string): ChainSocks5HopConfig {
  if (!raw || typeof raw !== 'object') {
    throw new Error(`${label}缺少配置`)
  }

  const hop = raw as Record<string, unknown>
  const protocol = String(hop.protocol || 'socks5').trim().toLowerCase()
  if (protocol !== 'socks5' && protocol !== 'http') {
    throw new Error(`${label}仅支持 http / socks5`)
  }

  const server = String(hop.server || '').trim()
  if (!server) {
    throw new Error(`${label}缺少 server`)
  }

  const portValue = Number(hop.port)
  if (!Number.isInteger(portValue) || portValue < 1 || portValue > 65535) {
    throw new Error(`${label}缺少有效 port`)
  }

  const username = String(hop.username || '').trim()
  const password = hop.password === undefined || hop.password === null ? '' : String(hop.password)
  if (password && !username) {
    throw new Error(`${label}填写 password 时请同时填写 username`)
  }

  return {
    protocol: protocol === 'http' ? 'http' : 'socks5',
    server,
    port: portValue,
    username: username || undefined,
    password: password || undefined,
  }
}

export function parseChainImportJSON(raw: string): { form: ChainImportForm; groupName: string } {
  const text = raw.trim()
  if (!text) {
    throw new Error('请输入链式代理 JSON')
  }

  let payload: Record<string, unknown>
  try {
    payload = JSON.parse(text) as Record<string, unknown>
  } catch {
    throw new Error('JSON 格式无效')
  }

  if (!payload || typeof payload !== 'object' || Array.isArray(payload)) {
    throw new Error('JSON 根节点必须是对象')
  }

  const first = parseChainQuickImportHop(payload.first, '第一层')
  const second = parseChainQuickImportHop(payload.second, '第二层')
  const localPort = parseOptionalChainPort(payload.localPort, 'localPort')
  const proxyName = String(payload.name ?? payload.proxyName ?? '').trim()
  const groupName = String(payload.group ?? payload.groupName ?? '').trim()

  return {
    form: {
      proxyName,
      localPort: localPort ? String(localPort) : '',
      first: {
        protocol: first.protocol,
        server: first.server,
        port: String(first.port),
        username: first.username || '',
        password: first.password || '',
      },
      second: {
        protocol: second.protocol,
        server: second.server,
        port: String(second.port),
        username: second.username || '',
        password: second.password || '',
      },
    },
    groupName,
  }
}

export function buildImportCandidatesFromClash(parsedProxies: ClashProxy[], prefix: string): ImportCandidate[] {
  return parsedProxies.map((proxy, index) => ({
    proxyName: resolveImportedProxyName(proxy, index, prefix),
    proxyConfig: proxyToYaml(proxy),
  }))
}

export function buildImportPreview(candidates: ImportCandidate[], groupName: string): ProxyDisplayInfo[] {
  return candidates.map((candidate, index) => {
    const info = parseProxyInfo(candidate.proxyConfig)
    return {
      proxyId: `preview-${index}`,
      proxyName: candidate.proxyName,
      proxyConfig: candidate.proxyConfig,
      groupName: candidate.groupName || groupName,
      sourceId: '',
      sourceUrl: '',
      sourceAutoRefresh: false,
      sourceRefreshIntervalM: 0,
      sourceLastRefreshAt: '',
      type: info.type || '-',
      server: info.server || '-',
      port: info.port || 0,
    }
  })
}

export function parseTimestampMs(value: string): number {
  const trimmed = value.trim()
  if (!trimmed) return 0
  const timestamp = Date.parse(trimmed)
  return Number.isFinite(timestamp) ? timestamp : 0
}

export function normalizeRefreshIntervalM(value: number): number {
  if (!Number.isFinite(value)) return 0
  if (value <= 0) return 0
  if (value < 5) return 5
  if (value > 24 * 60) return 24 * 60
  return Math.round(value)
}

export function sourceHostLabel(sourceURL: string): string {
  const raw = sourceURL.trim()
  if (!raw) return ''
  try {
    const parsed = new URL(raw)
    return parsed.host || raw
  } catch {
    return raw
  }
}

function normalizeSourceURL(sourceURL: string): string {
  const raw = sourceURL.trim()
  if (!raw) return ''
  try {
    const parsed = new URL(raw)
    parsed.hash = ''
    return parsed.toString()
  } catch {
    return raw
  }
}

function buildStableSourceID(sourceURL: string, sourceNamePrefix: string): string {
  const key = `${normalizeSourceURL(sourceURL)}|||${sourceNamePrefix.trim()}`
  let hash = 5381
  for (let index = 0; index < key.length; index += 1) {
    hash = ((hash << 5) + hash) ^ key.charCodeAt(index)
  }
  return `src-${(hash >>> 0).toString(36)}`
}

export function resolveImportSourceID(list: BrowserProxy[], sourceURL: string, sourceNamePrefix: string): string {
  const normalizedURL = normalizeSourceURL(sourceURL)
  const normalizedPrefix = sourceNamePrefix.trim()
  const existing = list.find((item) =>
    normalizeSourceURL(item.sourceUrl || '') === normalizedURL &&
    (item.sourceNamePrefix || '').trim() === normalizedPrefix &&
    (item.sourceId || '').trim() !== '',
  )
  if (existing?.sourceId?.trim()) {
    return existing.sourceId.trim()
  }
  return buildStableSourceID(sourceURL, sourceNamePrefix)
}

export function collectURLImportSources(list: BrowserProxy[]): URLImportSourceMeta[] {
  const sourceMap = new Map<string, URLImportSourceMeta>()
  for (const item of list) {
    const sourceId = (item.sourceId || '').trim()
    const sourceUrl = (item.sourceUrl || '').trim()
    if (!sourceId || !sourceUrl) continue

    const existing = sourceMap.get(sourceId)
    const currentLastRefreshAt = item.sourceLastRefreshAt || ''
    if (!existing) {
      sourceMap.set(sourceId, {
        sourceId,
        sourceUrl,
        sourceNamePrefix: (item.sourceNamePrefix || '').trim(),
        sourceGroupName: (item.groupName || '').trim(),
        sourceDnsServers: (item.dnsServers || '').trim(),
        sourceAutoRefresh: !!item.sourceAutoRefresh,
        sourceRefreshIntervalM: normalizeRefreshIntervalM(Number(item.sourceRefreshIntervalM || 0)),
        sourceLastRefreshAt: currentLastRefreshAt,
      })
      continue
    }

    if (
      parseTimestampMs(currentLastRefreshAt) > parseTimestampMs(existing.sourceLastRefreshAt) &&
      currentLastRefreshAt.trim()
    ) {
      existing.sourceLastRefreshAt = currentLastRefreshAt
    }
  }
  return Array.from(sourceMap.values())
}

export function nextProxyID(): string {
  return `proxy-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`
}

export function resolveImportedProxyName(proxy: ClashProxy, index: number, prefix: string): string {
  const rawName = (proxy.name || '').trim() || `导入代理 ${index + 1}`
  return prefix ? `${prefix}-${rawName}` : rawName
}

export function createExistingProxyIDPicker(oldSourceProxies: BrowserProxy[]) {
  const exactMap = new Map<string, BrowserProxy[]>()
  const nameMap = new Map<string, BrowserProxy[]>()

  oldSourceProxies.forEach((item) => {
    const exactKey = `${item.proxyName}|||${item.proxyConfig}`
    const exactList = exactMap.get(exactKey) || []
    exactList.push(item)
    exactMap.set(exactKey, exactList)

    const nameKey = item.proxyName
    const nameList = nameMap.get(nameKey) || []
    nameList.push(item)
    nameMap.set(nameKey, nameList)
  })

  return (name: string, configText: string): string | null => {
    const exactKey = `${name}|||${configText}`
    const exactList = exactMap.get(exactKey)
    if (exactList && exactList.length > 0) {
      const item = exactList.shift()
      if (item?.proxyId) return item.proxyId
    }

    const nameList = nameMap.get(name)
    if (nameList && nameList.length > 0) {
      const item = nameList.shift()
      if (item?.proxyId) return item.proxyId
    }
    return null
  }
}

export function buildRefreshedSourceProxies(
  parsedProxies: ClashProxy[],
  oldSourceProxies: BrowserProxy[],
  meta: URLImportSourceMeta,
  refreshedAt: string,
): BrowserProxy[] {
  const pickExisting = createExistingProxyIDPicker(oldSourceProxies)
  const prefix = meta.sourceNamePrefix.trim()
  const sourceGroupName = meta.sourceGroupName.trim()
  const sourceDnsServers = meta.sourceDnsServers.trim()

  return parsedProxies.map((proxy, index) => {
    const proxyName = resolveImportedProxyName(proxy, index, prefix)
    const proxyConfig = proxyToYaml(proxy)
    const proxyId = pickExisting(proxyName, proxyConfig) || nextProxyID()

    return {
      proxyId,
      proxyName,
      proxyConfig,
      dnsServers: sourceDnsServers || undefined,
      groupName: sourceGroupName || undefined,
      sourceId: meta.sourceId,
      sourceUrl: meta.sourceUrl,
      sourceNamePrefix: prefix || undefined,
      sourceAutoRefresh: meta.sourceAutoRefresh,
      sourceRefreshIntervalM: meta.sourceRefreshIntervalM,
      sourceLastRefreshAt: refreshedAt,
    }
  })
}
