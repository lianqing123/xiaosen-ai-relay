<template>
  <AppLayout>
    <div class="kiro-page">
      <header class="kiro-hero">
        <div class="kiro-hero-copy">
          <p class="kiro-kicker">Kiro Gateway</p>
          <h1>Kiro 接入</h1>
          <div class="kiro-status-row">
            <span :class="['kiro-live-chip', status?.gateway_healthy ? 'is-online' : 'is-offline']">
              <span class="kiro-live-dot"></span>
              {{ status?.gateway_healthy ? '网关在线' : '等待激活' }}
            </span>
            <span class="kiro-muted">{{ status?.kiro_home || '/opt/kiro-gateway' }}</span>
          </div>
        </div>
        <div class="kiro-orbit" aria-hidden="true">
          <span class="orbit-ring ring-a"></span>
          <span class="orbit-ring ring-b"></span>
          <span class="orbit-core">
            <Icon name="server" size="lg" />
          </span>
        </div>
      </header>

      <div v-if="errorMessage" class="kiro-alert kiro-alert-error">
        <Icon name="exclamationTriangle" size="sm" />
        <span>{{ errorMessage }}</span>
      </div>

      <div v-if="successMessage" class="kiro-alert kiro-alert-success">
        <Icon name="checkCircle" size="sm" />
        <span>{{ successMessage }}</span>
      </div>

      <section class="kiro-overview">
        <div class="kiro-metric">
          <span>凭据数量</span>
          <strong>{{ status?.credential_count ?? '-' }}</strong>
        </div>
        <div class="kiro-metric">
          <span>凭据文件</span>
          <strong>{{ status?.credentials_file_exists ? '已生成' : '未生成' }}</strong>
        </div>
        <div class="kiro-metric">
          <span>激活脚本</span>
          <strong>{{ status?.activation_script_exists ? '可用' : '缺失' }}</strong>
        </div>
        <div class="kiro-metric">
          <span>Kiro CLI</span>
          <strong>{{ status?.cli_available ? '可用' : '未安装' }}</strong>
        </div>
        <button type="button" class="kiro-refresh" :disabled="loading" title="刷新状态" @click="loadStatus">
          <Icon name="refresh" size="sm" :class="{ 'kiro-spin': loading }" />
          <span>刷新</span>
        </button>
      </section>

      <main class="kiro-grid">
        <section class="kiro-panel kiro-import-panel">
          <div class="kiro-section-head">
            <div>
              <p class="kiro-kicker">Credentials</p>
              <h2>导入凭据</h2>
            </div>
            <span class="kiro-step">01</span>
          </div>

          <div class="kiro-mode-tabs" role="tablist" aria-label="凭据来源">
            <button
              v-for="item in credentialModes"
              :key="item.value"
              type="button"
              :class="{ active: credentialType === item.value }"
              @click="credentialType = item.value"
            >
              <Icon :name="item.icon" size="sm" />
              <span>{{ item.label }}</span>
            </button>
          </div>

          <form class="kiro-form" @submit.prevent="submit(false)">
            <label v-if="credentialType === 'refresh_token'" class="kiro-field">
              <span>Refresh Token</span>
              <textarea
                v-model.trim="refreshToken"
                class="input kiro-token-input"
                autocomplete="off"
                spellcheck="false"
                placeholder='粘贴 refresh token、{"refreshToken":"...","provider":"Google"}，或多个 token 的 JSON 数组'
              ></textarea>
              <small>支持直接粘贴 Google/Kiro token JSON；多个账号可用 JSON 数组，或一行一个 token/JSON。</small>
            </label>

            <label v-else class="kiro-field">
              <span>{{ credentialType === 'json' ? 'JSON 凭据文件' : 'SQLite 凭据文件' }}</span>
              <div class="kiro-file-drop">
                <Icon name="upload" size="lg" />
                <input
                  type="file"
                  :accept="fileAccept"
                  @change="handleFileChange"
                />
                <strong>{{ selectedFile?.name || '选择文件' }}</strong>
                <small>
                  {{ selectedFile ? formatFileSize(selectedFile.size) : credentialType === 'json' ? 'kiro-auth-token.json / AWS SSO token JSON' : 'kiro-cli data.sqlite3' }}
                </small>
              </div>
              <small v-if="credentialType === 'json'">至少包含 refreshToken；完整缓存可带 accessToken、expiresAt、profileArn、region；SSO 还需要 clientId 和 clientSecret。</small>
            </label>

            <div class="kiro-form-grid">
              <label class="kiro-field">
                <span>Region</span>
                <input v-model.trim="region" class="input" placeholder="us-east-1" />
              </label>
            </div>

            <div class="kiro-form-grid">
              <label class="kiro-field">
                <span>Profile ARN</span>
                <input v-model.trim="profileArn" class="input" placeholder="可选" />
              </label>

              <label class="kiro-field">
                <span>API Region</span>
                <input v-model.trim="apiRegion" class="input" placeholder="可选" />
              </label>
            </div>

            <label class="kiro-check-row">
              <input v-model="appendCredential" type="checkbox" />
              <span>保留已有凭据并追加</span>
            </label>

            <div class="kiro-actions">
              <button type="submit" class="btn btn-secondary" :disabled="submitting || !canSubmit">
                <Icon name="upload" size="sm" />
                <span>{{ submitting ? '导入中' : '导入' }}</span>
              </button>
              <button type="button" class="btn btn-primary" :disabled="submitting || !canSubmit" @click="submit(true)">
                <Icon name="bolt" size="sm" />
                <span>{{ submitting ? '处理中' : '导入并激活' }}</span>
              </button>
            </div>
          </form>
        </section>

        <section class="kiro-panel">
          <div class="kiro-section-head">
            <div>
              <p class="kiro-kicker">Runtime</p>
              <h2>状态与日志</h2>
            </div>
            <button type="button" class="btn btn-primary btn-sm" :disabled="activating || !status?.credentials_file_exists" @click="activateGateway">
              <Icon name="play" size="sm" />
              <span>{{ activating ? '激活中' : '激活线路' }}</span>
            </button>
          </div>

          <div class="kiro-health">
            <div>
              <span>网关健康</span>
              <strong>{{ status?.gateway_healthy ? '正常' : '未启动' }}</strong>
            </div>
            <p>{{ status?.gateway_health_message || '暂无健康检查结果' }}</p>
          </div>

          <div class="kiro-credential-list">
            <article v-for="(credential, index) in credentials" :key="`${credential.type}-${index}`" class="kiro-credential-row">
              <div class="kiro-credential-icon">
                <Icon :name="credential.type === 'refresh_token' ? 'key' : 'document'" size="sm" />
              </div>
              <div>
                <strong>{{ credentialLabel(credential) }}</strong>
                <small>{{ credentialHint(credential) }}</small>
              </div>
              <span :class="['kiro-pill', credential.enabled === false ? 'muted' : 'active']">
                {{ credential.enabled === false ? '停用' : '启用' }}
              </span>
            </article>
            <div v-if="!credentials.length" class="kiro-empty">
              <Icon name="infoCircle" size="sm" />
              <span>暂无凭据</span>
            </div>
          </div>

          <div class="kiro-terminal">
            <div class="kiro-terminal-head">
              <span></span>
              <span></span>
              <span></span>
              <strong>activation output</strong>
            </div>
            <pre>{{ activationOutput || '等待激活输出' }}</pre>
          </div>
        </section>
      </main>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import { adminAPI } from '@/api/admin'
