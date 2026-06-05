<template>
  <AppLayout>
    <div class="kiro-status-page">
      <header class="kiro-status-hero">
        <div>
          <p class="kiro-status-kicker">Kiro Gateway</p>
          <h1>Kiro 状态总览</h1>
          <div class="kiro-status-meta">
            <span :class="['kiro-status-chip', gatewayStatusClass]">
              <span class="kiro-status-dot"></span>
              {{ gatewayStatusText }}
            </span>
            <span class="kiro-status-path">最后同步 {{ quotaFetchedAt }}</span>
          </div>
        </div>
        <div class="kiro-status-actions">
          <RouterLink to="/admin/channels/kiro" class="kiro-status-link">
            <Icon name="cog" size="sm" />
            <span>Kiro 接入配置</span>
          </RouterLink>
          <button type="button" class="kiro-status-button" :disabled="loading" @click="loadAll">
            <Icon name="refresh" size="sm" :class="{ 'kiro-status-spin': loading }" />
            <span>刷新</span>
          </button>
        </div>
      </header>

      <section class="kiro-command-board">
        <article :class="['kiro-quota-command', quotaTone]">
          <div class="kiro-command-top">
            <div>
              <span>额度余量</span>
              <strong>{{ quotaSummary.primary }}</strong>
            </div>
            <b>{{ remainingPercentLabel }}</b>
          </div>
          <div class="kiro-command-meter" :aria-label="quotaSummary.hint">
            <span :style="{ width: remainingPercentWidth }"></span>
          </div>
          <p>{{ quotaSummary.hint }}</p>
          <small>{{ quotaResetSummary }}</small>
        </article>

        <div class="kiro-health-stack">
          <article class="kiro-health-tile">
            <span>账号健康</span>
            <strong>{{ accountHealthSummary }}</strong>
            <small>{{ accountHealthHint }}</small>
          </article>
          <article class="kiro-health-tile">
            <span>线路调度</span>
            <strong>{{ scheduleSummary }}</strong>
            <small>{{ scheduleHint }}</small>
          </article>
          <article class="kiro-health-tile">
            <span>今日调用</span>
            <strong>{{ todayStats?.requests ?? '-' }}</strong>
            <small>{{ todayStats ? `${formatNumber(todayStats.tokens)} tokens · $${formatCost(todayStats.cost)}` : '暂无统计' }}</small>
          </article>
        </div>
      </section>

      <div v-if="errorMessage" class="kiro-status-alert error">
        <Icon name="exclamationTriangle" size="sm" />
        <span>{{ errorMessage }}</span>
      </div>
      <div v-if="actionMessage" class="kiro-status-alert success">
        <Icon name="checkCircle" size="sm" />
        <span>{{ actionMessage }}</span>
      </div>
      <div v-if="healthWarning" class="kiro-status-alert warning">
        <Icon name="exclamationTriangle" size="sm" />
        <span>{{ healthWarning }}</span>
      </div>

      <section class="kiro-status-panel kiro-account-management">
        <div class="kiro-status-section-head">
          <div>
            <p class="kiro-status-kicker">Account Desk</p>
            <h2>Kiro 账号管理</h2>
          </div>
          <span class="kiro-section-badge">{{ kiroAccountSummary }}</span>
        </div>

        <div class="kiro-account-controls">
          <label>
            <span>筛选账号</span>
            <input v-model.trim="kiroAccountSearch" type="search" placeholder="账号名 / 分组 / Base URL" />
          </label>
          <label>
            <span>状态</span>
            <select v-model="kiroStatusFilter">
              <option value="all">全部状态</option>
              <option value="active">仅正常</option>
              <option value="problem">异常 / 暂停</option>
            </select>
          </label>
          <button type="button" class="kiro-status-button" :disabled="loading" @click="loadAll">
            <Icon name="refresh" size="sm" :class="{ 'kiro-status-spin': loading }" />
            <span>同步账号</span>
          </button>
        </div>

        <div class="kiro-management-table" role="table" aria-label="Kiro 账号管理">
          <div class="kiro-management-table-head" role="row">
            <span>账号</span>
            <span>状态</span>
            <span>调度</span>
            <span>并发</span>
            <span>分组</span>
            <span>最后使用</span>
            <span>操作</span>
          </div>
          <article
            v-for="row in filteredKiroAccounts"
            :key="row.id"
            :class="['kiro-management-row', { selected: account?.id === row.id }]"
            role="row"
          >
            <div>
              <button type="button" class="kiro-row-title" @click="selectKiroAccount(row)">
                <strong>{{ row.name }}</strong>
                <small>#{{ row.id }} · {{ row.platform }} / {{ row.type }}</small>
              </button>
            </div>
            <AccountStatusIndicator :account="row" />
            <button
              type="button"
              class="kiro-schedulable-toggle"
              :class="{ on: row.schedulable }"
              :disabled="isBusyFor('schedulable', row.id)"
              @click="toggleSchedulable(row)"
            >
              {{ row.schedulable ? '参与调度' : '已暂停' }}
            </button>
            <AccountCapacityCell :account="row" />
            <AccountGroupsCell :groups="row.groups || []" :max-display="2" />
            <span>{{ formatDate(row.last_used_at) }}</span>
            <div class="kiro-row-actions">
              <button type="button" :disabled="isBusyFor('test', row.id)" @click="testKiroAccount(row)">测试</button>
              <button type="button" :disabled="isBusyFor('recover', row.id)" @click="recoverKiroState(row)">恢复</button>
              <button type="button" @click="selectKiroAccount(row)">详情</button>
            </div>
          </article>
          <div v-if="!filteredKiroAccounts.length" class="kiro-empty-mini">
            没有匹配的 Kiro 账号；确认账号 extra.source 为 kiro-gateway 或账号名包含 Kiro。
          </div>
        </div>
      </section>

      <main class="kiro-status-main">
        <section class="kiro-status-panel">
          <div class="kiro-status-section-head">
            <div>
              <p class="kiro-status-kicker">Official Credits</p>
              <h2>账号额度明细</h2>
            </div>
            <span class="kiro-section-badge">{{ officialQuotaBadge }}</span>
          </div>

          <div v-if="hasOfficialQuota" class="kiro-account-table" role="table" aria-label="Kiro 官方账号额度">
            <div class="kiro-account-table-head" role="row">
              <span>账号</span>
              <span>套餐</span>
              <span>剩余额度</span>
              <span>已用 / 总额</span>
              <span>重置时间</span>
              <span>状态</span>
            </div>
            <article v-for="row in accountQuotaRows" :key="row.key" class="kiro-account-table-row" role="row">
              <div>
                <strong>{{ row.name }}</strong>
                <small>{{ row.identity }}</small>
              </div>
              <span>{{ row.plan }}</span>
              <strong>{{ row.remaining }}</strong>
              <span>{{ row.usage }}</span>
              <span>{{ row.resetAt }}</span>
              <b :class="['kiro-row-status', row.tone]">{{ row.status }}</b>
            </article>
          </div>
          <div v-else class="kiro-empty-panel compact">
            <Icon name="infoCircle" size="lg" />
            <strong>Kiro 官方额度暂不可用</strong>
            <span>{{ officialQuota?.error || '刷新后仍不可用时，请检查 token 是否过期。' }}</span>
          </div>
        </section>

        <aside class="kiro-status-panel kiro-actions-panel">
          <div class="kiro-status-section-head">
            <div>
              <p class="kiro-status-kicker">Actions</p>
              <h2>常用操作</h2>
            </div>
          </div>

          <div v-if="account" class="kiro-primary-actions">
            <button data-testid="kiro-test-account" type="button" class="kiro-action-button primary" :disabled="isBusy('test')" @click="testKiroAccount()">
              <Icon name="play" size="sm" />
              <span>{{ isBusy('test') ? '测试中' : '测试线路' }}</span>
            </button>
            <button type="button" class="kiro-action-button" :disabled="isBusy('activate') || !gatewayStatus?.credentials_file_exists" @click="activateGateway">
              <Icon name="bolt" size="sm" />
              <span>{{ isBusy('activate') ? '激活中' : '重新激活' }}</span>
            </button>
            <button data-testid="kiro-toggle-schedulable" type="button" class="kiro-action-button" :disabled="isBusy('schedulable')" @click="toggleSchedulable()">
              <Icon name="ban" size="sm" />
              <span>{{ account.schedulable ? '暂停调度' : '启用调度' }}</span>
            </button>
            <button data-testid="kiro-recover-state" type="button" class="kiro-action-button" :disabled="isBusy('recover')" @click="recoverKiroState()">
              <Icon name="refresh" size="sm" />
              <span>{{ isBusy('recover') ? '恢复中' : '恢复状态' }}</span>
            </button>
          </div>

          <div v-else class="kiro-empty-panel compact">
            <Icon name="infoCircle" size="lg" />
            <strong>还没有 Kiro Gateway 账号</strong>
            <span>先导入 token 并激活线路。</span>
            <RouterLink to="/admin/channels/kiro" class="kiro-action-button primary">去接入</RouterLink>
          </div>

          <div v-if="testResult" class="kiro-test-result">
            <strong>{{ testResult.success ? '测试通过' : '测试失败' }}</strong>
            <span>{{ testResult.message }}</span>
            <small v-if="testResult.latency_ms">{{ testResult.latency_ms }}ms</small>
          </div>
        </aside>
      </main>

      <details class="kiro-diagnostic-panel">
        <summary>
          <span>高级诊断</span>
          <small>凭据、sub2 本地额度、模型和运行输出</small>
        </summary>

        <div class="kiro-diagnostic-grid">
          <section class="kiro-status-panel account-panel">
            <div class="kiro-status-section-head">
              <div>
                <p class="kiro-status-kicker">Sub2 Account</p>
                <h2>{{ account?.name || 'Kiro Gateway' }}</h2>
              </div>
              <AccountStatusIndicator v-if="account" :account="account" />
            </div>

            <div v-if="account" class="kiro-account-body">
              <div class="kiro-account-strip">
                <div>
                  <span>账号 ID</span>
                  <strong>#{{ account.id }}</strong>
                </div>
                <div>
                  <span>平台 / 类型</span>
                  <strong>{{ account.platform }} · {{ account.type }}</strong>
                </div>
                <div>
                  <span>优先级</span>
                  <strong>{{ account.priority }}</strong>
                </div>
                <div>
                  <span>倍率</span>
                  <strong>{{ account.rate_multiplier ?? 1 }}</strong>
                </div>
              </div>

              <div class="kiro-account-grid">
                <div class="kiro-account-box">
                  <span>调度状态</span>
                  <strong>{{ account.schedulable ? '参与调度' : '暂停调度' }}</strong>
                </div>
                <div class="kiro-account-box">
                  <span>并发容量</span>
                  <AccountCapacityCell :account="account" />
                </div>
                <div class="kiro-account-box">
                  <span>所属分组</span>
                  <AccountGroupsCell :groups="account.groups || []" :max-display="4" />
                </div>
                <div class="kiro-account-box">
                  <span>今日统计</span>
                  <AccountTodayStatsCell :stats="todayStats" :loading="statsLoading" :error="statsError" />
                </div>
              </div>

              <div class="kiro-quota-panel official">
                <div class="kiro-status-section-head compact">
                  <div>
                    <p class="kiro-status-kicker">Official Credits</p>
                    <h3>Kiro 官方额度</h3>
                  </div>
                  <span>{{ officialQuotaBadge }}</span>
                </div>
                <div v-if="hasOfficialQuota" class="kiro-quota-list">
                  <article class="kiro-quota-row">
                    <div class="kiro-quota-row-head">
                      <span>官方剩余</span>
                      <strong>{{ formatCreditAmount(officialQuota!.remaining) }} credits</strong>
                    </div>
                    <div class="kiro-quota-bar" :aria-label="`官方额度已使用 ${formatCreditAmount(officialQuota!.current_usage)}，剩余 ${formatCreditAmount(officialQuota!.remaining)}`">
                      <span :style="{ width: officialQuotaProgressWidth }"></span>
                    </div>
                    <small>
                      已用 {{ formatCreditAmount(officialQuota!.current_usage) }} / {{ formatCreditAmount(officialQuota!.usage_limit) }} credits
                      <template v-if="officialQuota?.next_reset_at"> · {{ formatQuotaReset(officialQuota.next_reset_at) }}</template>
                    </small>
                  </article>
                </div>
                <div v-else class="kiro-empty-mini">{{ officialQuota?.error || '等待官方额度数据' }}</div>
              </div>

              <div class="kiro-quota-panel">
                <div class="kiro-status-section-head compact">
                  <div>
                    <p class="kiro-status-kicker">Sub2 Quota</p>
                    <h3>sub2 本地额度</h3>
                  </div>
                  <span>{{ hasQuotaLimits ? `${quotaItems.length} 项` : '未设置上限' }}</span>
                </div>
                <div v-if="hasQuotaLimits" class="kiro-quota-list">
                  <article v-for="item in quotaItems" :key="item.key" class="kiro-quota-row">
                    <div class="kiro-quota-row-head">
                      <span>{{ item.label }}</span>
                      <strong>${{ formatQuotaAmount(item.remaining) }}</strong>
                    </div>
                    <div class="kiro-quota-bar" :aria-label="`${item.label} 已使用 ${formatQuotaAmount(item.used)}，剩余 ${formatQuotaAmount(item.remaining)}`">
                      <span :style="{ width: quotaProgressWidth(item) }"></span>
                    </div>
                    <small>
                      已用 ${{ formatQuotaAmount(item.used) }} / ${{ formatQuotaAmount(item.limit) }}
                      <template v-if="item.resetAt"> · {{ formatQuotaReset(item.resetAt) }}</template>
                    </small>
                  </article>
                </div>
                <div v-else class="kiro-empty-mini">未设置 sub2 账号级额度，无法计算剩余；Kiro 官方未返回可用余额。</div>
              </div>

              <div class="kiro-account-details">
                <div>
                  <span>Base URL</span>
                  <code>{{ String(account.credentials?.base_url || '-') }}</code>
                </div>
                <div>
                  <span>最后使用</span>
                  <strong>{{ formatDate(account.last_used_at) }}</strong>
                </div>
                <div>
                  <span>更新时间</span>
                  <strong>{{ formatDate(account.updated_at) }}</strong>
                </div>
              </div>

              <div v-if="account.error_message" class="kiro-status-alert error compact">
                <Icon name="exclamationTriangle" size="sm" />
                <span>{{ account.error_message }}</span>
              </div>

              <div class="kiro-models">
                <div class="kiro-status-section-head compact">
                  <div>
                    <p class="kiro-status-kicker">Models</p>
                    <h3>可用模型</h3>
                  </div>
                  <span>{{ models.length }} 个</span>
                </div>
                <div class="kiro-model-list">
                  <span v-for="model in models" :key="model.id">{{ model.display_name || model.id }}</span>
                  <span v-if="!models.length" class="muted">暂无模型数据</span>
                </div>
              </div>

              <div class="kiro-action-bar">
                <button type="button" class="btn btn-secondary" :disabled="isBusy('rate-limit')" @click="clearRateLimit()">
                  <Icon name="xCircle" size="sm" />
                  <span>清除限流</span>
                </button>
                <button v-if="isTempUnschedulable" type="button" class="btn btn-secondary" :disabled="isBusy('temp')" @click="resetTempUnschedulable()">
                  <Icon name="clock" size="sm" />
                  <span>清除临时暂停</span>
                </button>
              </div>
            </div>
          </section>

          <aside class="kiro-status-panel">
            <div class="kiro-status-section-head">
              <div>
                <p class="kiro-status-kicker">Runtime</p>
                <h2>网关运行时</h2>
              </div>
            </div>

            <div class="kiro-runtime-list">
              <div>
                <span>凭据文件</span>
                <strong>{{ gatewayStatus?.credentials_file_exists ? '已生成' : '缺失' }}</strong>
              </div>
              <div>
                <span>激活脚本</span>
                <strong>{{ gatewayStatus?.activation_script_exists ? '可用' : '缺失' }}</strong>
              </div>
              <div>
                <span>Kiro CLI</span>
                <strong>{{ gatewayStatus?.cli_available ? '可用' : '未安装' }}</strong>
              </div>
            </div>

            <div class="kiro-credential-list">
              <article v-for="(credential, index) in credentials" :key="`${credential.type}-${index}`">
                <div>
                  <strong>凭据 {{ index + 1 }}</strong>
                  <small>{{ credentialHint(credential) }}</small>
                </div>
                <span :class="credential.enabled === false ? 'muted' : 'active'">
                  {{ credential.enabled === false ? '停用' : '启用' }}
                </span>
              </article>
              <div v-if="!credentials.length" class="kiro-empty-mini">暂无凭据</div>
            </div>

            <div class="kiro-terminal">
              <div class="kiro-terminal-head">
                <span></span>
                <span></span>
                <span></span>
                <strong>运行输出</strong>
              </div>
              <pre>{{ activationOutput || gatewayStatus?.gateway_health_message || '等待状态输出' }}</pre>
            </div>
          </aside>
        </div>
      </details>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { RouterLink } from 'vue-router'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import AccountStatusIndicator from '@/components/account/AccountStatusIndicator.vue'
