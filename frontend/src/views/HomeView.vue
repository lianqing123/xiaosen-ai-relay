<template>
  <div v-if="homeContent" class="min-h-screen">
    <iframe
      v-if="isHomeContentUrl"
      :src="homeContent.trim()"
      class="h-screen w-full border-0"
      allowfullscreen
    ></iframe>
    <div v-else v-html="homeContent"></div>
  </div>

  <div v-else class="codex-home">
    <div class="home-texture" aria-hidden="true"></div>
    <header class="home-header" :class="{ 'is-scrolled': isScrolled }">
      <div class="header-shell">
        <router-link to="/home" class="brand-mark">
          <img v-if="siteLogo" :src="siteLogo" alt="Logo" class="brand-logo" />
          <span class="brand-name">{{ brandName }}</span>
        </router-link>

        <div class="header-actions">
          <router-link :to="loginPath" class="header-link">
            {{ navLogin }}
          </router-link>
          <router-link :to="dashboardPath" class="header-cta">
            {{ navCta }}
          </router-link>
          <button
            type="button"
            class="header-menu"
            :aria-label="menuAriaLabel"
            :aria-expanded="menuOpen ? 'true' : 'false'"
            @click="menuOpen = !menuOpen"
          >
            <span></span>
            <span></span>
            <span></span>
          </button>
        </div>

        <div v-if="menuOpen" class="header-panel">
          <button type="button" class="panel-link" @click="jumpTo('hero')">{{ homeLabel }}</button>
          <button type="button" class="panel-link" @click="jumpTo('focus')">{{ focusLabel }}</button>
          <router-link to="/key-usage" class="panel-link" @click="menuOpen = false">
            {{ statusLabel }}
          </router-link>
          <a
            v-if="docUrl"
            :href="docUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="panel-link"
            @click="menuOpen = false"
          >
            {{ docsLabel }}
          </a>
        </div>
      </div>
    </header>

    <section id="hero" class="hero-section">
      <div class="hero-media">
        <img src="/codex-hero.png" alt="Codex Hero" class="hero-image" />
        <div class="hero-edge-fade"></div>
        <div class="hero-vignette"></div>
        <div class="hero-bottom-fade"></div>
        <div class="hero-watermark-cover"></div>
      </div>

      <div class="hero-shell">
        <div class="hero-copy reveal-block is-visible">
          <h1 class="hero-title">
            <span class="title-line title-line--cn">{{ heroTitleLine1 }}</span>
            <span class="title-line title-line--latin">{{ heroTitleLine2 }}</span>
          </h1>
          <p class="hero-description">{{ heroDescription }}</p>
          <div class="hero-actions">
            <router-link :to="dashboardPath" class="hero-primary">
              {{ navCta }}
            </router-link>
            <router-link to="/key-usage" class="hero-secondary">{{ statusLabel }}</router-link>
          </div>
        </div>
      </div>
    </section>

    <main class="story-layout">
      <section id="focus" class="story-section reveal-block">
        <div class="story-shell">
          <div class="story-ambient story-ambient--focus" aria-hidden="true">
            <span></span>
            <span></span>
          </div>
          <div class="story-content">
            <h2 class="story-title">
              <span class="title-line title-line--cn">
                {{ storyOneTitlePrefix }} <span class="title-inline-latin">{{ storyOneTitleBrand }}</span>
              </span>
              <span class="title-line">{{ storyOneTitleLine2 }}</span>
            </h2>
            <p class="story-body">{{ storyOneBody }}</p>
          </div>
        </div>
      </section>

      <section class="story-section reveal-block">
        <div class="story-shell">
          <div class="story-ambient story-ambient--stable" aria-hidden="true">
            <span></span>
            <span></span>
            <span></span>
          </div>
          <div class="story-content">
            <h2 class="story-title">
              <span class="title-line">{{ storyTwoTitleLine1 }}</span>
              <span class="title-line">{{ storyTwoTitleLine2 }}</span>
            </h2>
            <p class="story-body">{{ storyTwoBody }}</p>
            <div class="status-strip" aria-label="Codex connection status">
              <span class="status-item">
                <span class="status-dot"></span>
                Route stable
              </span>
              <span class="status-item">READY</span>
              <span class="status-item">No interruption</span>
            </div>
          </div>
        </div>
      </section>
    </main>

    <footer class="home-footer">
      <div class="footer-shell">
        <span>{{ currentYear }} {{ brandName }}</span>
        <a
          v-if="docUrl"
          :href="docUrl"
          target="_blank"
          rel="noopener noreferrer"
          class="footer-link"
        >
          {{ docsLabel }}
        </a>
      </div>
    </footer>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import { useAuthStore, useAppStore } from '@/stores'

