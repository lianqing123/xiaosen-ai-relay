import { describe, expect, it, beforeEach, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import KeysView from '../KeysView.vue'

const keysList = vi.hoisted(() => vi.fn())
const keysUpdate = vi.hoisted(() => vi.fn())
const getAvailableGroups = vi.hoisted(() => vi.fn())
const getUserGroupRates = vi.hoisted(() => vi.fn())
const getActiveSubscriptions = vi.hoisted(() => vi.fn())
const getPlans = vi.hoisted(() => vi.fn())
const updateProfile = vi.hoisted(() => vi.fn())
const showError = vi.hoisted(() => vi.fn())
const showSuccess = vi.hoisted(() => vi.fn())
const authState = vi.hoisted(() => ({
  user: {
    id: 1,
    username: 'tester',
    email: 'tester@example.com',
    role: 'user',
    balance: 12.34,
    concurrency: 5,
    status: 'active',
    allowed_groups: null,
    balance_notify_enabled: true,
    balance_notify_threshold: null,
    balance_notify_extra_emails: [],
    subscription_overage_payg_enabled: false,
    created_at: '2026-05-20T00:00:00Z',
    updated_at: '2026-05-20T00:00:00Z',
  },
  refreshUser: vi.fn(),
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, unknown>) => {
        const messages: Record<string, string> = {
          'keys.preferredSubscriptionLabel': 'Claude 扣费套餐',
          'keys.preferredSubscriptionHint': '选择 Claude 优先扣哪个套餐',
          'keys.autoSubscriptionBilling': '自动选择',
          'keys.subscriptionOptionUnlimited': `${params?.name ?? ''} 不限额`,
          'keys.claudeBillingPlan': 'Claude 扣费订阅',
          'keys.claudeBillingColumn': 'Claude扣费',
          'keys.claudeBillingAuto': '自动选择',
          'keys.claudeBillingManual': '手动指定',
          'keys.claudeBillingPlanFirst': '套餐优先',
          'keys.noClaudeBillingSubscription': '当前无可用于 Claude 扣费的套餐，请购买或续费套餐。',
          'keys.oldClaudeSubscriptionNotSelectable': '当前 Claude 订阅可直接用于扣费。',
          'keys.claudeBalanceBillingActive': '余额计费中',
          'keys.claudeBalanceBillingText': `未绑定套餐时 Claude 会直接扣账户余额。当前余额 $${params?.balance ?? ''}`,
          'keys.claudeBillingUnavailable': '无法扣费',
          'keys.claudeNoBalanceBillingText': '当前没有可用套餐，且账户余额不足。请充值余额或购买套餐后再使用 Claude。',
          'keys.claudeOveragePayg': '套餐用完扣余额',
          'keys.claudeOveragePaygOn': '已开启',
          'keys.claudeOveragePaygOff': '未开启',
          'keys.claudeOveragePaygSaved': 'Claude 超额扣余额设置已保存',
          'keys.claudeBillingPlanUpdated': 'Claude 扣费套餐已更新',
          'keys.failedToUpdateClaudeBillingPlan': '更新 Claude 扣费套餐失败',
        }
        return messages[key] || key
      },
    }),
  }
})

vi.mock('@/api', () => ({
  keysAPI: {
    list: keysList,
    update: keysUpdate,
  },
  authAPI: {
    getPublicSettings: vi.fn().mockResolvedValue({}),
  },
  usageAPI: {
    getDashboardApiKeysUsage: vi.fn().mockResolvedValue({ stats: {} }),
  },
  userGroupsAPI: {
    getAvailable: getAvailableGroups,
    getUserGroupRates,
  },
  userAPI: {
    updateProfile,
  },
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => authState,
}))

vi.mock('@/api/payment', () => ({
  paymentAPI: {
    getPlans,
  },
}))

vi.mock('@/api/subscriptions', () => ({
  default: {
    getActiveSubscriptions,
  },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError,
    showSuccess,
  }),
}))

vi.mock('@/stores/onboarding', () => ({
  useOnboardingStore: () => ({
    isCurrentStep: vi.fn(() => false),
    nextStep: vi.fn(),
  }),
}))

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({
    copyToClipboard: vi.fn(),
  }),
}))

const passthroughStub = {
  template: '<div><slot /><slot name="filters" /><slot name="actions" /><slot name="table" /><slot name="pagination" /><slot name="footer" /></div>',
}

const dataTableStub = {
  props: ['data'],
  template: `
    <div>
      <div v-for="row in data" :key="row.id" data-test="table-row">
        <slot name="cell-group" :row="row" />
        <slot name="cell-claude_billing" :row="row" />
      </div>
      <slot name="empty" />
    </div>
  `,
}

const selectStub = {
  props: ['modelValue', 'options'],
  emits: ['update:modelValue'],
  template: `
    <select
      :data-test="$attrs['data-test'] || 'select'"
      :value="modelValue ?? ''"
      @change="$emit('update:modelValue', Number($event.target.value))"
    >
      <option v-for="option in options" :key="option.value" :value="option.value">{{ option.label }}</option>
    </select>
  `,
}

