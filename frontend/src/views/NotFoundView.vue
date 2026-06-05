<template>
  <div class="not-found-page">
    <div class="not-found-texture" aria-hidden="true"></div>

    <main class="not-found-shell">
      <section class="not-found-copy">
        <router-link to="/home" class="not-found-brand">{{ siteName }}</router-link>
        <p class="not-found-kicker">ROUTE CHECK</p>
        <h1>这条路径没有接入。</h1>
        <p>
          当前地址没有匹配到可用页面。你可以回到首页、进入控制台，或打开文档确认入口配置。
        </p>

        <div class="not-found-actions">
          <button type="button" class="not-found-secondary" @click="goBack">返回上一页</button>
          <router-link to="/home" class="not-found-primary">返回首页</router-link>
          <router-link to="/login" class="not-found-secondary">进入控制台</router-link>
        </div>
      </section>

      <aside class="not-found-panel" aria-label="404 route trace">
        <div class="panel-head">
          <span></span>
          <span></span>
          <span></span>
        </div>
        <div class="route-code">
          <p><span>HTTP</span> 404</p>
          <p><span>PATH</span> {{ currentPath }}</p>
          <p><span>STATE</span> not_mapped</p>
        </div>
        <div class="route-flow">
          <div>
            <strong>Request</strong>
            <span>browser</span>
          </div>
          <i></i>
          <div>
            <strong>Router</strong>
            <span>no match</span>
          </div>
          <i></i>
          <div>
            <strong>Next</strong>
            <span>home / console</span>
          </div>
        </div>
      </aside>
    </main>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAppStore } from '@/stores'

const router = useRouter()
const route = useRoute()
const appStore = useAppStore()

const siteName = computed(() => appStore.cachedPublicSettings?.site_name || appStore.siteName || 'Codex Access')
const currentPath = computed(() => route.fullPath || '/')

function goBack(): void {
  router.back()
}
</script>

<style scoped>
@import '@fontsource/noto-serif-sc/chinese-simplified-500.css';
@import '@fontsource/cormorant-garamond/latin-500.css';

.not-found-page {
  position: relative;
  min-height: 100dvh;
  overflow: hidden;
  background:
    radial-gradient(circle at 0% 0%, rgba(225, 203, 179, 0.34), transparent 30%),
    radial-gradient(circle at 96% 14%, rgba(189, 158, 122, 0.16), transparent 26%),
    #f6f1e8;
  color: #221a15;
}

.not-found-texture {
  position: fixed;
  inset: 0;
  pointer-events: none;
  background-image:
    linear-gradient(rgba(31, 24, 20, 0.032) 1px, transparent 1px),
    linear-gradient(90deg, rgba(31, 24, 20, 0.026) 1px, transparent 1px);
  background-size: 92px 92px;
  mask-image: linear-gradient(180deg, transparent 0%, black 20%, black 72%, transparent 100%);
  opacity: 0.34;
}

.not-found-shell {
  position: relative;
  z-index: 1;
  display: grid;
  min-height: 100dvh;
  max-width: 1280px;
  margin: 0 auto;
  grid-template-columns: minmax(0, 1fr) minmax(20rem, 0.82fr);
  gap: clamp(2rem, 6vw, 6rem);
  align-items: center;
  padding: 4rem 1.4rem;
}

.not-found-brand {
  display: inline-flex;
  margin-bottom: clamp(2.5rem, 8vh, 6rem);
  color: #221a15;
  font-size: 0.95rem;
  font-weight: 800;
  text-decoration: none;
}

.not-found-kicker {
  margin: 0 0 1.1rem;
  color: rgba(39, 29, 23, 0.52);
  font-size: 0.78rem;
  font-weight: 800;
  letter-spacing: 0.16em;
}

.not-found-copy h1 {
  max-width: 44rem;
  margin: 0;
  font-family: 'Noto Serif SC', 'Source Han Serif SC', 'Songti SC', serif;
  font-size: clamp(3.8rem, 7vw, 6.7rem);
  font-weight: 500;
  line-height: 0.94;
  letter-spacing: 0;
  text-wrap: balance;
}

.not-found-copy p:not(.not-found-kicker) {
  max-width: 34rem;
  margin: 1.5rem 0 0;
  color: rgba(39, 29, 23, 0.68);
  font-size: 1.06rem;
  line-height: 1.9;
}

.not-found-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 0.85rem;
  margin-top: 2rem;
}

