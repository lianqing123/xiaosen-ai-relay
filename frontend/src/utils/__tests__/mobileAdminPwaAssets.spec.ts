import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const testDir = dirname(fileURLToPath(import.meta.url))
const mainSource = readFileSync(resolve(testDir, '../../main.ts'), 'utf8')
const serviceWorkerSource = readFileSync(resolve(testDir, '../../../public/pwa-sw.js'), 'utf8')
const staleEntryBootloaders = [
  '../../../public/assets/index-d2UMdzKe.js',
  '../../../public/assets/index-CaL1l5UN.js'
].map((relativePath) => readFileSync(resolve(testDir, relativePath), 'utf8'))

describe('mobile admin PWA asset cache busting', () => {
  it('uses the current service worker registration and cache version', () => {
    expect(mainSource).toContain("navigator.serviceWorker.register('/pwa-sw.js?v=20260522-kiro-native-v32'")
    expect(mainSource).toContain("const CURRENT_PWA_CACHE_NAME = 'codex-access-mobile-admin-v32'")
    expect(mainSource).toContain('clearLegacyPwaCaches')
    expect(mainSource).toContain('window.location.reload()')
    expect(serviceWorkerSource).toContain("const CACHE_NAME = 'codex-access-mobile-admin-v32'")
    expect(serviceWorkerSource).toContain("event.data.type === 'SKIP_WAITING'")
    expect(serviceWorkerSource).toContain('client.navigate(client.url)')
    expect(serviceWorkerSource).toContain("url.pathname.startsWith('/assets/')")
  })

  it('keeps known stale entry bootloaders for cached clients', () => {
    for (const source of staleEntryBootloaders) {
      expect(source).toContain('clearStalePwaState')
      expect(source).toContain('navigator.serviceWorker.getRegistrations')
      expect(source).toContain("name.startsWith('codex-access-mobile-admin-')")
      expect(source).toContain('pwa_recovered_at')
      expect(source).toContain('window.location.replace')
    }
  })
})
