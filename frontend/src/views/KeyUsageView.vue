<template>
  <div class="key-usage-page">
    <div class="usage-texture" aria-hidden="true"></div>

    <header class="usage-header">
      <nav class="usage-nav">
        <router-link to="/home" class="usage-brand">
          <img v-if="siteLogo" :src="siteLogo" alt="Logo" class="usage-logo" />
          <span>{{ siteName }}</span>
        </router-link>

        <div class="usage-nav-actions">
          <router-link to="/home" class="usage-nav-link">首页</router-link>
          <a
            v-if="docUrl"
            :href="docUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="usage-nav-link"
          >
            文档
          </a>
          <router-link to="/login" class="usage-nav-cta">进入控制台</router-link>
        </div>
      </nav>
    </header>

    <main class="usage-main">
      <section class="usage-hero">
        <div class="usage-hero-copy fade-up">
          <p class="usage-kicker">KEY STATUS</p>
          <h1>检查 Key 状态。</h1>
          <p class="usage-lede">
            查询余额、额度、Token 使用、模型消耗和重置时间。请求只用于只读统计，不在前台保存密钥。
          </p>

          <div class="usage-query-panel">
            <label class="usage-input-label" for="usage-api-key">API Key</label>
            <div class="usage-input-row">
              <div class="usage-input-wrap">
                <svg class="usage-input-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">
                  <rect x="3" y="11" width="18" height="10" rx="2" />
                  <path d="M7 11V7a5 5 0 0 1 10 0v4" />
                </svg>
                <input
                  id="usage-api-key"
                  v-model="apiKey"
                  :type="keyVisible ? 'text' : 'password'"
                  :placeholder="t('keyUsage.placeholder')"
                  class="usage-input"
                  @keydown.enter="queryKey"
                />
                <button
                  type="button"
                  class="usage-eye"
                  :aria-label="keyVisible ? '隐藏密钥' : '显示密钥'"
                  @click="keyVisible = !keyVisible"
                >
                  <svg v-if="!keyVisible" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">
                    <path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94" />
                    <path d="M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19" />
                    <path d="M14.12 14.12a3 3 0 0 1-4.24-4.24" />
                    <line x1="1" y1="1" x2="23" y2="23" />
                  </svg>
                  <svg v-else viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">
                    <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z" />
                    <circle cx="12" cy="12" r="3" />
                  </svg>
                </button>
              </div>

              <button
                type="button"
                class="usage-query-btn"
                :disabled="isQuerying"
                @click="queryKey"
              >
                <span v-if="isQuerying" class="usage-spinner"></span>
                {{ isQuerying ? t('keyUsage.querying') : t('keyUsage.query') }}
              </button>
            </div>
            <p class="usage-privacy">{{ t('keyUsage.privacyNote') }}</p>

            <div v-if="showDatePicker" class="usage-range">
              <span>{{ t('keyUsage.dateRange') }}</span>
              <button
                v-for="range in dateRanges"
                :key="range.key"
                type="button"
                class="usage-range-btn"
                :class="{ 'is-active': currentRange === range.key }"
                @click="setDateRange(range.key)"
              >
                {{ range.label }}
              </button>
              <div v-if="currentRange === 'custom'" class="usage-custom-range">
                <input v-model="customStartDate" type="date" class="usage-date-input" />
                <span>-</span>
                <input v-model="customEndDate" type="date" class="usage-date-input" />
                <button type="button" class="usage-range-apply" @click="queryKey">
                  {{ t('keyUsage.apply') }}
                </button>
              </div>
            </div>
          </div>
        </div>

        <aside class="usage-product-card fade-up fade-up-delay-1" aria-label="API usage preview">
          <div class="usage-card-bar">
            <span></span>
            <span></span>
            <span></span>
          </div>
          <div class="usage-endpoint">
            <span>BASE URL</span>
            <strong>{{ baseUrl }}</strong>
          </div>
          <div class="usage-code-lines">
            <p><span>GET</span> /v1/usage</p>
            <p><span>AUTH</span> Bearer sk-...hidden</p>
            <p><span>MODE</span> read_only</p>
          </div>
          <div class="usage-metrics-preview">
            <div>
              <span>Quota</span>
              <strong>68.4%</strong>
            </div>
            <div>
              <span>Tokens</span>
              <strong>184,920</strong>
            </div>
            <div>
              <span>Models</span>
              <strong>7</strong>
            </div>
          </div>
          <div class="usage-route-flow">
            <span>Key</span>
            <i></i>
            <span>Gateway</span>
            <i></i>
            <span>Usage Log</span>
          </div>
        </aside>
      </section>

      <section class="usage-principles">
        <div class="usage-principle">
          <span>01</span>
          <strong>只读查询</strong>
          <p>使用 `/v1/usage` 获取状态，不改变密钥、额度或账号池配置。</p>
        </div>
        <div class="usage-principle">
          <span>02</span>
          <strong>按配置呈现</strong>
          <p>余额、订阅、限额和模型统计以当前部署返回的数据为准。</p>
        </div>
        <div class="usage-principle">
          <span>03</span>
          <strong>开发者可读</strong>
          <p>把 Token、费用、限速和模型消耗拆开看，减少排查成本。</p>
        </div>
      </section>

      <section v-if="showResults" class="usage-results">
        <div v-if="showLoading" class="usage-loading-grid">
          <div class="usage-skeleton-card">
            <div class="skeleton h-5 w-28"></div>
            <div class="skeleton h-44 w-44 rounded-full"></div>
          </div>
          <div class="usage-skeleton-card">
            <div class="skeleton h-5 w-24"></div>
            <div class="skeleton h-44 w-44 rounded-full"></div>
          </div>
          <div class="usage-skeleton-card usage-skeleton-wide">
            <div class="skeleton h-5 w-36"></div>
            <div class="skeleton h-4 w-full"></div>
            <div class="skeleton h-4 w-4/5"></div>
            <div class="skeleton h-4 w-2/3"></div>
          </div>
        </div>

        <div v-else-if="resultData" class="usage-result-stack">
          <div v-if="statusInfo" class="usage-status-pill fade-up">
            <span class="pulse-dot" :class="statusInfo.isActive ? 'is-ok' : 'is-stop'"></span>
            <strong>{{ statusInfo.label }}</strong>
            <em>{{ statusInfo.statusText }}</em>
          </div>

          <div v-if="ringItems.length > 0" :class="ringGridClass">
            <div
              v-for="(ring, i) in ringItems"
              :key="i"
              class="usage-ring-card fade-up"
              :class="`fade-up-delay-${Math.min(i + 1, 4)}`"
            >
              <div class="usage-ring-head">
                <h3>{{ ring.title }}</h3>
                <span>{{ ring.iconType }}</span>
              </div>
              <div class="usage-ring-wrap">
                <svg class="usage-ring" viewBox="0 0 160 160">
                  <circle cx="80" cy="80" r="68" fill="none" :stroke="ringTrackColor" stroke-width="10" />
                  <circle
                    class="progress-ring"
                    cx="80"
                    cy="80"
                    r="68"
                    fill="none"
                    :stroke="`url(#ring-grad-${i})`"
                    stroke-width="10"
                    stroke-linecap="round"
                    :stroke-dasharray="CIRCUMFERENCE.toFixed(2)"
                    :stroke-dashoffset="getRingOffset(ring)"
                  />
                  <defs>
                    <linearGradient :id="`ring-grad-${i}`" x1="0%" y1="0%" x2="100%" y2="100%">
                      <stop offset="0%" :stop-color="RING_GRADIENTS[i % 4].from" />
                      <stop offset="100%" :stop-color="RING_GRADIENTS[i % 4].to" />
                    </linearGradient>
                  </defs>
                </svg>
                <div class="usage-ring-center">
                  <template v-if="ring.isBalance">
                    <strong :style="{ color: RING_GRADIENTS[i % 4].from }">{{ ring.amount }}</strong>
                  </template>
                  <template v-else>
                    <strong>{{ displayPcts[i] ?? 0 }}%</strong>
                    <span>{{ t('keyUsage.used') }}</span>
                    <em :style="{ color: RING_GRADIENTS[i % 4].from }">{{ ring.amount }}</em>
                    <small v-if="ring.resetAt && formatResetTime(ring.resetAt)">
                      重置 {{ formatResetTime(ring.resetAt) }}
                    </small>
                  </template>
                </div>
              </div>
            </div>
          </div>

          <div v-if="detailRows.length > 0" class="usage-detail-panel fade-up fade-up-delay-3">
            <div class="usage-panel-head">
              <span>DETAIL</span>
              <h3>{{ t('keyUsage.detailInfo') }}</h3>
            </div>
            <div class="usage-detail-list">
              <div v-for="(row, i) in detailRows" :key="i" class="usage-detail-row">
                <div>
                  <span class="usage-detail-icon">
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" v-html="row.iconSvg"></svg>
                  </span>
                  <span>{{ row.label }}</span>
                </div>
                <strong :class="row.valueClass">{{ row.value }}</strong>
              </div>
            </div>
          </div>

          <div v-if="usageStatCells.length > 0" class="usage-stats-panel fade-up fade-up-delay-3">
            <div class="usage-panel-head">
              <span>TOKENS</span>
              <h3>{{ t('keyUsage.tokenStats') }}</h3>
            </div>
            <div class="usage-stat-grid">
              <div v-for="(cell, i) in usageStatCells" :key="i" class="usage-stat-cell">
                <span>{{ cell.label }}</span>
                <strong>{{ cell.value }}</strong>
              </div>
            </div>
          </div>

          <div v-if="modelStats.length > 0" class="usage-model-panel fade-up fade-up-delay-4">
            <div class="usage-panel-head">
              <span>MODELS</span>
              <h3>{{ t('keyUsage.modelStats') }}</h3>
            </div>
            <div class="usage-table-wrap">
              <table>
                <thead>
                  <tr>
                    <th>{{ t('keyUsage.model') }}</th>
                    <th>{{ t('keyUsage.requests') }}</th>
                    <th>{{ t('keyUsage.inputTokens') }}</th>
                    <th>{{ t('keyUsage.outputTokens') }}</th>
                    <th>{{ t('keyUsage.cacheCreationTokens') }}</th>
                    <th>{{ t('keyUsage.cacheReadTokens') }}</th>
                    <th>{{ t('keyUsage.totalTokens') }}</th>
                    <th>{{ t('keyUsage.cost') }}</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="(m, i) in modelStats" :key="i">
                    <td>{{ m.model || '-' }}</td>
                    <td>{{ fmtNum(m.requests) }}</td>
                    <td>{{ fmtNum(m.input_tokens) }}</td>
                    <td>{{ fmtNum(m.output_tokens) }}</td>
                    <td>{{ fmtNum(m.cache_creation_tokens) }}</td>
                    <td>{{ fmtNum(m.cache_read_tokens) }}</td>
                    <td>{{ fmtNum(m.total_tokens) }}</td>
                    <td>{{ usd(m.actual_cost != null ? m.actual_cost : m.cost) }}</td>
                  </tr>
                </tbody>
              </table>
            </div>
          </div>
        </div>
      </section>
    </main>

    <footer class="usage-footer">
      <span>{{ currentYear }} {{ siteName }}</span>
      <router-link to="/home">返回首页</router-link>
    </footer>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores'