import type { KiroCredentialEntry, KiroGatewayStatus } from '@/api/admin/kiro'
import { extractApiErrorMessage } from '@/utils/apiError'

type CredentialType = 'refresh_token' | 'json' | 'sqlite'

const credentialType = ref<CredentialType>('refresh_token')
const refreshToken = ref('')
const selectedFile = ref<File | null>(null)
const region = ref('us-east-1')
const profileArn = ref('')
const apiRegion = ref('')
const appendCredential = ref(true)
const status = ref<KiroGatewayStatus | null>(null)
const loading = ref(false)
const submitting = ref(false)
const activating = ref(false)
const errorMessage = ref('')
const successMessage = ref('')
const activationOutput = ref('')

const credentialModes: Array<{ value: CredentialType; label: string; icon: 'key' | 'document' }> = [
  { value: 'refresh_token', label: 'Token', icon: 'key' },
  { value: 'json', label: 'JSON', icon: 'document' },
  { value: 'sqlite', label: 'SQLite', icon: 'document' }
]

const fileAccept = computed(() => credentialType.value === 'json' ? '.json,application/json' : '.sqlite,.sqlite3,.db')
const credentials = computed(() => status.value?.credentials ?? [])
const canSubmit = computed(() => {
  if (credentialType.value === 'refresh_token') return refreshToken.value.length > 0
  return selectedFile.value !== null
})

