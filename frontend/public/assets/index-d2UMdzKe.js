(function () {
  async function clearStalePwaState() {
    try {
      if ('serviceWorker' in navigator) {
        const registrations = await navigator.serviceWorker.getRegistrations()
        await Promise.all(registrations.map((registration) => registration.unregister()))
      }
    } catch (_) {
      // Best-effort stale PWA recovery.
    }

    try {
      if ('caches' in window) {
        const cacheNames = await caches.keys()
        await Promise.all(
          cacheNames
            .filter((name) => name.startsWith('codex-access-mobile-admin-'))
            .map((name) => caches.delete(name))
        )
      }
    } catch (_) {
      // Best-effort stale PWA recovery.
    }
  }

  clearStalePwaState().finally(function () {
    const url = new URL(window.location.href)
    url.searchParams.set('pwa_recovered_at', String(Date.now()))
    window.location.replace(url.toString())
  })
})()
