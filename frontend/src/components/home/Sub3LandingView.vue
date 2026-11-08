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
import { publicCardAPI, type PublicCardBalance } from '@/api/publicCard'
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

const publicCardStyles = [
  '',
  '.promo-card-panel{margin-top:22px;max-width:560px;border:1px solid rgb(var(--brand-400) / .18);border-radius:16px;background:linear-gradient(135deg,rgb(var(--brand-400) / .08),rgba(53,208,255,.055));box-shadow:0 18px 60px rgba(0,0,0,.28),inset 0 0 0 1px rgba(255,255,255,.05);padding:16px}',
  '.promo-card-head{display:flex;justify-content:space-between;gap:14px;align-items:flex-start;margin-bottom:13px}',
  '.promo-card-kicker{font-family:var(--mono);font-size:11px;color:var(--accent);letter-spacing:1.4px;text-transform:uppercase}',
  '.promo-card-title{margin-top:4px;font-size:17px;font-weight:700;color:#fff}',
  '.promo-card-sub{margin-top:4px;color:var(--text-3);font-size:12.5px;line-height:1.55}',
  '.promo-card-actions{display:flex;gap:10px;flex-wrap:wrap}',
  '.promo-card-grid{display:grid;grid-template-columns:minmax(0,1fr) minmax(0,1fr);gap:10px;margin-top:12px}',
  '.promo-card-field{display:flex;flex-direction:column;gap:7px}',
  '.promo-card-field label{font-size:12px;color:var(--text-3)}',
  '.promo-card-field input,.promo-card-keybox textarea{width:100%;border:1px solid rgba(255,255,255,.12);background:rgba(5,12,20,.72);color:#fff;border-radius:10px;padding:11px 12px;font-family:var(--mono);font-size:12px;outline:none}',
  '.promo-card-field input:focus,.promo-card-keybox textarea:focus{border-color:rgb(var(--brand-400) / .5);box-shadow:0 0 0 3px rgb(var(--brand-400) / .08)}',
  '.promo-card-result{margin-top:12px;border-top:1px solid rgba(255,255,255,.08);padding-top:12px;color:var(--text-2);font-size:13px;line-height:1.7}',
  '.promo-card-result.error{color:#ff8b9a}',
  '.promo-card-result.success{color:#d9ffe7}',
  '.promo-card-balance{display:grid;grid-template-columns:repeat(auto-fit,minmax(92px,1fr));gap:8px}',
  '.promo-card-metric{border:1px solid rgba(255,255,255,.08);border-radius:10px;background:rgba(255,255,255,.035);padding:9px}',
  '.promo-card-metric span{display:block;color:var(--text-3);font-size:11px}',
  '.promo-card-metric b{display:block;margin-top:3px;color:#fff;font-family:var(--display);font-size:15px;white-space:nowrap;overflow:hidden;text-overflow:ellipsis}',
  '.promo-card-keybox{margin-top:10px;display:grid;grid-template-columns:minmax(0,1fr) auto;gap:8px;align-items:start}',
  '.promo-card-keybox textarea{min-height:64px;resize:vertical;line-height:1.45}',
  '.promo-card-connection{margin-top:12px;display:grid;gap:8px}',
  '.promo-card-connection-row{display:grid;grid-template-columns:92px minmax(0,1fr) auto;gap:8px;align-items:center}',
  '.promo-card-connection-label{color:var(--text-3);font-size:12px}',
  '.promo-card-connection-value{min-width:0;border:1px solid rgba(255,255,255,.12);background:rgba(5,12,20,.72);color:#fff;border-radius:10px;padding:10px 12px;font-family:var(--mono);font-size:12px;overflow-wrap:anywhere}',
  '.promo-card-copy{height:42px}',
  '.promo-key-modal{position:fixed;inset:0;z-index:10000;display:flex;align-items:center;justify-content:center;padding:18px;background:rgba(1,7,13,.82);backdrop-filter:blur(10px)}',
  '.promo-key-modal-card{width:min(680px,100%);border:1px solid rgb(var(--brand-400) / .32);border-radius:20px;background:linear-gradient(145deg,#0b1923,#071018);box-shadow:0 30px 100px rgba(0,0,0,.65),0 0 50px rgb(var(--brand-400) / .08);padding:22px;color:#fff}',
  '.promo-key-modal-title{font-family:var(--display);font-size:24px;font-weight:800}',
  '.promo-key-modal-sub{margin-top:6px;color:var(--text-3);font-size:13px;line-height:1.6}',
  '.promo-key-modal-value{display:block;margin-top:14px;border:1px solid rgba(255,255,255,.14);border-radius:12px;background:#050c14;padding:14px;color:#dfffea;font-family:var(--mono);font-size:12px;line-height:1.65;overflow-wrap:anywhere;user-select:all}',
  '.promo-key-modal-actions{display:flex;justify-content:flex-end;gap:10px;margin-top:16px;flex-wrap:wrap}',
  '@media (max-width:760px){.promo-card-head{display:block}.promo-card-actions{margin-top:12px}.promo-card-grid,.promo-card-balance,.promo-card-keybox,.promo-card-connection-row{grid-template-columns:1fr}.promo-card-connection-row .promo-card-copy{width:100%}}',
].join('\n')