const authStore = useAuthStore()
const appStore = useAppStore()

const siteLogo = computed(() => appStore.cachedPublicSettings?.site_logo || appStore.siteLogo || '')
const docUrl = computed(() => appStore.cachedPublicSettings?.doc_url || appStore.docUrl || '')
const homeContent = computed(() => appStore.cachedPublicSettings?.home_content || '')
const brandName = computed(() => appStore.cachedPublicSettings?.site_name || appStore.siteName || 'Codex Access')

const isAuthenticated = computed(() => authStore.isAuthenticated)
const isAdmin = computed(() => authStore.isAdmin)
const dashboardPath = computed(() => (isAuthenticated.value ? (isAdmin.value ? '/admin/dashboard' : '/dashboard') : '/login'))
const loginPath = computed(() => (isAuthenticated.value ? dashboardPath.value : '/login'))
const currentYear = computed(() => new Date().getFullYear())
const isScrolled = ref(false)
const menuOpen = ref(false)
const menuAriaLabel = '\u6253\u5f00\u83dc\u5355'
const homeLabel = '\u9996\u9875'
const focusLabel = '\u4f18\u5316'
const statusLabel = '\u67e5\u770b\u72b6\u6001'
const docsLabel = '\u6587\u6863'
const navLogin = computed(() => (isAuthenticated.value ? '\u63a7\u5236\u53f0' : '\u767b\u5f55'))
const navCta = computed(() => (isAuthenticated.value ? '\u8fdb\u5165\u63a7\u5236\u53f0' : '\u7acb\u5373\u63a5\u5165'))
const heroTitleLine1 = '\u66f4\u7a33\u5b9a\u5730\u8fde\u63a5'
const heroTitleLine2 = 'Codex'
const heroDescription = '\u4e3a\u957f\u671f\u4f7f\u7528\u800c\u8bbe\u8ba1\u7684 Codex \u8fde\u63a5\u4f53\u9a8c\u3002'
const storyOneTitlePrefix = '\u53ea\u4e3a'
const storyOneTitleBrand = 'Codex'
const storyOneTitleLine2 = '\u4f18\u5316\u3002'
const storyOneBody = '\u4e0d\u505a\u6742\u4e71\u805a\u5408\uff0c\u4e0d\u6269\u5f20\u6210\u6a21\u578b\u76ee\u5f55\uff0c\u53ea\u628a\u63a5\u5165\u3001\u7a33\u5b9a\u6027\u548c\u4f53\u9a8c\u6536\u7d27\u5230\u4e00\u6761\u66f4\u5e72\u51c0\u7684\u8def\u5f84\u4e0a\u3002'
const storyTwoTitleLine1 = '\u5c11\u4e00\u70b9\u6742\u97f3\u3002'
const storyTwoTitleLine2 = '\u591a\u4e00\u70b9\u786e\u5b9a\u6027\u3002'
const storyTwoBody = '\u54cd\u5e94\u66f4\u5e73\u987a\uff0c\u8fde\u63a5\u66f4\u8fde\u7eed\uff0c\u4f7f\u7528\u8fc7\u7a0b\u66f4\u5c11\u88ab\u6253\u65ad\u3002\u5bf9 Codex \u6765\u8bf4\uff0c\u8fd9\u5df2\u7ecf\u8db3\u591f\u91cd\u8981\u3002'

const isHomeContentUrl = computed(() => {
  const content = homeContent.value.trim()
  return content.startsWith('http://') || content.startsWith('https://')
})

let observer: IntersectionObserver | null = null

function handleScroll() {
  isScrolled.value = window.scrollY > 18
  if (menuOpen.value && window.scrollY > 24) {
    menuOpen.value = false
  }
}

function jumpTo(id: string) {
  menuOpen.value = false
  const el = document.getElementById(id)
  if (!el) return
  el.scrollIntoView({ behavior: 'smooth', block: 'start' })
}

function handleClick(event: MouseEvent) {
  const target = event.target as HTMLElement | null
  if (target?.closest('.header-shell')) return
  menuOpen.value = false
}