const { t, locale } = useI18n()
const appStore = useAppStore()

const siteName = computed(() => appStore.cachedPublicSettings?.site_name || appStore.siteName || 'Codex Access')
const siteLogo = computed(() => appStore.cachedPublicSettings?.site_logo || appStore.siteLogo || '')
const docUrl = computed(() => appStore.cachedPublicSettings?.doc_url || appStore.docUrl || '')
const currentYear = computed(() => new Date().getFullYear())
const baseUrl = computed(() => (typeof window === 'undefined' ? '/v1' : `${window.location.origin}/v1`))

const isDark = ref(document.documentElement.classList.contains('dark'))
const apiKey = ref('')
const keyVisible = ref(false)
const isQuerying = ref(false)
const showResults = ref(false)
const showLoading = ref(false)
const showDatePicker = ref(false)
// eslint-disable-next-line @typescript-eslint/no-explicit-any
const resultData = ref<any>(null)
const now = ref(new Date())
let resetTimer: ReturnType<typeof setInterval> | null = null

type DateRangeKey = 'today' | '7d' | '30d' | 'custom'
const currentRange = ref<DateRangeKey>('today')
const customStartDate = ref('')
const customEndDate = ref('')

const dateRanges = computed(() => [
  { key: 'today' as const, label: t('keyUsage.dateRangeToday') },
  { key: '7d' as const, label: t('keyUsage.dateRange7d') },
  { key: '30d' as const, label: t('keyUsage.dateRange30d') },
  { key: 'custom' as const, label: t('keyUsage.dateRangeCustom') },
])

function setDateRange(key: DateRangeKey) {
  currentRange.value = key
  if (key !== 'custom') {
    queryKey()
  }
}