import AccountCapacityCell from '@/components/account/AccountCapacityCell.vue'
import AccountGroupsCell from '@/components/account/AccountGroupsCell.vue'
import AccountTodayStatsCell from '@/components/account/AccountTodayStatsCell.vue'
import { adminAPI } from '@/api/admin'
import type { KiroCredentialEntry, KiroGatewayStatus, KiroOfficialQuotaAccount } from '@/api/admin/kiro'
import type { Account, ClaudeModel, WindowStats } from '@/types'
import { extractApiErrorMessage } from '@/utils/apiError'
import { formatDateTime } from '@/utils/format'

type ActionName = 'test' | 'recover' | 'schedulable' | 'rate-limit' | 'temp' | 'activate'
type QuotaKind = 'total' | 'daily' | 'weekly'

interface QuotaItem {
  key: QuotaKind
  label: string
  limit: number
  used: number
  remaining: number
  utilization: number
  resetAt: string | null
}

const gatewayStatus = ref<KiroGatewayStatus | null>(null)
const account = ref<Account | null>(null)
const kiroAccounts = ref<Account[]>([])
const kiroAccountSearch = ref('')
const kiroStatusFilter = ref<'all' | 'active' | 'problem'>('all')
const busyAccountId = ref<number | null>(null)
const selectedKiroAccountId = ref<number | null>(null)
const todayStats = ref<WindowStats | null>(null)
const models = ref<ClaudeModel[]>([])
const loading = ref(false)
const statsLoading = ref(false)
const statsError = ref('')
const busyAction = ref<ActionName | null>(null)
const errorMessage = ref('')
const actionMessage = ref('')
const activationOutput = ref('')
const testResult = ref<{ success: boolean; message: string; latency_ms?: number } | null>(null)

