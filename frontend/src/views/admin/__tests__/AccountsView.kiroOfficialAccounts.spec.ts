import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

import { describe, expect, it } from 'vitest'

const source = readFileSync(resolve(__dirname, '../AccountsView.vue'), 'utf8')

describe('AccountsView Kiro scope', () => {
  it('keeps official Kiro account quota rows visible beside the native account table', () => {
    expect(source).toContain('loadKiroOfficialStatus')
    expect(source).toContain('adminAPI.kiro.getStatus')
    expect(source).toContain('officialQuotaRows')
    expect(source).toContain('Kiro 官方账号池')
    expect(source).toContain('账号额度明细')
  })
})