function getDateParams(): string {
  const current = new Date()
  const fmt = (d: Date) => d.toISOString().split('T')[0]

  if (currentRange.value === 'custom') {
    if (customStartDate.value && customEndDate.value) {
      return `start_date=${customStartDate.value}&end_date=${customEndDate.value}`
    }
    return ''
  }

  const end = fmt(current)
  let start: string
  switch (currentRange.value) {
    case 'today':
      start = end
      break
    case '7d':
      start = fmt(new Date(current.getTime() - 7 * 86400000))
      break
    case '30d':
      start = fmt(new Date(current.getTime() - 30 * 86400000))
      break
    default:
      start = fmt(new Date(current.getTime() - 30 * 86400000))
  }
  return `start_date=${start}&end_date=${end}`
}

const CIRCUMFERENCE = 2 * Math.PI * 68
const RING_GRADIENTS = [
  { from: '#7c5a3b', to: '#d2ad7f' },
  { from: '#53614a', to: '#a9b78f' },
  { from: '#8b5b4b', to: '#d4a08a' },
  { from: '#2b211b', to: '#b99a78' },
]

const ringAnimated = ref(false)
const displayPcts = ref<number[]>([])

const ringTrackColor = computed(() => isDark.value ? '#2a221d' : '#e8ddce')

interface RingItem {
  title: string
  pct: number
  amount: string
  isBalance?: boolean
  iconType: 'clock' | 'calendar' | 'dollar'
  resetAt?: string | null
}

function getRingOffset(ring: RingItem): number {
  if (!ringAnimated.value) return CIRCUMFERENCE
  if (ring.isBalance) return 0
  return CIRCUMFERENCE - (Math.min(ring.pct, 100) / 100) * CIRCUMFERENCE
}

function triggerRingAnimation(items: RingItem[]) {
  ringAnimated.value = false
  displayPcts.value = items.map(() => 0)

  nextTick(() => {
    requestAnimationFrame(() => {
      setTimeout(() => {
        ringAnimated.value = true

        const duration = 1000
        const startTime = performance.now()
        const targets = items.map(item => item.isBalance ? 0 : item.pct)

        function tick() {
          const elapsed = performance.now() - startTime
          const p = Math.min(elapsed / duration, 1)
          const ease = 1 - Math.pow(1 - p, 3)
          displayPcts.value = targets.map(target => Math.round(ease * target))
          if (p < 1) requestAnimationFrame(tick)
        }
        requestAnimationFrame(tick)
      }, 50)
    })
  })
}

const statusInfo = computed(() => {
  const data = resultData.value
  if (!data) return null

  if (data.mode === 'quota_limited') {
    const isValid = data.isValid !== false
    const statusMap: Record<string, string> = {
      active: 'Active',
      quota_exhausted: 'Quota Exhausted',
      expired: 'Expired',
    }
    return {
      label: t('keyUsage.quotaMode'),
      statusText: statusMap[data.status] || data.status || 'Unknown',
      isActive: isValid && data.status === 'active',
    }
  }

  return {
    label: data.planName || t('keyUsage.walletBalance'),
    statusText: 'Active',
    isActive: true,
  }
})

const ringItems = computed<RingItem[]>(() => {
  const data = resultData.value
  if (!data) return []

  const items: RingItem[] = []

  if (data.mode === 'quota_limited') {
    if (data.quota) {
      const pct = data.quota.limit > 0 ? Math.min(Math.round((data.quota.used / data.quota.limit) * 100), 100) : 0
      items.push({ title: t('keyUsage.totalQuota'), pct, amount: `${usd(data.quota.used)} / ${usd(data.quota.limit)}`, iconType: 'dollar' })
    }
    if (data.rate_limits) {
      const windowLabels: Record<string, string> = { '5h': t('keyUsage.limit5h'), '1d': t('keyUsage.limitDaily'), '7d': t('keyUsage.limit7d') }
      const windowIcons: Record<string, 'clock' | 'calendar'> = { '5h': 'clock', '1d': 'calendar', '7d': 'calendar' }
      for (const rl of data.rate_limits) {
        const pct = rl.limit > 0 ? Math.min(Math.round((rl.used / rl.limit) * 100), 100) : 0
        items.push({
          title: windowLabels[rl.window] || rl.window,
          pct,
          amount: `${usd(rl.used)} / ${usd(rl.limit)}`,
          iconType: windowIcons[rl.window] || 'clock',
          resetAt: rl.reset_at,
        })
      }
    }
  } else {
    if (data.subscription) {
      const sub = data.subscription
      const limits = [
        { label: t('keyUsage.limitDaily'), usage: sub.daily_usage_usd, limit: sub.daily_limit_usd },
        { label: t('keyUsage.limitWeekly'), usage: sub.weekly_usage_usd, limit: sub.weekly_limit_usd },
        { label: t('keyUsage.limitMonthly'), usage: sub.monthly_usage_usd, limit: sub.monthly_limit_usd },
      ]
      for (const l of limits) {
        if (l.limit != null && l.limit > 0) {
          const pct = Math.min(Math.round((l.usage / l.limit) * 100), 100)
          items.push({ title: l.label, pct, amount: `${usd(l.usage)} / ${usd(l.limit)}`, iconType: 'calendar' })
        }
      }
    }
    if (!data.subscription && data.balance != null) {
      items.push({ title: t('keyUsage.walletBalance'), pct: 0, amount: usd(data.balance), isBalance: true, iconType: 'dollar' })
    }
  }

  return items
})

const ringGridClass = computed(() => {
  const len = ringItems.value.length
  if (len === 1) return 'usage-ring-grid usage-ring-grid--one'
  if (len === 2) return 'usage-ring-grid usage-ring-grid--two'
  return 'usage-ring-grid'
})

interface DetailRow {
  iconBg: string
  iconColor: string
  iconSvg: string
  label: string
  value: string
  valueClass: string
}

function getUsageColor(pct: number): string {
  if (pct > 90) return 'text-rose-500'
  if (pct > 70) return 'text-amber-600'
  return 'text-emerald-700'
}

