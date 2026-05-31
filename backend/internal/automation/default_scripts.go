package automation

const DualInstanceRuntimeScriptID = "dual-instance-runtime-switch"

func DefaultScripts() []ScriptRecord {
	return []ScriptRecord{
		{
			ID:          DualInstanceRuntimeScriptID,
			Name:        "双实例启动与 Runtime 切换",
			Description: "通过 Launch API 分别启动两个实例，切换 Runtime 会话后交给 OpenClaw 执行。",
			Type:        "launch-api",
			Status:      "ready",
			EntryFile:   "index.cjs",
			Tags:        []string{"Launch API", "OpenClaw", "双实例"},
			ParamsText: `{
	"browsers": [
	  {
	    "code": "BUYER_001",
	    "skipDefaultStartUrls": true,
	    "startUrls": ["https://finance.sina.com.cn/"]
	  },
	  {
	    "code": "BUYER_002",
	    "skipDefaultStartUrls": true,
	    "startUrls": ["https://map.baidu.com/"]
	  }
	],
	"timeoutMs": 45000
}`,
			ScriptText: `export async function run({ baseUrl, apiKey, params, log }) {
  const normalizeCode = (value, fallback) =>
    String(value || fallback || '').trim().toUpperCase()
  const normalizeStringArray = (value) =>
    Array.isArray(value)
      ? value
          .map((item) => String(item || '').trim())
          .filter(Boolean)
      : []
  const normalizeBrowserInput = (value, fallbackCode, fallbackStartUrls, defaultSkip) => {
    const raw = value && typeof value === 'object' ? value : {}
    const code = normalizeCode(raw.code || raw.launchCode, fallbackCode)
    if (!code) {
      return null
    }
    const startUrls = normalizeStringArray(raw.startUrls)
    const fallbackUrls = normalizeStringArray(fallbackStartUrls)
    const launchArgs = normalizeStringArray(raw.launchArgs)

    return {
      code,
      skipDefaultStartUrls:
        raw.skipDefaultStartUrls !== undefined
          ? raw.skipDefaultStartUrls !== false
          : defaultSkip,
      startUrls: startUrls.length > 0 ? startUrls : fallbackUrls,
      launchArgs,
    }
  }

  const timeoutMs = Number.isFinite(Number(params.timeoutMs))
    ? Math.max(1000, Math.round(Number(params.timeoutMs)))
    : 45000
  const defaultSkipDefaultStartUrls = params.skipDefaultStartUrls !== false

  let browsers = Array.isArray(params.browsers)
    ? params.browsers
        .map((item, index) =>
          normalizeBrowserInput(
            item,
            ['BUYER_001', 'BUYER_002'][index] || '',
            ['https://finance.sina.com.cn/', 'https://map.baidu.com/'][index] || [],
            defaultSkipDefaultStartUrls,
          ),
        )
        .filter(Boolean)
    : []

  if (browsers.length === 0) {
    browsers = [
      normalizeBrowserInput(
        { code: params.primaryCode, skipDefaultStartUrls: params.skipDefaultStartUrls },
        'BUYER_001',
        ['https://finance.sina.com.cn/'],
        defaultSkipDefaultStartUrls,
      ),
      normalizeBrowserInput(
        { code: params.secondaryCode, skipDefaultStartUrls: params.skipDefaultStartUrls },
        'BUYER_002',
        ['https://map.baidu.com/'],
        defaultSkipDefaultStartUrls,
      ),
    ].filter(Boolean)
  }

  if (browsers.length === 0) {
    throw new Error('params.browsers 不能为空')
  }

  const headers = {
    'Content-Type': 'application/json',
    ...(apiKey ? { 'X-Ant-Api-Key': apiKey } : {}),
  }

  const post = async (path, payload) => {
    const response = await fetch(baseUrl + path, {
      method: 'POST',
      headers,
      body: JSON.stringify(payload),
    })
    const text = await response.text()
    let body = text
    try {
      body = text ? JSON.parse(text) : null
    } catch {
      body = text
    }
    if (!response.ok) {
      throw new Error(path + ' failed: ' + response.status + ' ' + text)
    }
    return body
  }

  const sessions = []

  for (const browser of browsers) {
    const sessionResult = await post('/api/runtime/session', {
      selector: { code: browser.code, matchMode: 'unique' },
      skipDefaultStartUrls: browser.skipDefaultStartUrls,
      ...(browser.startUrls.length > 0 ? { startUrls: browser.startUrls } : {}),
      ...(browser.launchArgs.length > 0 ? { launchArgs: browser.launchArgs } : {}),
      timeoutMs,
    })

    sessions.push(sessionResult)
  }

  const browserCodes = browsers.map((item) => item.code)
  log('browserCodes', browserCodes)

  return {
    ok: true,
    summary: browserCodes.length + ' 个浏览器已就绪：' + browserCodes.join(' / '),
    browserCodes,
    sessions,
  }
}`,
			Notes: "先通过接口启动两个实例并切换 Runtime 会话；随后把实例信息交给 OpenClaw 执行自动化动作。",
			Source: ScriptSource{
				Type: "builtin",
				URI:  "repo://backend/internal/automation/default_scripts.go",
				Ref:  "HEAD",
				Path: DualInstanceRuntimeScriptID,
			},
		},
		{
			ID:          "news-query-txt",
			Name:        "查询新闻并写 TXT",
			Description: "通过 Bing 搜索新闻关键词，提取结果并写入本地 txt 文件。",
			Type:        "playwright-cdp",
			Status:      "ready",
			EntryFile:   "index.cjs",
			Tags:        []string{"Playwright", "新闻", "TXT"},
			ParamsText: `{
  "keyword": "OpenAI",
  "limit": 10,
  "timeRange": "week",
  "outputFileName": "openai-news.txt",
  "timeoutMs": 30000,
  "waitAfterLoadMs": 1500,
  "captureScreenshot": false
}`,
			ScriptText: `const fs = require('fs')

const DEFAULT_EXCLUDED_DOMAINS = [
  'zhihu.com',
  'baidu.com',
  'qq.com',
  '36kr.com',
  'apifox.com',
  'chatgpt-chinese.com',
  'openwebui.cn',
  'open-openai.com',
  'xiniushu.com',
  'reddit.com',
  'quora.com',
  'tieba.baidu.com',
  'weibo.com',
  'x.com',
  'twitter.com',
  'youtube.com',
  'bilibili.com',
  'douyin.com',
  'xiaohongshu.com',
]

function normalizeInt(value, fallback, min, max) {
  const parsed = Number(value)
  if (!Number.isFinite(parsed)) {
    return fallback
  }

  const rounded = Math.round(parsed)
  if (rounded < min) {
    return min
  }
  if (rounded > max) {
    return max
  }
  return rounded
}

function normalizeText(value) {
  return String(value || '').trim()
}

function normalizeDomainList(value) {
  if (!Array.isArray(value)) {
    return []
  }

  const deduped = new Set()
  for (const item of value) {
    const normalized = normalizeText(item).replace(/^https?:\/\//, '').replace(/^www\./, '').toLowerCase()
    if (normalized) {
      deduped.add(normalized)
    }
  }
  return Array.from(deduped)
}

function buildDefaultQuery(keyword) {
  const normalizedKeyword = normalizeText(keyword) || 'OpenAI'
  if (/[\u3400-\u9fff]/.test(normalizedKeyword)) {
    return normalizedKeyword + ' 新闻'
  }
  return normalizedKeyword + ' news'
}

function buildFallbackQueries(keyword, baseQuery) {
  const normalizedKeyword = normalizeText(keyword) || 'OpenAI'
  const normalizedBaseQuery = normalizeText(baseQuery)
  const candidates = [
    normalizedBaseQuery,
  ]

  if (/[\u3400-\u9fff]/.test(normalizedKeyword)) {
    candidates.push(normalizedKeyword + ' 最新新闻')
  } else {
    candidates.push(normalizedKeyword + ' latest news')
  }

  const deduped = new Set()
  for (const item of candidates) {
    const normalized = normalizeText(item)
    if (normalized) {
      deduped.add(normalized)
    }
  }
  return Array.from(deduped)
}

function buildSearchQuery(baseQuery, excludedDomains) {
  const normalizedBaseQuery = normalizeText(baseQuery)
  const normalizedDomains = normalizeDomainList(excludedDomains)
  const parts = [normalizedBaseQuery]

  for (const domain of normalizedDomains) {
    parts.push('-site:' + domain)
  }

  return parts.filter(Boolean).join(' ')
}

function mapTimeRangeToBingFilter(value) {
  switch (normalizeText(value).toLowerCase()) {
    case 'day':
    case '24h':
    case 'today':
      return 'ex1:"ez1"'
    case 'week':
      return 'ex1:"ez2"'
    case 'month':
      return 'ex1:"ez3"'
    default:
      return ''
  }
}

function buildSearchURL(query, timeRange, firstResultIndex) {
  const searchParams = new URLSearchParams({ q: query })
  const filter = mapTimeRangeToBingFilter(timeRange)
  if (filter) {
    searchParams.set('filters', filter)
  }
  if (Number.isFinite(firstResultIndex) && firstResultIndex > 1) {
    searchParams.set('first', String(firstResultIndex))
  }
  return 'https://www.bing.com/search?' + searchParams.toString()
}

function splitSnippet(snippet) {
  const normalized = normalizeText(snippet)
  if (!normalized) {
    return { publishedAt: '', summary: '' }
  }

  const match = normalized.match(/^([^·]{0,40})\s*·\s*(.+)$/)
  if (
    match &&
    /(前|分钟|小时|天前|周前|月前|昨天|\d{4}|\d{1,2}[/-]\d{1,2})/.test(match[1])
  ) {
    return {
      publishedAt: normalizeText(match[1]),
      summary: normalizeText(match[2]),
    }
  }

  return {
    publishedAt: '',
    summary: normalized,
  }
}

function parseHostname(rawUrl) {
  const normalized = normalizeText(rawUrl)
  if (!normalized) {
    return ''
  }

  try {
    return new URL(normalized).hostname.replace(/^www\./, '').toLowerCase()
  } catch {
    return ''
  }
}

function parsePathname(rawUrl) {
  const normalized = normalizeText(rawUrl)
  if (!normalized) {
    return ''
  }

  try {
    const pathname = new URL(normalized).pathname.replace(/\/+/g, '/').toLowerCase()
    if (!pathname) {
      return ''
    }
    return pathname === '/' ? pathname : pathname.replace(/\/$/, '')
  } catch {
    return ''
  }
}

function looksLikeQuestionTitle(title) {
  const normalized = normalizeText(title)
  if (!normalized) {
    return false
  }

  if (/[？?]/.test(normalized)) {
    return true
  }

  return /^(如何|为什么|怎么看|怎样|怎么|是否|有没有|谁能|请问|评价|如何评价|如何看待|为什么说)/.test(normalized)
}

function looksLikeAggregateText(text) {
  const normalized = normalizeText(text).toLowerCase()
  if (!normalized) {
    return false
  }

  return /(roundup|digest|flash report|llm news today|ai news today|daily ai news|news today|model releases)/.test(normalized)
}

function looksLikeListingPath(pathname) {
  const normalized = normalizeText(pathname).toLowerCase()
  if (!normalized || normalized === '/') {
    return false
  }

  if (/(^|\/)(tag|tags|topic|topics|category|categories|label|labels|brand|brands)(\/|$)/.test(normalized)) {
    return true
  }

  if (/(^|\/)(news|latest|headlines|insights)$/.test(normalized)) {
    return true
  }

  return /\/news\/(brand|brands|topic|topics|tag|tags)(\/|$)/.test(normalized)
}

function looksLikeListingText(text) {
  const normalized = normalizeText(text).toLowerCase()
  if (!normalized) {
    return false
  }

  return /(latest news|breaking headlines|news and insights|news and analysis|everything you need to know|get the latest|最新资讯|最新动态|实时追踪|热点快讯|快讯)/.test(normalized)
}

function isBlockedHostname(hostname) {
  const normalized = normalizeText(hostname).toLowerCase()
  if (!normalized) {
    return false
  }

  const blockedSuffixes = DEFAULT_EXCLUDED_DOMAINS
  const blockedKeywords = [
    'aitrack',
    'aitoolly',
    'aiflashreport',
    'llm-stats',
    'opentools',
  ]

  if (blockedSuffixes.some(function (suffix) {
    return normalized === suffix || normalized.endsWith('.' + suffix)
  })) {
    return true
  }

  return blockedKeywords.some(function (keyword) {
    return normalized.includes(keyword)
  })
}

function evaluateNewsItem(item) {
  const hostname = parseHostname(item.url)
  const pathname = parsePathname(item.url)
  const summary = normalizeText(item.summary)
  const source = normalizeText(item.source)
  const reasons = []

  if (!normalizeText(item.url)) {
    reasons.push('missing-url')
  }
  if (!hostname) {
    reasons.push('invalid-url')
  }
  if (hostname && isBlockedHostname(hostname)) {
    reasons.push('blocked-host')
  }
  if (!source) {
    reasons.push('missing-source')
  }
  if (summary.length < 20) {
    reasons.push('summary-too-short')
  }
  if (looksLikeQuestionTitle(item.title)) {
    reasons.push('question-title')
  }
  if (looksLikeAggregateText(item.title) || looksLikeAggregateText(summary)) {
    reasons.push('aggregate-page')
  }
  if (looksLikeListingPath(pathname) || looksLikeListingText(item.title) || looksLikeListingText(summary)) {
    reasons.push('listing-page')
  }

  return Object.assign({}, item, {
    hostname: hostname,
    pathname: pathname,
    qualityAccepted: reasons.length === 0,
    qualityReasons: reasons,
  })
}

function formatRejectedReason(reason) {
  switch (reason) {
    case 'missing-url':
      return '缺少链接'
    case 'invalid-url':
      return '链接无效'
    case 'blocked-host':
      return '来源站点已过滤'
    case 'missing-source':
      return '缺少来源'
    case 'summary-too-short':
      return '摘要过短'
    case 'question-title':
      return '标题更像问答'
    case 'aggregate-page':
      return '更像聚合页'
    case 'listing-page':
      return '更像列表页/专题页'
    default:
      return reason
  }
}

function formatReport(items, metadata) {
  const lines = [
    '新闻抓取结果',
    '查询词: ' + metadata.query,
    '抓取时间: ' + metadata.generatedAt,
    '搜索地址: ' + metadata.searchUrl,
    '原始结果: ' + metadata.rawCount,
    '通过校验: ' + items.length,
    '过滤数量: ' + metadata.rejectedItems.length,
    '',
  ]

  for (const item of items) {
    lines.push(item.rank + '. ' + item.title)
    if (item.source) {
      lines.push('来源: ' + item.source)
    }
    if (item.publishedAt) {
      lines.push('时间: ' + item.publishedAt)
    }
    lines.push('链接: ' + item.url)
    if (item.summary) {
      lines.push('摘要: ' + item.summary)
    }
    lines.push('')
  }

  if (metadata.rejectedItems.length > 0) {
    lines.push('被过滤结果（最多展示 5 条）')
    lines.push('')
    for (const item of metadata.rejectedItems.slice(0, 5)) {
      lines.push(item.rank + '. ' + item.title)
      if (item.hostname) {
        lines.push('站点: ' + item.hostname)
      }
      lines.push('原因: ' + item.qualityReasons.map(formatRejectedReason).join(' / '))
      lines.push('')
    }
  }

  return lines.join('\n')
}

function pickBestAttempt(current, candidate) {
  if (!current) {
    return candidate
  }

  if (candidate.acceptedItems.length !== current.acceptedItems.length) {
    return candidate.acceptedItems.length > current.acceptedItems.length ? candidate : current
  }

  if (candidate.distinctHostCount !== current.distinctHostCount) {
    return candidate.distinctHostCount > current.distinctHostCount ? candidate : current
  }

  if (candidate.rawItems.length !== current.rawItems.length) {
    return candidate.rawItems.length > current.rawItems.length ? candidate : current
  }

  return candidate
}

module.exports.run = async ({ launch, connect, selector, params, log, artifact }) => {
  const timeout = normalizeInt(params.timeoutMs, 30000, 1000, 120000)
  const waitAfterLoadMs = normalizeInt(params.waitAfterLoadMs, 1500, 0, 10000)
  const limit = normalizeInt(params.limit, 10, 1, 50)
  const maxPages = normalizeInt(params.maxPages, 3, 1, 5)
  const baseQuery = normalizeText(params.query) || buildDefaultQuery(params.keyword)
  const excludedDomains = normalizeDomainList(params.excludeDomains).length > 0
    ? normalizeDomainList(params.excludeDomains)
    : DEFAULT_EXCLUDED_DOMAINS
  const outputFileName = normalizeText(params.outputFileName) || 'news-results.txt'
  const scanLimit = Math.max(10, Math.min(20, limit * 2))
  const startUrls = Array.isArray(params.startUrls) && params.startUrls.length > 0
    ? params.startUrls
    : undefined

  const session = await launch({
    selector,
    startUrls,
    skipDefaultStartUrls: true,
  })

  const connection = await connect(session)
  const browser = connection.browser
  const context = connection.context || browser.contexts()[0]
  const page = await context.newPage()
  const closeRunnerPage = async function () {
    if (!page.isClosed()) {
      await page.close().catch(function () {})
    }
  }

  const searchCandidates = buildFallbackQueries(params.keyword, baseQuery)
  const minAcceptedCount = Math.min(limit, Math.max(2, Math.ceil(limit * 0.2)))
  const minDistinctHostCount = Math.min(3, minAcceptedCount)
  let bestAttempt = null

  try {
    for (const candidateQuery of searchCandidates) {
      const searchQuery = buildSearchQuery(candidateQuery, excludedDomains)
      const normalizedItems = []
      const seenUrls = new Set()
      let scannedPageCount = 0
      let firstSearchUrl = ''

      for (let pageIndex = 0; pageIndex < maxPages; pageIndex += 1) {
        const firstResultIndex = pageIndex * 10 + 1
        const searchUrl = buildSearchURL(searchQuery, params.timeRange, firstResultIndex)

        try {
          await page.goto(searchUrl, {
            waitUntil: 'domcontentloaded',
            timeout,
          })
          await page.waitForSelector('li.b_algo', { timeout })
        } catch (error) {
          if (pageIndex > 0 && normalizedItems.length > 0) {
            break
          }
          throw error
        }

        if (waitAfterLoadMs > 0) {
          await page.waitForTimeout(waitAfterLoadMs)
        }

        if (!firstSearchUrl) {
          firstSearchUrl = page.url()
        }

        const pageItems = await page.$$eval('li.b_algo', function (nodes, maxItems) {
          const clean = function (value) {
            return String(value || '').replace(/\s+/g, ' ').trim()
          }

          return nodes
            .slice(0, maxItems)
            .map(function (node) {
              const titleLink = node.querySelector('h2 a')
              const title = clean(titleLink && titleLink.textContent)
              const url = titleLink ? titleLink.href : ''
              const sourceNode = node.querySelector('.tptt')
              const source = clean(sourceNode && sourceNode.textContent)
              const citeNode = node.querySelector('.b_attribution cite')
              const cite = clean(citeNode && citeNode.textContent)
              const snippetNode = node.querySelector('.b_caption p')
              const snippet = clean(snippetNode && snippetNode.textContent)

              if (!title) {
                return null
              }

              return {
                title,
                url,
                source: source || cite,
                snippet,
              }
            })
            .filter(Boolean)
        }, scanLimit)

        let appendedCount = 0
        for (const item of pageItems) {
          const dedupeKey = normalizeText(item.url)
          if (!dedupeKey || seenUrls.has(dedupeKey)) {
            continue
          }

          seenUrls.add(dedupeKey)
          normalizedItems.push(
            evaluateNewsItem(
              Object.assign(
                {
                  rank: normalizedItems.length + 1,
                },
                item,
                splitSnippet(item.snippet)
              )
            )
          )
          appendedCount += 1
        }

        scannedPageCount += 1
        if (appendedCount === 0 || pageItems.length < 8) {
          break
        }
      }

      const acceptedItems = normalizedItems.filter(function (item) {
        return item.qualityAccepted
      }).slice(0, limit)
      const rejectedItems = normalizedItems.filter(function (item) {
        return !item.qualityAccepted
      })
      const distinctHostCount = new Set(
        acceptedItems
          .map(function (item) {
            return item.hostname
          })
          .filter(Boolean)
      ).size

      log('searchQuery', searchQuery)
      log('rawItemCount', normalizedItems.length)
      log('acceptedItemCount', acceptedItems.length)
      log('rejectedItemCount', rejectedItems.length)
      log('distinctHostCount', distinctHostCount)
      log('scannedPageCount', scannedPageCount)

      bestAttempt = pickBestAttempt(bestAttempt, {
        baseQuery: candidateQuery,
        searchQuery: searchQuery,
        searchUrl: firstSearchUrl || page.url(),
        rawItems: normalizedItems,
        acceptedItems: acceptedItems,
        rejectedItems: rejectedItems,
        distinctHostCount: distinctHostCount,
        scannedPageCount: scannedPageCount,
      })

      if (acceptedItems.length >= minAcceptedCount && distinctHostCount >= minDistinctHostCount) {
        break
      }
    }
  } catch (error) {
    await closeRunnerPage()
    throw error
  }

  if (!bestAttempt || bestAttempt.rawItems.length === 0) {
    await closeRunnerPage()
    throw new Error('未抓到新闻搜索结果，当前页面: ' + page.url())
  }

  const normalizedItems = bestAttempt.rawItems
  const acceptedItems = bestAttempt.acceptedItems
  const rejectedItems = bestAttempt.rejectedItems
  const distinctHostCount = bestAttempt.distinctHostCount
  const searchUrl = bestAttempt.searchUrl
  const scannedPageCount = bestAttempt.scannedPageCount || 1

  const outputName = outputFileName.toLowerCase().endsWith('.txt')
    ? outputFileName
    : outputFileName + '.txt'
  const outputPath = artifact(outputName)
  const reportText = formatReport(acceptedItems, {
    query: bestAttempt.baseQuery,
    generatedAt: new Date().toISOString(),
    searchUrl: searchUrl,
    rawCount: normalizedItems.length,
    rejectedItems: rejectedItems,
  })
  fs.writeFileSync(outputPath, reportText, 'utf8')

  let screenshotPath = ''
  if (params.captureScreenshot === true) {
    screenshotPath = artifact('news-search.png')
    await page.screenshot({
      path: screenshotPath,
      fullPage: true,
    })
  }

  log('outputPath', outputPath)
  await closeRunnerPage()

  if (acceptedItems.length < minAcceptedCount || distinctHostCount < minDistinctHostCount) {
    return {
      ok: false,
      summary: '新闻结果质量不足，仅 ' + acceptedItems.length + '/' + normalizedItems.length + ' 条通过校验',
      error: '搜索结果更像普通搜索、问答页或聚合页，未达到新闻抓取标准',
      query: bestAttempt.baseQuery,
      searchQuery: bestAttempt.searchQuery,
      searchUrl: searchUrl,
      outputPath,
      screenshotPath,
      rawItemCount: normalizedItems.length,
      itemCount: acceptedItems.length,
      rejectedCount: rejectedItems.length,
      distinctHostCount: distinctHostCount,
      scannedPageCount: scannedPageCount,
      firstTitle: acceptedItems[0] ? acceptedItems[0].title : '',
    }
  }

  return {
    ok: true,
    summary: '已筛出 ' + acceptedItems.length + ' 条有效新闻并写入 TXT',
    query: bestAttempt.baseQuery,
    searchQuery: bestAttempt.searchQuery,
    searchUrl: searchUrl,
    outputPath,
    screenshotPath,
    rawItemCount: normalizedItems.length,
    itemCount: acceptedItems.length,
    rejectedCount: rejectedItems.length,
    distinctHostCount: distinctHostCount,
    scannedPageCount: scannedPageCount,
    firstTitle: acceptedItems[0] ? acceptedItems[0].title : '',
  }
}`,
			Notes: "脚本会优先使用 Bing 搜索真实新闻结果，并自动追加时间过滤、排除问答/聚合站点、回退查询词和质量校验；只有达到新闻质量门槛时才会判定成功，并把结果写入本地 txt。执行时可直接点“创建 Demo 并执行”，成功后在结果里的 outputPath 查看文件。",
			Source: ScriptSource{
				Type: "builtin",
				URI:  "repo://backend/internal/automation/default_scripts.go",
				Ref:  "HEAD",
				Path: "news-query-txt",
			},
		},
	}
}
