import { beforeEach, describe, expect, it, vi } from 'vitest'

const { get } = vi.hoisted(() => ({
  get: vi.fn(),
}))

vi.mock('@/api/client', () => ({
  apiClient: {
    get,
  },
}))

import subscriptionsAPI from '@/api/subscriptions'

const codexSubscription = {
  id: 1,
  user_id: 1,
  group_id: 19,
  status: 'active' as const,
  daily_usage_usd: 0,
  weekly_usage_usd: 0,
  monthly_usage_usd: 2.89,
  daily_window_start: null,
  weekly_window_start: null,
  monthly_window_start: null,
  created_at: '2026-05-20T00:00:00Z',
  updated_at: '2026-05-20T00:00:00Z',
  expires_at: '2026-06-20T00:00:00Z',
  group: {
    id: 19,
    name: 'Codex 商务版',
    platform: 'openai',
    subscription_type: 'subscription',
  },
}

const standaloneClaudeSubscription = {
  id: 2,
  user_id: 1,
  group_id: 24,
  status: 'active' as const,
  daily_usage_usd: 0,
  weekly_usage_usd: 0,
  monthly_usage_usd: 0,
  daily_window_start: null,
  weekly_window_start: null,
  monthly_window_start: null,
  created_at: '2026-05-20T00:00:00Z',
  updated_at: '2026-05-20T00:00:00Z',
  expires_at: '2026-06-20T00:00:00Z',
  group: {
    id: 24,
    name: 'Claude',
    platform: 'anthropic',
    subscription_type: 'subscription',
  },
}

describe('subscriptions api', () => {
  beforeEach(() => {
    get.mockReset()
  })

  it('hides standalone Claude subscriptions from user-facing subscription lists', async () => {
    get.mockResolvedValue({ data: [codexSubscription, standaloneClaudeSubscription] })

    await expect(subscriptionsAPI.getActiveSubscriptions()).resolves.toEqual([codexSubscription])
    await expect(subscriptionsAPI.getMySubscriptions()).resolves.toEqual([codexSubscription])
  })
})
