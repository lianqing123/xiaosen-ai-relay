import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import router from './router'
import i18n, { initI18n } from './i18n'
import { useAppStore } from '@/stores/app'
import './style.css'

function initThemeClass() {
  const savedTheme = localStorage.getItem('theme')
  const shouldUseDark =
    savedTheme === 'dark' ||
    (!savedTheme && window.matchMedia('(prefers-color-scheme: dark)').matches)
  document.documentElement.classList.toggle('dark', shouldUseDark)
}

const PWA_CACHE_PREFIX = 'codex-access-mobile-admin-'
const CURRENT_PWA_CACHE_NAME = 'codex-access-mobile-admin-v32'
const PWA_FORCE_REFRESH_FLAG = 'pwa-claude-billing-v32-refreshed'

async function clearLegacyPwaCaches() {
  if (!('caches' in window)) {
    return
  }

  try {
    const cacheNames = await caches.keys()
    await Promise.all(
      cacheNames
        .filter((name) => name.startsWith(PWA_CACHE_PREFIX) && name !== CURRENT_PWA_CACHE_NAME)
        .map((name) => caches.delete(name))
    )
  } catch (error) {
    console.warn('[pwa] legacy cache cleanup failed:', error)
  }
}

function registerPwaServiceWorker() {
  if (import.meta.env.DEV || !('serviceWorker' in navigator)) {
    return
  }

  window.addEventListener('load', async () => {
    await clearLegacyPwaCaches()

    navigator.serviceWorker.register('/pwa-sw.js?v=20260522-kiro-native-v32', {
      scope: '/',
      updateViaCache: 'none'
    }).then((registration) => {
      registration.update().catch(() => undefined)
      registration.waiting?.postMessage({ type: 'SKIP_WAITING' })

      if (navigator.serviceWorker.controller && !sessionStorage.getItem(PWA_FORCE_REFRESH_FLAG)) {
        sessionStorage.setItem(PWA_FORCE_REFRESH_FLAG, '1')
        window.location.reload()
      }
    }).catch((error) => {
      console.warn('[pwa] service worker registration failed:', error)
    })
  })
}

async function bootstrap() {
  // Apply theme class globally before app mount to keep all routes consistent.
  initThemeClass()

  const app = createApp(App)
  const pinia = createPinia()
  app.use(pinia)

  // Initialize settings from injected config BEFORE mounting (prevents flash)
  // This must happen after pinia is installed but before router and i18n
  const appStore = useAppStore()
  appStore.initFromInjectedConfig()

  // Set document title immediately after config is loaded
  if (appStore.siteName && appStore.siteName !== 'Sub2API') {
    document.title = appStore.siteName
  }

  await initI18n()

  app.use(router)
  app.use(i18n)

  // 等待路由器完成初始导航后再挂载，避免竞态条件导致的空白渲染
  await router.isReady()
  app.mount('#app')
  registerPwaServiceWorker()
}

bootstrap()