.not-found-primary,
.not-found-secondary {
  display: inline-flex;
  min-width: 7.2rem;
  align-items: center;
  justify-content: center;
  border-radius: 9999px;
  padding: 0.95rem 1.35rem;
  font-weight: 800;
  text-decoration: none;
  transition: transform 160ms ease, background-color 160ms ease;
}

.not-found-primary {
  background: #271d17;
  color: #f8f3eb;
  box-shadow: 0 16px 34px rgba(30, 20, 14, 0.14);
}

.not-found-secondary {
  border: 1px solid rgba(31, 24, 20, 0.1);
  background: rgba(255, 255, 255, 0.52);
  color: #271d17;
}

button.not-found-secondary {
  cursor: pointer;
}

.not-found-primary:hover,
.not-found-secondary:hover {
  transform: translateY(-1px);
}

.not-found-panel {
  position: relative;
  overflow: hidden;
  border: 1px solid rgba(39, 29, 23, 0.11);
  border-radius: 2.4rem;
  padding: 1.7rem;
  background:
    linear-gradient(145deg, rgba(255, 252, 245, 0.8), rgba(234, 220, 201, 0.44)),
    rgba(255, 255, 255, 0.56);
  box-shadow: 0 30px 70px rgba(76, 54, 38, 0.13);
  backdrop-filter: blur(20px) saturate(140%);
}

.not-found-panel::before {
  content: '404';
  position: absolute;
  right: -0.3rem;
  top: -1.8rem;
  color: rgba(39, 29, 23, 0.05);
  font-family: 'Cormorant Garamond', 'Times New Roman', serif;
  font-size: 13rem;
  line-height: 1;
}

.panel-head,
.route-code,
.route-flow {
  position: relative;
}

.panel-head {
  display: flex;
  gap: 0.42rem;
  margin-bottom: 2rem;
}

.panel-head span {
  width: 0.58rem;
  height: 0.58rem;
  border-radius: 9999px;
  background: rgba(39, 29, 23, 0.2);
}

.route-code {
  display: grid;
  gap: 0.9rem;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
}

.route-code p {
  display: flex;
  gap: 1rem;
  margin: 0;
  color: rgba(39, 29, 23, 0.74);
  word-break: break-all;
}

.route-code span {
  min-width: 3.6rem;
  color: rgba(39, 29, 23, 0.45);
  font-size: 0.78rem;
  font-weight: 800;
  letter-spacing: 0.12em;
}

.route-flow {
  display: grid;
  grid-template-columns: 1fr auto 1fr auto 1fr;
  gap: 0.7rem;
  align-items: center;
  margin-top: 2.2rem;
}

.route-flow div {
  display: grid;
  min-height: 5rem;
  align-content: center;
  gap: 0.3rem;
  border: 1px solid rgba(39, 29, 23, 0.08);
  border-radius: 1.1rem;
  padding: 0.85rem;
  background: rgba(255, 255, 255, 0.42);
}

.route-flow strong {
  font-size: 0.86rem;
}

.route-flow span {
  color: rgba(39, 29, 23, 0.48);
  font-size: 0.74rem;
}

.route-flow i {
  width: 1.8rem;
  height: 1px;
  background: rgba(39, 29, 23, 0.14);
}

:deep(.dark) .not-found-page {
  background:
    radial-gradient(circle at 0% 0%, rgba(98, 73, 54, 0.24), transparent 30%),
    radial-gradient(circle at 96% 14%, rgba(122, 92, 65, 0.16), transparent 26%),
    #100d0b;
  color: #f1e8dc;
}

:deep(.dark) .not-found-brand,
:deep(.dark) .not-found-secondary {
  color: #f1e8dc;
}

:deep(.dark) .not-found-panel,
:deep(.dark) .not-found-secondary,
:deep(.dark) .route-flow div {
  border-color: rgba(255, 243, 230, 0.1);
  background: rgba(26, 21, 18, 0.72);
}

:deep(.dark) .not-found-copy p:not(.not-found-kicker),
:deep(.dark) .route-code p,
:deep(.dark) .route-flow span {
  color: rgba(241, 232, 220, 0.66);
}

@media (max-width: 900px) {
  .not-found-shell {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 640px) {
  .not-found-shell {
    padding: 3rem 1rem;
  }

  .not-found-copy h1 {
    font-size: clamp(3rem, 17vw, 4.8rem);
  }

  .not-found-actions,
  .not-found-primary,
  .not-found-secondary {
    width: 100%;
  }

  .route-flow {
    grid-template-columns: 1fr;
  }

  .route-flow i {
    width: 1px;
    height: 1rem;
    margin-left: 1rem;
  }
}
</style>