const dialogStub = {
  props: ['show'],
  template: '<div v-if="show" data-test="dialog"><slot /><slot name="footer" /></div>',
}

function mountView() {
  return mount(KeysView, {
    global: {
      stubs: {
        AppLayout: passthroughStub,
        TablePageLayout: passthroughStub,
        BaseDialog: dialogStub,
        DataTable: dataTableStub,
        Pagination: true,
        ConfirmDialog: true,
        EmptyState: true,
        Select: selectStub,
        SearchInput: true,
        Icon: true,
        UseKeyModal: true,
        EndpointPopover: true,
        GroupBadge: true,
        GroupOptionItem: true,
      },
    },
  })
}

function makeClaudeKey(preferredSubscriptionGroupId = 0) {
  return {
    id: 101,
    user_id: 1,
    key: 'sk-claude',
    name: 'Claude Key',
    group_id: 24,
    preferred_subscription_group_id: preferredSubscriptionGroupId,
    status: 'active',
    ip_whitelist: [],
    ip_blacklist: [],
    last_used_at: null,
    quota: 0,
    quota_used: 0,
    expires_at: null,
    created_at: '2026-05-20T00:00:00Z',
    updated_at: '2026-05-20T00:00:00Z',
    rate_limit_5h: 0,
    rate_limit_1d: 0,
    rate_limit_7d: 0,
    usage_5h: 0,
    usage_1d: 0,
    usage_7d: 0,
    window_5h_start: null,
    window_1d_start: null,
    window_7d_start: null,
    reset_5h_at: null,
    reset_1d_at: null,
    reset_7d_at: null,
    group: {
      id: 24,
      name: 'Claude',
      description: '',
      platform: 'anthropic',
      subscription_type: 'subscription',
      rate_multiplier: 1,
    },
  }
}

