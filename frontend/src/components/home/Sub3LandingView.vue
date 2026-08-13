<template>
  <div ref="host" class="sub3-landing-host" />
</template>

<script setup lang="ts">
import { inject, onBeforeUnmount, onMounted, ref, watch, type WatchStopHandle } from 'vue'
import { routerKey } from 'vue-router'
import rawHtml from '@/assets/sub3api-home.html?raw'
import { getLocale, i18n, setLocale } from '@/i18n'
import { buildGatewayUrl } from '@/api/client'
import { query as queryUserUsage, getDashboardStats as getUserDashboardStats } from '@/api/usage'
import {
  getDashboardOverview,
  listRequestDetails,
  type OpsDashboardOverview,
  type OpsRequestDetail,
  subscribeQPS,
} from '@/api/admin/ops'
import { list as listChannelMonitorViews, type UserMonitorView } from '@/api/channelMonitor'
import { getModelStats as getAdminModelStats } from '@/api/admin/dashboard'
import { useAppStore } from '@/stores'
import type { PublicSettings } from '@/types'

const host = ref<HTMLElement | null>(null)
const router = inject(routerKey, null)
const appStore = useAppStore()
let styleElement: HTMLStyleElement | null = null
let activityTimer: number | null = null
let animationFrame: number | null = null
let stopLocaleWatch: WatchStopHandle | null = null
let landingDataController: AbortController | null = null
let landingOnline: boolean | null = null
let channelSummaryTimer: number | null = null
let qpsUnsubscribe: (() => void) | null = null
let channelOnlineCount = 0
let channelTotalCount = 0
let channelMonitorKnown = false
let liveQps: number | null = null
const navigationHandlers: Array<{ element: HTMLAnchorElement; handler: (event: Event) => void }> = []
const cleanupCallbacks: Array<() => void> = []

type LandingLocale = 'zh' | 'en'

const landingMessages = {
  zh: {
    nav: ['服务状态', 'QQ群'],
    eyebrow: 'Sub3API · Private Gateway',
    description: '一个 <b>API Key</b>，统一调度所有已接入的 AI 模型。智能路由、自动容灾、按量计费，为个人与小型团队提供 <b>稳定 · 透明 · 安全</b> 的私有网关服务。',
    docs: '仓库下载',
    qqGroup: 'QQ群',
    stats: ['服务可用性', '已接入 AI 模型', '网关转发延迟'],
    checkingStatus: '正在检测网关',
    gatewayStatus: 'Gateway Online',
    offlineStatus: 'Gateway Offline',
    realtimeStatus: 'REAL-TIME · 实时状态',
    online: '● 在线',
    offline: '● 离线',
    requestLog: '请求日志',
    live: '实时',
    channelsOnline: '渠道在线',
    channelMonitorDisabled: '渠道监控未启用',
    noPublicStatusChecks: '暂无公开状态检查',
    publicStatusUnavailable: '公开状态暂不可用',
    metrics: ['监控', '并发', '保护'],
    featureTitles: ['<em>一键</em>接入', '<em>稳定</em>可靠', '密钥与用量<em>透明</em>'],
    featureDescriptions: [
      '获取一个 API 密钥，即可调用所有已接入的 AI 模型，无需分别申请。统一的接口风格，让接入成本趋近于零。',
      '智能调度多个上游账号，自动切换与负载均衡，故障自动容灾，告别频繁报错与排队等待。',
      '按 Key 分配配额与倍率，个人、家庭或小团队都能清楚看到每一次调用成本，账单一目了然。',
    ],
    featureActions: ['查看服务状态'],
    footer: ['服务状态', 'QQ群'],
  },
  en: {
    nav: ['Service Status', 'QQ Group'],
    eyebrow: 'Sub3API · Private Gateway',
    description: 'One <b>API key</b> connects every available AI model through unified routing, automatic failover, and usage-based billing. A <b>stable · transparent · secure</b> private gateway for individuals and small teams.',
    docs: 'Repository Download',
    qqGroup: 'QQ Group',
    stats: ['Service Availability', 'Connected AI Models', 'Gateway Latency'],
    checkingStatus: 'CHECKING GATEWAY',
    gatewayStatus: 'Gateway Online',
    offlineStatus: 'Gateway Offline',
    realtimeStatus: 'REAL-TIME · LIVE STATUS',
    online: '● ONLINE',
    offline: '● OFFLINE',
    requestLog: 'Request Log',
    live: 'LIVE',
    channelsOnline: 'channels online',
    channelMonitorDisabled: 'Channel monitoring disabled',
    noPublicStatusChecks: 'No public status checks',
    publicStatusUnavailable: 'Public status unavailable',
    metrics: ['Monitoring', 'Concurrency', 'Protected'],
    featureTitles: ['<em>One-key</em> access', '<em>Stable</em> routing', 'Transparent <em>usage</em>'],
    featureDescriptions: [
      'Use one API key for every connected AI model. A unified interface keeps integration work close to zero.',
      'Route traffic across multiple upstream accounts with automatic switching, load balancing, and failover.',
      'Assign quotas and rates per key, then track every request and cost clearly for individuals or teams.',
    ],
    featureActions: ['View Service Status'],
    footer: ['Service Status', 'QQ Group'],
  },
} as const

