<template>
  <AppLayout>
    <div class="custom-page-layout">
      <div class="custom-page-frame">
        <div class="custom-page-frame-head">
          <div>
            <span>EMBEDDED PAGE</span>
            <strong>{{ menuItem?.label || t('customPage.title') }}</strong>
          </div>
          <p>外部页面以嵌入方式打开，权限、Token 和主题参数按当前登录态注入。</p>
        </div>

        <div v-if="loading" class="flex h-full items-center justify-center py-12">
          <div class="custom-loading">
            <span></span>
            <p>正在加载</p>
          </div>
        </div>

        <div
          v-else-if="!menuItem"
          class="custom-empty-state"
        >
          <div class="max-w-md">
            <div class="custom-empty-icon">
              <Icon name="link" size="lg" class="text-gray-400" />
            </div>
            <h3>
              {{ t('customPage.notFoundTitle') }}
            </h3>
            <p>
              {{ t('customPage.notFoundDesc') }}
            </p>
          </div>
        </div>

        <div v-else-if="!isValidUrl" class="custom-empty-state">
          <div class="max-w-md">
            <div class="custom-empty-icon">
              <Icon name="link" size="lg" class="text-gray-400" />
            </div>
            <h3>
              {{ t('customPage.notConfiguredTitle') }}
            </h3>
            <p>
              {{ t('customPage.notConfiguredDesc') }}
            </p>
          </div>
        </div>

        <div v-else class="custom-embed-shell">
          <a
            :href="embeddedUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="btn btn-secondary btn-sm custom-open-fab"
          >
            <Icon name="externalLink" size="sm" class="mr-1.5" :stroke-width="2" />
            {{ t('customPage.openInNewTab') }}
          </a>
          <iframe
            :src="embeddedUrl"
            class="custom-embed-frame"
            allowfullscreen
          ></iframe>
        </div>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores'
import { useAuthStore } from '@/stores/auth'
import { useAdminSettingsStore } from '@/stores/adminSettings'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import { buildEmbeddedUrl, detectTheme } from '@/utils/embedded-url'

const { t, locale } = useI18n()
const route = useRoute()
const appStore = useAppStore()
const authStore = useAuthStore()
const adminSettingsStore = useAdminSettingsStore()

const loading = ref(false)
const pageTheme = ref<'light' | 'dark'>('light')
let themeObserver: MutationObserver | null = null

const menuItemId = computed(() => route.params.id as string)

const menuItem = computed(() => {
  const id = menuItemId.value
  // Try public settings first (contains user-visible items)
  const publicItems = appStore.cachedPublicSettings?.custom_menu_items ?? []
  const found = publicItems.find((item) => item.id === id) ?? null
  if (found) return found
  // For admin users, also check admin settings (contains admin-only items)
  if (authStore.isAdmin) {
    return adminSettingsStore.customMenuItems.find((item) => item.id === id) ?? null
  }
  return null
})

const embeddedUrl = computed(() => {
  if (!menuItem.value) return ''
  return buildEmbeddedUrl(
    menuItem.value.url,
    authStore.user?.id,
    authStore.token,
    pageTheme.value,
    locale.value,
  )
})

const isValidUrl = computed(() => {
  const url = embeddedUrl.value
  return url.startsWith('http://') || url.startsWith('https://')
})

onMounted(async () => {
  pageTheme.value = detectTheme()

  if (typeof document !== 'undefined') {
    themeObserver = new MutationObserver(() => {
      pageTheme.value = detectTheme()
    })
    themeObserver.observe(document.documentElement, {
      attributes: true,
      attributeFilter: ['class'],
    })
  }

  if (appStore.publicSettingsLoaded) return
  loading.value = true
  try {
    await appStore.fetchPublicSettings()
  } finally {
    loading.value = false
  }
})

onUnmounted(() => {
  if (themeObserver) {
    themeObserver.disconnect()
    themeObserver = null
  }
})
</script>

<style scoped>
.custom-page-layout {
  display: flex;
  min-height: calc(100vh - 64px - 4rem);
  flex-direction: column;
}

.custom-page-frame {
  position: relative;
  display: flex;
  min-height: 0;
  flex: 1;
  flex-direction: column;
  overflow: hidden;
  border: 1px solid rgba(39, 29, 23, 0.08);
  border-radius: 2rem;
  background:
    radial-gradient(circle at 0% 0%, rgba(225, 203, 179, 0.24), transparent 30%),
    linear-gradient(145deg, rgba(255, 252, 245, 0.86), rgba(246, 241, 232, 0.66));
  box-shadow: 0 24px 60px rgba(76, 54, 38, 0.09);
}

