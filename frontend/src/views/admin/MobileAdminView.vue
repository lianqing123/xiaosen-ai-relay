<template>
  <div class="mobile-admin-shell">
    <header class="mobile-admin-header">
      <div>
        <p class="mobile-admin-kicker">{{ siteName }} · iOS</p>
        <h1>手机控制台</h1>
      </div>
      <button class="icon-action" type="button" :disabled="loading" aria-label="刷新" @click="loadOverview">
        <Icon name="refresh" size="md" :class="{ 'spin-once': loading }" />
      </button>
    </header>

    <section v-if="showInstallHint" class="install-card">
      <div>
        <p class="install-title">安装到 iPhone</p>
        <p class="install-copy">在 Safari 分享菜单里选择“添加到主屏幕”，下次可直接从桌面进入后台。</p>
      </div>
      <button class="install-close" type="button" aria-label="关闭安装提示" @click="dismissInstallHint">
        <Icon name="x" size="sm" />
      </button>
    </section>

    <main class="mobile-admin-main">
      <section id="overview" class="hero-panel">
        <div class="hero-copy">
          <span class="live-pill">
            <span class="live-dot"></span>
            {{ realtime ? '实时在线' : '等待数据' }}
          </span>
          <h2>{{ formatNumber(stats?.today_requests) }}</h2>
          <p>今日请求 · 当前 {{ formatNumber(realtime?.active_requests) }} 个活跃请求</p>
        </div>
        <div class="hero-stack" aria-label="近段请求趋势">
          <span
            v-for="(bar, index) in trendBars"
            :key="`${bar.label}-${index}`"
            class="trend-bar"
            :style="{ height: bar.height }"
          ></span>
        </div>
      </section>

      <section class="metric-grid" aria-label="核心指标">
        <article v-for="metric in coreMetrics" :key="metric.label" class="metric-card">
          <div class="metric-icon">
            <Icon :name="metric.icon" size="sm" />
          </div>
          <p>{{ metric.label }}</p>
          <strong>{{ metric.value }}</strong>
          <span>{{ metric.hint }}</span>
        </article>
      </section>

      <section class="quick-panel" aria-label="快捷操作">
        <div class="section-head">
          <div>
            <p class="section-kicker">Actions</p>
            <h2>快捷操作</h2>
          </div>
          <span class="version-chip">v{{ version || '-' }}</span>
        </div>
        <div class="quick-grid">
          <button v-for="action in quickActions" :key="action.path" type="button" @click="go(action.path)">
            <Icon :name="action.icon" size="md" />
            <span>{{ action.label }}</span>
            <small>{{ action.hint }}</small>
          </button>
        </div>
      </section>

      <section id="users" class="data-panel">
        <div class="section-head">
          <div>
            <p class="section-kicker">Users</p>
            <h2>用户快查</h2>
          </div>
          <button class="text-action" type="button" @click="go('/admin/users')">完整管理</button>
        </div>
        <form class="search-row" @submit.prevent="searchUsers">
          <Icon name="search" size="sm" />
          <input v-model.trim="userSearch" type="search" placeholder="邮箱 / 用户名 / ID" />
          <button type="submit" :disabled="usersLoading">查找</button>
        </form>
        <div v-if="users.length" class="compact-list">
          <button v-for="user in users" :key="user.id" type="button" class="user-row" @click="go('/admin/users')">
            <span class="avatar-mark">{{ getUserInitial(user) }}</span>
            <span class="row-main">
              <strong>{{ user.email || user.username || `用户 ${user.id}` }}</strong>
              <small>ID {{ user.id }} · {{ user.status === 'active' ? '正常' : '禁用' }}</small>
            </span>
            <span class="row-value">{{ formatMoney(user.balance) }}</span>
          </button>
        </div>
        <div v-else class="empty-line">{{ usersLoading ? '正在查询用户' : '暂无匹配用户' }}</div>
      </section>

      <section id="channels" class="data-panel">
        <div class="section-head">
          <div>
            <p class="section-kicker">Channels</p>
            <h2>通道状态</h2>
          </div>
          <button class="text-action" type="button" @click="go('/admin/channels/monitor')">查看全部</button>
        </div>
        <div v-if="monitors.length" class="monitor-list">
          <article v-for="monitor in monitors" :key="monitor.id" class="monitor-card">
            <div>
              <strong>{{ monitor.name }}</strong>
              <small>{{ monitor.primary_model }} · {{ monitor.provider }}</small>
            </div>
            <div class="monitor-side">
              <span :class="['status-pill', monitorStatusClass(monitor.primary_status)]">
                {{ monitorStatusLabel(monitor.primary_status) }}
              </span>
              <small>{{ monitor.primary_latency_ms ?? '-' }} ms</small>
            </div>
          </article>
        </div>
        <div v-else class="empty-line">暂无通道监控数据</div>
      </section>

      <section id="finance" class="data-panel cleanup-panel">
        <div class="section-head">
          <div>
            <p class="section-kicker">Finance</p>
            <h2>余额与返利</h2>
          </div>
          <button class="text-action" type="button" @click="go('/admin/orders/dashboard')">支付概览</button>
        </div>
        <div class="cleanup-card">
          <div>
            <span class="cleanup-label">闲置余额</span>
            <strong>{{ formatMoney(cleanupPreview?.total_balance) }}</strong>
            <small>{{ formatNumber(cleanupPreview?.count) }} 个候选用户 · 仅预览未清理</small>
          </div>
          <button type="button" @click="go('/admin/users')">处理</button>
        </div>
        <div class="finance-grid">
          <span>
            <small>今日实际扣费</small>
            <strong>{{ formatMoney(stats?.today_actual_cost) }}</strong>
          </span>
          <span>
            <small>今日账号成本</small>
            <strong>{{ formatMoney(stats?.today_account_cost) }}</strong>
          </span>
        </div>
      </section>

      <section id="system" class="data-panel system-panel">
        <div class="section-head">
          <div>
            <p class="section-kicker">System</p>
            <h2>系统快照</h2>
          </div>
          <button class="text-action" type="button" @click="go('/admin/settings')">设置</button>
        </div>
        <div class="system-lines">
          <span><b>{{ formatNumber(stats?.total_users) }}</b><small>总用户</small></span>
          <span><b>{{ formatNumber(stats?.normal_accounts) }}</b><small>正常账号</small></span>
          <span><b>{{ formatDuration(stats?.uptime) }}</b><small>运行时间</small></span>
        </div>
      </section>

      <p v-if="overviewError" class="error-strip">{{ overviewError }}</p>
    </main>

    <nav class="bottom-tabs" aria-label="手机后台导航">
      <button v-for="tab in tabs" :key="tab.id" type="button" @click="scrollToSection(tab.id)">
        <Icon :name="tab.icon" size="sm" />
        <span>{{ tab.label }}</span>
      </button>
    </nav>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { adminAPI } from '@/api/admin'
