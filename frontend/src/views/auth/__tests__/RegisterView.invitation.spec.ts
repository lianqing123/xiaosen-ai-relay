import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import RegisterView from '@/views/auth/RegisterView.vue'

const {
  routeState,
  getPublicSettingsMock,
  validateInvitationCodeMock,
  showErrorMock
} = vi.hoisted(() => ({
  routeState: {
    query: {} as Record<string, string | string[] | null | undefined>
  },
  getPublicSettingsMock: vi.fn(),
  validateInvitationCodeMock: vi.fn(),
  showErrorMock: vi.fn()
}))

vi.mock('vue-router', () => ({
  useRouter: () => ({
    push: vi.fn()
  }),
  useRoute: () => routeState
}))

vi.mock('vue-i18n', () => ({
  createI18n: () => ({
    global: {
      t: (key: string) => key
    }
  }),
  useI18n: () => ({
    t: (key: string) => key,
    locale: { value: 'zh' }
  })
}))

vi.mock('@/stores', () => ({
  useAuthStore: () => ({
    register: vi.fn()
  }),
  useAppStore: () => ({
    showSuccess: vi.fn(),
    showError: (...args: any[]) => showErrorMock(...args)
  })
}))

vi.mock('@/api/auth', () => ({
  getPublicSettings: (...args: any[]) => getPublicSettingsMock(...args),
  isWeChatWebOAuthEnabled: () => false,
  validatePromoCode: vi.fn(),
  validateInvitationCode: (...args: any[]) => validateInvitationCodeMock(...args)
}))

describe('RegisterView invitation links', () => {
  beforeEach(() => {
    routeState.query = {}
    getPublicSettingsMock.mockReset()
    validateInvitationCodeMock.mockReset()
    showErrorMock.mockReset()
    getPublicSettingsMock.mockResolvedValue({
      registration_enabled: true,
      email_verify_enabled: false,
      promo_code_enabled: false,
      invitation_code_enabled: true,
      turnstile_enabled: false,
      turnstile_site_key: '',
      site_name: 'Codex Access',
      linuxdo_oauth_enabled: false,
      wechat_oauth_enabled: false,
      oidc_oauth_enabled: false,
      oidc_oauth_provider_name: 'OIDC',
      registration_email_suffix_whitelist: []
    })
    validateInvitationCodeMock.mockResolvedValue({ valid: true })
  })

  it('prefills and validates invitation_code from shared register links', async () => {
    routeState.query = {
      invitation_code: ' INVITE-123 '
    }

    const wrapper = mount(RegisterView, {
      global: {
        stubs: {
          AuthLayout: { template: '<div><slot /><slot name="footer" /></div>' },
          Icon: true,
          TurnstileWidget: true,
          LinuxDoOAuthSection: true,
          WechatOAuthSection: true,
          OidcOAuthSection: true,
          RouterLink: true,
          transition: false
        }
      }
    })

    await flushPromises()

    expect((wrapper.get('#invitation_code').element as HTMLInputElement).value).toBe('INVITE-123')
    expect(validateInvitationCodeMock).toHaveBeenCalledWith('INVITE-123')
  })

  it('prefills and validates old affiliate links as invitation codes', async () => {
    routeState.query = {
      aff: ' AFF123 '
    }

    const wrapper = mount(RegisterView, {
      global: {
        stubs: {
          AuthLayout: { template: '<div><slot /><slot name="footer" /></div>' },
          Icon: true,
          TurnstileWidget: true,
          LinuxDoOAuthSection: true,
          WechatOAuthSection: true,
          OidcOAuthSection: true,
          RouterLink: true,
          transition: false
        }
      }
    })

    await flushPromises()

    expect((wrapper.get('#invitation_code').element as HTMLInputElement).value).toBe('AFF123')
    expect(validateInvitationCodeMock).toHaveBeenCalledWith('AFF123')
  })
})