const credentials = computed(() => gatewayStatus.value?.credentials ?? [])
const officialQuota = computed(() => gatewayStatus.value?.official_quota ?? null)
const hasOfficialQuota = computed(() => officialQuota.value?.available === true && numericQuota(officialQuota.value?.usage_limit) > 0)
const officialQuotaAccounts = computed<KiroOfficialQuotaAccount[]>(() => {
  return (officialQuota.value?.accounts ?? []).filter((item) => item.available && !item.deduplicated)
})
const officialQuotaVisibleAccounts = computed<KiroOfficialQuotaAccount[]>(() => {
  return (officialQuota.value?.accounts ?? []).filter((item) => !item.deduplicated)
})
const officialQuotaBadge = computed(() => {
  if (hasOfficialQuota.value) {
    const count = officialQuota.value?.unique_user_count || officialQuotaAccounts.value.length
    return count > 1 ? `${count} 个账号` : '官方实时'
  }
  return officialQuota.value?.error ? '读取失败' : '未获取'
})
const officialQuotaProgressWidth = computed(() => {
  const quota = officialQuota.value
  if (!quota || quota.usage_limit <= 0) return '0%'
  return `${Math.min(Math.max((quota.current_usage / quota.usage_limit) * 100, 0), 100)}%`
})
const remainingPercent = computed(() => {
  const quota = officialQuota.value
  if (!quota || quota.usage_limit <= 0) return 0
  return Math.min(Math.max((quota.remaining / quota.usage_limit) * 100, 0), 100)
})
const remainingPercentLabel = computed(() => hasOfficialQuota.value ? `${formatPercent(remainingPercent.value)}%` : '--')
const remainingPercentWidth = computed(() => `${remainingPercent.value}%`)
const quotaTone = computed(() => {
  if (!hasOfficialQuota.value) return 'unknown'
  if (remainingPercent.value <= 15) return 'danger'
  if (remainingPercent.value <= 35) return 'warning'
  return 'good'
})
const quotaFetchedAt = computed(() => officialQuota.value?.fetched_at ? formatDate(officialQuota.value.fetched_at) : '等待数据')
const quotaResetSummary = computed(() => {
  if (officialQuota.value?.next_reset_at) return formatQuotaReset(officialQuota.value.next_reset_at)
  return hasOfficialQuota.value ? '官方未返回重置时间' : '等待官方额度'
})
const knownAccountCount = computed(() => officialQuota.value?.account_count || credentials.value.length || kiroAccounts.value.length)
const accountHealthSummary = computed(() => {
  const total = knownAccountCount.value
  if (!total) return '未接入'
  return `${healthyAccountCount.value} / ${total}`
})
const healthyAccountCount = computed(() => {
  if (officialQuotaVisibleAccounts.value.length) {
    return officialQuotaVisibleAccounts.value.filter((item) => item.available).length
  }
  return kiroAccounts.value.filter(isKiroAccountHealthy).length
})
const unhealthyAccountCount = computed(() => Math.max(knownAccountCount.value - healthyAccountCount.value, 0))
const accountHealthHint = computed(() => {
  if (!knownAccountCount.value) return '暂无 token 凭据'
  if (unhealthyAccountCount.value > 0) return `${unhealthyAccountCount.value} 个账号需要处理`
  return officialQuotaVisibleAccounts.value.length ? '全部账号可读取官方额度' : '本地调度状态正常'
})
const scheduleSummary = computed(() => {
  if (!account.value) return '未创建'
  if (!gatewayStatus.value?.gateway_healthy) return '网关异常'
  return account.value.schedulable ? '可调度' : '已暂停'
})
const scheduleHint = computed(() => {
  if (!account.value) return '未找到 Kiro Gateway sub2 账号'
  if (account.value.error_message) return account.value.error_message
  if (isTempUnschedulable.value) return '存在临时暂停'
  return account.value.schedulable ? `并发 ${account.value.current_concurrency ?? 0}/${account.value.concurrency}` : '不会参与用户请求'
})
const kiroAccountSummary = computed(() => `${filteredKiroAccounts.value.length} / ${kiroAccounts.value.length} 个账号`)
const filteredKiroAccounts = computed(() => {
  const keyword = kiroAccountSearch.value.toLowerCase()
  return kiroAccounts.value.filter((item) => {
    if (kiroStatusFilter.value === 'active' && !isKiroAccountHealthy(item)) return false
    if (kiroStatusFilter.value === 'problem' && isKiroAccountHealthy(item)) return false
    if (!keyword) return true

    const groupNames = (item.groups || []).map((group) => group.name).join(' ')
    const baseURL = String(item.credentials?.base_url || '')
    return [
      item.name,
      item.status,
      item.error_message || '',
      groupNames,
      baseURL
    ].some((value) => value.toLowerCase().includes(keyword))
  })
})
const healthWarning = computed(() => {
  if (officialQuota.value?.error) return officialQuota.value.error
  if (unhealthyAccountCount.value > 0) return `${unhealthyAccountCount.value} 个 Kiro 账号未能读取官方额度`
  if (account.value?.error_message) return account.value.error_message
  if (gatewayStatus.value && !gatewayStatus.value.gateway_healthy) return gatewayStatus.value.gateway_health_message || 'Kiro 网关健康检查异常'
  return ''
})
const accountQuotaRows = computed(() => {
  return officialQuotaVisibleAccounts.value.map((item, index) => ({
    key: item.user_id || item.credential_sha256 || `account-${index}`,
    name: `账号 ${String(index + 1).padStart(2, '0')}`,
    identity: item.error || (item.user_id ? `User ${shortId(item.user_id)}` : '已脱敏'),
    plan: item.subscription_title || officialQuota.value?.subscription_title || 'Kiro',
    remaining: `${formatCreditAmount(item.remaining)} credits`,
    usage: `${formatCreditAmount(item.current_usage)} / ${formatCreditAmount(item.usage_limit)} credits`,
    resetAt: item.next_reset_at ? formatDate(item.next_reset_at) : '-',
    status: item.available ? '正常' : '异常',
    tone: item.available ? 'good' : 'danger'
  }))
})
const gatewayStatusClass = computed(() => gatewayStatus.value?.gateway_healthy ? 'online' : 'offline')
const gatewayStatusText = computed(() => gatewayStatus.value?.gateway_healthy ? '网关在线' : '网关异常')
const isTempUnschedulable = computed(() => {
  if (!account.value?.temp_unschedulable_until) return false
  return new Date(account.value.temp_unschedulable_until).getTime() > Date.now()
})
const quotaItems = computed<QuotaItem[]>(() => {
  if (!account.value) return []
  return [
    makeQuotaItem('total', '总额度', account.value.quota_limit, account.value.quota_used, null),
    makeQuotaItem('daily', '今日额度', account.value.quota_daily_limit, account.value.quota_daily_used, quotaResetAt('daily')),
    makeQuotaItem('weekly', '本周额度', account.value.quota_weekly_limit, account.value.quota_weekly_used, quotaResetAt('weekly'))
  ].filter((item): item is QuotaItem => item !== null)
})
const hasQuotaLimits = computed(() => quotaItems.value.length > 0)
const quotaSummary = computed(() => {
  if (!account.value) {
    return { primary: '-', hint: '等待账号数据' }
  }
  if (hasOfficialQuota.value && officialQuota.value) {
    return {
      primary: `${formatCreditAmount(officialQuota.value.remaining)} credits`,
      hint: `官方已用 ${formatCreditAmount(officialQuota.value.current_usage)} / ${formatCreditAmount(officialQuota.value.usage_limit)} · ${officialQuota.value.subscription_title || 'Kiro'}`
    }
  }
  if (officialQuota.value?.error) {
    return { primary: '读取失败', hint: officialQuota.value.error }
  }
  if (!hasQuotaLimits.value) {
    return { primary: '未设置上限', hint: '无法计算剩余；Kiro 官方未返回可用余额' }
  }
  const tightest = [...quotaItems.value].sort((left, right) => left.remaining - right.remaining)[0]
  return {
    primary: `$${formatQuotaAmount(tightest.remaining)}`,
    hint: `${tightest.label}剩余 / 上限 $${formatQuotaAmount(tightest.limit)}`
  }
})

