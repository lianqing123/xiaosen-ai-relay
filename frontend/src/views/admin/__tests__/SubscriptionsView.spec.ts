import { beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent, h } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'

import SubscriptionsView from '../SubscriptionsView.vue'

const {
  listSubscriptions,
  getGroups,
  searchUsers,
  assignSubscription,
  resetQuota,
  showError,
  showSuccess
} = vi.hoisted(() => ({
  listSubscriptions: vi.fn(),
  getGroups: vi.fn(),
  searchUsers: vi.fn(),
  assignSubscription: vi.fn(),
  resetQuota: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn()
}))

function createAdminApiMock() {
  return {
  adminAPI: {
    subscriptions: {
      list: listSubscriptions,
      assign: assignSubscription,
      resetQuota
    },
    groups: {
      getAll: getGroups
    },
    usage: {
      searchUsers
    }
  }
}
}

vi.mock('@/api/admin', () => createAdminApiMock())
vi.mock('@/api/admin/index', () => createAdminApiMock())
vi.mock('@/api/admin/usage', () => ({}))
vi.mock('@/api/client', () => ({
  apiClient: {
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
    delete: vi.fn()
  }
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError,
    showSuccess
  })
}))

vi.mock('@/utils/format', () => ({
  formatDateOnly: (value: string) => value
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, unknown>) => {
        if (!params) return key
        return `${key}:${JSON.stringify(params)}`
      }
    })
  }
})

const LayoutStub = defineComponent({
  setup(_, { slots }) {
    return () => h('div', slots.default?.())
  }
})

const TablePageLayoutStub = defineComponent({
  setup(_, { slots }) {
    return () => h('div', [slots.filters?.(), slots.table?.(), slots.pagination?.()])
  }
})

const DataTableStub = defineComponent({
  setup(_, { slots }) {
    return () => h('div', slots.empty?.())
  }
})

const BaseDialogStub = defineComponent({
  props: {
    show: Boolean,
    title: String
  },
  emits: ['close'],
  setup(props, { slots }) {
    return () => (props.show ? h('section', [slots.default?.(), slots.footer?.()]) : null)
  }
})

const SelectStub = defineComponent({
  props: {
    modelValue: [String, Number],
    options: Array
  },
  emits: ['update:modelValue', 'change'],
  setup(props, { emit }) {
    return () =>
      h(
        'select',
        {
          value: props.modelValue ?? '',
          onChange: (event: Event) => {
            const value = (event.target as HTMLSelectElement).value
            const normalized = /^\d+$/.test(value) ? Number(value) : value
            emit('update:modelValue', normalized)
            emit('change', normalized)
          }
        },
        ((props.options as Array<{ value: string | number; label: string }>) || []).map((option) =>
          h('option', { value: option.value }, option.label)
        )
      )
  }
})

describe('admin SubscriptionsView assignment form', () => {
  beforeEach(() => {
    listSubscriptions.mockReset()
    getGroups.mockReset()
    searchUsers.mockReset()
    assignSubscription.mockReset()
    resetQuota.mockReset()
    showError.mockReset()
    showSuccess.mockReset()

    listSubscriptions.mockResolvedValue({
      items: [],
      total: 0,
      pages: 0,
      page: 1,
      page_size: 20
    })
    getGroups.mockResolvedValue([
      {
        id: 8,
        name: 'Codex Pro',
        description: 'subscription group',
        platform: 'openai',
        subscription_type: 'subscription',
        status: 'active',
        rate_multiplier: 1
      }
    ])
    resetQuota.mockResolvedValue({})
  })

  it('lets admins choose common validity day presets when assigning a subscription', async () => {
    const wrapper = mount(SubscriptionsView, {
      global: {
        stubs: {
          AppLayout: LayoutStub,
          TablePageLayout: TablePageLayoutStub,
          DataTable: DataTableStub,
          Pagination: true,
          BaseDialog: BaseDialogStub,
          ConfirmDialog: true,
          EmptyState: true,
          Select: SelectStub,
          GroupBadge: true,
          GroupOptionItem: true,
          Icon: true
        }
      }
    })

    await flushPromises()
    await wrapper
      .findAll('button')
      .find((button) => button.text().includes('admin.subscriptions.assignSubscription'))!
      .trigger('click')

    const preset90 = wrapper.find('[data-test="assign-validity-preset-90"]')
    expect(preset90.exists()).toBe(true)
    await preset90.trigger('click')

    const input = wrapper.find<HTMLInputElement>('[data-test="assign-validity-days-input"]')
    expect(input.element.value).toBe('90')
  })

  it('submits the selected quota reset window for an active subscription', async () => {
    const activeSubscription = {
      id: 42,
      status: 'active',
      user: { id: 7, email: 'user@example.com' },
      group: {
        id: 8,
        name: 'Codex Pro',
        platform: 'openai',
        subscription_type: 'subscription',
        daily_limit_usd: 10,
        weekly_limit_usd: 50,
        monthly_limit_usd: 100
      }
    }

    listSubscriptions.mockResolvedValue({
      items: [activeSubscription],
      total: 1,
      pages: 1,
      page: 1,
      page_size: 20
    })

    const DataTableWithActionsStub = defineComponent({
      props: {
        data: Array
      },
      setup(props, { slots }) {
        return () =>
          h(
            'div',
            ((props.data as any[]) || []).map((row) =>
              h('div', { key: row.id }, [
                slots['cell-usage']?.({ row }),
                slots['cell-actions']?.({ row })
              ])
            )
          )
      }
    })

    const wrapper = mount(SubscriptionsView, {
      global: {
        stubs: {
          AppLayout: LayoutStub,
          TablePageLayout: TablePageLayoutStub,
          DataTable: DataTableWithActionsStub,
          Pagination: true,
          BaseDialog: BaseDialogStub,
          ConfirmDialog: true,
          EmptyState: true,
          Select: SelectStub,
          GroupBadge: true,
          GroupOptionItem: true,
          Icon: true
        }
      }
    })

    await flushPromises()
    expect(wrapper.find('[data-test="reset-quota-inline-42"]').exists()).toBe(true)
    await wrapper.find('[data-test="reset-quota-inline-42"]').trigger('click')

    expect(wrapper.find('[data-test="reset-quota-option-all"]').attributes('aria-pressed')).toBe('false')
    expect(resetQuota).not.toHaveBeenCalled()

    await wrapper.find('[data-test="reset-quota-option-monthly"]').trigger('click')
    await flushPromises()

    expect(resetQuota).toHaveBeenCalledWith(42, {
      daily: false,
      weekly: false,
      monthly: true
    })
  })
})