function normalizeLocale(locale: string): LandingLocale {
  return locale.startsWith('zh') ? 'zh' : 'en'
}

function setTextList(selector: string, values: readonly string[]) {
  host.value?.querySelectorAll<HTMLElement>(selector).forEach((element, index) => {
    if (values[index]) element.textContent = values[index]
  })
}

function setLabelPreservingChildren(element: HTMLElement | null, label: string) {
  if (!element) return
  const textNode = Array.from(element.childNodes).find((node) => node.nodeType === Node.TEXT_NODE)
  if (textNode) textNode.textContent = ` ${label} `
}

function applyLandingLocale(locale: LandingLocale) {
  if (!host.value) return
  const messages = landingMessages[locale]

  setTextList('.nav-links a', messages.nav)
  setLabelPreservingChildren(host.value.querySelector<HTMLElement>('.eyebrow'), messages.eyebrow)
  const description = host.value.querySelector<HTMLElement>('.hero-sub')
  if (description) description.innerHTML = messages.description
  const docs = host.value.querySelector<HTMLElement>('.hero-cta .btn-ghost')
  if (docs) docs.textContent = messages.docs
  setTextList('.hero-stats .lbl', messages.stats)
  setLabelPreservingChildren(
    host.value.querySelector<HTMLElement>('.status-text'),
    landingOnline === null ? messages.checkingStatus : landingOnline ? messages.gatewayStatus : messages.offlineStatus,
  )
  const realtimeStatus = host.value.querySelector<HTMLElement>('.status-text small')
  if (realtimeStatus) realtimeStatus.textContent = messages.realtimeStatus
  const online = host.value.querySelector<HTMLElement>('.realtime-tag')
  if (online) online.textContent = messages.online
  const requestLog = host.value.querySelector<HTMLElement>('.activity-head > span:first-child')
  if (requestLog) requestLog.textContent = messages.requestLog
  const live = host.value.querySelector<HTMLElement>('.activity-head .live')
  if (live) live.textContent = messages.live
  setTextList('.metric .m-lbl', messages.metrics)
  host.value.querySelectorAll<HTMLElement>('.f-card h3').forEach((element, index) => {
    if (messages.featureTitles[index]) element.innerHTML = messages.featureTitles[index]
  })
  setTextList('.f-card p', messages.featureDescriptions)
  host.value.querySelectorAll<HTMLElement>('.f-more').forEach((element, index) => {
    if (messages.featureActions[index]) setLabelPreservingChildren(element, messages.featureActions[index])
  })
  setTextList('.ft-right a', messages.footer)
  host.value.querySelectorAll<HTMLButtonElement>('.lang button').forEach((button) => {
    button.classList.toggle('active', button.dataset.locale === locale)
    button.setAttribute('aria-pressed', String(button.dataset.locale === locale))
  })
  if (landingOnline !== null) applyHealthStatus(landingOnline)
  if (channelMonitorKnown) applyChannelSummary(channelOnlineCount, channelTotalCount, channelMonitorKnown)
}