const detailRows = computed<DetailRow[]>(() => {
  const data = resultData.value
  if (!data) return []

  const rows: DetailRow[] = []
  const ICON_SHIELD = '<path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/>'
  const ICON_CALENDAR = '<rect x="3" y="4" width="18" height="18" rx="2" ry="2"/><line x1="16" y1="2" x2="16" y2="6"/><line x1="8" y1="2" x2="8" y2="6"/><line x1="3" y1="10" x2="21" y2="10"/>'
  const ICON_DOLLAR = '<line x1="12" y1="1" x2="12" y2="23"/><path d="M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"/>'
  const ICON_CHECK = '<polyline points="20 6 9 17 4 12"/>'

  if (data.mode === 'quota_limited') {
    if (data.quota) {
      const remainColor = data.quota.remaining <= 0 ? 'text-rose-500'
        : data.quota.remaining < data.quota.limit * 0.1 ? 'text-amber-600'
        : 'text-emerald-700'
      rows.push({
        iconBg: 'bg-emerald-500/10', iconColor: 'text-emerald-500', iconSvg: ICON_SHIELD,
        label: t('keyUsage.remainingQuota'), value: usd(data.quota.remaining), valueClass: remainColor,
      })
    }
    if (data.expires_at) {
      const daysLeft = data.days_until_expiry
      let expiryStr = formatDate(data.expires_at)
      if (daysLeft != null) {
        expiryStr += daysLeft > 0 ? ` ${t('keyUsage.daysLeft', { days: daysLeft })}` : daysLeft === 0 ? ` ${t('keyUsage.todayExpires')}` : ''
      }
      rows.push({
        iconBg: 'bg-amber-500/10', iconColor: 'text-amber-500', iconSvg: ICON_CALENDAR,
        label: t('keyUsage.expiresAt'), value: expiryStr, valueClass: '',
      })
    }
    if (data.rate_limits) {
      const windowMap: Record<string, string> = { '5h': '5H', '1d': locale.value === 'zh' ? '日' : 'D', '7d': '7D' }
      for (const rl of data.rate_limits) {
        const pct = rl.limit > 0 ? (rl.used / rl.limit) * 100 : 0
        let valueStr = `${usd(rl.used)} / ${usd(rl.limit)}`
        const resetStr = formatResetTime(rl.reset_at)
        if (resetStr) {
          valueStr += `（重置 ${resetStr}）`
        }
        rows.push({
          iconBg: 'bg-primary-500/10', iconColor: 'text-primary-500', iconSvg: ICON_DOLLAR,
          label: `${t('keyUsage.usedQuota')} (${windowMap[rl.window] || rl.window})`,
          value: valueStr,
          valueClass: getUsageColor(pct),
        })
      }
    }
  } else {
    rows.push({
      iconBg: 'bg-emerald-500/10', iconColor: 'text-emerald-500', iconSvg: ICON_CHECK,
      label: t('keyUsage.subscriptionType'), value: data.planName || t('keyUsage.walletBalance'), valueClass: '',
    })

    if (data.subscription) {
      const sub = data.subscription
      if (sub.daily_limit_usd > 0) {
        const pct = (sub.daily_usage_usd / sub.daily_limit_usd) * 100
        rows.push({
          iconBg: 'bg-primary-500/10', iconColor: 'text-primary-500', iconSvg: ICON_DOLLAR,
          label: `${t('keyUsage.usedQuota')} (${locale.value === 'zh' ? '日' : 'D'})`, value: `${usd(sub.daily_usage_usd)} / ${usd(sub.daily_limit_usd)}`, valueClass: getUsageColor(pct),
        })
      }
      if (sub.weekly_limit_usd > 0) {
        const pct = (sub.weekly_usage_usd / sub.weekly_limit_usd) * 100
        rows.push({
          iconBg: 'bg-indigo-500/10', iconColor: 'text-indigo-500', iconSvg: ICON_DOLLAR,
          label: `${t('keyUsage.usedQuota')} (${locale.value === 'zh' ? '周' : 'W'})`, value: `${usd(sub.weekly_usage_usd)} / ${usd(sub.weekly_limit_usd)}`, valueClass: getUsageColor(pct),
        })
      }
      if (sub.monthly_limit_usd > 0) {
        const pct = (sub.monthly_usage_usd / sub.monthly_limit_usd) * 100
        rows.push({
          iconBg: 'bg-emerald-500/10', iconColor: 'text-emerald-500', iconSvg: ICON_DOLLAR,
          label: `${t('keyUsage.usedQuota')} (${locale.value === 'zh' ? '月' : 'M'})`, value: `${usd(sub.monthly_usage_usd)} / ${usd(sub.monthly_limit_usd)}`, valueClass: getUsageColor(pct),
        })
      }
      if (sub.expires_at) {
        rows.push({
          iconBg: 'bg-amber-500/10', iconColor: 'text-amber-500', iconSvg: ICON_CALENDAR,
          label: t('keyUsage.subscriptionExpires'), value: formatDate(sub.expires_at), valueClass: '',
        })
      }
    }

    const remainColor = data.remaining != null
      ? (data.remaining <= 0 ? 'text-rose-500' : data.remaining < 10 ? 'text-amber-600' : 'text-emerald-700')
      : ''
    rows.push({
      iconBg: 'bg-emerald-500/10', iconColor: 'text-emerald-500', iconSvg: ICON_SHIELD,
      label: t('keyUsage.remainingQuota'), value: data.remaining != null ? usd(data.remaining) : '-', valueClass: remainColor,
    })
  }

  return rows
})

interface StatCell {
  label: string
  value: string
}

const usageStatCells = computed<StatCell[]>(() => {
  const usage = resultData.value?.usage
  if (!usage) return []

  const today = usage.today || {}
  const total = usage.total || {}

  return [
    { label: t('keyUsage.todayRequests'), value: fmtNum(today.requests) },
    { label: t('keyUsage.todayInputTokens'), value: fmtNum(today.input_tokens) },
    { label: t('keyUsage.todayOutputTokens'), value: fmtNum(today.output_tokens) },
    { label: t('keyUsage.todayTokens'), value: fmtNum(today.total_tokens) },
    { label: t('keyUsage.todayCacheCreation'), value: fmtNum(today.cache_creation_tokens) },
    { label: t('keyUsage.todayCacheRead'), value: fmtNum(today.cache_read_tokens) },
    { label: t('keyUsage.todayCost'), value: usd(today.actual_cost) },
    { label: t('keyUsage.rpmTpm'), value: `${usage.rpm || 0} / ${usage.tpm || 0}` },
    { label: t('keyUsage.totalRequests'), value: fmtNum(total.requests) },
    { label: t('keyUsage.totalInputTokens'), value: fmtNum(total.input_tokens) },
    { label: t('keyUsage.totalOutputTokens'), value: fmtNum(total.output_tokens) },
    { label: t('keyUsage.totalTokensLabel'), value: fmtNum(total.total_tokens) },
    { label: t('keyUsage.totalCacheCreation'), value: fmtNum(total.cache_creation_tokens) },
    { label: t('keyUsage.totalCacheRead'), value: fmtNum(total.cache_read_tokens) },
    { label: t('keyUsage.totalCost'), value: usd(total.actual_cost) },
    { label: t('keyUsage.avgDuration'), value: usage.average_duration_ms ? `${Math.round(usage.average_duration_ms)} ms` : '-' },
  ]
})

