import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import KiroGatewayView from '../KiroGatewayView.vue'

const { activate, getStatus, importCredentials } = vi.hoisted(() => ({
  activate: vi.fn(),
  getStatus: vi.fn(),
  importCredentials: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    kiro: {
      getStatus,
      importCredentials,
      activate
    }
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

describe('KiroGatewayView token import flow', () => {
  beforeEach(() => {
    vi.useRealTimers()
    activate.mockReset()
    getStatus.mockReset()
    importCredentials.mockReset()
    getStatus.mockResolvedValue({
      kiro_home: '/opt/kiro-gateway',
      credentials_file_exists: false,
      credential_count: 0,
      credentials: [],
      activation_script_exists: true,
      cli_available: true,
      gateway_healthy: false,
      gateway_health_message: 'not started'
    })
    importCredentials.mockResolvedValue({
      message: 'credentials imported',
      credential_count: 1,
      credentials: [],
      two_factor_accepted: false
    })
    activate.mockResolvedValue({
      message: 'kiro gateway activated',
      output: 'activated'
    })
  })

  it('does not show the official login flow', async () => {
    const windowOpen = vi.spyOn(window, 'open').mockReturnValue(null)

    const wrapper = mount(KiroGatewayView)
    await flushPromises()

    expect(wrapper.text()).not.toContain('Kiro 官方网页登录')
    expect(wrapper.find('[data-testid="kiro-official-login-button"]').exists()).toBe(false)
    expect(wrapper.find('.kiro-login-link').exists()).toBe(false)
    expect(wrapper.find('input[type="password"]').exists()).toBe(false)
    expect(windowOpen).not.toHaveBeenCalled()

    windowOpen.mockRestore()
  })

  it('extracts refreshToken from a pasted Google provider token JSON', async () => {
    const wrapper = mount(KiroGatewayView)
    await flushPromises()

    await wrapper.get('textarea').setValue('{"refreshToken":"kiro-google-refresh","provider":"Google"}')
    await wrapper.get('form').trigger('submit.prevent')
    await flushPromises()

    expect(importCredentials).toHaveBeenCalledWith(expect.objectContaining({
      type: 'refresh_token',
      refreshToken: 'kiro-google-refresh',
      append: true,
      activate: false
    }))
  })

  it('rejects token JSON without refreshToken before calling the API', async () => {
    const wrapper = mount(KiroGatewayView)
    await flushPromises()

    await wrapper.get('textarea').setValue('{"provider":"Google"}')
    await wrapper.get('form').trigger('submit.prevent')
    await flushPromises()

    expect(importCredentials).not.toHaveBeenCalled()
    expect(wrapper.text()).toContain('Token JSON 缺少 refreshToken')
  })

  it('imports multiple pasted token JSON objects and activates once at the end', async () => {
    const wrapper = mount(KiroGatewayView)
    await flushPromises()

    await wrapper.get('textarea').setValue(JSON.stringify([
      { refreshToken: 'first-refresh', provider: 'Google' },
      { refreshToken: 'second-refresh', provider: 'Google' }
    ]))
    await wrapper.get('.btn.btn-primary').trigger('click')
    await flushPromises()

    expect(importCredentials).toHaveBeenCalledTimes(2)
    expect(importCredentials).toHaveBeenNthCalledWith(1, expect.objectContaining({
      type: 'refresh_token',
      refreshToken: 'first-refresh',
      append: true,
      activate: false
    }))
    expect(importCredentials).toHaveBeenNthCalledWith(2, expect.objectContaining({
      type: 'refresh_token',
      refreshToken: 'second-refresh',
      append: true,
      activate: false
    }))
    expect(activate).toHaveBeenCalledTimes(1)
  })

  it('imports line-numbered token JSON rows as separate credentials', async () => {
    const wrapper = mount(KiroGatewayView)
    await flushPromises()

    await wrapper.get('textarea').setValue([
      '11  {"refreshToken":"numbered-first","provider":"Google"}',
      '12  {"refreshToken":"numbered-second","provider":"Google"}'
    ].join('\n'))
    await wrapper.get('form').trigger('submit.prevent')
    await flushPromises()

    expect(importCredentials).toHaveBeenCalledTimes(2)
    expect(importCredentials).toHaveBeenNthCalledWith(1, expect.objectContaining({
      refreshToken: 'numbered-first'
    }))
    expect(importCredentials).toHaveBeenNthCalledWith(2, expect.objectContaining({
      refreshToken: 'numbered-second'
    }))
  })

  it('imports one plain refresh token without recursion', async () => {
    const wrapper = mount(KiroGatewayView)
    await flushPromises()

    await wrapper.get('textarea').setValue('plain-refresh-token')
    await wrapper.get('form').trigger('submit.prevent')
    await flushPromises()

    expect(importCredentials).toHaveBeenCalledTimes(1)
    expect(importCredentials).toHaveBeenCalledWith(expect.objectContaining({
      refreshToken: 'plain-refresh-token'
    }))
  })
})