import type { DashboardSnapshotV2Stats } from '@/api/admin/dashboard'
import type { ChannelMonitor, MonitorStatus } from '@/api/admin/channelMonitor'
import type { InactiveBalanceCleanupPreview } from '@/api/admin/users'
import type { AdminUser, TrendDataPoint } from '@/types'
import Icon from '@/components/icons/Icon.vue'
import { useAppStore } from '@/stores/app'
import {
  getMobileAdminInstallState,
  type MobileAdminInstallState
} from '@/utils/mobileAdminPwa'

type IconName = InstanceType<typeof Icon>['$props']['name']

const router = useRouter()
const appStore = useAppStore()

const loading = ref(false)
const usersLoading = ref(false)
const overviewError = ref('')
const userSearch = ref('')
const stats = ref<DashboardSnapshotV2Stats | null>(null)
const trend = ref<TrendDataPoint[]>([])
const realtime = ref<{
  active_requests: number
  requests_per_minute: number
  average_response_time: number
  error_rate: number
} | null>(null)
const cleanupPreview = ref<InactiveBalanceCleanupPreview | null>(null)
const users = ref<AdminUser[]>([])
const monitors = ref<ChannelMonitor[]>([])
const version = ref('')
const installState = ref<MobileAdminInstallState>({
  isIos: false,
  isStandalone: false,
  canShowInstallHint: false
})
const installHintDismissed = ref(false)

