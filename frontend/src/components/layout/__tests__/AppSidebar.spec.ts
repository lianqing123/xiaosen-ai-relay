import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const componentPath = resolve(dirname(fileURLToPath(import.meta.url)), '../AppSidebar.vue')
const componentSource = readFileSync(componentPath, 'utf8')
const stylePath = resolve(dirname(fileURLToPath(import.meta.url)), '../../../style.css')
const styleSource = readFileSync(stylePath, 'utf8')

describe('AppSidebar custom SVG styles', () => {
  it('does not override uploaded SVG fill or stroke colors', () => {
    expect(componentSource).toContain('.sidebar-svg-icon {')
    expect(componentSource).toContain('color: currentColor;')
    expect(componentSource).toContain('display: block;')
    expect(componentSource).not.toContain('stroke: currentColor;')
    expect(componentSource).not.toContain('fill: none;')
  })
})

describe('AppSidebar header styles', () => {
  it('does not clip the version badge dropdown', () => {
    const sidebarHeaderBlockMatch = styleSource.match(/\.sidebar-header\s*\{[\s\S]*?\n {2}\}/)
    const sidebarBrandBlockMatch = componentSource.match(/\.sidebar-brand\s*\{[\s\S]*?\n\}/)

    expect(sidebarHeaderBlockMatch).not.toBeNull()
    expect(sidebarBrandBlockMatch).not.toBeNull()
    expect(sidebarHeaderBlockMatch?.[0]).not.toContain('@apply overflow-hidden;')
    expect(sidebarBrandBlockMatch?.[0]).not.toContain('overflow: hidden;')
  })
})

describe('AppSidebar admin navigation', () => {
  it('shows Kiro Gateway as a direct admin menu entry', () => {
    const channelGroupMatch = componentSource.match(
      /path: '\/admin\/channels'[\s\S]*?children:\s*\[([\s\S]*?)\n\s*\],\n\s*\},/
    )

    expect(channelGroupMatch).not.toBeNull()
    expect(channelGroupMatch?.[1]).not.toContain("/admin/channels/kiro")
    expect(componentSource).toContain(
      "    { path: '/admin/channels/kiro', label: t('nav.kiroGateway'), icon: ServerIcon },\n" +
        "    { path: '/admin/kiro/status', label: t('nav.kiroAccountStatus'), icon: SignalIcon },\n" +
        "    { path: '/admin/subscriptions', label: t('nav.subscriptions'), icon: CreditCardIcon },"
    )
  })
})
