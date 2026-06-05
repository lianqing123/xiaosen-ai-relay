import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import MobileAdminView from '../MobileAdminView.vue'

const {
  getSnapshotV2,
  getRealtimeMetrics,
  previewInactiveBalanceCleanup,
  listUsers,
  listChannelMonitors,
  getVersion,
  push
} = vi.hoisted(() => ({
  getSnapshotV2: vi.fn(),
  getRealtimeMetrics: vi.fn(),
  previewInactiveBalanceCleanup: vi.fn(),
  listUsers: vi.fn(),
  listChannelMonitors: vi.fn(),
  getVersion: vi.fn(),
  push: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    dashboard: {
      getSnapshotV2,
      getRealtimeMetrics
    },
    users: {
      previewInactiveBalanceCleanup,
      list: listUsers
    },
    channelMonitor: {
      list: listChannelMonitors
    },
    system: {
      getVersion
    }
  }
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({
    user: {
      id: 1,
      email: 'admin@example.com',
      username: 'Admin',
      role: 'admin'
    }
  })
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: vi.fn(),
    siteName: 'Codex Access'
  })
}))

vi.mock('vue-router', () => ({
  useRouter: () => ({ push })
}))

describe('MobileAdminView', () => {
  beforeEach(() => {
    getSnapshotV2.mockReset()
    getRealtimeMetrics.mockReset()
    previewInactiveBalanceCleanup.mockReset()
    listUsers.mockReset()
    listChannelMonitors.mockReset()
    getVersion.mockReset()
    push.mockReset()

    getSnapshotV2.mockResolvedValue({
      stats: {
        total_users: 87,
        today_new_users: 6,
        active_users: 24,
        hourly_active_users: 5,
        stats_updated_at: '2026-05-11T08:30:00Z',
        stats_stale: false,
        total_api_keys: 42,
        active_api_keys: 39,
        total_accounts: 18,
        normal_accounts: 15,
        error_accounts: 2,
        ratelimit_accounts: 1,
        overload_accounts: 0,
        total_requests: 120400,
        total_input_tokens: 2300,
        total_output_tokens: 1700,
        total_cache_creation_tokens: 300,
        total_cache_read_tokens: 900,
        total_tokens: 5200,
        total_cost: 281.4,
        total_actual_cost: 198.2,
        total_account_cost: 121.8,
        today_requests: 1432,
        today_input_tokens: 300,
        today_output_tokens: 210,
        today_cache_creation_tokens: 40,
        today_cache_read_tokens: 90,
        today_tokens: 640,
        today_cost: 14.5,
        today_actual_cost: 9.8,
        today_account_cost: 5.4,
        average_duration_ms: 842,
        uptime: 86400,
        rpm: 19.7,
        tpm: 8400
      },
      trend: [],
      models: []
    })
    getRealtimeMetrics.mockResolvedValue({
      active_requests: 3,
      requests_per_minute: 21.4,
      average_response_time: 836,
      error_rate: 0.018
    })
    previewInactiveBalanceCleanup.mockResolvedValue({
      days: 3,
      cutoff_at: '2026-05-08T00:00:00Z',
      count: 12,
      total_balance: 328.42,
      candidates: []
    })
    listUsers.mockResolvedValue({
      items: [],
      total: 0,
      page: 1,
      page_size: 6,
      pages: 0
    })
    listChannelMonitors.mockResolvedValue({
      items: [
        {
          id: 1,
          name: 'OpenAI 主通道',
          provider: 'openai',
          endpoint: 'https://api.openai.com',
          api_key_masked: 'sk-***',
          primary_model: 'gpt-5.1',
          extra_models: [],
          group_name: 'default',
          enabled: true,
          interval_seconds: 60,
          last_checked_at: '2026-05-11T08:32:00Z',
          created_by: 1,
          created_at: '2026-05-01T00:00:00Z',
          updated_at: '2026-05-11T08:32:00Z',
          primary_status: 'operational',
          primary_latency_ms: 720,
          availability_7d: 99.2,
          extra_models_status: [],
          template_id: null,
          extra_headers: {},
          body_override_mode: 'off',
          body_override: null
        }
      ],
      total: 1,
      page: 1,
      page_size: 5,
      pages: 1
    })
    getVersion.mockResolvedValue({ version: '2026.05.11' })
  })

  it('loads the mobile admin overview from existing admin APIs', async () => {
    const wrapper = mount(MobileAdminView, {
      global: {
        stubs: {
          Icon: true,
          RouterLink: true
        }
      }
    })

    await flushPromises()

    expect(getSnapshotV2).toHaveBeenCalledWith(expect.objectContaining({
      include_stats: true,
      include_trend: true,
      include_model_stats: true
    }))
    expect(previewInactiveBalanceCleanup).toHaveBeenCalledWith(3, 5)
    expect(listChannelMonitors).toHaveBeenCalledWith({ page: 1, page_size: 5 })
    expect(wrapper.text()).toContain('手机控制台')
    expect(wrapper.text()).toContain('1,432')
    expect(wrapper.text()).toContain('配额重置')
    expect(wrapper.text()).toContain('闲置余额')
    expect(wrapper.text()).toContain('OpenAI 主通道')
  })
})