const siteName = computed(() => appStore.siteName || 'Codex Access')
const showInstallHint = computed(() => installState.value.canShowInstallHint && !installHintDismissed.value)

const cacheRate = computed(() => {
  const source = stats.value
  if (!source) return 0
  const cacheTokens = source.today_cache_read_tokens || 0
  const billableContext =
    (source.today_input_tokens || 0) + (source.today_cache_creation_tokens || 0) + cacheTokens
  if (billableContext <= 0) return 0
  return cacheTokens / billableContext
})

const coreMetrics = computed<Array<{ label: string; value: string; hint: string; icon: IconName }>>(() => [
  {
    label: '实际扣费',
    value: formatMoney(stats.value?.today_actual_cost),
    hint: '今日',
    icon: 'creditCard'
  },
  {
    label: '缓存率',
    value: formatPercent(cacheRate.value),
    hint: '今日上下文',
    icon: 'database'
  },
  {
    label: '异常账号',
    value: formatNumber(stats.value?.error_accounts),
    hint: `${formatNumber(stats.value?.ratelimit_accounts)} 个限流`,
    icon: 'exclamationTriangle'
  },
  {
    label: 'RPM',
    value: formatNumber(Math.round(stats.value?.rpm || realtime.value?.requests_per_minute || 0)),
    hint: `${formatLatency(realtime.value?.average_response_time || stats.value?.average_duration_ms)} 平均`,
    icon: 'bolt'
  }
])

const trendBars = computed(() => {
  const points = trend.value.slice(-12)
  const max = Math.max(...points.map((point) => point.requests), 1)
  if (!points.length) {
    return Array.from({ length: 12 }, (_, index) => ({
      label: `empty-${index}`,
      height: `${12 + (index % 4) * 8}%`
    }))
  }
  return points.map((point) => ({
    label: point.date,
    height: `${Math.max(10, Math.round((point.requests / max) * 100))}%`
  }))
})

const quickActions: Array<{ label: string; hint: string; path: string; icon: IconName }> = [
  { label: '用户', hint: '余额与状态', path: '/admin/users', icon: 'users' },
  { label: '订阅', hint: '配额重置', path: '/admin/subscriptions', icon: 'creditCard' },
  { label: '账号', hint: '上游账号', path: '/admin/accounts', icon: 'server' },
  { label: '通道', hint: '健康监控', path: '/admin/channels/monitor', icon: 'server' },
  { label: '订单', hint: '充值订阅', path: '/admin/orders/dashboard', icon: 'creditCard' },
  { label: '用量', hint: '日志统计', path: '/admin/usage', icon: 'chartBar' },
  { label: '设置', hint: '系统配置', path: '/admin/settings', icon: 'cog' }
]

const tabs: Array<{ id: string; label: string; icon: IconName }> = [
  { id: 'overview', label: '概览', icon: 'home' },
  { id: 'users', label: '用户', icon: 'users' },
  { id: 'channels', label: '通道', icon: 'server' },
  { id: 'finance', label: '财务', icon: 'creditCard' },
  { id: 'system', label: '系统', icon: 'cog' }
]

async function loadOverview() {
  loading.value = true
  overviewError.value = ''

  const [snapshotResult, realtimeResult, cleanupResult, userResult, monitorResult, versionResult] =
    await Promise.allSettled([
      adminAPI.dashboard.getSnapshotV2({
        include_stats: true,
        include_trend: true,
        include_model_stats: true,
        granularity: 'hour'
      }),
      adminAPI.dashboard.getRealtimeMetrics(),
      adminAPI.users.previewInactiveBalanceCleanup(3, 5),
      adminAPI.users.list(1, 6, {
        include_subscriptions: true,
        sort_by: 'created_at',
        sort_order: 'desc'
      }),
      adminAPI.channelMonitor.list({ page: 1, page_size: 5 }),
      adminAPI.system.getVersion()
    ])

  if (snapshotResult.status === 'fulfilled') {
    stats.value = snapshotResult.value.stats || null
    trend.value = snapshotResult.value.trend || []
  }
  if (realtimeResult.status === 'fulfilled') realtime.value = realtimeResult.value
  if (cleanupResult.status === 'fulfilled') cleanupPreview.value = cleanupResult.value
  if (userResult.status === 'fulfilled') users.value = userResult.value.items || []
  if (monitorResult.status === 'fulfilled') monitors.value = monitorResult.value.items || []
  if (versionResult.status === 'fulfilled') version.value = versionResult.value.version

  if ([snapshotResult, realtimeResult, cleanupResult].some((result) => result.status === 'rejected')) {
    overviewError.value = '部分数据暂时不可用，请稍后刷新。'
    appStore.showError?.('手机后台部分数据加载失败')
  }

  loading.value = false
}

