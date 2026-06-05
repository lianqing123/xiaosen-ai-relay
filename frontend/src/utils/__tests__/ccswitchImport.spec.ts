import { describe, expect, it } from 'vitest'

import { buildCcswitchProviderImportUrl } from '../ccswitchImport'
import type { ApiKey } from '@/types'

describe('buildCcswitchProviderImportUrl', () => {
  it('exports Kiro Claude provider with concrete models and config', () => {
    const row = {
      id: 347,
      key: 'sk-test-kiro',
      name: 'kiro',
      group_id: 24,
      group: {
        id: 24,
        name: 'Kiro 专属线路',
        platform: 'anthropic'
      }
    } as ApiKey

    const url = buildCcswitchProviderImportUrl({
      row,
      baseUrl: 'https://api.zhongzhuan.pro',
      providerName: '中转站',
      clientType: 'claude'
    })

    const parsed = new URL(url)
    expect(parsed.protocol).toBe('ccswitch:')
    expect(parsed.hostname).toBe('v1')
    expect(parsed.pathname).toBe('/import')

    const params = parsed.searchParams
    expect(params.get('resource')).toBe('provider')
    expect(params.get('app')).toBe('claude')
    expect(params.get('endpoint')).toBe('https://api.zhongzhuan.pro')
    expect(params.get('apiKey')).toBe('sk-test-kiro')
    expect(params.get('model')).toBe('claude-sonnet-4.6')
    expect(params.get('sonnetModel')).toBe('claude-sonnet-4.6')
    expect(params.get('haikuModel')).toBe('claude-haiku-4.5')
    expect(params.get('opusModel')).toBe('claude-opus-4.7')
    expect(params.get('configFormat')).toBe('json')

    const config = JSON.parse(decodeBase64Utf8(params.get('config') || ''))
    expect(config.env).toMatchObject({
      ANTHROPIC_API_KEY: 'sk-test-kiro',
      ANTHROPIC_AUTH_TOKEN: 'sk-test-kiro',
      ANTHROPIC_BASE_URL: 'https://api.zhongzhuan.pro',
      ANTHROPIC_MODEL: 'claude-sonnet-4.6',
      ANTHROPIC_DEFAULT_SONNET_MODEL: 'claude-sonnet-4.6',
      ANTHROPIC_DEFAULT_HAIKU_MODEL: 'claude-haiku-4.5',
      ANTHROPIC_DEFAULT_OPUS_MODEL: 'claude-opus-4.7'
    })
  })
})

function decodeBase64Utf8(value: string): string {
  const binary = atob(value)
  const bytes = Uint8Array.from(binary, (char) => char.charCodeAt(0))
  return new TextDecoder().decode(bytes)
}