async function loadStatus() {
  loading.value = true
  errorMessage.value = ''
  try {
    status.value = await adminAPI.kiro.getStatus()
  } catch (error: unknown) {
    errorMessage.value = extractApiErrorMessage(error, '读取 Kiro 状态失败')
  } finally {
    loading.value = false
  }
}

function handleFileChange(event: Event) {
  const target = event.target as HTMLInputElement
  selectedFile.value = target.files?.[0] ?? null
  if (!selectedFile.value) return
  const lower = selectedFile.value.name.toLowerCase()
  if (lower.endsWith('.sqlite') || lower.endsWith('.sqlite3') || lower.endsWith('.db')) {
    credentialType.value = 'sqlite'
  } else if (lower.endsWith('.json')) {
    credentialType.value = 'json'
  }
}

async function submit(activate: boolean) {
  if (!canSubmit.value) return
  submitting.value = true
  errorMessage.value = ''
  successMessage.value = ''
  try {
    if (credentialType.value === 'refresh_token') {
      const tokens = parseRefreshTokenInputs(refreshToken.value)
      for (const [index, token] of tokens.entries()) {
        await adminAPI.kiro.importCredentials({
          type: credentialType.value,
          refreshToken: token,
          file: null,
          profileArn: profileArn.value,
          region: region.value,
          apiRegion: apiRegion.value,
          append: appendCredential.value || index > 0,
          activate: false
        })
      }
      if (activate) {
        const activation = await adminAPI.kiro.activate()
        activationOutput.value = activation.output || activation.message
      }
      successMessage.value = activate ? `已导入 ${tokens.length} 个凭据，Kiro 专属线路已接入` : `已导入 ${tokens.length} 个凭据`
    } else {
      const result = await adminAPI.kiro.importCredentials({
        type: credentialType.value,
        refreshToken: '',
        file: selectedFile.value,
        profileArn: profileArn.value,
        region: region.value,
        apiRegion: apiRegion.value,
        append: appendCredential.value,
        activate
      })
      successMessage.value = result.activation ? '凭据已导入，Kiro 专属线路已接入' : '凭据已导入'
      activationOutput.value = result.activation?.output || activationOutput.value
    }
    refreshToken.value = ''
    await loadStatus()
  } catch (error: unknown) {
    errorMessage.value = extractApiErrorMessage(error, '导入 Kiro 凭据失败')
  } finally {
    submitting.value = false
  }
}

function parseRefreshTokenInputs(input: string): string[] {
  const text = input.trim()
  if (!text) return []
  if (text.startsWith('[')) {
    let payload: unknown
    try {
      payload = JSON.parse(text)
    } catch {
      throw new Error('Token JSON 格式不正确')
    }
    if (!Array.isArray(payload)) throw new Error('Token JSON 必须是数组')
    const tokens = payload.map(readRefreshTokenFromPayload)
    if (!tokens.length) throw new Error('Token JSON 数组不能为空')
    return tokens
  }

  const lines = text
    .split(/\r?\n/)
    .map(normalizeTokenLine)
    .filter(Boolean)
  if (lines.length > 1) {
    return text
      .split(/\r?\n/)
      .map(normalizeTokenLine)
      .filter(Boolean)
      .map(parseRefreshTokenLine)
      .flat()
  }

  return parseRefreshTokenLine(lines[0] || text)
}