function isBusy(action: ActionName): boolean {
  return busyAction.value === action
}

function isBusyFor(action: ActionName, accountId: number): boolean {
  return busyAction.value === action && busyAccountId.value === accountId
}

async function loadAll() {
  loading.value = true
  errorMessage.value = ''
  try {
    const [statusResult, accountResult] = await Promise.all([
      adminAPI.kiro.getStatus(),
      adminAPI.accounts.list(1, 100, {
        platform: 'anthropic',
        type: 'apikey'
      })
    ])
    gatewayStatus.value = statusResult
    kiroAccounts.value = accountResult.items.filter(isKiroAccount)
    const nextAccount = findKiroAccount(kiroAccounts.value)
    selectedKiroAccountId.value = nextAccount?.id ?? null
    account.value = nextAccount
    await loadAccountDetails()
  } catch (error: unknown) {
    errorMessage.value = extractApiErrorMessage(error, '读取 Kiro 账号状态失败')
  } finally {
    loading.value = false
  }
}

function findKiroAccount(accounts: Account[]): Account | null {
  if (selectedKiroAccountId.value) {
    const selected = accounts.find((item) => item.id === selectedKiroAccountId.value)
    if (selected) return selected
  }
  const candidates = accounts.filter((item) => item.extra?.source === 'kiro-gateway' || item.name === 'Kiro Gateway')
  return candidates.find((item) => item.name === 'Kiro Gateway' && item.status === 'active')
    || candidates.find((item) => item.name === 'Kiro Gateway')
    || candidates.find((item) => item.extra?.source === 'kiro-gateway' && item.status === 'active')
    || candidates[0]
    || accounts[0]
    || null
}

