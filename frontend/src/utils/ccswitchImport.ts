import type { ApiKey } from '@/types'

type CcsClientType = 'claude' | 'gemini'

interface BuildCcswitchProviderImportUrlOptions {
  row: ApiKey
  baseUrl: string
  providerName: string
  clientType?: CcsClientType
  usageScript?: string
}

const KIRO_MODELS = {
  main: 'claude-sonnet-4.6',
  sonnet: 'claude-sonnet-4.6',
  haiku: 'claude-haiku-4.5',
  opus: 'claude-opus-4.7'
} as const

export function buildCcswitchProviderImportUrl({
  row,
  baseUrl,
  providerName,
  clientType = 'claude',
  usageScript
}: BuildCcswitchProviderImportUrlOptions): string {
  const platform = row.group?.platform || 'anthropic'
  const endpoint = resolveCcsEndpoint(baseUrl, platform)
  const app = resolveCcsApp(platform, clientType)

  const params = new URLSearchParams({
    resource: 'provider',
    app,
    name: providerName,
    homepage: baseUrl,
    endpoint,
    apiKey: row.key,
    usageEnabled: 'true',
    usageAutoInterval: '30'
  })

  if (usageScript) {
    params.set('usageScript', encodeBase64Utf8(usageScript))
  }

  if (isKiroKey(row, platform) && app === 'claude') {
    appendKiroClaudeConfig(params, row.key, endpoint)
  }

  return `ccswitch://v1/import?${params.toString()}`
}

function resolveCcsApp(platform: string, clientType: CcsClientType): string {
  if (platform === 'antigravity') {
    return clientType === 'gemini' ? 'gemini' : 'claude'
  }

  if (platform === 'openai') return 'codex'
  if (platform === 'gemini') return 'gemini'
  return 'claude'
}

function resolveCcsEndpoint(baseUrl: string, platform: string): string {
  if (platform === 'antigravity') {
    return `${trimTrailingSlash(baseUrl)}/antigravity`
  }

  return trimTrailingSlash(baseUrl)
}

function isKiroKey(row: ApiKey, platform: string): boolean {
  if (platform !== 'anthropic') return false
  const groupName = row.group?.name || ''
  return row.group_id === 24 || /kiro/i.test(groupName) || groupName.includes('Kiro')
}

function appendKiroClaudeConfig(params: URLSearchParams, apiKey: string, endpoint: string): void {
  params.set('model', KIRO_MODELS.main)
  params.set('sonnetModel', KIRO_MODELS.sonnet)
  params.set('haikuModel', KIRO_MODELS.haiku)
  params.set('opusModel', KIRO_MODELS.opus)
  params.set('configFormat', 'json')
  params.set(
    'config',
    encodeBase64Utf8(
      JSON.stringify({
        env: {
          ANTHROPIC_API_KEY: apiKey,
          ANTHROPIC_AUTH_TOKEN: apiKey,
          ANTHROPIC_BASE_URL: endpoint,
          ANTHROPIC_MODEL: KIRO_MODELS.main,
          ANTHROPIC_DEFAULT_SONNET_MODEL: KIRO_MODELS.sonnet,
          ANTHROPIC_DEFAULT_HAIKU_MODEL: KIRO_MODELS.haiku,
          ANTHROPIC_DEFAULT_OPUS_MODEL: KIRO_MODELS.opus
        }
      })
    )
  )
}

function trimTrailingSlash(value: string): string {
  return value.replace(/\/+$/, '')
}

function encodeBase64Utf8(value: string): string {
  if (typeof btoa === 'function') {
    const bytes = new TextEncoder().encode(value)
    let binary = ''
    for (const byte of bytes) {
      binary += String.fromCharCode(byte)
    }
    return btoa(binary)
  }

  return encodeBase64Bytes(new TextEncoder().encode(value))
}

function encodeBase64Bytes(bytes: Uint8Array): string {
  const alphabet = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/'
  let output = ''

  for (let i = 0; i < bytes.length; i += 3) {
    const a = bytes[i]
    const b = bytes[i + 1]
    const c = bytes[i + 2]
    const triple = (a << 16) | ((b || 0) << 8) | (c || 0)

    output += alphabet[(triple >> 18) & 63]
    output += alphabet[(triple >> 12) & 63]
    output += i + 1 < bytes.length ? alphabet[(triple >> 6) & 63] : '='
    output += i + 2 < bytes.length ? alphabet[triple & 63] : '='
  }

  return output
}