function applyQQGroupSettings(settings?: PublicSettings | null) {
  const url = settings?.qq_group_url?.trim() || ''
  const number = settings?.qq_group_number?.trim() || ''
  const navLabel = `${landingMessages[normalizeLocale(getLocale())].qqGroup}：${number}`
  host.value?.querySelectorAll<HTMLAnchorElement>('.qq-group-link').forEach((element) => {
    if (!url || !number) {
      element.style.display = 'none'
      return
    }
    element.style.display = ''
    element.href = url
    element.setAttribute('target', '_blank')
    element.setAttribute('rel', 'noopener noreferrer')
    if (element.closest('.nav-links')) element.textContent = navLabel
  })
}

function applyVersionBadge(settings?: PublicSettings | null) {
  const version = settings?.version?.trim()
  if (!version || !host.value) return
  const label = version.startsWith('v') ? version : `v${version}`
  host.value.querySelectorAll<HTMLElement>('.monitor-title .badge').forEach((element) => {
    element.textContent = label
  })
}

type LandingActivity = {
  statusCode: number
  label: string
  createdAt: string
  kind: 'success' | 'error'
}

function getPersistedViewer() {
  const token = localStorage.getItem('auth_token')
  if (!token) return { authenticated: false, admin: false }

  try {
    const user = JSON.parse(localStorage.getItem('auth_user') || 'null') as { role?: string } | null
    return { authenticated: true, admin: user?.role === 'admin' }
  } catch {
    return { authenticated: true, admin: false }
  }
}

function formatActivityTime(value: string) {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '--:--:--'
  return date.toLocaleTimeString([], { hour12: false })
}

function renderActivityRows(items: LandingActivity[], emptyLabel = 'No recent requests') {
  const activityList = host.value?.querySelector<HTMLElement>('#actList')
  if (!activityList) return

  activityList.replaceChildren()
  if (!items.length) {
    const empty = document.createElement('div')
    empty.className = 'act-item'
    empty.textContent = emptyLabel
    empty.style.color = 'var(--text-3)'
    activityList.appendChild(empty)
    return
  }

  items.slice(0, 4).forEach((item, index) => {
    const element = document.createElement('div')
    element.className = 'act-item'
    element.style.animationDelay = `${index * 0.1}s`

    const status = document.createElement('span')
    status.textContent = String(item.statusCode)
    status.className = item.kind === 'error' || item.statusCode >= 400 ? 'error' : 'ok'
    status.style.color = item.kind === 'error' || item.statusCode >= 400 ? '#ff6b7d' : ''

    const label = document.createElement('span')
    label.textContent = item.label

    const timestamp = document.createElement('span')
    timestamp.className = 'ts'
    timestamp.textContent = formatActivityTime(item.createdAt)

    element.append(status, label, timestamp)
    activityList.appendChild(element)
  })
}

function setLandingValue(selector: string, value: string) {
  const element = host.value?.querySelector<HTMLElement>(selector)
  if (element) element.textContent = value
}

function applyHealthStatus(online: boolean) {
  landingOnline = online
  const locale = normalizeLocale(getLocale())
  const statusText = host.value?.querySelector<HTMLElement>('.status-text')
  if (statusText) {
    statusText.dataset.connected = String(online)
    setLabelPreservingChildren(
      statusText,
      online ? landingMessages[locale].gatewayStatus : landingMessages[locale].offlineStatus,
    )
  }
  const statusTag = host.value?.querySelector<HTMLElement>('.realtime-tag')
  if (statusTag) statusTag.dataset.connected = String(online)
  updateRealtimeTag()
}

