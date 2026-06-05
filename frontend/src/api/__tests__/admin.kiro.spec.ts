import { beforeEach, describe, expect, it, vi } from 'vitest'

const { get, post } = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
}))

vi.mock('@/api/client', () => ({
  apiClient: {
    get,
    post,
  },
}))

import {
  getCLILoginSession,
  startCLILogin,
  type KiroCLILoginInput,
  type KiroCLILoginSession,
} from '@/api/admin/kiro'

type Assert<T extends true> = T
type IsExact<T, U> = (
  (<G>() => G extends T ? 1 : 2) extends (<G>() => G extends U ? 1 : 2)
    ? ((<G>() => G extends U ? 1 : 2) extends (<G>() => G extends T ? 1 : 2) ? true : false)
    : false
)

type ExpectedKiroCLILoginInput = {
  provider: 'builder' | 'google' | 'github' | 'identity_center'
  identityProviderUrl?: string
  region?: string
  append?: boolean
  activate?: boolean
}

type ExpectedKiroCLILoginSession = {
  id: string
  status: 'starting' | 'waiting' | 'importing' | 'imported' | 'activated' | 'failed'
  provider: string
  verification_url?: string
  user_code?: string
  message?: string
  output?: string
  error?: string
  started_at: string
  updated_at: string
  credential_count?: number
}

const inputContractExact: Assert<IsExact<KiroCLILoginInput, ExpectedKiroCLILoginInput>> = true
const sessionContractExact: Assert<IsExact<KiroCLILoginSession, ExpectedKiroCLILoginSession>> = true

describe('admin kiro api cli login', () => {
  beforeEach(() => {
    get.mockReset()
    post.mockReset()
  })

  it('starts Kiro CLI device login with backend field names', async () => {
    const session: KiroCLILoginSession = {
      id: 'kiro_123',
      status: 'waiting',
      provider: 'builder',
      verification_url: 'https://device.sso.us-east-1.amazonaws.com/',
      user_code: 'ABCD-EFGH',
      started_at: '2026-05-16T00:00:00Z',
      updated_at: '2026-05-16T00:00:01Z',
    }
    post.mockResolvedValue({ data: session })

    const result = await startCLILogin({
      provider: 'identity_center',
      identityProviderUrl: 'https://example.awsapps.com/start',
      region: 'us-east-1',
      append: true,
      activate: true,
    })

    expect(post).toHaveBeenCalledWith('/admin/kiro/login/start', {
      provider: 'identity_center',
      identity_provider_url: 'https://example.awsapps.com/start',
      region: 'us-east-1',
      append: true,
      activate: true,
    }, { timeout: 180000 })
    expect(result).toEqual(session)
  })

  it('polls Kiro CLI login session by id', async () => {
    const session: KiroCLILoginSession = {
      id: 'kiro_123',
      status: 'imported',
      provider: 'builder',
      started_at: '2026-05-16T00:00:00Z',
      updated_at: '2026-05-16T00:00:01Z',
      credential_count: 1,
    }
    get.mockResolvedValue({ data: session })

    const result = await getCLILoginSession('kiro_123')

    expect(get).toHaveBeenCalledWith('/admin/kiro/login/kiro_123')
    expect(result).toEqual(session)
  })

  it('keeps login request and response types aligned with the backend contract', () => {
    expect(inputContractExact).toBe(true)
    expect(sessionContractExact).toBe(true)
  })
})