async function searchUsers() {
  usersLoading.value = true
  try {
    const response = await adminAPI.users.list(1, 6, {
      search: userSearch.value || undefined,
      include_subscriptions: true,
      sort_by: userSearch.value ? undefined : 'created_at',
      sort_order: 'desc'
    })
    users.value = response.items || []
  } catch {
    overviewError.value = '用户查询失败，请稍后重试。'
  } finally {
    usersLoading.value = false
  }
}

function go(path: string) {
  router.push(path)
}

function scrollToSection(id: string) {
  document.getElementById(id)?.scrollIntoView({ behavior: 'smooth', block: 'start' })
}

function dismissInstallHint() {
  installHintDismissed.value = true
  localStorage.setItem('mobile_admin_install_hint_dismissed', '1')
}

function getUserInitial(user: AdminUser): string {
  const value = user.username || user.email || String(user.id)
  return value.trim().slice(0, 1).toUpperCase()
}

function formatNumber(value: number | null | undefined): string {
  if (value === null || value === undefined || Number.isNaN(value)) return '0'
  return new Intl.NumberFormat('zh-CN', { maximumFractionDigits: 1 }).format(value)
}

function formatMoney(value: number | null | undefined): string {
  if (value === null || value === undefined || Number.isNaN(value)) return '¥0.00'
  return `¥${new Intl.NumberFormat('zh-CN', {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2
  }).format(value)}`
}

function formatPercent(value: number | null | undefined): string {
  if (value === null || value === undefined || Number.isNaN(value)) return '0%'
  const normalized = value <= 1 ? value * 100 : value
  return `${new Intl.NumberFormat('zh-CN', { maximumFractionDigits: 1 }).format(normalized)}%`
}

function formatLatency(value: number | null | undefined): string {
  if (!value || Number.isNaN(value)) return '0 ms'
  return `${Math.round(value)} ms`
}

function formatDuration(seconds: number | null | undefined): string {
  if (!seconds || Number.isNaN(seconds)) return '0h'
  const days = Math.floor(seconds / 86400)
  const hours = Math.floor((seconds % 86400) / 3600)
  if (days > 0) return `${days}d ${hours}h`
  return `${hours}h`
}

function monitorStatusLabel(status: MonitorStatus | ''): string {
  if (status === 'operational') return '正常'
  if (status === 'degraded') return '波动'
  if (status === 'failed' || status === 'error') return '异常'
  return '未知'
}

function monitorStatusClass(status: MonitorStatus | ''): string {
  if (status === 'operational') return 'status-ok'
  if (status === 'degraded') return 'status-warn'
  if (status === 'failed' || status === 'error') return 'status-bad'
  return 'status-muted'
}

onMounted(() => {
  installState.value = getMobileAdminInstallState()
  installHintDismissed.value = localStorage.getItem('mobile_admin_install_hint_dismissed') === '1'
  loadOverview()
})
</script>

<style scoped>
.mobile-admin-shell {
  min-height: 100dvh;
  padding: calc(18px + env(safe-area-inset-top)) 16px calc(96px + env(safe-area-inset-bottom));
  color: #17322f;
  background:
    linear-gradient(135deg, rgba(255, 255, 255, 0.82) 0%, rgba(250, 246, 237, 0.94) 48%, rgba(239, 249, 246, 0.9) 100%),
    repeating-linear-gradient(90deg, rgba(15, 118, 110, 0.05) 0 1px, transparent 1px 28px);
}