function initRevealObserver() {
  const targets = document.querySelectorAll<HTMLElement>('.reveal-block')
  observer?.disconnect()
  observer = new IntersectionObserver(
    (entries) => {
      entries.forEach((entry) => {
        if (entry.isIntersecting) {
          entry.target.classList.add('is-visible')
        }
      })
    },
    {
      threshold: 0.18,
      rootMargin: '0px 0px -10% 0px',
    }
  )
  targets.forEach((target) => observer?.observe(target))
}

onMounted(async () => {
  authStore.checkAuth()

  if (!appStore.publicSettingsLoaded) {
    await appStore.fetchPublicSettings()
  }

  document.title = brandName.value
  handleScroll()
  window.addEventListener('scroll', handleScroll, { passive: true })
  window.addEventListener('click', handleClick)
  await nextTick()
  initRevealObserver()
})

onBeforeUnmount(() => {
  window.removeEventListener('scroll', handleScroll)
  window.removeEventListener('click', handleClick)
  observer?.disconnect()
})
</script>

<style scoped>
@import '@fontsource/noto-serif-sc/chinese-simplified-500.css';
@import '@fontsource/cormorant-garamond/latin-500.css';

.codex-home {
  position: relative;
  min-height: 100vh;
  background:
    radial-gradient(circle at 0% 0%, rgba(225, 203, 179, 0.3), transparent 28%),
    radial-gradient(circle at 100% 12%, rgba(224, 197, 167, 0.14), transparent 24%),
    #f6f1e8;
  color: #1f1814;
  overflow: hidden;
}

.home-texture {
  position: fixed;
  inset: 0;
  z-index: 0;
  pointer-events: none;
  background-image:
    linear-gradient(rgba(31, 24, 20, 0.032) 1px, transparent 1px),
    linear-gradient(90deg, rgba(31, 24, 20, 0.026) 1px, transparent 1px);
  background-size: 92px 92px;
  mask-image: linear-gradient(180deg, transparent 0%, black 18%, black 72%, transparent 100%);
  opacity: 0.34;
}

.home-header {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  z-index: 40;
  padding: 1.1rem 1.4rem 0;
  transition: padding 180ms ease;
}

.header-shell {
  position: relative;
  margin: 0 auto;
  display: flex;
  max-width: 1280px;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  border-radius: 9999px;
  border: 1px solid transparent;
  padding: 0.9rem 1.25rem;
  transition:
    background-color 180ms ease,
    border-color 180ms ease,
    box-shadow 180ms ease,
    backdrop-filter 180ms ease;
}

.home-header.is-scrolled .header-shell {
  background: rgba(246, 241, 232, 0.56);
  border-color: rgba(44, 33, 25, 0.08);
  box-shadow: 0 12px 40px rgba(67, 48, 34, 0.08);
  backdrop-filter: blur(18px) saturate(145%);
}

.brand-mark {
  display: inline-flex;
  align-items: center;
  gap: 0.8rem;
  color: inherit;
  text-decoration: none;
}

.brand-logo {
  height: 2rem;
  width: 2rem;
  border-radius: 9999px;
  object-fit: contain;
  background: rgba(255, 255, 255, 0.68);
  box-shadow: 0 10px 24px rgba(42, 28, 18, 0.08);
}

.brand-name {
  font-size: 0.92rem;
  font-weight: 700;
  letter-spacing: 0;
}

.header-actions {
  display: inline-flex;
  align-items: center;
  gap: 0.9rem;
}

.header-link,
.footer-link,
.panel-link {
  color: rgba(31, 24, 20, 0.76);
  text-decoration: none;
  transition: color 150ms ease;
}

.header-link:hover,
.footer-link:hover,
.panel-link:hover {
  color: #1f1814;
}

.header-cta,
.hero-primary,
.hero-secondary {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 9999px;
  text-decoration: none;
  transition:
    transform 150ms ease,
    background-color 150ms ease,
    border-color 150ms ease,
    color 150ms ease;
}

.header-cta {
  min-width: 6.1rem;
  padding: 0.72rem 1.1rem;
  border: 1px solid rgba(31, 24, 20, 0.08);
  background: rgba(255, 255, 255, 0.58);
  color: #1f1814;
}

.header-cta:hover,
.hero-primary:hover,
.hero-secondary:hover {
  transform: translateY(-1px);
}