const landingMessages = {
  zh: {
    nav: ['服务状态', '管理登录', 'QQ群'],
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
    nav: ['Service Status', 'Admin Login', 'QQ Group'],
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

function formatPublicCardMoney(value: number) {
  if (!Number.isFinite(value)) return '--'
  if (value < 0) return '无限'
  return '$' + value.toFixed(value >= 10 ? 2 : 4)
}

function publicCardStatusLabel(status: string) {
  switch (status) {
    case 'active':
      return '可用'
    case 'quota_exhausted':
      return '额度已用完'
    case 'expired':
      return '已过期'
    case 'disabled':
      return '已停用'
    case 'unused':
      return '未兑换'
    case 'used':
      return '已兑换'
    default:
      return status || '--'
  }
}

function setPublicCardResultText(result: HTMLElement | null, message: string, kind: 'idle' | 'success' | 'error' = 'idle') {
  if (!result) return
  result.classList.toggle('success', kind === 'success')
  result.classList.toggle('error', kind === 'error')
  result.textContent = message
}

function publicCardErrorMessage(error: unknown) {
  const maybe = error as { message?: unknown }
  return typeof maybe?.message === 'string' && maybe.message
    ? maybe.message
    : '操作失败，请检查卡密或 SK 后重试。'
}

async function copyPublicCardText(value: string) {
  if (navigator.clipboard?.writeText) {
    await navigator.clipboard.writeText(value)
    return
  }
  const fallback = document.createElement('textarea')
  fallback.value = value
  fallback.style.position = 'fixed'
  fallback.style.opacity = '0'
  document.body.appendChild(fallback)
  fallback.select()
  const copied = typeof document.execCommand === 'function' && document.execCommand('copy')
  fallback.remove()
  if (!copied) throw new Error('clipboard unavailable')
}

function renderPublicCardBalance(result: HTMLElement | null, balance: PublicCardBalance, apiKey?: string) {
  if (!result) return
  result.classList.add('success')
  result.classList.remove('error')
  result.replaceChildren()

  const metrics = document.createElement('div')
  metrics.className = 'promo-card-balance'
  const rows = [
    ['状态', publicCardStatusLabel(balance.status)],
    ['并发', String(balance.concurrency || 1)],
    ['总额度', formatPublicCardMoney(balance.quota)],
    ['已使用', formatPublicCardMoney(balance.quota_used)],
    ['剩余额度', formatPublicCardMoney(balance.quota_remaining)],
  ]
  rows.forEach(([label, value]) => {
    const item = document.createElement('div')
    item.className = 'promo-card-metric'
    const labelEl = document.createElement('span')
    labelEl.textContent = label
    const valueEl = document.createElement('b')
    valueEl.textContent = value
    item.append(labelEl, valueEl)
    metrics.appendChild(item)
  })
  result.appendChild(metrics)

  if (!apiKey) return

  const connection = document.createElement('div')
  connection.className = 'promo-card-connection'
  const apiBase = buildGatewayUrl('/v1')

  const addConnectionRow = (label: string, value: string, copyLabel: string) => {
    const row = document.createElement('div')
    row.className = 'promo-card-connection-row'
    const labelEl = document.createElement('span')
    labelEl.className = 'promo-card-connection-label'
    labelEl.textContent = label
    const valueEl = document.createElement('code')
    valueEl.className = 'promo-card-connection-value'
    valueEl.textContent = value
    const copy = document.createElement('button')
    copy.type = 'button'
    copy.className = 'btn btn-primary promo-card-copy'
    copy.textContent = copyLabel
    const copyValue = async () => {
      try {
        await copyPublicCardText(value)
        copy.textContent = '已复制'
        window.setTimeout(() => {
          copy.textContent = copyLabel
        }, 1800)
      } catch {
        copy.textContent = '请手动复制'
      }
    }
    copy.addEventListener('click', copyValue)
    cleanupCallbacks.push(() => copy.removeEventListener('click', copyValue))
    row.append(labelEl, valueEl, copy)
    connection.appendChild(row)
  }

  addConnectionRow('API 地址', apiBase, '复制地址')
  addConnectionRow('SK / API Key', apiKey, '复制 SK')
  result.appendChild(connection)

  const hint = document.createElement('div')
  hint.style.marginTop = '8px'
  hint.style.color = 'var(--text-3)'
  hint.textContent = '在 OpenAI 兼容客户端中，将 API 地址和 SK 分别填入 Base URL 与 API Key。完整 SK 只显示一次，请立即保存。'
  result.appendChild(hint)

  showPublicCardKeyModal(apiKey, apiBase)
}

function showPublicCardKeyModal(apiKey: string, apiBase: string) {
  document.querySelector('.promo-key-modal')?.remove()

  const overlay = document.createElement('div')
  overlay.className = 'promo-key-modal'
  overlay.setAttribute('role', 'dialog')
  overlay.setAttribute('aria-modal', 'true')

  const card = document.createElement('div')
  card.className = 'promo-key-modal-card'
  const title = document.createElement('div')
  title.className = 'promo-key-modal-title'
  title.textContent = '兑换成功，SK 已生成'
  const sub = document.createElement('div')
  sub.className = 'promo-key-modal-sub'
  sub.textContent = '完整 SK 只展示这一次。请立即复制保存，然后把接口地址填入客户端的 Base URL。'
  const baseValue = document.createElement('code')
  baseValue.className = 'promo-key-modal-value'
  baseValue.textContent = `API 地址：${apiBase}`
  const keyValue = document.createElement('code')
  keyValue.className = 'promo-key-modal-value'
  keyValue.textContent = `SK / API Key：${apiKey}`
  const actions = document.createElement('div')
  actions.className = 'promo-key-modal-actions'
  const close = document.createElement('button')
  close.type = 'button'
  close.className = 'btn btn-ghost'
  close.textContent = '我已保存，关闭'
  const copy = document.createElement('button')
  copy.type = 'button'
  copy.className = 'btn btn-primary'
  copy.textContent = '复制 SK'

  const remove = () => overlay.remove()
  close.addEventListener('click', remove)
  overlay.addEventListener('click', (event) => {
    if (event.target === overlay) remove()
  })
  copy.addEventListener('click', async () => {
    try {
      await copyPublicCardText(apiKey)
      copy.textContent = 'SK 已复制'
    } catch {
      copy.textContent = '请手动复制上方 SK'
    }
  })

  actions.append(close, copy)
  card.append(title, sub, baseValue, keyValue, actions)
  overlay.appendChild(card)
  document.body.appendChild(overlay)
  cleanupCallbacks.push(remove)
}

function setupPublicCardPanel() {
  if (!host.value || host.value.querySelector('#promoCardPanel')) return
  const anchor = host.value.querySelector<HTMLElement>('.hero-cta')
  if (!anchor) return

  const panel = document.createElement('section')
  panel.id = 'promoCardPanel'
  panel.className = 'promo-card-panel'
  panel.innerHTML = [
    '<div class="promo-card-head">',
    '  <div>',
    '    <div class="promo-card-kicker">PROMO CARD ACCESS</div>',
    '    <div class="promo-card-title">卡密兑换与 SK 余额查询</div>',
    '    <div class="promo-card-sub">输入卡密可兑换一把带额度的 SK；使用卡密或 SK 都能查询剩余额度。</div>',
    '  </div>',
    '  <div class="promo-card-actions">',
    '    <button type="button" class="btn btn-primary" id="promoRedeemBtn">兑换卡密</button>',
    '    <button type="button" class="btn btn-ghost" id="promoQueryBtn">查询余额</button>',
    '  </div>',
    '</div>',
    '<div class="promo-card-grid">',
    '  <div class="promo-card-field">',
    '    <label for="promoCardCode">卡密</label>',
    '    <input id="promoCardCode" autocomplete="off" placeholder="输入后台生成的卡密">',
    '  </div>',
    '  <div class="promo-card-field">',
    '    <label for="promoApiKey">SK / 卡密查询</label>',
    '    <input id="promoApiKey" autocomplete="off" placeholder="粘贴 sk-... 或卡密查询余额">',
    '  </div>',
    '</div>',
    '<div class="promo-card-result" id="promoCardResult">推广卡兑换后会自动生成 SK；余额用完后 SK 会自动失效。</div>',
  ].join('')
  anchor.insertAdjacentElement('afterend', panel)

  const codeInput = panel.querySelector<HTMLInputElement>('#promoCardCode')
  const keyInput = panel.querySelector<HTMLInputElement>('#promoApiKey')
  const redeemButton = panel.querySelector<HTMLButtonElement>('#promoRedeemBtn')
  const queryButton = panel.querySelector<HTMLButtonElement>('#promoQueryBtn')
  const result = panel.querySelector<HTMLElement>('#promoCardResult')

  const redeem = async () => {
    const code = codeInput?.value.trim() || ''
    if (!code) {
      setPublicCardResultText(result, '请先输入卡密。', 'error')
      return
    }
    if (redeemButton) redeemButton.disabled = true
    setPublicCardResultText(result, '正在兑换卡密，请稍等...', 'idle')
    try {
      const data = await publicCardAPI.redeem(code)
      if (keyInput) keyInput.value = data.api_key
      renderPublicCardBalance(result, data, data.api_key)
    } catch (error) {
      setPublicCardResultText(result, publicCardErrorMessage(error), 'error')
    } finally {
      if (redeemButton) redeemButton.disabled = false
    }
  }

  const query = async () => {
    const credential = codeInput?.value.trim() || keyInput?.value.trim() || ''
    if (!credential) {
      setPublicCardResultText(result, '请先输入卡密或粘贴 SK。', 'error')
      return
    }
    if (queryButton) queryButton.disabled = true
    setPublicCardResultText(result, '正在查询余额...', 'idle')
    try {
      const data = await publicCardAPI.query(credential)
      renderPublicCardBalance(result, data)
    } catch (error) {
      setPublicCardResultText(result, publicCardErrorMessage(error), 'error')
    } finally {
      if (queryButton) queryButton.disabled = false
    }
  }

  redeemButton?.addEventListener('click', redeem)
  queryButton?.addEventListener('click', query)
  codeInput?.addEventListener('keydown', (event) => {
    if (event.key === 'Enter') void redeem()
  })
  keyInput?.addEventListener('keydown', (event) => {
    if (event.key === 'Enter') void query()
  })
  cleanupCallbacks.push(() => redeemButton?.removeEventListener('click', redeem))
  cleanupCallbacks.push(() => queryButton?.removeEventListener('click', query))
}

function setupLandingInteractions() {
  if (!host.value) return
  setupPublicCardPanel()

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
  styleElement.textContent = sourceStyle + publicCardStyles
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