function isKiroAccount(item: Account): boolean {
  const extra = item.extra as Record<string, unknown> | undefined
  const credentials = item.credentials as Record<string, unknown> | undefined
  const baseURL = typeof credentials?.base_url === 'string' ? credentials.base_url : ''
  return extra?.source === 'kiro-gateway'
    || extra?.deployment_path === '/opt/kiro-gateway'
    || item.name.toLowerCase().includes('kiro')
    || baseURL.includes('kiro-upstream')
    || baseURL.includes('kiro-gateway')
}

function isKiroAccountHealthy(item: Account): boolean {
  if (item.status !== 'active') return false
  if (!item.schedulable) return false
  if (item.error_message) return false
  if (item.rate_limit_reset_at) return false
  if (item.overload_until && new Date(item.overload_until).getTime() > Date.now()) return false
  if (item.temp_unschedulable_until && new Date(item.temp_unschedulable_until).getTime() > Date.now()) return false
  return true
}

async function selectKiroAccount(nextAccount: Account) {
  selectedKiroAccountId.value = nextAccount.id
  account.value = nextAccount
  await loadAccountDetails()
}

function replaceKiroAccount(updated: Account) {
  kiroAccounts.value = kiroAccounts.value.map((item) => item.id === updated.id ? updated : item)
  if (account.value?.id === updated.id) {
    account.value = updated
  }
}

function makeQuotaItem(
  key: QuotaKind,
  label: string,
  limitValue: number | null | undefined,
  usedValue: number | null | undefined,
  resetAt: string | null
): QuotaItem | null {
  const limit = numericQuota(limitValue)
  if (limit <= 0) return null
  const used = numericQuota(usedValue)
  const remaining = Math.max(limit - used, 0)
  return {
    key,
    label,
    limit,
    used,
    remaining,
    utilization: limit > 0 ? (used / limit) * 100 : 0,
    resetAt
  }
}

function numericQuota(value: number | null | undefined): number {
  return typeof value === 'number' && Number.isFinite(value) ? Math.max(value, 0) : 0
}

function quotaResetAt(kind: Exclude<QuotaKind, 'total'>): string | null {
  const current = account.value
  if (!current) return null

  const directResetAt = kind === 'daily' ? current.quota_daily_reset_at : current.quota_weekly_reset_at
  if (directResetAt) return directResetAt

  const extra = current.extra as Record<string, unknown> | undefined
  const resetAtKey = kind === 'daily' ? 'quota_daily_reset_at' : 'quota_weekly_reset_at'
  const extraResetAt = extra?.[resetAtKey]
  if (typeof extraResetAt === 'string' && extraResetAt) return extraResetAt

  const startKey = kind === 'daily' ? 'quota_daily_start' : 'quota_weekly_start'
  const startValue = extra?.[startKey]
  if (typeof startValue !== 'string' || !startValue) return null

  const start = new Date(startValue)
  if (Number.isNaN(start.getTime())) return null
  const periodMs = kind === 'daily' ? 24 * 60 * 60 * 1000 : 7 * 24 * 60 * 60 * 1000
  return new Date(start.getTime() + periodMs).toISOString()
}

async function loadAccountDetails() {
  todayStats.value = null
  models.value = []
  statsError.value = ''
  if (!account.value) return

  statsLoading.value = true
  try {
    const [stats, availableModels] = await Promise.all([
      adminAPI.accounts.getTodayStats(account.value.id).catch((error: unknown) => {
        statsError.value = extractApiErrorMessage(error, '读取今日统计失败')
        return null
      }),
      adminAPI.accounts.getAvailableModels(account.value.id).catch(() => [])
    ])
    todayStats.value = stats
    models.value = availableModels
  } finally {
    statsLoading.value = false
  }
}

async function withAccountAction(action: ActionName, target: Account | null, handler: (current: Account) => Promise<void>) {
  if (!target) return
  busyAction.value = action
  busyAccountId.value = target.id
  errorMessage.value = ''
  actionMessage.value = ''
  try {
    await handler(target)
  } catch (error: unknown) {
    errorMessage.value = extractApiErrorMessage(error, 'Kiro 账号操作失败')
  } finally {
    busyAction.value = null
    busyAccountId.value = null
  }
}

async function testKiroAccount(target: Account | null = account.value) {
  await withAccountAction('test', target, async (current) => {
    const result = await adminAPI.accounts.testAccount(current.id)
    testResult.value = result
    actionMessage.value = result.success ? 'Kiro 账号测试通过' : 'Kiro 账号测试失败'
  })
}

async function recoverKiroState(target: Account | null = account.value) {
  await withAccountAction('recover', target, async (current) => {
    replaceKiroAccount(await adminAPI.accounts.recoverState(current.id))
    actionMessage.value = 'Kiro 账号状态已恢复'
    await loadAll()
  })
}

async function toggleSchedulable(target: Account | null = account.value) {
  await withAccountAction('schedulable', target, async (current) => {
    const nextSchedulable = !current.schedulable
    replaceKiroAccount(await adminAPI.accounts.setSchedulable(current.id, nextSchedulable))
    actionMessage.value = nextSchedulable ? 'Kiro 账号已启用调度' : 'Kiro 账号已暂停调度'
    await loadAccountDetails()
  })
}

async function clearRateLimit() {
  await withAccountAction('rate-limit', account.value, async (current) => {
    replaceKiroAccount(await adminAPI.accounts.clearRateLimit(current.id))
    actionMessage.value = 'Kiro 账号限流状态已清除'
    await loadAccountDetails()
  })
}

async function resetTempUnschedulable() {
  await withAccountAction('temp', account.value, async (current) => {
    await adminAPI.accounts.resetTempUnschedulable(current.id)
    actionMessage.value = 'Kiro 账号临时暂停已清除'
    await loadAll()
  })
}

async function activateGateway() {
  busyAction.value = 'activate'
  errorMessage.value = ''
  actionMessage.value = ''
  try {
    const result = await adminAPI.kiro.activate()
    activationOutput.value = result.output || result.message
    actionMessage.value = 'Kiro 专属线路已激活'
    await loadAll()
  } catch (error: unknown) {
    errorMessage.value = extractApiErrorMessage(error, '激活 Kiro Gateway 失败')
  } finally {
    busyAction.value = null
  }
}

function credentialHint(credential: KiroCredentialEntry): string {
  const parts = [credential.region, credential.api_region, credential.profile_arn].filter(Boolean)
  return parts.length ? parts.join(' · ') : credential.path || '已脱敏'
}

function formatDate(value: string | null | undefined): string {
  if (!value) return '-'
  return formatDateTime(value)
}

function formatNumber(value: number): string {
  return new Intl.NumberFormat('zh-CN').format(value)
}

function formatCost(value: number): string {
  return value.toFixed(4)
}

function formatQuotaAmount(value: number): string {
  return value.toFixed(2)
}

function formatCreditAmount(value: number): string {
  return new Intl.NumberFormat('zh-CN', {
    minimumFractionDigits: Number.isInteger(value) ? 0 : 2,
    maximumFractionDigits: 2
  }).format(value)
}