.header-menu {
  display: inline-flex;
  flex-direction: column;
  justify-content: center;
  gap: 0.24rem;
  width: 2.2rem;
  height: 2.2rem;
  border: 0;
  background: transparent;
  color: #1f1814;
  cursor: pointer;
  padding: 0;
}

.header-menu span {
  display: block;
  width: 1rem;
  height: 1.5px;
  margin: 0 auto;
  border-radius: 9999px;
  background: currentColor;
}

.header-panel {
  position: absolute;
  top: calc(100% + 0.7rem);
  right: 0;
  min-width: 11rem;
  display: grid;
  gap: 0.2rem;
  padding: 0.6rem;
  border: 1px solid rgba(31, 24, 20, 0.08);
  border-radius: 1rem;
  background: rgba(250, 246, 239, 0.92);
  box-shadow: 0 22px 44px rgba(67, 48, 34, 0.14);
  backdrop-filter: blur(18px);
}

.panel-link {
  display: flex;
  align-items: center;
  justify-content: flex-start;
  width: 100%;
  border: 0;
  background: transparent;
  border-radius: 0.8rem;
  padding: 0.7rem 0.85rem;
  font: inherit;
  cursor: pointer;
}

.panel-link:hover {
  background: rgba(31, 24, 20, 0.05);
}

.hero-section {
  position: relative;
  min-height: 100dvh;
  overflow: hidden;
}

.hero-media {
  position: absolute;
  inset: 0;
}

.hero-image {
  position: absolute;
  top: 0;
  right: 0;
  width: 74%;
  height: 100%;
  object-fit: cover;
  object-position: right center;
  transform: scale(1.02);
}

.hero-edge-fade {
  position: absolute;
  inset: 0 auto 0 26%;
  width: 30%;
  background: linear-gradient(90deg, rgba(246, 241, 232, 0.92) 0%, rgba(246, 241, 232, 0.34) 42%, rgba(246, 241, 232, 0) 100%);
  pointer-events: none;
}

.hero-vignette {
  position: absolute;
  inset: 0;
  background:
    linear-gradient(90deg, rgba(246, 241, 232, 0.97) 0%, rgba(246, 241, 232, 0.94) 22%, rgba(246, 241, 232, 0.78) 38%, rgba(246, 241, 232, 0.22) 57%, rgba(246, 241, 232, 0.08) 100%),
    linear-gradient(180deg, rgba(246, 241, 232, 0.08) 0%, rgba(246, 241, 232, 0) 56%, rgba(246, 241, 232, 0.08) 100%);
}

.hero-bottom-fade {
  position: absolute;
  left: 0;
  right: 0;
  bottom: 0;
  height: 16vh;
  background: linear-gradient(180deg, rgba(246, 241, 232, 0) 0%, rgba(246, 241, 232, 0.95) 100%);
}

.hero-watermark-cover {
  position: absolute;
  right: 0;
  bottom: 0;
  width: min(24rem, 28vw);
  height: min(21rem, 24vw);
  pointer-events: none;
  background:
    radial-gradient(ellipse at 82% 72%, rgba(246, 241, 232, 1) 0%, rgba(246, 241, 232, 0.99) 30%, rgba(246, 241, 232, 0.72) 52%, rgba(246, 241, 232, 0) 76%);
  filter: blur(10px);
  transform: translate(8%, 14%);
}

.hero-shell {
  position: relative;
  z-index: 1;
  display: flex;
  align-items: center;
  min-height: 100vh;
  max-width: 1280px;
  margin: 0 auto;
  padding: 7.8rem 1.4rem 3rem clamp(1.8rem, 3.6vw, 3.8rem);
}

.hero-copy {
  position: relative;
  max-width: 42rem;
  padding-left: 0;
  margin-left: 0;
}

.hero-copy::before {
  content: '';
  position: absolute;
  z-index: -1;
  inset: -3.5rem -7rem -2.4rem -4rem;
  pointer-events: none;
  background:
    radial-gradient(ellipse at 56% 48%, rgba(246, 241, 232, 0.78) 0%, rgba(246, 241, 232, 0.46) 38%, rgba(246, 241, 232, 0) 72%);
  filter: blur(22px);
}

.hero-title,
.story-title {
  margin: 0;
  font-family: 'Noto Serif SC', 'Source Han Serif SC', 'Songti SC', serif;
  font-weight: 500;
  letter-spacing: 0;
  line-height: 0.9;
  text-wrap: balance;
}