function parseRefreshTokenLine(input: string): string[] {
  const text = normalizeTokenLine(input)
  if (!text.startsWith('{')) return [text]

  let payload: unknown
  try {
    payload = JSON.parse(text)
  } catch {
    throw new Error('Token JSON 格式不正确')
  }
  if (!payload || typeof payload !== 'object') {
    throw new Error('Token JSON 必须是对象')
  }

  return [readRefreshTokenFromPayload(payload)]
}

function normalizeTokenLine(input: string): string {
  return input.trim().replace(/^\d+[.)]?\s+(?=\{)/, '')
}

function readRefreshTokenFromPayload(payload: unknown): string {
  if (!payload || typeof payload !== 'object' || Array.isArray(payload)) {
    throw new Error('Token JSON 必须是对象')
  }
  const token = (payload as { refreshToken?: unknown }).refreshToken
  if (typeof token !== 'string' || token.trim() === '') {
    throw new Error('Token JSON 缺少 refreshToken')
  }
  return token.trim()
}

async function activateGateway() {
  activating.value = true
  errorMessage.value = ''
  successMessage.value = ''
  try {
    const result = await adminAPI.kiro.activate()
    activationOutput.value = result.output || result.message
    successMessage.value = 'Kiro 专属线路已接入'
    await loadStatus()
  } catch (error: unknown) {
    errorMessage.value = extractApiErrorMessage(error, '激活 Kiro Gateway 失败')
  } finally {
    activating.value = false
  }
}

function credentialLabel(credential: KiroCredentialEntry): string {
  if (credential.type === 'refresh_token') return `Refresh Token ${credential.refresh_token_sha256 || ''}`.trim()
  return credential.path?.split('/').pop() || credential.type.toUpperCase()
}

function credentialHint(credential: KiroCredentialEntry): string {
  const parts = [credential.region, credential.api_region, credential.profile_arn].filter(Boolean)
  return parts.length ? parts.join(' · ') : credential.path || '已脱敏'
}