// eslint-disable-next-line @typescript-eslint/no-explicit-any
const modelStats = computed<any[]>(() => resultData.value?.model_stats || [])

function usd(value: number | null | undefined): string {
  if (value == null || value < 0) return '-'
  return '$' + Number(value).toFixed(2)
}

function fmtNum(val: number | null | undefined): string {
  if (val == null) return '-'
  return val.toLocaleString()
}

function formatDate(iso: string | null | undefined): string {
  if (!iso) return '-'
  const d = new Date(iso)
  const loc = locale.value === 'zh' ? 'zh-CN' : 'en-US'
  return d.toLocaleDateString(loc, { year: 'numeric', month: 'long', day: 'numeric' })
}

async function fetchUsage(key: string) {
  const dateParams = getDateParams()
  const url = '/v1/usage' + (dateParams ? '?' + dateParams : '')
  const res = await fetch(url, {
    headers: { 'Authorization': 'Bearer ' + key },
  })
  if (!res.ok) {
    const body = await res.json().catch(() => null)
    const msg = body?.error?.message || body?.message || `${t('keyUsage.queryFailed')} (${res.status})`
    throw new Error(msg)
  }
  return await res.json()
}

async function queryKey() {
  if (isQuerying.value) return
  const key = apiKey.value.trim()
  if (!key) {
    appStore.showInfo(t('keyUsage.enterApiKey'))
    return
  }

  isQuerying.value = true
  showResults.value = true
  showLoading.value = true
  resultData.value = null

  try {
    const data = await fetchUsage(key)
    resultData.value = data
    showLoading.value = false
    showDatePicker.value = true

    nextTick(() => {
      triggerRingAnimation(ringItems.value)
    })

    appStore.showSuccess(t('keyUsage.querySuccess'))
  } catch (err) {
    showResults.value = false
    showLoading.value = false
    appStore.showError((err as Error).message || t('keyUsage.queryFailedRetry'))
  } finally {
    isQuerying.value = false
  }
}

function initTheme() {
  const savedTheme = localStorage.getItem('theme')
  if (savedTheme === 'dark' || (!savedTheme && window.matchMedia('(prefers-color-scheme: dark)').matches)) {
    isDark.value = true
    document.documentElement.classList.add('dark')
  }
}

function formatResetTime(resetAt: string | null | undefined): string {
  if (!resetAt) return ''
  const diff = new Date(resetAt).getTime() - now.value.getTime()
  if (diff <= 0) return t('keyUsage.resetNow')
  const days = Math.floor(diff / 86400000)
  const hours = Math.floor((diff % 86400000) / 3600000)
  const mins = Math.floor((diff % 3600000) / 60000)
  if (days > 0) return `${days}d ${hours}h`
  if (hours > 0) return `${hours}h ${mins}m`
  return `${mins}m`
}

onMounted(() => {
  initTheme()
  if (!appStore.publicSettingsLoaded) {
    appStore.fetchPublicSettings()
  }
  resetTimer = setInterval(() => { now.value = new Date() }, 60000)
})

onUnmounted(() => {
  if (resetTimer) clearInterval(resetTimer)
})
</script>

<style scoped>
@import '@fontsource/noto-serif-sc/chinese-simplified-500.css';
@import '@fontsource/cormorant-garamond/latin-500.css';

.key-usage-page {
  position: relative;
  min-height: 100vh;
  overflow: hidden;
  background:
    radial-gradient(circle at 2% 0%, rgba(225, 203, 179, 0.34), transparent 28%),
    radial-gradient(circle at 92% 16%, rgba(189, 158, 122, 0.16), transparent 26%),
    #f6f1e8;
  color: #221a15;
}

.usage-texture {
  position: fixed;
  inset: 0;
  pointer-events: none;
  background-image:
    linear-gradient(rgba(31, 24, 20, 0.03) 1px, transparent 1px),
    linear-gradient(90deg, rgba(31, 24, 20, 0.024) 1px, transparent 1px);
  background-size: 92px 92px;
  mask-image: linear-gradient(180deg, transparent 0%, black 18%, black 74%, transparent 100%);
  opacity: 0.32;
}

.usage-header,
.usage-main,
.usage-footer {
  position: relative;
  z-index: 1;
}

.usage-header {
  padding: 1.1rem 1.4rem 0;
}

.usage-nav {
  display: flex;
  max-width: 1280px;
  margin: 0 auto;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  border: 1px solid rgba(44, 33, 25, 0.08);
  border-radius: 9999px;
  padding: 0.9rem 1.25rem;
  background: rgba(246, 241, 232, 0.56);
  box-shadow: 0 12px 40px rgba(67, 48, 34, 0.08);
  backdrop-filter: blur(18px) saturate(145%);
}

.usage-brand,
.usage-nav-actions,
.usage-nav-link,
.usage-nav-cta {
  display: inline-flex;
  align-items: center;
}

.usage-brand {
  gap: 0.75rem;
  color: inherit;
  text-decoration: none;
  font-size: 0.92rem;
  font-weight: 700;
}

.usage-logo {
  width: 2rem;
  height: 2rem;
  border-radius: 9999px;
  object-fit: contain;
  background: rgba(255, 255, 255, 0.68);
}

.usage-nav-actions {
  gap: 0.9rem;
}

.usage-nav-link,
.usage-footer a {
  color: rgba(31, 24, 20, 0.68);
  font-size: 0.92rem;
  text-decoration: none;
  transition: color 160ms ease;
}

.usage-nav-link:hover,
.usage-footer a:hover {
  color: #1f1814;
}

.usage-nav-cta,
.usage-query-btn {
  justify-content: center;
  border: 0;
  border-radius: 9999px;
  background: #271d17;
  color: #f8f3eb;
  font-weight: 700;
  text-decoration: none;
  box-shadow: 0 16px 34px rgba(30, 20, 14, 0.14);
  transition: transform 160ms ease, opacity 160ms ease;
}

.usage-nav-cta {
  min-width: 6.8rem;
  padding: 0.72rem 1.08rem;
}

.usage-nav-cta:hover,
.usage-query-btn:hover {
  transform: translateY(-1px);
}

.usage-main {
  max-width: 1280px;
  margin: 0 auto;
  padding: 7rem 1.4rem 5rem;
}

.usage-hero {
  display: grid;
  grid-template-columns: minmax(0, 1.08fr) minmax(22rem, 0.92fr);
  gap: clamp(2rem, 5vw, 5.5rem);
  align-items: center;
}

.usage-kicker {
  margin: 0 0 1.2rem;
  color: rgba(39, 29, 23, 0.54);
  font-size: 0.78rem;
  font-weight: 800;
  letter-spacing: 0.16em;
}