describe('KeysView Claude preferred subscription selector', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    keysList.mockResolvedValue({ items: [], total: 0, pages: 0 })
    keysUpdate.mockResolvedValue({})
    updateProfile.mockImplementation(async (payload) => ({
      ...authState.user,
      ...payload,
      updated_at: '2026-05-20T00:01:00Z',
    }))
    authState.user = {
      id: 1,
      username: 'tester',
      email: 'tester@example.com',
      role: 'user',
      balance: 12.34,
      concurrency: 5,
      status: 'active',
      allowed_groups: null,
      balance_notify_enabled: true,
      balance_notify_threshold: null,
      balance_notify_extra_emails: [],
      subscription_overage_payg_enabled: false,
      created_at: '2026-05-20T00:00:00Z',
      updated_at: '2026-05-20T00:00:00Z',
    }
    authState.refreshUser.mockResolvedValue(authState.user)
    getAvailableGroups.mockResolvedValue([
      {
        id: 24,
        name: 'Claude',
        description: '',
        platform: 'anthropic',
        subscription_type: 'subscription',
        rate_multiplier: 1,
      },
      {
        id: 12,
        name: 'Codex Pro',
        description: '',
        platform: 'openai',
        subscription_type: 'subscription',
        rate_multiplier: 1,
      },
      {
        id: 19,
        name: 'Codex 商务版',
        description: '',
        platform: 'openai',
        subscription_type: 'subscription',
        rate_multiplier: 1,
      },
    ])
    getUserGroupRates.mockResolvedValue({})
    getActiveSubscriptions.mockResolvedValue([
      {
        id: 88,
        group_id: 12,
        status: 'active',
        monthly_usage_usd: 0,
        group: {
          id: 12,
          name: 'Codex Pro',
          platform: 'openai',
          subscription_type: 'subscription',
        },
      },
    ])
    getPlans.mockResolvedValue({
      data: [
        {
          id: 7,
          group_id: 12,
          included_group_ids: [24],
        },
      ],
    })
  })

  it('shows the selector when one active package can bill the Claude route', async () => {
    const wrapper = mountView()
    await flushPromises()

    ;(wrapper.vm as any).showCreateModal = true
    ;(wrapper.vm as any).formData.group_id = 24
    await nextTick()

    expect(wrapper.text()).toContain('Claude 扣费套餐')
    expect(wrapper.text()).toContain('Codex Pro')
  })

  it('uses group metadata from the group list when an active subscription omits group details', async () => {
    getActiveSubscriptions.mockResolvedValue([
      {
        id: 88,
        group_id: 12,
        status: 'active',
        monthly_usage_usd: 0,
      },
    ])
    keysList.mockResolvedValue({ items: [makeClaudeKey(0)], total: 1, pages: 1 })

    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.find('[data-test="claude-billing-select"]').text()).toContain('Codex Pro')
  })

  it('keeps an owned Codex subscription selectable when the public plan list omits that plan', async () => {
    getActiveSubscriptions.mockResolvedValue([
      {
        id: 99,
        group_id: 19,
        status: 'active',
        monthly_usage_usd: 0,
        group: {
          id: 19,
          name: 'Codex 商务版',
          platform: 'openai',
          subscription_type: 'subscription',
        },
      },
    ])
    getPlans.mockResolvedValue({
      data: [
        {
          id: 7,
          group_id: 12,
          included_group_ids: [24],
        },
      ],
    })
    keysList.mockResolvedValue({ items: [makeClaudeKey(0)], total: 1, pages: 1 })

    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.find('[data-test="claude-billing-select"]').text()).toContain('Codex 商务版')
  })

  it('shows the selected Claude billing subscription on key rows', async () => {
    keysList.mockResolvedValue({ items: [makeClaudeKey(12)], total: 1, pages: 1 })

    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.text()).toContain('Claude 扣费订阅')
    expect((wrapper.vm as any).columns.some((column: { key: string }) => column.key === 'claude_billing')).toBe(true)
    expect(wrapper.text()).toContain('手动指定')
    expect(wrapper.find('[data-test="claude-billing-select"]').text()).toContain('Codex Pro')
  })

  it('updates the Claude billing subscription from the key row selector', async () => {
    getAvailableGroups.mockResolvedValue([
      {
        id: 24,
        name: 'Claude',
        description: '',
        platform: 'anthropic',
        subscription_type: 'subscription',
        rate_multiplier: 1,
      },
      {
        id: 12,
        name: 'Codex Pro',
        description: '',
        platform: 'openai',
        subscription_type: 'subscription',
        rate_multiplier: 1,
      },
      {
        id: 19,
        name: '商务版',
        description: '',
        platform: 'openai',
        subscription_type: 'subscription',
        rate_multiplier: 1,
      },
    ])
    getActiveSubscriptions.mockResolvedValue([
      {
        id: 88,
        group_id: 12,
        status: 'active',
        monthly_usage_usd: 0,
        group: {
          id: 12,
          name: 'Codex Pro',
          platform: 'openai',
          subscription_type: 'subscription',
        },
      },
      {
        id: 99,
        group_id: 19,
        status: 'active',
        monthly_usage_usd: 0,
        group: {
          id: 19,
          name: '商务版',
          platform: 'openai',
          subscription_type: 'subscription',
        },
      },
    ])
    getPlans.mockResolvedValue({
      data: [
        { id: 7, group_id: 12, included_group_ids: [24] },
        { id: 8, group_id: 19, included_group_ids: [24] },
      ],
    })
    keysList.mockResolvedValue({ items: [makeClaudeKey(0)], total: 1, pages: 1 })

    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('[data-test="claude-billing-select"]').setValue('19')
    await flushPromises()

    expect(keysUpdate).toHaveBeenCalledWith(101, { preferred_subscription_group_id: 19 })
    expect(showSuccess).toHaveBeenCalledWith('Claude 扣费套餐已更新')
  })

  it('shows an explicit warning when a Claude key has no usable billing subscription', async () => {
    authState.user.balance = 0
    getActiveSubscriptions.mockResolvedValue([])
    getPlans.mockResolvedValue({ data: [] })
    keysList.mockResolvedValue({ items: [makeClaudeKey(0)], total: 1, pages: 1 })

    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.text()).toContain('当前没有可用套餐，且账户余额不足。请充值余额或购买套餐后再使用 Claude。')
  })

  it('shows balance billing when a Claude key has no package but the user has balance', async () => {
    authState.user.balance = 12.34
    getActiveSubscriptions.mockResolvedValue([])
    getPlans.mockResolvedValue({ data: [] })
    keysList.mockResolvedValue({ items: [makeClaudeKey(0)], total: 1, pages: 1 })

    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.text()).toContain('余额计费中')
    expect(wrapper.text()).toContain('当前余额 $12.34')
    expect(wrapper.text()).not.toContain('当前无可用于 Claude 扣费的套餐，请购买或续费套餐。')
  })

  it('lets users enable balance fallback for Claude subscription overage from the key page', async () => {
    keysList.mockResolvedValue({ items: [makeClaudeKey(0)], total: 1, pages: 1 })

    const wrapper = mountView()
    await flushPromises()

    const toggle = wrapper.find('[data-test="claude-overage-toggle"]')
    expect(toggle.exists()).toBe(true)

    await toggle.setValue(true)
    await flushPromises()

    expect(updateProfile).toHaveBeenCalledWith({ subscription_overage_payg_enabled: true })
    expect(authState.user.subscription_overage_payg_enabled).toBe(true)
    expect(showSuccess).toHaveBeenCalledWith('Claude 超额扣余额设置已保存')
  })

  it('does not show a direct Claude subscription as a billing package choice', async () => {
    getActiveSubscriptions.mockResolvedValue([
      {
        id: 124,
        group_id: 24,
        status: 'active',
        monthly_usage_usd: 0,
        group: {
          id: 24,
          name: 'Claude',
          platform: 'anthropic',
          subscription_type: 'subscription',
        },
      },
    ])
    getPlans.mockResolvedValue({
      data: [
        {
          id: 7,
          group_id: 12,
          included_group_ids: [24],
        },
      ],
    })
    keysList.mockResolvedValue({ items: [makeClaudeKey(0)], total: 1, pages: 1 })

    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.find('[data-test="claude-billing-select"]').exists()).toBe(false)
    expect(wrapper.text()).toContain('当前余额 $12.34')
    expect(wrapper.text()).not.toContain('检测到旧 Claude 订阅')
  })
})