.title-line {
  display: block;
  white-space: nowrap;
}

.title-line--cn {
  font-family: 'Noto Serif SC', 'Source Han Serif SC', 'Songti SC', serif;
}

.title-line--latin,
.title-inline-latin {
  font-family: 'Cormorant Garamond', 'Times New Roman', serif;
  font-weight: 500;
  letter-spacing: 0;
}

.hero-title {
  max-width: 39rem;
  font-size: clamp(4rem, 5.65vw, 6.05rem);
}

.hero-description,
.story-body {
  color: rgba(39, 29, 23, 0.72);
  font-size: 1.12rem;
  line-height: 1.8;
}

.hero-description {
  max-width: 28rem;
  margin: 1.8rem 0 0;
}

.hero-actions {
  display: flex;
  align-items: center;
  gap: 1rem;
  margin-top: 2rem;
}

.hero-primary {
  min-width: 7.2rem;
  padding: 0.95rem 1.45rem;
  background: #271d17;
  color: #f8f3eb;
  box-shadow: 0 16px 34px rgba(30, 20, 14, 0.16);
}

.hero-secondary {
  min-width: 7.2rem;
  padding: 0.95rem 1.45rem;
  border: 1px solid rgba(31, 24, 20, 0.1);
  background: rgba(255, 255, 255, 0.52);
  color: #271d17;
}

.story-layout {
  position: relative;
  z-index: 1;
  padding: 0 1.4rem 6rem;
}

.story-section {
  padding: 4rem 0 5rem;
}

.story-shell {
  position: relative;
  max-width: 1280px;
  margin: 0 auto;
  padding-left: clamp(24rem, calc(15rem + 13vw), 31.75rem);
}

.story-content {
  max-width: 60rem;
}

.story-title {
  font-size: clamp(3.3rem, 6.2vw, 6rem);
}

.story-body {
  max-width: 34rem;
  margin-top: 1.6rem;
}

.story-ambient {
  position: absolute;
  inset: 0 auto auto 0;
  width: min(21rem, 24vw);
  height: 14rem;
  pointer-events: none;
  opacity: 0.36;
}

.story-ambient span {
  position: absolute;
  display: block;
  background: rgba(31, 24, 20, 0.18);
}

.story-ambient--focus span:nth-child(1) {
  left: 2rem;
  top: 2.4rem;
  width: 58%;
  height: 1px;
}

.story-ambient--focus span:nth-child(2) {
  left: 2rem;
  top: 2.4rem;
  width: 1px;
  height: 8rem;
  background: linear-gradient(180deg, rgba(31, 24, 20, 0.22), transparent);
}

.story-ambient--stable span:nth-child(1) {
  left: 3rem;
  top: 1.2rem;
  width: 72%;
  height: 1px;
}

.story-ambient--stable span:nth-child(2),
.story-ambient--stable span:nth-child(3) {
  top: 0.94rem;
  width: 0.48rem;
  height: 0.48rem;
  border-radius: 9999px;
  background: rgba(201, 111, 74, 0.82);
}

.story-ambient--stable span:nth-child(2) {
  left: 3rem;
}

.story-ambient--stable span:nth-child(3) {
  left: calc(3rem + 72%);
}

.status-strip {
  display: inline-flex;
  align-items: center;
  gap: 0.85rem;
  margin-top: 2.2rem;
  padding-top: 1rem;
  border-top: 1px solid rgba(31, 24, 20, 0.12);
  color: rgba(31, 24, 20, 0.5);
  font-size: 0.78rem;
  line-height: 1;
}

.status-item {
  display: inline-flex;
  align-items: center;
  gap: 0.38rem;
  white-space: nowrap;
}

.status-dot {
  width: 0.42rem;
  height: 0.42rem;
  border-radius: 9999px;
  background: #4f7d63;
  box-shadow: 0 0 0 0 rgba(79, 125, 99, 0.18);
  animation: status-breathe 3.2s ease-in-out infinite;
}

.home-footer {
  position: relative;
  z-index: 1;
  padding: 0 1.4rem 2rem;
}

.footer-shell {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  max-width: 1280px;
  margin: 0 auto;
  padding-top: 1.5rem;
  border-top: 1px solid rgba(31, 24, 20, 0.08);
  font-size: 0.92rem;
  color: rgba(31, 24, 20, 0.52);
}