.usage-hero h1 {
  max-width: 44rem;
  margin: 0;
  font-family: 'Noto Serif SC', 'Source Han Serif SC', 'Songti SC', serif;
  font-size: clamp(3.6rem, 5.2vw, 5.35rem);
  font-weight: 500;
  line-height: 0.94;
  letter-spacing: 0;
  text-wrap: balance;
}

.usage-lede {
  max-width: 37rem;
  margin: 1.5rem 0 0;
  color: rgba(39, 29, 23, 0.7);
  font-size: 1.08rem;
  line-height: 1.9;
}

.usage-query-panel {
  max-width: 45rem;
  margin-top: 2.4rem;
  padding: 1rem;
  border: 1px solid rgba(44, 33, 25, 0.1);
  border-radius: 1.8rem;
  background: rgba(255, 252, 245, 0.62);
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.5), 0 20px 48px rgba(77, 55, 38, 0.08);
  backdrop-filter: blur(16px);
}

.usage-input-label {
  display: block;
  margin: 0 0 0.65rem 0.2rem;
  color: rgba(39, 29, 23, 0.58);
  font-size: 0.78rem;
  font-weight: 800;
  letter-spacing: 0.12em;
}

.usage-input-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 0.8rem;
}

.usage-input-wrap {
  position: relative;
}

.usage-input-icon,
.usage-eye svg {
  width: 1.1rem;
  height: 1.1rem;
}

.usage-input-icon {
  position: absolute;
  left: 1rem;
  top: 50%;
  color: rgba(39, 29, 23, 0.42);
  transform: translateY(-50%);
}

.usage-input {
  width: 100%;
  height: 3.35rem;
  border: 1px solid rgba(39, 29, 23, 0.1);
  border-radius: 9999px;
  padding: 0 3.2rem 0 2.9rem;
  background: rgba(255, 255, 255, 0.72);
  color: #221a15;
  font-size: 0.95rem;
  outline: none;
  transition: border-color 160ms ease, box-shadow 160ms ease, background-color 160ms ease;
}

.usage-input::placeholder {
  color: rgba(39, 29, 23, 0.36);
}

.usage-input:focus,
.usage-date-input:focus {
  border-color: rgba(124, 90, 59, 0.52);
  box-shadow: 0 0 0 4px rgba(124, 90, 59, 0.12);
}

.usage-eye {
  position: absolute;
  top: 50%;
  right: 1rem;
  display: inline-flex;
  width: 2rem;
  height: 2rem;
  align-items: center;
  justify-content: center;
  border: 0;
  background: transparent;
  color: rgba(39, 29, 23, 0.46);
  cursor: pointer;
  transform: translateY(-50%);
}

.usage-query-btn {
  display: inline-flex;
  min-width: 7rem;
  height: 3.35rem;
  align-items: center;
  gap: 0.5rem;
  padding: 0 1.35rem;
  cursor: pointer;
}

.usage-query-btn:disabled {
  cursor: not-allowed;
  opacity: 0.68;
  transform: none;
}

.usage-spinner {
  width: 0.9rem;
  height: 0.9rem;
  border: 2px solid rgba(255, 255, 255, 0.38);
  border-top-color: #fff;
  border-radius: 9999px;
  animation: spin 0.8s linear infinite;
}

.usage-privacy {
  margin: 0.8rem 0 0;
  color: rgba(39, 29, 23, 0.48);
  font-size: 0.82rem;
}

.usage-range,
.usage-custom-range {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.5rem;
}

.usage-range {
  margin-top: 1rem;
  padding-top: 1rem;
  border-top: 1px solid rgba(39, 29, 23, 0.08);
  color: rgba(39, 29, 23, 0.56);
  font-size: 0.82rem;
}

.usage-range-btn,
.usage-range-apply,
.usage-date-input {
  border: 1px solid rgba(39, 29, 23, 0.1);
  border-radius: 9999px;
  background: rgba(255, 255, 255, 0.58);
  color: #2d241e;
  font-size: 0.78rem;
}

.usage-range-btn,
.usage-range-apply {
  padding: 0.5rem 0.8rem;
  cursor: pointer;
}

.usage-range-btn.is-active,
.usage-range-apply {
  border-color: #2b211b;
  background: #2b211b;
  color: #f8f3eb;
}

.usage-date-input {
  min-height: 2rem;
  padding: 0.35rem 0.65rem;
  outline: none;
}

.usage-product-card {
  position: relative;
  overflow: hidden;
  border: 1px solid rgba(39, 29, 23, 0.11);
  border-radius: 2.4rem;
  padding: clamp(1.35rem, 3vw, 2rem);
  background:
    linear-gradient(145deg, rgba(255, 252, 245, 0.78), rgba(234, 220, 201, 0.42)),
    rgba(255, 255, 255, 0.55);
  box-shadow: 0 30px 70px rgba(76, 54, 38, 0.13);
  backdrop-filter: blur(20px) saturate(140%);
}

.usage-product-card::before {
  content: '';
  position: absolute;
  inset: -24% -16% auto auto;
  width: 18rem;
  height: 18rem;
  border-radius: 9999px;
  background: radial-gradient(circle, rgba(203, 167, 121, 0.24), transparent 62%);
}

.usage-card-bar,
.usage-endpoint,
.usage-code-lines,
.usage-metrics-preview,
.usage-route-flow {
  position: relative;
}

.usage-card-bar {
  display: flex;
  gap: 0.42rem;
  margin-bottom: 1.4rem;
}

.usage-card-bar span {
  width: 0.58rem;
  height: 0.58rem;
  border-radius: 9999px;
  background: rgba(39, 29, 23, 0.2);
}

.usage-endpoint {
  display: grid;
  gap: 0.45rem;
  padding-bottom: 1.2rem;
  border-bottom: 1px solid rgba(39, 29, 23, 0.08);
}

.usage-endpoint span,
.usage-code-lines span,
.usage-metrics-preview span {
  color: rgba(39, 29, 23, 0.5);
  font-size: 0.72rem;
  font-weight: 800;
  letter-spacing: 0.12em;
}

.usage-endpoint strong {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: clamp(1rem, 2vw, 1.25rem);
  font-weight: 700;
  word-break: break-all;
}

.usage-code-lines {
  display: grid;
  gap: 0.75rem;
  margin-top: 1.35rem;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 0.88rem;
}

.usage-code-lines p {
  display: flex;
  gap: 0.8rem;
  margin: 0;
  color: rgba(39, 29, 23, 0.76);
}

.usage-code-lines span {
  min-width: 3.2rem;
}

.usage-metrics-preview {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 0.75rem;
  margin-top: 1.6rem;
}