.mobile-admin-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  margin: 0 auto 14px;
  max-width: 720px;
}

.mobile-admin-kicker,
.section-kicker {
  margin: 0 0 4px;
  color: #0f766e;
  font-size: 11px;
  font-weight: 800;
  letter-spacing: 0.12em;
  text-transform: uppercase;
}

.mobile-admin-header h1,
.section-head h2 {
  margin: 0;
  color: #12201e;
  font-weight: 850;
  letter-spacing: 0;
}

.mobile-admin-header h1 {
  font-size: 28px;
  line-height: 1.08;
}

.icon-action {
  display: inline-flex;
  height: 44px;
  width: 44px;
  align-items: center;
  justify-content: center;
  border: 1px solid rgba(15, 118, 110, 0.18);
  border-radius: 18px;
  background: rgba(255, 255, 255, 0.78);
  color: #0f766e;
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.8), 0 14px 28px rgba(27, 51, 44, 0.08);
  transition: transform 180ms ease, border-color 180ms ease;
}

.icon-action:active,
.quick-grid button:active,
.cleanup-card button:active,
.text-action:active,
.bottom-tabs button:active {
  transform: scale(0.97);
}

.spin-once {
  animation: spinOnce 900ms linear infinite;
}

.install-card,
.hero-panel,
.quick-panel,
.data-panel,
.metric-card {
  border: 1px solid rgba(15, 118, 110, 0.14);
  background: rgba(255, 255, 255, 0.74);
  box-shadow: 0 18px 45px rgba(21, 50, 43, 0.08);
  backdrop-filter: blur(16px);
}

.install-card {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 14px;
  max-width: 720px;
  margin: 0 auto 14px;
  padding: 14px;
  border-radius: 24px;
  animation: liftIn 420ms ease both;
}

.install-title {
  margin: 0 0 3px;
  font-size: 14px;
  font-weight: 800;
}

.install-copy {
  margin: 0;
  color: #57716d;
  font-size: 12px;
  line-height: 1.55;
}

.install-close {
  display: inline-flex;
  height: 30px;
  width: 30px;
  flex: 0 0 auto;
  align-items: center;
  justify-content: center;
  border: 0;
  border-radius: 999px;
  background: rgba(15, 118, 110, 0.08);
  color: #0f766e;
}

.mobile-admin-main {
  display: grid;
  gap: 14px;
  max-width: 720px;
  margin: 0 auto;
}

.hero-panel {
  display: grid;
  grid-template-columns: minmax(0, 1.4fr) minmax(104px, 0.8fr);
  gap: 16px;
  overflow: hidden;
  min-height: 176px;
  padding: 20px;
  border-radius: 30px;
  background:
    linear-gradient(135deg, rgba(255, 255, 255, 0.86), rgba(245, 251, 248, 0.74)),
    linear-gradient(160deg, rgba(245, 158, 11, 0.14), rgba(20, 184, 166, 0.1));
  animation: liftIn 380ms ease both;
}

.hero-copy {
  display: flex;
  min-width: 0;
  flex-direction: column;
  justify-content: space-between;
}

.live-pill {
  display: inline-flex;
  width: fit-content;
  align-items: center;
  gap: 7px;
  border: 1px solid rgba(15, 118, 110, 0.18);
  border-radius: 999px;
  padding: 7px 10px;
  color: #0f766e;
  font-size: 12px;
  font-weight: 750;
  background: rgba(240, 253, 250, 0.76);
}

.live-dot {
  position: relative;
  height: 7px;
  width: 7px;
  border-radius: 999px;
  background: #0d9488;
}

.live-dot::after {
  content: '';
  position: absolute;
  inset: -5px;
  border-radius: inherit;
  border: 1px solid rgba(13, 148, 136, 0.26);
  animation: breathe 1.8s ease-in-out infinite;
}

.hero-copy h2 {
  margin: 18px 0 6px;
  font-size: 46px;
  line-height: 0.95;
  font-weight: 900;
  letter-spacing: 0;
}

