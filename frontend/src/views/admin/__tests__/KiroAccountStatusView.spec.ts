import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import KiroAccountStatusView from '../KiroAccountStatusView.vue'

const {
  getStatus,
  list,
  testAccount,
  recoverState,
  setSchedulable,
  clearRateLimit,
  resetTempUnschedulable,
  getTodayStats,
  getAvailableModels,
  activate
} = vi.hoisted(() => ({
  getStatus: vi.fn(),
  list: vi.fn(),
  testAccount: vi.fn(),
  recoverState: vi.fn(),
  setSchedulable: vi.fn(),
  clearRateLimit: vi.fn(),
  resetTempUnschedulable: vi.fn(),
  getTodayStats: vi.fn(),
  getAvailableModels: vi.fn(),
  activate: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    kiro: {
      getStatus,
      activate
    },
    accounts: {
      list,
      testAccount,
      recoverState,
      setSchedulable,
      clearRateLimit,
      resetTempUnschedulable,
      getTodayStats,
      getAvailableModels
    }
  }
}))

vi.mock('vue-router', () => ({
  RouterLink: {
    props: ['to'],
    template: '<a :href="typeof to === \'string\' ? to : \'#\'"><slot /></a>'
  }
}))

vi.mock('@/components/layout/AppLayout.vue', () => ({
  default: {
    template: '<div><slot /></div>'
  }
}))

vi.mock('@/components/icons/Icon.vue', () => ({
  default: {
    props: ['name', 'size'],
    template: '<span class="icon-stub" />'
  }
}))

vi.mock('@/components/account/AccountStatusIndicator.vue', () => ({
  default: {
    props: ['account'],
    template: '<span class="status-stub">{{ account.status }}</span>'
  }
}))

vi.mock('@/components/account/AccountCapacityCell.vue', () => ({
  default: {
    props: ['account'],
    template: '<span class="capacity-stub">{{ account.current_concurrency ?? 0 }}/{{ account.concurrency }}</span>'
  }
}))

vi.mock('@/components/account/AccountGroupsCell.vue', () => ({
  default: {
    props: ['groups'],
    template: '<span class="groups-stub">{{ (groups || []).map((group) => group.name).join(", ") }}</span>'
  }
}))

vi.mock('@/components/account/AccountTodayStatsCell.vue', () => ({
  default: {
    props: ['stats'],
    template: '<span class="today-stats-stub">{{ stats ? stats.requests : "-" }}</span>'
  }
}))

const kiroAccount = {
  id: 180,
  name: 'Kiro Gateway',
  platform: 'anthropic',
  type: 'apikey',
  credentials: {
    base_url: 'http://127.0.0.1:8000'
  },
  extra: {
    source: 'kiro-gateway'
  },
  proxy_id: null,
  concurrency: 2,
  current_concurrency: 1,
  priority: 70,
  status: 'active',
  error_message: null,
  last_used_at: '2026-05-17T01:00:00Z',
  expires_at: null,
  auto_pause_on_expired: true,
  created_at: '2026-05-16T00:00:00Z',
  updated_at: '2026-05-17T00:00:00Z',
  groups: [{ id: 24, name: 'Kiro 专属线路', platform: 'anthropic' }],
  group_ids: [24],
  schedulable: true,
  quota_limit: 100,
  quota_used: 25,
  quota_daily_limit: 20,
  quota_daily_used: 6,
  quota_weekly_limit: 60,
  quota_weekly_used: 17,
  quota_daily_reset_at: '2026-05-18T00:00:00Z',
  quota_weekly_reset_at: '2026-05-24T00:00:00Z',
  rate_limited_at: null,
  rate_limit_reset_at: null,
  overload_until: null,
  temp_unschedulable_until: null,
  temp_unschedulable_reason: null,
  session_window_start: null,
  session_window_end: null,
  session_window_status: null
}

const disabledTestKiroAccount = {
  ...kiroAccount,
  id: 177,
  name: 'Kiro Gateway 内网测试',
  status: 'disabled',
  schedulable: false,
  current_concurrency: 0
}