.usage-metrics-preview div {
  display: grid;
  gap: 0.35rem;
  border: 1px solid rgba(39, 29, 23, 0.08);
  border-radius: 1.1rem;
  padding: 0.85rem;
  background: rgba(255, 255, 255, 0.42);
}

.usage-metrics-preview strong {
  font-family: 'Cormorant Garamond', 'Times New Roman', serif;
  font-size: 1.55rem;
  font-weight: 500;
}

.usage-route-flow {
  display: flex;
  align-items: center;
  gap: 0.7rem;
  margin-top: 1.6rem;
  color: rgba(39, 29, 23, 0.58);
  font-size: 0.82rem;
}

.usage-route-flow span {
  white-space: nowrap;
}

.usage-route-flow i {
  flex: 1;
  min-width: 1.4rem;
  height: 1px;
  background: linear-gradient(90deg, rgba(39, 29, 23, 0.16), rgba(39, 29, 23, 0.04));
}

.usage-principles {
  display: grid;
  grid-template-columns: 1.2fr 1fr 1fr;
  gap: 1px;
  margin-top: 5rem;
  overflow: hidden;
  border: 1px solid rgba(39, 29, 23, 0.08);
  border-radius: 2rem;
  background: rgba(39, 29, 23, 0.08);
}

.usage-principle {
  min-height: 11rem;
  padding: 1.4rem;
  background: rgba(255, 252, 245, 0.58);
}

.usage-principle span {
  color: rgba(39, 29, 23, 0.36);
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 0.78rem;
}

.usage-principle strong {
  display: block;
  margin-top: 1.8rem;
  font-size: 1.15rem;
}

.usage-principle p {
  max-width: 22rem;
  margin: 0.7rem 0 0;
  color: rgba(39, 29, 23, 0.6);
  line-height: 1.7;
}

.usage-results {
  margin-top: 5rem;
}

.usage-loading-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 1rem;
}

.usage-skeleton-card {
  display: grid;
  justify-items: center;
  gap: 1.4rem;
  border: 1px solid rgba(39, 29, 23, 0.09);
  border-radius: 1.8rem;
  padding: 2rem;
  background: rgba(255, 252, 245, 0.6);
}

.usage-skeleton-wide {
  grid-column: 1 / -1;
  justify-items: stretch;
}

.usage-result-stack {
  display: grid;
  gap: 1.2rem;
}

.usage-status-pill {
  display: inline-flex;
  width: fit-content;
  align-items: center;
  gap: 0.65rem;
  border: 1px solid rgba(39, 29, 23, 0.08);
  border-radius: 9999px;
  padding: 0.7rem 1rem;
  background: rgba(255, 252, 245, 0.72);
  box-shadow: 0 16px 34px rgba(76, 54, 38, 0.08);
}

.usage-status-pill strong {
  font-size: 0.9rem;
}

.usage-status-pill em {
  color: rgba(39, 29, 23, 0.5);
  font-style: normal;
  font-size: 0.78rem;
}

.usage-ring-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 1rem;
}

.usage-ring-grid--one {
  grid-template-columns: minmax(0, 25rem);
}

.usage-ring-grid--two {
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.usage-ring-card,
.usage-detail-panel,
.usage-stats-panel,
.usage-model-panel {
  overflow: hidden;
  border: 1px solid rgba(39, 29, 23, 0.09);
  border-radius: 1.8rem;
  background: rgba(255, 252, 245, 0.68);
  box-shadow: 0 18px 46px rgba(76, 54, 38, 0.08);
  backdrop-filter: blur(14px);
}

.usage-ring-card {
  padding: 1.5rem;
}

.usage-ring-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
}

.usage-ring-head h3 {
  margin: 0;
  color: rgba(39, 29, 23, 0.58);
  font-size: 0.78rem;
  font-weight: 800;
  letter-spacing: 0.12em;
  text-transform: uppercase;
}

.usage-ring-head span {
  color: rgba(39, 29, 23, 0.38);
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 0.7rem;
}

.usage-ring-wrap {
  position: relative;
  width: 11rem;
  height: 11rem;
  margin: 1.2rem auto 0;
}

.usage-ring {
  width: 11rem;
  height: 11rem;
}

.usage-ring-center {
  position: absolute;
  inset: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  text-align: center;
}

.usage-ring-center strong {
  font-family: 'Cormorant Garamond', 'Times New Roman', serif;
  color: #221a15;
  font-size: 2.35rem;
  font-weight: 500;
  line-height: 1;
}

.usage-ring-center span,
.usage-ring-center small {
  color: rgba(39, 29, 23, 0.48);
  font-size: 0.72rem;
}

.usage-ring-center em {
  margin-top: 0.25rem;
  font-style: normal;
  font-size: 0.82rem;
  font-weight: 800;
}

.usage-panel-head {
  padding: 1.25rem 1.4rem;
  border-bottom: 1px solid rgba(39, 29, 23, 0.08);
}

.usage-panel-head span {
  color: rgba(39, 29, 23, 0.42);
  font-size: 0.72rem;
  font-weight: 800;
  letter-spacing: 0.14em;
}

.usage-panel-head h3 {
  margin: 0.2rem 0 0;
  font-size: 1.1rem;
}

.usage-detail-list {
  display: grid;
}

.usage-detail-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1.4rem;
  padding: 1rem 1.4rem;
  border-bottom: 1px solid rgba(39, 29, 23, 0.06);
}

.usage-detail-row:last-child {
  border-bottom: 0;
}

.usage-detail-row > div {
  display: inline-flex;
  min-width: 0;
  align-items: center;
  gap: 0.8rem;
  color: rgba(39, 29, 23, 0.68);
}

.usage-detail-icon {
  display: inline-flex;
  width: 2rem;
  height: 2rem;
  flex: 0 0 auto;
  align-items: center;
  justify-content: center;
  border-radius: 0.7rem;
  background: rgba(39, 29, 23, 0.06);
  color: #7c5a3b;
}

.usage-detail-icon svg {
  width: 1rem;
  height: 1rem;
}

.usage-detail-row strong,
.usage-stat-cell strong,
.usage-table-wrap td:not(:first-child) {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-variant-numeric: tabular-nums;
}

.usage-detail-row strong {
  text-align: right;
  font-size: 0.92rem;
}

.usage-stat-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  background: rgba(39, 29, 23, 0.06);
  gap: 1px;
}

.usage-stat-cell {
  display: grid;
  gap: 0.45rem;
  padding: 1rem 1.1rem;
  background: rgba(255, 252, 245, 0.72);
}

.usage-stat-cell span {
  color: rgba(39, 29, 23, 0.52);
  font-size: 0.78rem;
}