function applyOverviewMetrics(overview: OpsDashboardOverview | null, modelCount: number | null) {
  if (!overview) return

  const availability = Number.isFinite(overview.sla) ? `${(overview.sla * 100).toFixed(1)}%` : '--'
  const models = modelCount === null ? '--' : `${modelCount}+`
  const latency = overview.duration.avg_ms ?? overview.duration.p50_ms
  const latencyLabel = latency === null || latency === undefined ? '--' : `${Math.round(latency)}ms`
  setLandingValue('.hero-stats .stat:nth-child(1) .num', availability)
  setLandingValue('.hero-stats .stat:nth-child(2) .num', models)
  setLandingValue('.hero-stats .stat:nth-child(3) .num', latencyLabel)

  const redis = overview.system_metrics?.redis_ok
  setLandingValue('.metric:nth-child(2) .m-val', redis === undefined || redis === null ? '--' : redis ? 'Ready' : 'Down')
  setLandingValue('.metric:nth-child(3) .m-val', window.location.protocol === 'https:' ? 'TLS' : 'HTTP')
}

function updateRealtimeTag() {
  const tag = host.value?.querySelector<HTMLElement>('.realtime-tag')
  if (!tag) return
  const locale = normalizeLocale(getLocale())
  const base = landingOnline ? landingMessages[locale].online : landingMessages[locale].offline
  tag.textContent = landingOnline && liveQps !== null ? base + ' · ' + liveQps.toFixed(1) + ' QPS' : base
}

function applyChannelSummary(online: number, total: number, known: boolean) {
  const line = host.value?.querySelector<HTMLElement>('.channel-line')
  const text = host.value?.querySelector<HTMLElement>('#chText')
  if (!line || !text) return
  channelOnlineCount = online
  channelTotalCount = total
  channelMonitorKnown = known
  const locale = normalizeLocale(getLocale())
  if (!known || total === 0) {
    line.classList.remove('online', 'degraded', 'offline')
    line.classList.add('offline')
    text.textContent = landingMessages[locale].channelMonitorDisabled
    return
  }
  const any = online > 0
  const ok = any && online === total
  line.classList.toggle('online', ok)
  line.classList.toggle('degraded', any && !ok)
  line.classList.toggle('offline', !any)
  text.textContent = online + '/' + total + ' ' + landingMessages[locale].channelsOnline
}

async function refreshChannelSummary() {
  if (!host.value || /jsdom/i.test(navigator.userAgent)) return
  try {
    const res = await listChannelMonitorViews()
    const items: UserMonitorView[] = res.items || []
    const online = items.filter((it) => it.primary_status === 'operational').length
    applyChannelSummary(online, items.length, true)
  } catch {
    applyChannelSummary(0, 0, false)
  }
}

function setupLiveQps() {
  const viewer = getPersistedViewer()
  if (!viewer.authenticated || !viewer.admin) return
  qpsUnsubscribe = subscribeQPS(
    (data: { data?: { qps?: unknown } }) => {
      const qps = Number(data?.data?.qps)
      if (Number.isFinite(qps)) {
        liveQps = qps
        updateRealtimeTag()
      }
    },
    {
      staleTimeoutMs: 60_000,
      onFatalClose: () => {
        liveQps = null
        updateRealtimeTag()
      },
    },
  )
}

function mapOpsActivity(item: OpsRequestDetail): LandingActivity {
  const platform = item.platform?.trim()
  const model = item.model?.trim()
  const label = [platform, model].filter(Boolean).join(' / ') || 'API request'
  return {
    statusCode: item.status_code || (item.kind === 'error' ? 500 : 200),
    label,
    createdAt: item.created_at,
    kind: item.kind,
  }
}

function mapUserActivity(item: Awaited<ReturnType<typeof queryUserUsage>>['items'][number]): LandingActivity {
  const endpoint = item.inbound_endpoint?.trim() || item.model?.trim() || 'API request'
  const parts = endpoint.split(/\s+/)
  const label = parts.length > 1 ? `${parts[0]} ${parts.slice(1).join(' ')}` : endpoint
  return { statusCode: 200, label, createdAt: item.created_at, kind: 'success' }
}

function statusCodeForMonitorStatus(status: string) {
  switch (status) {
    case 'operational':
      return 200
    case 'degraded':
      return 206
    case 'failed':
    case 'error':
      return 503
    default:
      return 200
  }
}

function isMonitorStatusOnline(status: string) {
  return status === 'operational'
}