.hero-copy p {
  margin: 0;
  color: #607a76;
  font-size: 13px;
}

.hero-stack {
  display: flex;
  align-items: flex-end;
  justify-content: flex-end;
  gap: 5px;
  min-height: 118px;
  padding: 10px 0 2px;
}

.trend-bar {
  width: 8px;
  min-height: 10px;
  border-radius: 999px;
  background: linear-gradient(180deg, #f59e0b, #0d9488);
  opacity: 0.82;
  animation: riseBar 680ms cubic-bezier(0.16, 1, 0.3, 1) both;
}

.metric-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
}

.metric-card {
  min-height: 132px;
  border-radius: 24px;
  padding: 14px;
  animation: liftIn 420ms ease both;
}

.metric-icon {
  display: inline-flex;
  height: 34px;
  width: 34px;
  align-items: center;
  justify-content: center;
  border-radius: 14px;
  color: #0f766e;
  background: rgba(15, 118, 110, 0.08);
}

.metric-card p,
.metric-card span {
  margin: 10px 0 0;
  color: #6a807b;
  font-size: 12px;
}

.metric-card strong {
  display: block;
  margin-top: 5px;
  color: #12201e;
  font-size: 24px;
  font-weight: 880;
  line-height: 1;
}

.quick-panel,
.data-panel {
  border-radius: 28px;
  padding: 16px;
  animation: liftIn 460ms ease both;
}

.section-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
  margin-bottom: 14px;
}

.section-head h2 {
  font-size: 18px;
}

.version-chip,
.text-action {
  border: 1px solid rgba(15, 118, 110, 0.14);
  border-radius: 999px;
  padding: 8px 11px;
  color: #0f766e;
  font-size: 12px;
  font-weight: 750;
  background: rgba(240, 253, 250, 0.72);
}

.text-action {
  border: 0;
}

.quick-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 9px;
}

.quick-grid button {
  display: flex;
  min-height: 92px;
  min-width: 0;
  flex-direction: column;
  align-items: flex-start;
  justify-content: space-between;
  border: 1px solid rgba(15, 118, 110, 0.12);
  border-radius: 22px;
  padding: 12px;
  color: #12312d;
  background: rgba(255, 255, 255, 0.64);
  text-align: left;
  transition: transform 180ms ease, background 180ms ease;
}

.quick-grid button svg {
  color: #0f766e;
}

.quick-grid span {
  margin-top: 8px;
  font-size: 13px;
  font-weight: 800;
}

.quick-grid small {
  color: #71837f;
  font-size: 11px;
}

.search-row {
  display: grid;
  grid-template-columns: 18px minmax(0, 1fr) auto;
  align-items: center;
  gap: 8px;
  border: 1px solid rgba(15, 118, 110, 0.15);
  border-radius: 20px;
  padding: 8px 8px 8px 12px;
  background: rgba(255, 255, 255, 0.66);
  color: #0f766e;
}

.search-row input {
  min-width: 0;
  border: 0;
  outline: 0;
  color: #152925;
  background: transparent;
  font-size: 14px;
}

.search-row button,
.cleanup-card button {
  border: 0;
  border-radius: 15px;
  padding: 9px 12px;
  color: #fff;
  font-size: 12px;
  font-weight: 800;
  background: #0f766e;
}

.compact-list,
.monitor-list,
.system-lines,
.finance-grid {
  display: grid;
  gap: 10px;
  margin-top: 12px;
}

.user-row,
.monitor-card,
.cleanup-card,
.finance-grid span,
.system-lines span {
  border: 1px solid rgba(15, 118, 110, 0.11);
  border-radius: 20px;
  background: rgba(255, 255, 255, 0.62);
}

.user-row {
  display: grid;
  grid-template-columns: 40px minmax(0, 1fr) auto;
  align-items: center;
  gap: 10px;
  width: 100%;
  padding: 10px;
  text-align: left;
}

.avatar-mark {
  display: inline-flex;
  height: 40px;
  width: 40px;
  align-items: center;
  justify-content: center;
  border-radius: 16px;
  color: #0f766e;
  font-size: 15px;
  font-weight: 900;
  background: rgba(15, 118, 110, 0.1);
}