.usage-stat-cell strong {
  color: #221a15;
  font-size: 0.94rem;
}

.usage-table-wrap {
  overflow-x: auto;
}

.usage-table-wrap table {
  width: 100%;
  min-width: 58rem;
  border-collapse: collapse;
}

.usage-table-wrap th,
.usage-table-wrap td {
  padding: 0.92rem 1rem;
  border-bottom: 1px solid rgba(39, 29, 23, 0.06);
  text-align: right;
  white-space: nowrap;
}

.usage-table-wrap th:first-child,
.usage-table-wrap td:first-child {
  text-align: left;
}

.usage-table-wrap th {
  color: rgba(39, 29, 23, 0.48);
  font-size: 0.72rem;
  font-weight: 800;
  letter-spacing: 0.08em;
}

.usage-table-wrap td {
  color: rgba(39, 29, 23, 0.72);
  font-size: 0.86rem;
}

.usage-footer {
  display: flex;
  max-width: 1280px;
  margin: 0 auto;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  border-top: 1px solid rgba(31, 24, 20, 0.08);
  padding: 1.5rem 1.4rem 2rem;
  color: rgba(31, 24, 20, 0.52);
  font-size: 0.92rem;
}

.progress-ring {
  transition: stroke-dashoffset 1.2s cubic-bezier(0.4, 0, 0.2, 1);
  transform: rotate(-90deg);
  transform-origin: 50% 50%;
}

@keyframes shimmer-kv {
  0% {
    background-position: -200% 0;
  }
  100% {
    background-position: 200% 0;
  }
}

.skeleton {
  border-radius: 0.7rem;
  background: linear-gradient(90deg, rgba(216, 202, 183, 0.58) 25%, rgba(246, 241, 232, 0.8) 50%, rgba(216, 202, 183, 0.58) 75%);
  background-size: 200% 100%;
  animation: shimmer-kv 1.8s ease-in-out infinite;
}

@keyframes fade-up-kv {
  from {
    opacity: 0;
    transform: translateY(18px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.fade-up {
  animation: fade-up-kv 0.62s cubic-bezier(0.16, 1, 0.3, 1) forwards;
}

.fade-up-delay-1 {
  animation-delay: 0.1s;
  opacity: 0;
}

.fade-up-delay-2 {
  animation-delay: 0.18s;
  opacity: 0;
}

.fade-up-delay-3 {
  animation-delay: 0.26s;
  opacity: 0;
}

.fade-up-delay-4 {
  animation-delay: 0.34s;
  opacity: 0;
}

@keyframes pulse-dot-kv {
  0%,
  100% {
    box-shadow: 0 0 0 0 currentColor;
    opacity: 1;
  }
  50% {
    box-shadow: 0 0 0 8px transparent;
    opacity: 0.72;
  }
}

.pulse-dot {
  width: 0.55rem;
  height: 0.55rem;
  border-radius: 9999px;
  animation: pulse-dot-kv 2.2s ease-in-out infinite;
}

.pulse-dot.is-ok {
  background: #557b5f;
  color: rgba(85, 123, 95, 0.34);
}

.pulse-dot.is-stop {
  background: #b24d4d;
  color: rgba(178, 77, 77, 0.32);
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

:deep(.dark) .key-usage-page {
  background:
    radial-gradient(circle at 2% 0%, rgba(98, 73, 54, 0.24), transparent 28%),
    radial-gradient(circle at 92% 16%, rgba(122, 92, 65, 0.16), transparent 26%),
    #100d0b;
  color: #f1e8dc;
}

:deep(.dark) .usage-nav,
:deep(.dark) .usage-query-panel,
:deep(.dark) .usage-product-card,
:deep(.dark) .usage-principle,
:deep(.dark) .usage-ring-card,
:deep(.dark) .usage-detail-panel,
:deep(.dark) .usage-stats-panel,
:deep(.dark) .usage-model-panel,
:deep(.dark) .usage-status-pill,
:deep(.dark) .usage-skeleton-card {
  border-color: rgba(255, 243, 230, 0.1);
  background: rgba(26, 21, 18, 0.72);
}

:deep(.dark) .usage-input,
:deep(.dark) .usage-range-btn,
:deep(.dark) .usage-date-input,
:deep(.dark) .usage-metrics-preview div,
:deep(.dark) .usage-stat-cell {
  border-color: rgba(255, 243, 230, 0.1);
  background: rgba(255, 255, 255, 0.06);
  color: #f1e8dc;
}

:deep(.dark) .usage-hero h1,
:deep(.dark) .usage-ring-center strong,
:deep(.dark) .usage-stat-cell strong {
  color: #f1e8dc;
}

:deep(.dark) .usage-lede,
:deep(.dark) .usage-privacy,
:deep(.dark) .usage-principle p,
:deep(.dark) .usage-detail-row > div,
:deep(.dark) .usage-table-wrap td,
:deep(.dark) .usage-nav-link,
:deep(.dark) .usage-footer,
:deep(.dark) .usage-footer a {
  color: rgba(241, 232, 220, 0.66);
}

@media (max-width: 980px) {
  .usage-main {
    padding-top: 4.8rem;
  }

  .usage-hero,
  .usage-principles {
    grid-template-columns: 1fr;
  }

  .usage-product-card {
    order: -1;
  }

  .usage-ring-grid,
  .usage-ring-grid--two {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .usage-stat-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 680px) {
  .usage-header {
    padding: 0.85rem 0.85rem 0;
  }

  .usage-nav {
    border-radius: 1.4rem;
    padding: 0.8rem;
  }

  .usage-nav-actions {
    gap: 0.55rem;
  }

  .usage-nav-link {
    display: none;
  }

  .usage-main {
    padding: 3.8rem 1rem 4rem;
  }

  .usage-hero h1 {
    font-size: clamp(3rem, 17vw, 4.8rem);
  }

  .usage-input-row {
    grid-template-columns: 1fr;
  }

  .usage-query-btn {
    width: 100%;
  }

  .usage-product-card {
    border-radius: 1.7rem;
  }

  .usage-metrics-preview,
  .usage-loading-grid,
  .usage-ring-grid,
  .usage-ring-grid--one,
  .usage-ring-grid--two,
  .usage-stat-grid {
    grid-template-columns: 1fr;
  }

  .usage-detail-row {
    align-items: flex-start;
    flex-direction: column;
  }

  .usage-detail-row strong {
    text-align: left;
  }

  .usage-footer {
    flex-direction: column;
    align-items: flex-start;
  }
}

@media (prefers-reduced-motion: reduce) {
  .fade-up,
  .pulse-dot,
  .skeleton,
  .progress-ring {
    animation: none;
    transition: none;
  }
}
</style>