function formatFileSize(size: number): string {
  if (size < 1024) return `${size} B`
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`
  return `${(size / 1024 / 1024).toFixed(1)} MB`
}

onMounted(loadStatus)
</script>

<style scoped>
.kiro-page {
  max-width: 1180px;
  margin: 0 auto;
  padding: 0.25rem 0 2rem;
  color: #1f2937;
}

.kiro-hero {
  position: relative;
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 1.5rem;
  align-items: center;
  min-height: 190px;
  overflow: hidden;
  border: 1px solid rgba(226, 215, 200, 0.86);
  border-radius: 1.25rem;
  background:
    linear-gradient(135deg, rgba(255, 252, 245, 0.96), rgba(247, 244, 237, 0.86)),
    radial-gradient(circle at 18% 18%, rgba(20, 184, 166, 0.14), transparent 30%),
    radial-gradient(circle at 82% 28%, rgba(245, 158, 11, 0.16), transparent 32%);
  box-shadow: 0 28px 72px -52px rgba(88, 60, 32, 0.56);
  padding: 2rem;
}

.dark .kiro-hero {
  border-color: rgba(71, 85, 105, 0.55);
  background:
    linear-gradient(135deg, rgba(15, 23, 42, 0.94), rgba(30, 41, 59, 0.84)),
    radial-gradient(circle at 18% 18%, rgba(20, 184, 166, 0.15), transparent 30%),
    radial-gradient(circle at 82% 28%, rgba(245, 158, 11, 0.1), transparent 32%);
}

.kiro-hero-copy h1 {
  margin-top: 0.45rem;
  font-size: clamp(2rem, 5vw, 4.25rem);
  font-weight: 760;
  line-height: 0.95;
  letter-spacing: 0;
  color: #111827;
}

.dark .kiro-hero-copy h1,
.dark .kiro-section-head h2,
.dark .kiro-metric strong,
.dark .kiro-health strong,
.dark .kiro-credential-row strong {
  color: #f8fafc;
}

.kiro-kicker {
  margin: 0;
  color: #0f766e;
  font-size: 0.72rem;
  font-weight: 800;
  letter-spacing: 0.14em;
  text-transform: uppercase;
}

.dark .kiro-kicker {
  color: #5eead4;
}

.kiro-status-row {
  display: flex;
  flex-wrap: wrap;
  gap: 0.75rem;
  align-items: center;
  margin-top: 1.25rem;
}

.kiro-live-chip,
.kiro-pill {
  display: inline-flex;
  align-items: center;
  gap: 0.45rem;
  border-radius: 999px;
  border: 1px solid rgba(15, 118, 110, 0.18);
  background: rgba(240, 253, 250, 0.86);
  color: #0f766e;
  padding: 0.45rem 0.75rem;
  font-size: 0.78rem;
  font-weight: 700;
}

.kiro-live-chip.is-offline {
  border-color: rgba(217, 119, 6, 0.22);
  background: rgba(255, 251, 235, 0.92);
  color: #b45309;
}

.kiro-live-dot {
  width: 0.45rem;
  height: 0.45rem;
  border-radius: 999px;
  background: currentColor;
  animation: kiro-pulse 1.8s ease-in-out infinite;
}

.kiro-muted {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  color: #78716c;
  font-size: 0.8rem;
}

.dark .kiro-muted {
  color: #94a3b8;
}

.kiro-orbit {
  position: relative;
  width: 132px;
  aspect-ratio: 1;
}

.orbit-ring,
.orbit-core {
  position: absolute;
  inset: 0;
  display: grid;
  place-items: center;
  border-radius: 999px;
}

.orbit-ring {
  border: 1px solid rgba(15, 118, 110, 0.22);
}

.ring-a {
  animation: kiro-orbit 16s linear infinite;
  background: conic-gradient(from 90deg, rgba(20, 184, 166, 0.2), transparent 42%, rgba(245, 158, 11, 0.26), transparent 74%);
}

.ring-b {
  inset: 18px;
  border-style: dashed;
  animation: kiro-orbit 11s linear infinite reverse;
}

.orbit-core {
  inset: 38px;
  color: #0f766e;
  background: rgba(255, 255, 255, 0.72);
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.9), 0 18px 40px -28px rgba(15, 118, 110, 0.72);
}

.kiro-alert {
  display: flex;
  align-items: center;
  gap: 0.65rem;
  margin-top: 1rem;
  border-radius: 1rem;
  padding: 0.85rem 1rem;
  font-size: 0.9rem;
}

.kiro-alert-error {
  border: 1px solid rgba(248, 113, 113, 0.35);
  background: rgba(254, 242, 242, 0.9);
  color: #b91c1c;
}

.kiro-alert-success {
  border: 1px solid rgba(45, 212, 191, 0.35);
  background: rgba(240, 253, 250, 0.9);
  color: #0f766e;
}

.kiro-overview {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr)) auto;
  gap: 0.75rem;
  align-items: stretch;
  margin-top: 1rem;
}

.kiro-metric,
.kiro-refresh,
.kiro-panel {
  border: 1px solid rgba(226, 215, 200, 0.78);
  background: rgba(255, 255, 255, 0.86);
  box-shadow: 0 20px 54px -44px rgba(88, 60, 32, 0.52);
}

.dark .kiro-metric,
.dark .kiro-refresh,
.dark .kiro-panel {
  border-color: rgba(71, 85, 105, 0.56);
  background: rgba(15, 23, 42, 0.72);
}

.kiro-metric {
  border-radius: 1rem;
  padding: 1rem;
}

.kiro-metric span,
.kiro-health span,
.kiro-field span {
  color: #78716c;
  font-size: 0.78rem;
  font-weight: 700;
}

.dark .kiro-metric span,
.dark .kiro-health span,
.dark .kiro-field span {
  color: #cbd5e1;
}

.kiro-metric strong {
  display: block;
  margin-top: 0.35rem;
  color: #1f2937;
  font-size: 1.05rem;
}

.kiro-refresh {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  min-width: 7rem;
  border-radius: 1rem;
  color: #0f766e;
  font-size: 0.88rem;
  font-weight: 800;
  transition: transform 0.2s ease, border-color 0.2s ease;
}

.kiro-refresh:hover {
  border-color: rgba(15, 118, 110, 0.3);
  transform: translateY(-1px);
}

.kiro-grid {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(360px, 0.82fr);
  gap: 1rem;
  margin-top: 1rem;
}

.kiro-panel {
  border-radius: 1.25rem;
  padding: 1.25rem;
}

.kiro-section-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 1rem;
  margin-bottom: 1.2rem;
}

.kiro-section-head h2 {
  margin-top: 0.2rem;
  color: #1f2937;
  font-size: 1.18rem;
  font-weight: 760;
  letter-spacing: 0;
}

.kiro-step {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  color: rgba(15, 118, 110, 0.42);
  font-size: 1.8rem;
  font-weight: 800;
}

.kiro-device-login {
  display: grid;
  gap: 0.9rem;
  margin-bottom: 1.1rem;
  border: 1px solid rgba(15, 118, 110, 0.16);
  border-radius: 1rem;
  background:
    linear-gradient(135deg, rgba(240, 253, 250, 0.72), rgba(255, 251, 235, 0.45)),
    rgba(255, 255, 255, 0.58);
  padding: 1rem;
}

.dark .kiro-device-login {
  border-color: rgba(45, 212, 191, 0.22);
  background:
    linear-gradient(135deg, rgba(20, 184, 166, 0.12), rgba(245, 158, 11, 0.08)),
    rgba(15, 23, 42, 0.56);
}

.kiro-device-head,
.kiro-device-title,
.kiro-login-options,
.kiro-login-link {
  display: flex;
  align-items: center;
}

.kiro-device-head {
  justify-content: space-between;
  gap: 0.85rem;
}

.kiro-device-title {
  min-width: 0;
  gap: 0.7rem;
}

.kiro-device-title strong,
.kiro-device-title small {
  display: block;
}

.kiro-device-title strong {
  color: #1f2937;
  font-size: 0.94rem;
}

.dark .kiro-device-title strong {
  color: #f8fafc;
}

.kiro-device-title small {
  margin-top: 0.12rem;
  color: #78716c;
  font-size: 0.76rem;
}

.dark .kiro-device-title small {
  color: #94a3b8;
}

.kiro-device-icon,
.kiro-icon-button {
  display: grid;
  place-items: center;
  width: 2.25rem;
  height: 2.25rem;
  border-radius: 0.8rem;
  border: 1px solid rgba(15, 118, 110, 0.14);
  background: rgba(255, 255, 255, 0.72);
  color: #0f766e;
}

.kiro-login-options {
  flex-wrap: wrap;
  gap: 0.85rem;
}

.kiro-device-card {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 0.7rem;
  align-items: center;
  border: 1px solid rgba(226, 215, 200, 0.75);
  border-radius: 0.9rem;
  background: rgba(255, 255, 255, 0.72);
  padding: 0.85rem;
}

.dark .kiro-device-card {
  border-color: rgba(71, 85, 105, 0.55);
  background: rgba(15, 23, 42, 0.66);
}

.kiro-device-card span {
  color: #78716c;
  font-size: 0.72rem;
  font-weight: 750;
}

.kiro-device-card strong {
  display: block;
  margin-top: 0.18rem;
  color: #111827;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 1.08rem;
  overflow-wrap: anywhere;
}

.dark .kiro-device-card strong {
  color: #f8fafc;
}

.kiro-icon-button {
  cursor: pointer;
  transition: transform 0.18s ease, border-color 0.18s ease;
}

.kiro-icon-button:hover:not(:disabled) {
  border-color: rgba(15, 118, 110, 0.32);
  transform: translateY(-1px);
}

.kiro-icon-button:disabled {
  cursor: not-allowed;
  opacity: 0.45;
}

.kiro-login-link {
  grid-column: 1 / -1;
  justify-content: center;
  gap: 0.45rem;
  min-height: 2.55rem;
  border-radius: 0.8rem;
  background: #0f766e;
  color: #fff;
  font-size: 0.84rem;
  font-weight: 800;
  text-decoration: none;
}

.kiro-device-card p {
  grid-column: 1 / -1;
  margin: 0;
  overflow-wrap: anywhere;
  color: #78716c;
  font-size: 0.78rem;
}

.dark .kiro-device-card p {
  color: #94a3b8;
}

.kiro-login-button {
  width: 100%;
}

.kiro-mode-tabs {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 0.5rem;
  margin-bottom: 1rem;
}

.kiro-mode-tabs button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 0.45rem;
  min-height: 2.75rem;
  border: 1px solid rgba(226, 215, 200, 0.9);
  border-radius: 0.9rem;
  background: rgba(250, 250, 249, 0.78);
  color: #57534e;
  font-weight: 750;
  transition: transform 0.18s ease, background 0.18s ease, color 0.18s ease;
}

.kiro-mode-tabs button:hover,
.kiro-mode-tabs button.active {
  transform: translateY(-1px);
  background: #0f766e;
  color: white;
}

.kiro-form {
  display: grid;
  gap: 1rem;
}

.kiro-field {
  display: grid;
  gap: 0.5rem;
}

.kiro-field small {
  color: #78716c;
  font-size: 0.76rem;
}

.kiro-token-input {
  min-height: 8.5rem;
  resize: vertical;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
}

.kiro-file-drop {
  position: relative;
  display: grid;
  place-items: center;
  gap: 0.45rem;
  min-height: 10rem;
  overflow: hidden;
  border: 1px dashed rgba(15, 118, 110, 0.32);
  border-radius: 1.1rem;
  background:
    linear-gradient(135deg, rgba(240, 253, 250, 0.72), rgba(255, 251, 235, 0.52)),
    repeating-linear-gradient(90deg, rgba(15, 118, 110, 0.05) 0, rgba(15, 118, 110, 0.05) 1px, transparent 1px, transparent 14px);
  color: #0f766e;
  text-align: center;
}

.kiro-file-drop input {
  position: absolute;
  inset: 0;
  cursor: pointer;
  opacity: 0;
}

.kiro-file-drop strong {
  max-width: min(100%, 28rem);
  overflow: hidden;
  color: #1f2937;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.dark .kiro-file-drop strong {
  color: #f8fafc;
}

.kiro-form-grid {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
  gap: 0.85rem;
}

.kiro-check-row {
  display: inline-flex;
  align-items: center;
  gap: 0.55rem;
  color: #57534e;
  font-size: 0.88rem;
  font-weight: 650;
}

.dark .kiro-check-row {
  color: #cbd5e1;
}

.kiro-check-row input {
  width: 1rem;
  height: 1rem;
  accent-color: #0f766e;
}

.kiro-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 0.75rem;
  justify-content: flex-end;
}

.kiro-health {
  display: grid;
  gap: 0.45rem;
  border-top: 1px solid rgba(226, 215, 200, 0.78);
  border-bottom: 1px solid rgba(226, 215, 200, 0.78);
  padding: 1rem 0;
}

.kiro-health div {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
}

.kiro-health p {
  margin: 0;
  overflow-wrap: anywhere;
  color: #78716c;
  font-size: 0.82rem;
}

.dark .kiro-health,
.dark .kiro-credential-row {
  border-color: rgba(71, 85, 105, 0.55);
}

.dark .kiro-health p {
  color: #94a3b8;
}

.kiro-credential-list {
  display: grid;
  gap: 0.65rem;
  margin-top: 1rem;
}

.kiro-credential-row {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  gap: 0.75rem;
  align-items: center;
  border: 1px solid rgba(226, 215, 200, 0.72);
  border-radius: 1rem;
  padding: 0.85rem;
  background: rgba(250, 250, 249, 0.62);
}

.kiro-credential-icon {
  display: grid;
  place-items: center;
  width: 2.25rem;
  height: 2.25rem;
  border-radius: 0.8rem;
  background: rgba(240, 253, 250, 0.92);
  color: #0f766e;
}

.kiro-credential-row strong,
.kiro-credential-row small {
  display: block;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.kiro-credential-row strong {
  color: #1f2937;
  font-size: 0.9rem;
}

.kiro-credential-row small {
  margin-top: 0.18rem;
  color: #78716c;
  font-size: 0.74rem;
}

.kiro-pill {
  padding: 0.32rem 0.55rem;
  font-size: 0.72rem;
}

.kiro-pill.active {
  background: rgba(240, 253, 250, 0.9);
}

.kiro-pill.muted {
  border-color: rgba(120, 113, 108, 0.18);
  background: rgba(245, 245, 244, 0.9);
  color: #78716c;
}

.kiro-empty {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  border: 1px dashed rgba(168, 162, 158, 0.65);
  border-radius: 1rem;
  padding: 1rem;
  color: #78716c;
  font-size: 0.86rem;
}

.kiro-terminal {
  overflow: hidden;
  margin-top: 1rem;
  border-radius: 1rem;
  background: #111827;
  color: #e5e7eb;
}

.kiro-terminal-head {
  display: flex;
  align-items: center;
  gap: 0.4rem;
  border-bottom: 1px solid rgba(255, 255, 255, 0.08);
  padding: 0.65rem 0.8rem;
}

.kiro-terminal-head span {
  width: 0.55rem;
  height: 0.55rem;
  border-radius: 999px;
  background: #f97316;
}

.kiro-terminal-head span:nth-child(2) {
  background: #facc15;
}

.kiro-terminal-head span:nth-child(3) {
  background: #14b8a6;
}

.kiro-terminal-head strong {
  margin-left: 0.35rem;
  color: #94a3b8;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 0.72rem;
  font-weight: 700;
}

.kiro-terminal pre {
  min-height: 8.5rem;
  max-height: 18rem;
  margin: 0;
  overflow: auto;
  padding: 0.9rem;
  white-space: pre-wrap;
  word-break: break-word;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 0.76rem;
  line-height: 1.6;
}

.kiro-spin {
  animation: kiro-spin 0.8s linear infinite;
}

@keyframes kiro-pulse {
  0%,
  100% {
    box-shadow: 0 0 0 0 currentColor;
    opacity: 0.72;
  }
  50% {
    box-shadow: 0 0 0 7px transparent;
    opacity: 1;
  }
}

@keyframes kiro-orbit {
  to {
    transform: rotate(360deg);
  }
}

@keyframes kiro-spin {
  to {
    transform: rotate(360deg);
  }
}

@media (max-width: 900px) {
  .kiro-hero,
  .kiro-grid,
  .kiro-overview,
  .kiro-form-grid {
    grid-template-columns: 1fr;
  }

  .kiro-orbit {
    width: 104px;
  }

  .kiro-refresh {
    min-height: 3rem;
  }
}

@media (max-width: 640px) {
  .kiro-page {
    padding-bottom: 5rem;
  }

  .kiro-hero,
  .kiro-panel {
    border-radius: 1rem;
    padding: 1rem;
  }

  .kiro-actions {
    justify-content: stretch;
  }

  .kiro-actions .btn {
    flex: 1 1 100%;
  }
}
</style>