function isMonitorStatusError(status: string) {
  return status === 'failed' || status === 'error'
}

function applyPublicMonitorMetrics(items: UserMonitorView[], healthOnline: boolean) {
  const locale = normalizeLocale(getLocale())
  const modelNames = new Set<string>()
  const availabilityValues: number[] = []
  const latencyValues: number[] = []
  const activities: LandingActivity[] = []

  for (const item of items) {
    if (item.primary_model?.trim()) modelNames.add(item.primary_model.trim())
    for (const extra of item.extra_models || []) {
      if (extra.model?.trim()) modelNames.add(extra.model.trim())
      if (Number.isFinite(extra.latency_ms)) latencyValues.push(Number(extra.latency_ms))
    }
    if (Number.isFinite(item.availability_7d)) availabilityValues.push(Number(item.availability_7d))
    if (Number.isFinite(item.primary_latency_ms)) latencyValues.push(Number(item.primary_latency_ms))

    for (const point of item.timeline || []) {
      const createdAt = point.checked_at
      if (!createdAt || Number.isNaN(new Date(createdAt).getTime())) continue
      const provider = item.provider?.trim()
      const model = item.primary_model?.trim()
      const label = [provider, model].filter(Boolean).join(' / ') || 'Channel status check'
      activities.push({
        statusCode: statusCodeForMonitorStatus(point.status),
        label,
        createdAt,
        kind: isMonitorStatusError(point.status) ? 'error' : 'success',
      })
    }
  }

  const avgAvailability = availabilityValues.length
    ? availabilityValues.reduce((sum, value) => sum + value, 0) / availabilityValues.length
    : null
  const avgLatency = latencyValues.length
    ? latencyValues.reduce((sum, value) => sum + value, 0) / latencyValues.length
    : null
  const online = items.filter((item) => isMonitorStatusOnline(item.primary_status)).length

  setLandingValue(
    '.hero-stats .stat:nth-child(1) .num',
    avgAvailability === null ? (healthOnline ? 'Online' : 'Down') : `${avgAvailability.toFixed(1)}%`,
  )
  setLandingValue('.hero-stats .stat:nth-child(2) .num', modelNames.size ? `${modelNames.size}+` : '--')
  setLandingValue('.hero-stats .stat:nth-child(3) .num', avgLatency === null ? '--' : `${Math.round(avgLatency)}ms`)
  setLandingValue('.metric:nth-child(2) .m-val', healthOnline ? 'Ready' : 'Down')
  setLandingValue('.metric:nth-child(3) .m-val', window.location.protocol === 'https:' ? 'TLS' : 'HTTP')

  activities.sort((a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime())
  renderActivityRows(activities, landingMessages[locale].noPublicStatusChecks)
  applyChannelSummary(online, items.length, true)
}

async function refreshPublicLandingData(healthOnline: boolean, signal: AbortSignal) {
  const locale = normalizeLocale(getLocale())
  try {
    const res = await listChannelMonitorViews({ signal })
    if (signal.aborted) return
    applyPublicMonitorMetrics(res.items || [], healthOnline)
  } catch {
    if (signal.aborted) return
    renderActivityRows([], landingMessages[locale].publicStatusUnavailable)
    setLandingValue('.hero-stats .stat:nth-child(1) .num', healthOnline ? 'Online' : 'Down')
    setLandingValue('.hero-stats .stat:nth-child(2) .num', '--')
    setLandingValue('.hero-stats .stat:nth-child(3) .num', '--')
    setLandingValue('.metric:nth-child(2) .m-val', healthOnline ? 'Ready' : 'Down')
    setLandingValue('.metric:nth-child(3) .m-val', window.location.protocol === 'https:' ? 'TLS' : 'HTTP')
  }
  applyHealthStatus(healthOnline)
}

async function refreshLandingData() {
  if (!host.value || /jsdom/i.test(navigator.userAgent)) return
  if (document.hidden) return
  landingDataController?.abort()
  const controller = new AbortController()
  landingDataController = controller

  const healthOnline = await fetch(buildGatewayUrl('/health'), { signal: controller.signal })
    .then((response) => response.ok)
    .catch(() => false)
  if (controller.signal.aborted || landingDataController !== controller) return

  const viewer = getPersistedViewer()
  if (!viewer.authenticated) {
    await refreshPublicLandingData(healthOnline, controller.signal)
    return
  }

  try {
    if (viewer.admin) {
      const [requests, overview, models] = await Promise.all([
        listRequestDetails({ time_range: '1h', kind: 'all', page: 1, page_size: 4, sort: 'created_at_desc' }),
        getDashboardOverview({ time_range: '1h' }),
        getAdminModelStats({ start_date: new Date().toISOString().slice(0, 10), end_date: new Date().toISOString().slice(0, 10) }),
      ])
      renderActivityRows(requests.items.map(mapOpsActivity))
      applyOverviewMetrics(overview, models.models.length)
      applyHealthStatus(healthOnline)
    } else {
      const [requests, stats] = await Promise.all([
        queryUserUsage({ page: 1, page_size: 4, sort_by: 'created_at', sort_order: 'desc' }),
        getUserDashboardStats(),
      ])
      renderActivityRows(requests.items.map(mapUserActivity))
      const latency = Number.isFinite(stats.average_duration_ms) ? `${Math.round(stats.average_duration_ms)}ms` : '--'
      setLandingValue('.hero-stats .stat:nth-child(1) .num', '--')
      setLandingValue('.hero-stats .stat:nth-child(2) .num', '--')
      setLandingValue('.hero-stats .stat:nth-child(3) .num', latency)
      applyHealthStatus(healthOnline)
    }
  } catch (error) {
    if (!controller.signal.aborted) {
      console.error('Failed to load landing metrics:', error)
      renderActivityRows([], 'Unable to load recent requests')
      applyHealthStatus(healthOnline)
    }
  }
}

function setupLandingInteractions() {
  if (!host.value) return

  const nav = host.value.querySelector<HTMLElement>('#nav')
  const onScroll = () => nav?.classList.toggle('scrolled', window.scrollY > 24)
  window.addEventListener('scroll', onScroll, { passive: true })
  cleanupCallbacks.push(() => window.removeEventListener('scroll', onScroll))

  const monitorCard = host.value.querySelector<HTMLElement>('.monitor')
  if (monitorCard) {
    const openMonitor = () => {
      void router?.push('/monitor')
    }
    monitorCard.addEventListener('click', openMonitor)
    cleanupCallbacks.push(() => monitorCard.removeEventListener('click', openMonitor))
  }

  const navToggle = host.value.querySelector<HTMLButtonElement>('.nav-toggle')
  const toggleMobileNav = () => nav?.classList.toggle('mobile-open')
  navToggle?.addEventListener('click', toggleMobileNav)
  if (navToggle) cleanupCallbacks.push(() => navToggle.removeEventListener('click', toggleMobileNav))

  host.value.querySelectorAll<HTMLButtonElement>('.lang button').forEach((button) => {
    const selectLanguage = () => {
      const nextLocale = normalizeLocale(button.dataset.locale || 'en')
      applyLandingLocale(nextLocale)
      void setLocale(nextLocale)
    }
    button.addEventListener('click', selectLanguage)
    cleanupCallbacks.push(() => button.removeEventListener('click', selectLanguage))
  })

  void refreshLandingData()
  activityTimer = window.setInterval(() => void refreshLandingData(), 10_000)
  void refreshChannelSummary()
  channelSummaryTimer = window.setInterval(() => void refreshChannelSummary(), 30_000)
  setupLiveQps()

  const onVisibilityChange = () => {
    if (!document.hidden) {
      void refreshLandingData()
      void refreshChannelSummary()
    }
  }
  document.addEventListener('visibilitychange', onVisibilityChange)
  cleanupCallbacks.push(() => document.removeEventListener('visibilitychange', onVisibilityChange))

  if (/jsdom/i.test(navigator.userAgent)) return
  const canvas = host.value.querySelector<HTMLCanvasElement>('#particles')
  const context = canvas?.getContext('2d')
  if (!canvas || !context) return

  let width = 0
  let height = 0
  const resizeCanvas = () => {
    width = canvas.width = window.innerWidth
    height = canvas.height = window.innerHeight
  }
  resizeCanvas()
  window.addEventListener('resize', resizeCanvas)
  cleanupCallbacks.push(() => window.removeEventListener('resize', resizeCanvas))

  const dots = Array.from({ length: Math.min(90, Math.floor(window.innerWidth / 16)) }, () => ({
    x: Math.random() * window.innerWidth,
    y: Math.random() * window.innerHeight,
    radius: Math.random() * 1.8 + 0.4,
    velocityX: (Math.random() - 0.5) * 0.25,
    velocityY: -(Math.random() * 0.35 + 0.05),
    alpha: Math.random() * 0.5 + 0.12,
  }))
  const drawParticles = () => {
    context.clearRect(0, 0, width, height)
    dots.forEach((dot) => {
      dot.x += dot.velocityX
      dot.y += dot.velocityY
      if (dot.y < -6) {
        dot.y = height + 6
        dot.x = Math.random() * width
      }
      if (dot.x < -6) dot.x = width + 6
      if (dot.x > width + 6) dot.x = -6
      context.beginPath()
      context.arc(dot.x, dot.y, dot.radius, 0, Math.PI * 2)
      context.fillStyle = `rgba(150,190,255,${dot.alpha})`
      context.fill()
    })
    animationFrame = window.requestAnimationFrame(drawParticles)
  }
  drawParticles()
}

function mountLandingPage() {
  if (!host.value) return

  const parsed = new DOMParser().parseFromString(rawHtml, 'text/html')
  const sourceStyle = parsed.querySelector('style')?.textContent || ''

  styleElement = document.createElement('style')
  styleElement.dataset.sub3Landing = 'true'
  styleElement.textContent = sourceStyle
  document.head.appendChild(styleElement)

  const pageNodes = Array.from(parsed.body.childNodes).filter((node) => node.nodeName !== 'SCRIPT')
  host.value.replaceChildren(...pageNodes.map((node) => document.importNode(node, true)))

  host.value.querySelectorAll<HTMLAnchorElement>('a[href^="/"]').forEach((element) => {
    const handler = (event: Event) => {
      event.preventDefault()
      const href = element.getAttribute('href')
      if (href) router?.push(href)
    }
    element.addEventListener('click', handler)
    navigationHandlers.push({ element, handler })
  })
  setupLandingInteractions()
  applyLandingLocale(normalizeLocale(getLocale()))
  void Promise.resolve(appStore.fetchPublicSettings()).then((settings) => {
    applyQQGroupSettings(settings)
    applyVersionBadge(settings)
  })
  stopLocaleWatch = watch(
    () => i18n.global.locale.value,
    (locale) => applyLandingLocale(normalizeLocale(String(locale))),
  )
}

onMounted(mountLandingPage)

onBeforeUnmount(() => {
  navigationHandlers.forEach(({ element, handler }) => element.removeEventListener('click', handler))
  navigationHandlers.length = 0
  cleanupCallbacks.forEach((cleanup) => cleanup())
  cleanupCallbacks.length = 0
  if (activityTimer !== null) window.clearTimeout(activityTimer)
  if (activityTimer !== null) window.clearInterval(activityTimer)
  if (channelSummaryTimer !== null) window.clearInterval(channelSummaryTimer)
  qpsUnsubscribe?.()
  landingDataController?.abort()
  if (animationFrame !== null) window.cancelAnimationFrame(animationFrame)
  stopLocaleWatch?.()
  styleElement?.remove()
  host.value?.replaceChildren()
  activityTimer = null
  channelSummaryTimer = null
  qpsUnsubscribe = null
  landingDataController = null
  landingOnline = null
  animationFrame = null
  stopLocaleWatch = null
  styleElement = null
})
</script>

<style scoped>
.sub3-landing-host {
  position: relative;
  isolation: isolate;
  min-height: 100vh;
  width: 100%;
  overflow-x: hidden;
  background: #071018;
}
</style>
