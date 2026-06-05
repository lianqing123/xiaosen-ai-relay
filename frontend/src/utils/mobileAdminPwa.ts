export interface MobileAdminDeviceSignals {
  userAgent?: string
  platform?: string
  maxTouchPoints?: number
  standalone?: boolean
  displayModeStandalone?: boolean
}

export interface MobileAdminInstallState {
  isIos: boolean
  isStandalone: boolean
  canShowInstallHint: boolean
}

export function isIosLikeDevice(signals: MobileAdminDeviceSignals = readBrowserSignals()): boolean {
  const userAgent = signals.userAgent || ''
  const platform = signals.platform || ''
  const maxTouchPoints = signals.maxTouchPoints || 0

  return /iPhone|iPad|iPod/i.test(userAgent) || (platform === 'MacIntel' && maxTouchPoints > 1)
}

export function isStandaloneDisplay(signals: MobileAdminDeviceSignals = readBrowserSignals()): boolean {
  return Boolean(signals.standalone || signals.displayModeStandalone)
}

export function getMobileAdminInstallState(
  signals: MobileAdminDeviceSignals = readBrowserSignals()
): MobileAdminInstallState {
  const isIos = isIosLikeDevice(signals)
  const isStandalone = isStandaloneDisplay(signals)

  return {
    isIos,
    isStandalone,
    canShowInstallHint: isIos && !isStandalone
  }
}

export function readBrowserSignals(): MobileAdminDeviceSignals {
  if (typeof window === 'undefined' || typeof navigator === 'undefined') {
    return {}
  }

  return {
    userAgent: navigator.userAgent,
    platform: navigator.platform,
    maxTouchPoints: navigator.maxTouchPoints,
    standalone: Boolean((navigator as Navigator & { standalone?: boolean }).standalone),
    displayModeStandalone: window.matchMedia?.('(display-mode: standalone)').matches ?? false
  }
}