.reveal-block {
  opacity: 0;
  transform: translateY(28px);
  transition:
    opacity 700ms ease,
    transform 700ms ease;
}

.reveal-block.is-visible {
  opacity: 1;
  transform: translateY(0);
}

@keyframes status-breathe {
  0%,
  100% {
    box-shadow: 0 0 0 0 rgba(79, 125, 99, 0.16);
  }
  50% {
    box-shadow: 0 0 0 8px rgba(79, 125, 99, 0);
  }
}

:deep(.dark) .codex-home {
  background:
    radial-gradient(circle at 0% 0%, rgba(98, 73, 54, 0.24), transparent 28%),
    radial-gradient(circle at 100% 12%, rgba(122, 92, 65, 0.16), transparent 24%),
    #0f0d0c;
  color: #f1e8dc;
}

:deep(.dark) .home-header.is-scrolled .header-shell {
  background: rgba(18, 15, 13, 0.56);
  border-color: rgba(255, 243, 230, 0.08);
  box-shadow: 0 12px 40px rgba(0, 0, 0, 0.22);
}

:deep(.dark) .brand-logo,
:deep(.dark) .header-cta,
:deep(.dark) .hero-secondary,
:deep(.dark) .header-panel {
  background: rgba(255, 255, 255, 0.08);
}

:deep(.dark) .header-link,
:deep(.dark) .footer-link,
:deep(.dark) .panel-link,
:deep(.dark) .hero-description,
:deep(.dark) .story-body,
:deep(.dark) .footer-shell {
  color: rgba(241, 232, 220, 0.68);
}

:deep(.dark) .header-cta,
:deep(.dark) .hero-secondary {
  border-color: rgba(255, 243, 230, 0.08);
  color: #f1e8dc;
}

:deep(.dark) .header-menu {
  color: #f1e8dc;
}

:deep(.dark) .hero-vignette {
  background:
    linear-gradient(90deg, rgba(15, 13, 12, 0.94) 0%, rgba(15, 13, 12, 0.88) 20%, rgba(15, 13, 12, 0.64) 36%, rgba(15, 13, 12, 0.22) 58%, rgba(15, 13, 12, 0.06) 100%),
    linear-gradient(180deg, rgba(15, 13, 12, 0.1) 0%, rgba(15, 13, 12, 0) 56%, rgba(15, 13, 12, 0.1) 100%);
}

:deep(.dark) .hero-bottom-fade {
  background: linear-gradient(180deg, rgba(15, 13, 12, 0) 0%, rgba(15, 13, 12, 0.95) 100%);
}

@media (max-width: 1024px) {
  .hero-image {
    width: 100%;
    opacity: 0.42;
  }

  .hero-edge-fade,
  .story-ambient {
    display: none;
  }

  .hero-vignette {
    background:
      linear-gradient(180deg, rgba(246, 241, 232, 0.86) 0%, rgba(246, 241, 232, 0.76) 34%, rgba(246, 241, 232, 0.88) 100%);
  }

  .hero-shell,
  .story-shell {
    padding-left: 2rem;
  }

  .hero-copy {
    padding-left: 0;
    margin-left: 0;
  }

  .hero-copy::before {
    inset: -2.5rem -2rem -2rem -2rem;
    opacity: 0.72;
  }
}

@media (max-width: 768px) {
  .home-header {
    padding: 0.9rem 1rem 0;
  }

  .header-shell {
    padding: 0.8rem 1rem;
  }

  .header-actions {
    gap: 0.6rem;
  }

  .header-link {
    display: none;
  }

  .header-cta {
    min-width: auto;
    padding: 0.68rem 0.95rem;
  }

  .hero-shell {
    padding: 7rem 1rem 2rem;
  }

  .hero-title {
    font-size: clamp(3rem, 16vw, 4.6rem);
  }

  .title-line {
    white-space: normal;
  }

  .hero-description,
  .story-body {
    font-size: 1rem;
  }

  .hero-actions {
    flex-wrap: wrap;
  }

  .story-layout {
    padding: 0 1rem 4rem;
  }

  .story-shell {
    padding-left: 0;
  }

  .story-title {
    font-size: clamp(2.6rem, 14vw, 4.2rem);
  }

  .status-strip {
    flex-wrap: wrap;
    gap: 0.65rem;
  }

  .footer-shell {
    flex-direction: column;
    align-items: flex-start;
  }
}
</style>