describe('KiroAccountStatusView', () => {
  beforeEach(() => {
    getStatus.mockReset()
    list.mockReset()
    testAccount.mockReset()
    recoverState.mockReset()
    setSchedulable.mockReset()
    clearRateLimit.mockReset()
    resetTempUnschedulable.mockReset()
    getTodayStats.mockReset()
    getAvailableModels.mockReset()
    activate.mockReset()

    getStatus.mockResolvedValue({
      kiro_home: '/opt/kiro-gateway',
      credentials_file_exists: true,
      credential_count: 2,
      credentials: [],
      official_quota: {
        available: true,
        source: 'AmazonCodeWhispererService.GetUsageLimits',
        fetched_at: '2026-05-17T00:00:00Z',
        account_count: 1,
        unique_user_count: 1,
        subscription_title: 'KIRO PRO',
        subscription_type: 'Q_DEVELOPER_STANDALONE_PRO',
        overage_status: 'DISABLED',
        next_reset_at: '2026-06-01T00:00:00Z',
        display_name: 'Credits',
        currency: 'USD',
        current_usage: 225.09,
        usage_limit: 1000,
        remaining: 774.91,
        overage_charges: 0,
        overage_rate: 0.04,
        accounts: [{
          credential_sha256: 'ae9184bdc6f0fc48',
          user_id: 'user-1',
          available: true,
          subscription_title: 'KIRO PRO',
          subscription_type: 'Q_DEVELOPER_STANDALONE_PRO',
          overage_status: 'DISABLED',
          next_reset_at: '2026-06-01T00:00:00Z',
          display_name: 'Credits',
          currency: 'USD',
          current_usage: 225.09,
          usage_limit: 1000,
          remaining: 774.91,
          overage_charges: 0,
          overage_rate: 0.04
        }]
      },
      activation_script_exists: true,
      cli_available: true,
      gateway_healthy: true,
      gateway_health_message: 'ok'
    })
    list.mockResolvedValue({
      items: [kiroAccount],
      total: 1,
      page: 1,
      page_size: 10,
      pages: 1
    })
    getTodayStats.mockResolvedValue({ requests: 12, tokens: 3456, cost: 0.42 })
    getAvailableModels.mockResolvedValue([
      { id: 'claude-sonnet-4.6', display_name: 'Claude Sonnet 4.6' },
      { id: 'claude-opus-4.7', display_name: 'Claude Opus 4.7' }
    ])
    testAccount.mockResolvedValue({ success: true, message: 'OK', latency_ms: 230 })
    recoverState.mockResolvedValue({ ...kiroAccount, error_message: null })
    setSchedulable.mockResolvedValue({ ...kiroAccount, schedulable: false })
    clearRateLimit.mockResolvedValue({ ...kiroAccount, rate_limit_reset_at: null })
    resetTempUnschedulable.mockResolvedValue({ message: 'ok' })
    activate.mockResolvedValue({ message: 'activated', output: 'done' })
  })

  it('loads Kiro gateway status and the matching sub2 account', async () => {
    const wrapper = mount(KiroAccountStatusView)
    await flushPromises()

    expect(getStatus).toHaveBeenCalled()
    expect(list).toHaveBeenCalledWith(1, 100, expect.objectContaining({
      platform: 'anthropic',
      type: 'apikey'
    }))
    expect(wrapper.text()).toContain('Kiro 状态总览')
    expect(wrapper.text()).toContain('Kiro 账号管理')
    expect(wrapper.text()).toContain('Kiro Gateway')
    expect(wrapper.text()).toContain('Kiro 专属线路')
    expect(wrapper.text()).toContain('Claude Opus 4.7')
    expect(wrapper.text()).toContain('Kiro 官方额度')
    expect(wrapper.text()).toContain('774.91 credits')
    expect(wrapper.text()).toContain('225.09 / 1,000 credits')
    expect(wrapper.text()).toContain('KIRO PRO')
    expect(wrapper.text()).toContain('总额度')
    expect(wrapper.text()).toContain('$75.00')
    expect(wrapper.text()).toContain('今日额度')
    expect(wrapper.text()).toContain('$14.00')
    expect(wrapper.text()).toContain('本周额度')
    expect(wrapper.text()).toContain('$43.00')
  })

  it('prioritizes the operator overview before technical diagnostics', async () => {
    const wrapper = mount(KiroAccountStatusView)
    await flushPromises()

    const text = wrapper.text()
    expect(text).toContain('额度余量')
    expect(text).toContain('账号健康')
    expect(text).toContain('线路调度')
    expect(text).toContain('账号额度明细')
    expect(text).toContain('筛选账号')
    expect(text).toContain('同步账号')
    expect(text).toContain('常用操作')
    expect(text).toContain('高级诊断')
    expect(text).toContain('77.49%')
    expect(text.indexOf('额度余量')).toBeLessThan(text.indexOf('高级诊断'))
    expect(text.indexOf('账号额度明细')).toBeLessThan(text.indexOf('高级诊断'))
    expect(text).not.toContain('status output')
    expect(text).not.toContain('ae9184bdc6f0fc48')
  })

  it('uses the same account actions as the sub2 account page', async () => {
    const wrapper = mount(KiroAccountStatusView)
    await flushPromises()

    await wrapper.get('[data-testid="kiro-test-account"]').trigger('click')
    await flushPromises()
    expect(testAccount).toHaveBeenCalledWith(180)
    expect(wrapper.text()).toContain('OK')

    await wrapper.get('[data-testid="kiro-recover-state"]').trigger('click')
    await flushPromises()
    expect(recoverState).toHaveBeenCalledWith(180)

    await wrapper.get('[data-testid="kiro-toggle-schedulable"]').trigger('click')
    await flushPromises()
    expect(setSchedulable).toHaveBeenCalledWith(180, false)
  })

  it('prefers the active production Kiro account over disabled test accounts', async () => {
    list.mockResolvedValueOnce({
      items: [disabledTestKiroAccount, kiroAccount],
      total: 2,
      page: 1,
      page_size: 10,
      pages: 1
    })

    const wrapper = mount(KiroAccountStatusView)
    await flushPromises()

    expect(getTodayStats).toHaveBeenCalledWith(180)
    expect(wrapper.text()).toContain('#180')
    expect(wrapper.text()).toContain('#177')
    expect(wrapper.text().indexOf('Sub2 AccountKiro Gatewayactive')).toBeGreaterThan(-1)
  })

  it('does not show a fake remaining quota when the Kiro account has no sub2 quota limit', async () => {
    getStatus.mockResolvedValueOnce({
      kiro_home: '/opt/kiro-gateway',
      credentials_file_exists: true,
      credential_count: 1,
      credentials: [],
      official_quota: {
        available: false,
        source: 'AmazonCodeWhispererService.GetUsageLimits',
        fetched_at: '2026-05-17T00:00:00Z',
        error: 'official quota unavailable',
        account_count: 1,
        unique_user_count: 0,
        current_usage: 0,
        usage_limit: 0,
        remaining: 0,
        overage_charges: 0,
        overage_rate: 0
      },
      activation_script_exists: true,
      cli_available: true,
      gateway_healthy: true,
      gateway_health_message: 'ok'
    })
    const noLimitAccount = {
      ...kiroAccount,
      quota_limit: undefined,
      quota_used: undefined,
      quota_daily_limit: undefined,
      quota_daily_used: undefined,
      quota_weekly_limit: undefined,
      quota_weekly_used: undefined,
      quota_daily_reset_at: undefined,
      quota_weekly_reset_at: undefined
    }
    list.mockResolvedValueOnce({
      items: [noLimitAccount],
      total: 1,
      page: 1,
      page_size: 10,
      pages: 1
    })

    const wrapper = mount(KiroAccountStatusView)
    await flushPromises()

    expect(wrapper.text()).toContain('未设置上限')
    expect(wrapper.text()).toContain('无法计算剩余')
    expect(wrapper.text()).toContain('Kiro 官方未返回可用余额')
    expect(wrapper.text()).not.toContain('未限制')
  })
})