.row-main {
  min-width: 0;
}

.row-main strong,
.monitor-card strong {
  display: block;
  overflow: hidden;
  color: #142522;
  font-size: 13px;
  font-weight: 800;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.row-main small,
.monitor-card small,
.cleanup-card small,
.finance-grid small,
.system-lines small {
  color: #71837f;
  font-size: 11px;
}

.row-value {
  color: #0f766e;
  font-size: 13px;
  font-weight: 850;
}

.monitor-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 12px;
}

.monitor-side {
  display: grid;
  justify-items: end;
  gap: 4px;
}

.status-pill {
  border-radius: 999px;
  padding: 5px 8px;
  font-size: 11px;
  font-weight: 850;
}

.status-ok {
  color: #047857;
  background: rgba(16, 185, 129, 0.12);
}

.status-warn {
  color: #a16207;
  background: rgba(245, 158, 11, 0.15);
}

.status-bad {
  color: #b91c1c;
  background: rgba(239, 68, 68, 0.12);
}

.status-muted {
  color: #64748b;
  background: rgba(100, 116, 139, 0.12);
}

.cleanup-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 14px;
}

.cleanup-label {
  display: block;
  color: #0f766e;
  font-size: 12px;
  font-weight: 800;
}

.cleanup-card strong {
  display: block;
  margin: 4px 0;
  font-size: 26px;
  font-weight: 900;
}

.finance-grid {
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.finance-grid span,
.system-lines span {
  padding: 12px;
}

.finance-grid strong,
.system-lines b {
  display: block;
  margin-top: 4px;
  color: #132522;
  font-size: 17px;
}

.system-lines {
  grid-template-columns: repeat(3, minmax(0, 1fr));
}

.system-lines span {
  min-width: 0;
}

.system-lines b {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.empty-line,
.error-strip {
  margin: 12px 0 0;
  border-radius: 18px;
  padding: 14px;
  color: #6b7f7b;
  font-size: 13px;
  background: rgba(15, 118, 110, 0.06);
}

.error-strip {
  color: #b45309;
  background: rgba(245, 158, 11, 0.12);
}

.bottom-tabs {
  position: fixed;
  right: 14px;
  bottom: calc(10px + env(safe-area-inset-bottom));
  left: 14px;
  z-index: 20;
  display: grid;
  grid-template-columns: repeat(5, minmax(0, 1fr));
  gap: 4px;
  max-width: 720px;
  margin: 0 auto;
  border: 1px solid rgba(15, 118, 110, 0.14);
  border-radius: 24px;
  padding: 7px;
  background: rgba(255, 255, 255, 0.82);
  box-shadow: 0 18px 36px rgba(16, 44, 39, 0.16);
  backdrop-filter: blur(18px);
}

.bottom-tabs button {
  display: flex;
  min-width: 0;
  flex-direction: column;
  align-items: center;
  gap: 3px;
  border: 0;
  border-radius: 17px;
  padding: 8px 4px;
  color: #0f766e;
  font-size: 10px;
  font-weight: 750;
  background: transparent;
}

.bottom-tabs button:first-child {
  background: rgba(15, 118, 110, 0.1);
}

@keyframes liftIn {
  from {
    opacity: 0;
    transform: translateY(12px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

@keyframes breathe {
  0%,
  100% {
    opacity: 0;
    transform: scale(0.72);
  }
  45% {
    opacity: 1;
    transform: scale(1.08);
  }
}

@keyframes riseBar {
  from {
    opacity: 0;
    transform: scaleY(0.25);
    transform-origin: bottom;
  }
  to {
    opacity: 0.82;
    transform: scaleY(1);
    transform-origin: bottom;
  }
}

@keyframes spinOnce {
  to {
    transform: rotate(360deg);
  }
}

@media (min-width: 700px) {
  .mobile-admin-shell {
    padding-right: 24px;
    padding-left: 24px;
  }

  .metric-grid {
    grid-template-columns: repeat(4, minmax(0, 1fr));
  }
}
</style>