.custom-page-frame-head {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 1.5rem;
  border-bottom: 1px solid rgba(39, 29, 23, 0.08);
  padding: 1rem 1.2rem;
}

.custom-page-frame-head span {
  display: block;
  color: rgba(39, 29, 23, 0.45);
  font-size: 0.68rem;
  font-weight: 800;
  letter-spacing: 0.14em;
}

.custom-page-frame-head strong {
  display: block;
  margin-top: 0.2rem;
  color: #221a15;
  font-size: 1.05rem;
}

.custom-page-frame-head p {
  max-width: 33rem;
  margin: 0;
  color: rgba(39, 29, 23, 0.58);
  font-size: 0.86rem;
  line-height: 1.7;
}

.custom-loading {
  display: grid;
  justify-items: center;
  gap: 0.85rem;
  color: rgba(39, 29, 23, 0.58);
}

.custom-loading span {
  width: 2.4rem;
  height: 2.4rem;
  border: 2px solid rgba(39, 29, 23, 0.12);
  border-top-color: #7c5a3b;
  border-radius: 9999px;
  animation: custom-spin 0.8s linear infinite;
}

.custom-empty-state {
  display: flex;
  min-height: 26rem;
  align-items: center;
  justify-content: center;
  padding: 2.5rem;
  text-align: center;
}

.custom-empty-icon {
  display: flex;
  width: 3.2rem;
  height: 3.2rem;
  margin: 0 auto 1rem;
  align-items: center;
  justify-content: center;
  border: 1px solid rgba(39, 29, 23, 0.08);
  border-radius: 9999px;
  background: rgba(255, 255, 255, 0.52);
}

.custom-empty-state h3 {
  color: #221a15;
  font-size: 1.08rem;
  font-weight: 800;
}

.custom-empty-state p {
  margin-top: 0.5rem;
  color: rgba(39, 29, 23, 0.58);
  font-size: 0.92rem;
  line-height: 1.7;
}

.custom-embed-shell {
  position: relative;
  width: 100%;
  flex: 1;
  overflow: hidden;
  background: rgba(255, 255, 255, 0.46);
}

.custom-open-fab {
  position: absolute;
  right: 1rem;
  top: 1rem;
  z-index: 10;
  border-color: rgba(39, 29, 23, 0.08);
  background: rgba(255, 252, 245, 0.84);
  box-shadow: 0 14px 32px rgba(76, 54, 38, 0.12);
  backdrop-filter: blur(14px);
}

.custom-embed-frame {
  display: block;
  margin: 0;
  width: 100%;
  height: 100%;
  border: 0;
  border-radius: 0;
  box-shadow: none;
  background: transparent;
}

@keyframes custom-spin {
  to {
    transform: rotate(360deg);
  }
}

:global(.dark) .custom-page-frame {
  border-color: rgba(255, 243, 230, 0.1);
  background:
    radial-gradient(circle at 0% 0%, rgba(98, 73, 54, 0.22), transparent 30%),
    linear-gradient(145deg, rgba(26, 21, 18, 0.9), rgba(16, 13, 11, 0.82));
}

:global(.dark) .custom-page-frame-head,
:global(.dark) .custom-empty-icon {
  border-color: rgba(255, 243, 230, 0.1);
}

:global(.dark) .custom-page-frame-head strong,
:global(.dark) .custom-empty-state h3 {
  color: #f1e8dc;
}

:global(.dark) .custom-page-frame-head p,
:global(.dark) .custom-empty-state p,
:global(.dark) .custom-loading {
  color: rgba(241, 232, 220, 0.64);
}

:global(.dark) .custom-embed-shell,
:global(.dark) .custom-open-fab,
:global(.dark) .custom-empty-icon {
  background: rgba(255, 255, 255, 0.06);
}

@media (max-width: 720px) {
  .custom-page-layout {
    min-height: calc(100vh - 64px - 2rem);
  }

  .custom-page-frame {
    border-radius: 1.4rem;
  }

  .custom-page-frame-head {
    align-items: flex-start;
    flex-direction: column;
  }
}
</style>
