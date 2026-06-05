import { describe, expect, it } from 'vitest'

import {
  getMobileAdminInstallState,
  isIosLikeDevice,
  isStandaloneDisplay
} from '../mobileAdminPwa'

describe('mobile admin PWA helpers', () => {
  it('detects iPhone Safari as an installable non-standalone session', () => {
    const state = getMobileAdminInstallState({
      userAgent:
        'Mozilla/5.0 (iPhone; CPU iPhone OS 17_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.4 Mobile/15E148 Safari/604.1',
      standalone: false,
      displayModeStandalone: false
    })

    expect(state.isIos).toBe(true)
    expect(state.isStandalone).toBe(false)
    expect(state.canShowInstallHint).toBe(true)
  })

  it('hides install hints once launched from the iOS home screen', () => {
    const state = getMobileAdminInstallState({
      userAgent:
        'Mozilla/5.0 (iPhone; CPU iPhone OS 17_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.4 Mobile/15E148 Safari/604.1',
      standalone: true,
      displayModeStandalone: true
    })

    expect(state.isStandalone).toBe(true)
    expect(state.canShowInstallHint).toBe(false)
  })

  it('detects iPadOS desktop-class Safari as iOS-like', () => {
    expect(isIosLikeDevice({
      userAgent:
        'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.4 Safari/605.1.15',
      platform: 'MacIntel',
      maxTouchPoints: 5
    })).toBe(true)
  })

  it('supports display-mode standalone detection without navigator.standalone', () => {
    expect(isStandaloneDisplay({
      standalone: false,
      displayModeStandalone: true
    })).toBe(true)
  })
})