function formatPercent(value: number): string {
  return new Intl.NumberFormat('zh-CN', {
    minimumFractionDigits: Number.isInteger(value) ? 0 : 2,
    maximumFractionDigits: 2
  }).format(value)
}

function shortId(value: string): string {
  const trimmed = value.trim()
  if (trimmed.length <= 8) return trimmed
  return `${trimmed.slice(0, 4)}...${trimmed.slice(-4)}`
}

function quotaProgressWidth(item: QuotaItem): string {
  return `${Math.min(Math.max(item.utilization, 0), 100)}%`
}

function formatQuotaReset(value: string): string {
  return `重置 ${formatDate(value)}`
}

onMounted(loadAll)
</script>

<style scoped>
.kiro-status-page {
  max-width: 1240px;
  margin: 0 auto;
  padding: 0.25rem 0 2rem;
  color: #111827;
}

.kiro-status-hero,
.kiro-status-panel,
.kiro-quota-command,
.kiro-health-tile,
.kiro-diagnostic-panel {
  border: 1px solid rgba(226, 232, 240, 0.92);
  background: rgba(255, 255, 255, 0.94);
  box-shadow: 0 22px 56px -46px rgba(15, 23, 42, 0.38);
}

.dark .kiro-status-hero,
.dark .kiro-status-panel,
.dark .kiro-quota-command,
.dark .kiro-health-tile,
.dark .kiro-diagnostic-panel {
  border-color: rgba(71, 85, 105, 0.58);
  background: rgba(15, 23, 42, 0.8);
}

.kiro-status-hero {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  border-radius: 1rem;
  padding: 1.15rem 1.25rem;
}

.kiro-status-kicker {
  margin: 0;
  color: #0f766e;
  font-size: 0.68rem;
  font-weight: 800;
  letter-spacing: 0.12em;
  text-transform: uppercase;
}

.dark .kiro-status-kicker {
  color: #5eead4;
}

.kiro-status-hero h1,
.kiro-status-section-head h2,
.kiro-status-section-head h3 {
  margin: 0.2rem 0 0;
  color: #111827;
  font-weight: 780;
  letter-spacing: 0;
}

.kiro-status-hero h1 {
  font-size: clamp(1.65rem, 3.2vw, 2.85rem);
  line-height: 1.05;
}

.dark .kiro-status-hero h1,
.dark .kiro-status-section-head h2,
.dark .kiro-status-section-head h3,
.dark .kiro-command-top strong,
.dark .kiro-health-tile strong,
.dark .kiro-account-table-row strong,
.dark .kiro-account-strip strong,
.dark .kiro-account-box strong,
.dark .kiro-account-details strong,
.dark .kiro-runtime-list strong,
.dark .kiro-credential-list strong,
.dark .kiro-empty-panel strong,
.dark .kiro-test-result strong {
  color: #f8fafc;
}

.kiro-status-meta,
.kiro-status-actions,
.kiro-action-bar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.6rem;
}

.kiro-status-meta {
  margin-top: 0.75rem;
}

.kiro-status-chip,
.kiro-status-link,
.kiro-status-button,
.kiro-section-badge,
.kiro-row-status,
.kiro-action-button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 0.42rem;
  border-radius: 0.72rem;
  font-size: 0.82rem;
  font-weight: 760;
  white-space: nowrap;
}

.kiro-status-chip,
.kiro-section-badge {
  min-height: 2.25rem;
  padding: 0 0.78rem;
  border: 1px solid rgba(15, 118, 110, 0.18);
  color: #0f766e;
  background: rgba(240, 253, 250, 0.88);
}

.kiro-status-chip.offline {
  border-color: rgba(220, 38, 38, 0.2);
  color: #b91c1c;
  background: rgba(254, 242, 242, 0.92);
}

.kiro-status-dot {
  width: 0.45rem;
  height: 0.45rem;
  border-radius: 999px;
  background: currentColor;
  animation: kiro-status-pulse 1.8s ease-in-out infinite;
}

.kiro-status-path,
.kiro-account-details code {
  color: #64748b;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 0.8rem;
}

.kiro-status-link,
.kiro-status-button {
  min-height: 2.35rem;
  padding: 0 0.85rem;
  border: 1px solid rgba(15, 118, 110, 0.18);
  color: #0f766e;
  background: rgba(248, 250, 252, 0.88);
  transition: transform 0.18s ease, border-color 0.18s ease;
}

.kiro-status-link:hover,
.kiro-status-button:hover:not(:disabled),
.kiro-action-button:hover:not(:disabled) {
  border-color: rgba(15, 118, 110, 0.34);
  transform: translateY(-1px);
}

.kiro-command-board {
  display: grid;
  grid-template-columns: minmax(0, 1.7fr) minmax(330px, 0.85fr);
  gap: 0.85rem;
  margin-top: 0.85rem;
}

.kiro-quota-command {
  overflow: hidden;
  border-radius: 1.05rem;
  padding: 1.35rem;
  background:
    linear-gradient(135deg, rgba(240, 253, 250, 0.95), rgba(255, 255, 255, 0.96) 58%),
    #ffffff;
}

.kiro-quota-command.warning {
  background: linear-gradient(135deg, rgba(255, 251, 235, 0.96), rgba(255, 255, 255, 0.96) 58%);
}

.kiro-quota-command.danger {
  background: linear-gradient(135deg, rgba(254, 242, 242, 0.96), rgba(255, 255, 255, 0.96) 58%);
}

.kiro-command-top {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 1rem;
}

.kiro-command-top span,
.kiro-health-tile span,
.kiro-account-table-head span,
.kiro-account-strip span,
.kiro-account-box span,
.kiro-account-details span,
.kiro-runtime-list span {
  color: #64748b;
  font-size: 0.76rem;
  font-weight: 760;
}

.kiro-command-top strong {
  display: block;
  margin-top: 0.28rem;
  color: #111827;
  font-size: clamp(2.4rem, 6vw, 4.6rem);
  line-height: 0.95;
  letter-spacing: 0;
}

.kiro-command-top b {
  border-radius: 999px;
  padding: 0.45rem 0.68rem;
  color: #0f766e;
  background: rgba(204, 251, 241, 0.72);
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 0.95rem;
}

.kiro-quota-command.warning .kiro-command-top b {
  color: #92400e;
  background: rgba(254, 243, 199, 0.9);
}

.kiro-quota-command.danger .kiro-command-top b {
  color: #991b1b;
  background: rgba(254, 226, 226, 0.9);
}

.kiro-command-meter,
.kiro-quota-bar {
  overflow: hidden;
  height: 0.52rem;
  border-radius: 999px;
  background: rgba(226, 232, 240, 0.88);
}

.kiro-command-meter {
  margin-top: 1.35rem;
}

.kiro-command-meter span,
.kiro-quota-bar span {
  display: block;
  height: 100%;
  border-radius: inherit;
  background: #0f766e;
  transition: width 0.28s ease;
}

.kiro-quota-command.warning .kiro-command-meter span {
  background: #b45309;
}

.kiro-quota-command.danger .kiro-command-meter span {
  background: #b91c1c;
}

