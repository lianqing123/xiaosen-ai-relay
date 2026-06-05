import { describe, expect, it, vi } from 'vitest'
import { defineComponent } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'

const { createUserMock } = vi.hoisted(() => ({
  createUserMock: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    users: {
      create: createUserMock
    }
  }
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: vi.fn(),
    showSuccess: vi.fn()
  })
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

import UserCreateModal from '../UserCreateModal.vue'

const BaseDialogStub = defineComponent({
  props: {
    show: {
      type: Boolean,
      default: false
    }
  },
  template: '<div v-if="show"><slot /><slot name="footer" /></div>'
})

describe('UserCreateModal', () => {
  it('submits new users with the API concurrency default of 5', async () => {
    createUserMock.mockReset()
    createUserMock.mockResolvedValue({})

    const wrapper = mount(UserCreateModal, {
      props: {
        show: true
      },
      global: {
        stubs: {
          BaseDialog: BaseDialogStub,
          Icon: true
        }
      }
    })

    await wrapper.find('input[type="email"]').setValue('new@example.com')
    await wrapper.find('input[type="text"]').setValue('Password123!')
    await wrapper.get('form#create-user-form').trigger('submit.prevent')
    await flushPromises()

    expect(createUserMock).toHaveBeenCalledTimes(1)
    expect(createUserMock.mock.calls[0]?.[0]).toEqual(expect.objectContaining({
      email: 'new@example.com',
      concurrency: 5
    }))
  })
})
