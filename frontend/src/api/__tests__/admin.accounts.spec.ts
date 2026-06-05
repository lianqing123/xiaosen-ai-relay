import { beforeEach, describe, expect, it, vi } from 'vitest'

const { get } = vi.hoisted(() => ({
  get: vi.fn(),
}))

vi.mock('@/api/client', () => ({
  apiClient: {
    get,
  },
}))

import { list } from '@/api/admin/accounts'

describe('admin accounts api', () => {
  beforeEach(() => {
    get.mockReset()
  })

  it('passes source filters to the admin account list endpoint', async () => {
    get.mockResolvedValue({
      data: {
        items: [],
        total: 0,
        page: 1,
        page_size: 20,
        pages: 0,
      },
    })

    await list(1, 20, {
      platform: 'anthropic',
      type: 'apikey',
      source: 'kiro-gateway',
    })

    expect(get).toHaveBeenCalledWith('/admin/accounts', {
      params: {
        page: 1,
        page_size: 20,
        platform: 'anthropic',
        type: 'apikey',
        source: 'kiro-gateway',
      },
      signal: undefined,
    })
  })
})