.kiro-quota-command p,
.kiro-quota-command small,
.kiro-health-tile small,
.kiro-account-table-row small,
.kiro-test-result span,
.kiro-test-result small,
.kiro-empty-panel span,
.kiro-credential-list small {
  color: #64748b;
  font-size: 0.8rem;
}

.kiro-quota-command p {
  margin: 0.9rem 0 0.25rem;
}

.kiro-health-stack {
  display: grid;
  gap: 0.85rem;
}

.kiro-health-tile {
  border-radius: 1rem;
  padding: 1rem;
}

.kiro-health-tile strong,
.kiro-account-strip strong,
.kiro-account-box strong,
.kiro-account-details strong,
.kiro-runtime-list strong {
  display: block;
  margin-top: 0.25rem;
  color: #111827;
  font-size: 1.05rem;
}

.kiro-status-alert {
  display: flex;
  align-items: center;
  gap: 0.6rem;
  margin-top: 0.85rem;
  border-radius: 0.9rem;
  padding: 0.8rem 0.9rem;
  font-size: 0.88rem;
}

.kiro-status-alert.compact {
  margin-top: 0;
}

.kiro-status-alert.error {
  border: 1px solid rgba(248, 113, 113, 0.34);
  background: rgba(254, 242, 242, 0.86);
  color: #b91c1c;
}

.kiro-status-alert.success {
  border: 1px solid rgba(45, 212, 191, 0.35);
  background: rgba(240, 253, 250, 0.9);
  color: #0f766e;
}

.kiro-status-alert.warning {
  border: 1px solid rgba(251, 191, 36, 0.42);
  background: rgba(255, 251, 235, 0.92);
  color: #92400e;
}

.kiro-status-main,
.kiro-diagnostic-grid {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(310px, 0.42fr);
  gap: 0.85rem;
  margin-top: 0.85rem;
}

.kiro-status-panel {
  border-radius: 1rem;
  padding: 1rem;
}

.kiro-account-management {
  margin-top: 0.85rem;
}

.kiro-status-section-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 1rem;
  margin-bottom: 1rem;
}

.kiro-status-section-head.compact {
  margin-bottom: 0.7rem;
}

.kiro-status-section-head h2,
.kiro-status-section-head h3 {
  font-size: 1.08rem;
}

.kiro-account-table {
  overflow: hidden;
  border: 1px solid rgba(226, 232, 240, 0.9);
  border-radius: 0.9rem;
}

.kiro-account-controls {
  display: grid;
  grid-template-columns: minmax(260px, 1fr) minmax(160px, 0.28fr) auto;
  gap: 0.75rem;
  align-items: end;
  margin-bottom: 0.8rem;
}

.kiro-account-controls label {
  display: grid;
  gap: 0.35rem;
}

.kiro-account-controls label span {
  color: #64748b;
  font-size: 0.75rem;
  font-weight: 760;
}

.kiro-account-controls input,
.kiro-account-controls select {
  min-height: 2.4rem;
  border: 1px solid rgba(203, 213, 225, 0.95);
  border-radius: 0.72rem;
  background: rgba(248, 250, 252, 0.92);
  color: #0f172a;
  padding: 0 0.78rem;
  font-size: 0.86rem;
}

.dark .kiro-account-controls input,
.dark .kiro-account-controls select {
  border-color: rgba(71, 85, 105, 0.72);
  background: rgba(15, 23, 42, 0.72);
  color: #f8fafc;
}

.kiro-management-table {
  overflow: hidden;
  border: 1px solid rgba(226, 232, 240, 0.9);
  border-radius: 0.9rem;
}

.kiro-management-table-head,
.kiro-management-row {
  display: grid;
  grid-template-columns: minmax(190px, 1.3fr) 0.85fr 0.78fr 0.72fr minmax(140px, 1fr) 0.85fr minmax(180px, 1fr);
  gap: 0.8rem;
  align-items: center;
  padding: 0.82rem 0.95rem;
}

.kiro-management-table-head {
  background: rgba(248, 250, 252, 0.92);
}

.kiro-management-table-head span {
  color: #64748b;
  font-size: 0.76rem;
  font-weight: 760;
}

.kiro-management-row {
  border-top: 1px solid rgba(226, 232, 240, 0.86);
  color: #334155;
  font-size: 0.85rem;
}

.kiro-management-row.selected {
  background: rgba(240, 253, 250, 0.74);
}

.kiro-row-title {
  display: grid;
  gap: 0.18rem;
  border: 0;
  background: transparent;
  color: inherit;
  padding: 0;
  text-align: left;
}

.kiro-row-title strong {
  color: #111827;
  font-size: 0.92rem;
}

.kiro-row-title small {
  color: #64748b;
  font-size: 0.74rem;
}

.kiro-schedulable-toggle {
  min-height: 1.9rem;
  border: 1px solid rgba(148, 163, 184, 0.34);
  border-radius: 999px;
  background: rgba(241, 245, 249, 0.82);
  color: #475569;
  padding: 0 0.6rem;
  font-size: 0.76rem;
  font-weight: 760;
}

.kiro-schedulable-toggle.on {
  border-color: rgba(15, 118, 110, 0.28);
  background: rgba(204, 251, 241, 0.72);
  color: #0f766e;
}

.kiro-row-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 0.38rem;
}

.kiro-row-actions button {
  min-height: 1.9rem;
  border: 1px solid rgba(203, 213, 225, 0.86);
  border-radius: 0.55rem;
  background: rgba(255, 255, 255, 0.82);
  color: #334155;
  padding: 0 0.55rem;
  font-size: 0.76rem;
  font-weight: 760;
}

.kiro-row-actions button:hover:not(:disabled),
.kiro-schedulable-toggle:hover:not(:disabled),
.kiro-row-title:hover strong {
  color: #0f766e;
}

.kiro-account-table-head,
.kiro-account-table-row {
  display: grid;
  grid-template-columns: 1.1fr 0.9fr 1fr 1.2fr 1.1fr 0.7fr;
  gap: 0.8rem;
  align-items: center;
  padding: 0.82rem 0.95rem;
}

.kiro-account-table-head {
  background: rgba(248, 250, 252, 0.92);
}

.kiro-account-table-row {
  border-top: 1px solid rgba(226, 232, 240, 0.86);
  color: #334155;
  font-size: 0.86rem;
}

.kiro-account-table-row strong {
  color: #111827;
  font-size: 0.95rem;
}

.kiro-row-status {
  min-height: 1.85rem;
  padding: 0 0.55rem;
}

.kiro-row-status.good {
  color: #0f766e;
  background: rgba(204, 251, 241, 0.72);
}

.kiro-row-status.danger {
  color: #b91c1c;
  background: rgba(254, 226, 226, 0.82);
}

.kiro-primary-actions {
  display: grid;
  gap: 0.65rem;
}

.kiro-action-button {
  min-height: 2.75rem;
  border: 1px solid rgba(226, 232, 240, 0.95);
  color: #334155;
  background: rgba(248, 250, 252, 0.9);
  transition: transform 0.18s ease, border-color 0.18s ease, background 0.18s ease;
}

.kiro-action-button.primary {
  border-color: rgba(15, 118, 110, 0.2);
  color: #ffffff;
  background: #0f766e;
}

