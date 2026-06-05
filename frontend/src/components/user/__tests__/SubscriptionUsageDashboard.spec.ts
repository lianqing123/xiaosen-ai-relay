import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import SubscriptionUsageDashboard from '@/components/user/SubscriptionUsageDashboard.vue'
import type { UserSubscription } from '@/types'

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

function makeSubscription(overrides: Partial<UserSubscription> = {}): UserSubscription {
  return {
    id: 1,
    user_id: 7,
    group_id: 3,
    status: 'active',
    daily_usage_usd: 10,
    weekly_usage_usd: 25,
    monthly_usage_usd: 75,
    daily_window_start: '2026-04-30T00:00:00Z',
    weekly_window_start: '2026-04-27T00:00:00Z',
    monthly_window_start: '2026-04-01T00:00:00Z',
    created_at: '2026-04-20T00:00:00Z',
    updated_at: '2026-04-30T00:00:00Z',
    expires_at: '2026-05-30T00:00:00Z',
    group: {
      id: 3,
      name: 'Codex Standard',
      description: null,
      platform: 'openai',
      rate_multiplier: 0.6,
      is_exclusive: false,
      status: 'active',
      subscription_type: 'subscription',
      daily_limit_usd: 20,
      weekly_limit_usd: 100,
      monthly_limit_usd: 100,
      image_price_1k: null,
      image_price_2k: null,
      image_price_4k: null,
      claude_code_only: false,
      fallback_group_id: null,
      fallback_group_id_on_invalid_request: null,
      require_oauth_only: false,
      require_privacy_set: false,
      created_at: '2026-04-20T00:00:00Z',
      updated_at: '2026-04-20T00:00:00Z'
    },
    ...overrides
  }
}

describe('SubscriptionUsageDashboard', () => {
  it('renders daily, weekly, and monthly quota ratios for active subscriptions', () => {
    const wrapper = mount(SubscriptionUsageDashboard, {
      props: {
        subscriptions: [makeSubscription()]
      }
    })

    expect(wrapper.text()).toContain('userSubscriptions.usageDashboard.title')
    expect(wrapper.text()).toContain('50%')
    expect(wrapper.text()).toContain('25%')
    expect(wrapper.text()).toContain('75%')
    expect(wrapper.text()).toContain('$10.00 / $20.00')
    expect(wrapper.text()).toContain('$25.00 / $100.00')
    expect(wrapper.text()).toContain('$75.00 / $100.00')
  })

  it('shows an unlimited state when active subscriptions have no quota limits', () => {
    const wrapper = mount(SubscriptionUsageDashboard, {
      props: {
        subscriptions: [
          makeSubscription({
            daily_usage_usd: 0,
            weekly_usage_usd: 0,
            monthly_usage_usd: 0,
            group: {
              ...makeSubscription().group!,
              daily_limit_usd: null,
              weekly_limit_usd: null,
              monthly_limit_usd: null
            }
          })
        ]
      }
    })

    expect(wrapper.text()).toContain('userSubscriptions.usageDashboard.unlimited')
  })

  it('does not treat inactive subscriptions as unlimited usage', () => {
    const wrapper = mount(SubscriptionUsageDashboard, {
      props: {
        subscriptions: [
          makeSubscription({
            status: 'expired'
          })
        ]
      }
    })

    expect(wrapper.text()).toContain('userSubscriptions.usageDashboard.noActive')
    expect(wrapper.text()).not.toContain('userSubscriptions.usageDashboard.unlimited')
  })
})