.kiro-test-result,
.kiro-account-strip > div,
.kiro-account-box,
.kiro-account-details,
.kiro-quota-panel,
.kiro-quota-row,
.kiro-runtime-list > div,
.kiro-credential-list article {
  border: 1px solid rgba(226, 232, 240, 0.86);
  border-radius: 0.85rem;
  background: rgba(248, 250, 252, 0.72);
  padding: 0.85rem;
}

.kiro-test-result {
  display: grid;
  gap: 0.2rem;
  margin-top: 0.75rem;
}

.kiro-diagnostic-panel {
  margin-top: 0.85rem;
  border-radius: 1rem;
  padding: 0;
}

.kiro-diagnostic-panel summary {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.9rem;
  cursor: pointer;
  list-style: none;
  padding: 1rem;
}

.kiro-diagnostic-panel summary::-webkit-details-marker {
  display: none;
}

.kiro-diagnostic-panel summary span {
  color: #111827;
  font-size: 1rem;
  font-weight: 780;
}

.kiro-diagnostic-panel summary small {
  color: #64748b;
  font-size: 0.78rem;
}

.kiro-diagnostic-panel[open] {
  padding-bottom: 1rem;
}

.kiro-diagnostic-panel[open] summary {
  border-bottom: 1px solid rgba(226, 232, 240, 0.9);
}

.kiro-diagnostic-grid {
  padding: 0 1rem;
}

.kiro-account-body,
.kiro-runtime-list,
.kiro-credential-list,
.kiro-quota-list,
.kiro-account-details {
  display: grid;
  gap: 0.85rem;
}

.kiro-account-strip,
.kiro-account-grid {
  display: grid;
  gap: 0.65rem;
}

.kiro-account-strip {
  grid-template-columns: repeat(4, minmax(0, 1fr));
}

.kiro-account-grid {
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.kiro-account-details code {
  display: block;
  margin-top: 0.25rem;
  overflow-wrap: anywhere;
}

.kiro-quota-row-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
}

.kiro-quota-row-head span {
  color: #64748b;
  font-size: 0.76rem;
  font-weight: 760;
}

.kiro-quota-row-head strong {
  color: #111827;
  font-size: 1rem;
}

.kiro-quota-row small {
  display: block;
  margin-top: 0.38rem;
  color: #78716c;
  font-size: 0.74rem;
}

.kiro-quota-bar {
  margin-top: 0.55rem;
}

.kiro-model-list {
  display: flex;
  flex-wrap: wrap;
  gap: 0.45rem;
}

.kiro-model-list span,
.kiro-credential-list article > span {
  border-radius: 999px;
  background: rgba(240, 253, 250, 0.86);
  color: #0f766e;
  padding: 0.32rem 0.55rem;
  font-size: 0.74rem;
  font-weight: 760;
}

.kiro-model-list span.muted,
.kiro-credential-list article > span.muted,
.kiro-empty-mini {
  background: rgba(241, 245, 249, 0.84);
  color: #64748b;
}

.kiro-action-bar {
  gap: 0.5rem;
}

.kiro-credential-list article {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.7rem;
}

.kiro-credential-list strong,
.kiro-credential-list small {
  display: block;
}

.kiro-empty-panel {
  display: grid;
  place-items: center;
  gap: 0.7rem;
  min-height: 18rem;
  color: #64748b;
  text-align: center;
}

.kiro-empty-panel.compact {
  min-height: 10rem;
}

.kiro-empty-mini {
  border-radius: 0.85rem;
  padding: 0.85rem;
  text-align: center;
}

.kiro-terminal {
  overflow: hidden;
  margin-top: 0.85rem;
  border-radius: 0.9rem;
  background: #0f172a;
}

.kiro-terminal-head {
  display: flex;
  align-items: center;
  gap: 0.35rem;
  border-bottom: 1px solid rgba(148, 163, 184, 0.18);
  padding: 0.65rem 0.75rem;
}

.kiro-terminal-head span {
  width: 0.5rem;
  height: 0.5rem;
  border-radius: 999px;
  background: #fb7185;
}

.kiro-terminal-head span:nth-child(2) {
  background: #fbbf24;
}

.kiro-terminal-head span:nth-child(3) {
  background: #34d399;
}

.kiro-terminal-head strong {
  margin-left: 0.4rem;
  color: #94a3b8;
  font-size: 0.72rem;
}

.kiro-terminal pre {
  min-height: 8rem;
  max-height: 18rem;
  overflow: auto;
  margin: 0;
  padding: 0.85rem;
  color: #d1fae5;
  font-size: 0.78rem;
  line-height: 1.55;
  white-space: pre-wrap;
}

.dark .kiro-account-table,
.dark .kiro-management-table,
.dark .kiro-management-row,
.dark .kiro-account-table-row,
.dark .kiro-test-result,
.dark .kiro-account-strip > div,
.dark .kiro-account-box,
.dark .kiro-account-details,
.dark .kiro-quota-panel,
.dark .kiro-quota-row,
.dark .kiro-runtime-list > div,
.dark .kiro-credential-list article {
  border-color: rgba(71, 85, 105, 0.52);
  background: rgba(15, 23, 42, 0.56);
}

.dark .kiro-account-table-head,
.dark .kiro-management-table-head,
.dark .kiro-row-actions button,
.dark .kiro-schedulable-toggle,
.dark .kiro-action-button {
  background: rgba(30, 41, 59, 0.68);
}

.dark .kiro-command-meter,
.dark .kiro-quota-bar {
  background: rgba(51, 65, 85, 0.82);
}

.dark .kiro-account-table-row,
.dark .kiro-management-row,
.dark .kiro-row-title strong,
.dark .kiro-diagnostic-panel summary span {
  color: #f8fafc;
}

.kiro-status-spin {
  animation: kiro-status-spin 0.9s linear infinite;
}

@keyframes kiro-status-spin {
  to {
    transform: rotate(360deg);
  }
}

@keyframes kiro-status-pulse {
  0%, 100% {
    opacity: 0.5;
    transform: scale(0.9);
  }
  50% {
    opacity: 1;
    transform: scale(1.08);
  }
}

@media (max-width: 1040px) {
  .kiro-command-board,
  .kiro-status-main,
  .kiro-diagnostic-grid {
    grid-template-columns: 1fr;
  }

  .kiro-health-stack {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }
}

@media (max-width: 760px) {
  .kiro-status-hero,
  .kiro-diagnostic-panel summary {
    display: grid;
  }

  .kiro-health-stack,
  .kiro-account-strip,
  .kiro-account-grid {
    grid-template-columns: 1fr;
  }

  .kiro-account-controls,
  .kiro-management-table-head,
  .kiro-management-row {
    grid-template-columns: 1fr;
  }

  .kiro-management-table-head {
    display: none;
  }

  .kiro-management-row {
    gap: 0.58rem;
  }

  .kiro-account-table {
    display: grid;
    gap: 0.65rem;
    border: 0;
  }

  .kiro-account-table-head {
    display: none;
  }

  .kiro-account-table-row {
    grid-template-columns: 1fr;
    border: 1px solid rgba(226, 232, 240, 0.9);
    border-radius: 0.85rem;
  }

  .kiro-status-actions,
  .kiro-status-link,
  .kiro-status-button,
  .kiro-action-button,
  .kiro-action-bar .btn {
    width: 100%;
  }
}
</style>
