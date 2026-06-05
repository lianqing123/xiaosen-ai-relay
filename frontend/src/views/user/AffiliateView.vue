<template>
  <AppLayout>
    <div class="space-y-6">
      <div v-if="loading" class="space-y-4 py-2">
        <div class="h-44 animate-pulse rounded-[1.75rem] border border-stone-200/70 bg-white/70 dark:border-dark-700 dark:bg-dark-800/50"></div>
        <div class="grid gap-4 md:grid-cols-2">
          <div class="h-36 animate-pulse rounded-2xl border border-stone-200/70 bg-white/70 dark:border-dark-700 dark:bg-dark-800/50"></div>
          <div class="h-36 animate-pulse rounded-2xl border border-stone-200/70 bg-white/70 dark:border-dark-700 dark:bg-dark-800/50"></div>
        </div>
      </div>

      <template v-else-if="detail">
        <section class="affiliate-hero">
          <div class="affiliate-hero-glow" aria-hidden="true"></div>
          <div class="relative grid gap-6 lg:grid-cols-[1.35fr_0.65fr] lg:items-end">
            <div>
              <div class="inline-flex items-center gap-2 rounded-full border border-emerald-200/80 bg-white/70 px-3 py-1 text-xs font-medium text-emerald-800 shadow-[inset_0_1px_0_rgba(255,255,255,0.8)] dark:border-primary-800/50 dark:bg-dark-900/50 dark:text-primary-200">
                <span class="h-1.5 w-1.5 rounded-full bg-emerald-500"></span>
                CNY REBATE LEDGER
              </div>
              <h2 class="mt-5 text-2xl font-semibold tracking-tight text-stone-950 dark:text-white md:text-3xl">
                {{ t('affiliate.title') }}
              </h2>
              <p class="mt-2 max-w-2xl text-sm leading-6 text-stone-600 dark:text-dark-300">
                {{ t('affiliate.description') }}
              </p>
              <div class="mt-6 flex flex-wrap items-end gap-4">
                <div>
                  <p class="text-xs font-medium uppercase tracking-[0.18em] text-stone-500 dark:text-dark-400">
                    {{ t('affiliate.stats.availableQuota') }}
                  </p>
                  <p class="mt-1 font-mono text-4xl font-semibold tracking-tight text-emerald-800 dark:text-primary-200 md:text-5xl">
                    {{ formatCnyAmount(detail.aff_quota) }}
                  </p>
                </div>
                <div v-if="detail.aff_frozen_quota > 0" class="rounded-2xl border border-amber-200 bg-amber-50/80 px-4 py-3 text-sm text-amber-800 dark:border-amber-900/50 dark:bg-amber-950/30 dark:text-amber-200">
                  {{ t('affiliate.stats.frozenQuota') }}: {{ formatCnyAmount(detail.aff_frozen_quota) }}
                </div>
              </div>
            </div>

            <div class="grid grid-cols-3 gap-3 lg:grid-cols-1">
              <div class="affiliate-meter">
                <p>{{ t('affiliate.stats.rebateRate') }}</p>
                <strong>{{ formattedRebateRate }}%</strong>
              </div>
              <div class="affiliate-meter">
                <p>{{ t('affiliate.stats.invitedUsers') }}</p>
                <strong>{{ formatCount(detail.aff_count) }}</strong>
              </div>
              <div class="affiliate-meter">
                <p>{{ t('affiliate.stats.totalQuota') }}</p>
                <strong>{{ formatCnyAmount(detail.aff_history_quota) }}</strong>
              </div>
            </div>
          </div>
        </section>

        <div class="grid gap-4 lg:grid-cols-[1.15fr_0.85fr]">
          <section class="card overflow-hidden p-0">
            <div class="border-b border-stone-200/70 px-5 py-4 dark:border-dark-700">
              <h3 class="text-base font-semibold text-stone-950 dark:text-white">{{ t('affiliate.yourCode') }}</h3>
              <p class="mt-1 text-sm text-stone-500 dark:text-dark-400">{{ t('affiliate.tips.line1') }}</p>
            </div>
            <div class="grid gap-4 p-5 md:grid-cols-2">
              <div class="space-y-2">
                <p class="text-sm font-medium text-stone-700 dark:text-gray-300">{{ t('affiliate.yourCode') }}</p>
                <div class="flex min-h-[44px] items-center gap-2 rounded-2xl border border-stone-200 bg-stone-50/80 px-3 py-2 dark:border-dark-700 dark:bg-dark-900">
                  <code class="flex-1 truncate font-mono text-sm font-semibold text-stone-950 dark:text-white">{{ detail.aff_code }}</code>
                  <button class="btn btn-secondary btn-sm" @click="copyCode">
                    <Icon name="copy" size="sm" />
                    <span>{{ t('affiliate.copyCode') }}</span>
                  </button>
                </div>
              </div>

              <div class="space-y-2">
                <p class="text-sm font-medium text-stone-700 dark:text-gray-300">{{ t('affiliate.inviteLink') }}</p>
                <div class="flex min-h-[44px] items-center gap-2 rounded-2xl border border-stone-200 bg-stone-50/80 px-3 py-2 dark:border-dark-700 dark:bg-dark-900">
                  <code class="flex-1 truncate text-sm text-stone-600 dark:text-gray-300">{{ inviteLink }}</code>
                  <button class="btn btn-secondary btn-sm" @click="copyInviteLink">
                    <Icon name="copy" size="sm" />
                    <span>{{ t('affiliate.copyLink') }}</span>
                  </button>
                </div>
              </div>
            </div>
            <div class="mx-5 mb-5 rounded-2xl border border-emerald-200/70 bg-emerald-50/70 p-4 text-sm text-emerald-900 dark:border-primary-900/40 dark:bg-primary-900/20 dark:text-primary-200">
              <div class="flex gap-3">
                <Icon name="sparkles" size="sm" class="mt-0.5 flex-shrink-0 text-emerald-700 dark:text-primary-300" />
                <div class="space-y-1">
                  <p>{{ t('affiliate.tips.line2', { rate: `${formattedRebateRate}%` }) }}</p>
                  <p>{{ t('affiliate.tips.line3') }}</p>
                  <p v-if="detail.aff_frozen_quota > 0">{{ t('affiliate.tips.line4') }}</p>
                </div>
              </div>
            </div>
          </section>

          <section class="card overflow-hidden p-0">
            <div class="flex h-full flex-col">
              <div class="border-b border-stone-200/70 px-5 py-4 dark:border-dark-700">
                <h3 class="text-base font-semibold text-stone-950 dark:text-white">{{ t('affiliate.transfer.title') }}</h3>
                <p class="mt-1 text-sm leading-6 text-stone-500 dark:text-dark-400">{{ t('affiliate.transfer.description') }}</p>
              </div>
              <div class="flex flex-1 flex-col justify-between gap-5 p-5">
                <div class="rounded-2xl border border-stone-200 bg-white/70 p-4 dark:border-dark-700 dark:bg-dark-900/40">
                  <p class="text-xs font-medium uppercase tracking-[0.16em] text-stone-500 dark:text-dark-400">
                    {{ t('affiliate.stats.availableQuota') }}
                  </p>
                  <p class="mt-2 font-mono text-3xl font-semibold text-stone-950 dark:text-white">
                    {{ formatCnyAmount(detail.aff_quota) }}
                  </p>
                  <p v-if="detail.aff_quota <= 0" class="mt-2 text-sm text-amber-700 dark:text-amber-300">
                    {{ t('affiliate.transfer.empty') }}
                  </p>
                </div>
                <button
                  class="btn btn-primary w-full justify-center"
                  :disabled="transferring || detail.aff_quota <= 0"
                  @click="transferQuota"
                >
                  <Icon v-if="transferring" name="refresh" size="sm" class="animate-spin" />
                  <Icon v-else name="creditCard" size="sm" />
                  <span>{{ transferring ? t('affiliate.transfer.transferring') : t('affiliate.transfer.button') }}</span>
                </button>
              </div>
            </div>
          </section>
        </div>

        <section class="card overflow-hidden p-0">
          <div class="flex flex-col gap-2 border-b border-stone-200/70 px-5 py-4 dark:border-dark-700 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <h3 class="text-base font-semibold text-stone-950 dark:text-white">{{ t('affiliate.invitees.title') }}</h3>
              <p class="mt-1 text-sm text-stone-500 dark:text-dark-400">{{ t('affiliate.stats.rebateRateHint') }}</p>
            </div>
            <div class="rounded-full border border-stone-200 bg-stone-50 px-3 py-1 text-xs font-medium text-stone-600 dark:border-dark-700 dark:bg-dark-900 dark:text-dark-300">
              {{ t('affiliate.invitees.total', { count: formatCount(detail.invitees.length) }) }}
            </div>
          </div>
          <div v-if="detail.invitees.length === 0" class="m-5 rounded-2xl border border-dashed border-stone-300 p-8 text-center text-sm text-stone-500 dark:border-dark-700 dark:text-dark-400">
            {{ t('affiliate.invitees.empty') }}
          </div>
          <div v-else class="overflow-x-auto">
            <table class="w-full min-w-[760px] text-left text-sm">
              <thead class="bg-stone-50/80 text-xs uppercase tracking-wide text-stone-500 dark:bg-dark-900/60 dark:text-dark-400">
                <tr>
                  <th class="px-5 py-3 font-medium">{{ t('affiliate.invitees.columns.email') }}</th>
                  <th class="px-5 py-3 font-medium">{{ t('affiliate.invitees.columns.username') }}</th>
                  <th class="px-5 py-3 text-right font-medium">{{ t('affiliate.invitees.columns.recharged') }}</th>
                  <th class="px-5 py-3 text-right font-medium">{{ t('affiliate.invitees.columns.rebate') }}</th>
                  <th class="px-5 py-3 font-medium">{{ t('affiliate.invitees.columns.joinedAt') }}</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-stone-100 bg-white/50 dark:divide-dark-800 dark:bg-dark-900/20">
                <tr
                  v-for="item in detail.invitees"
                  :key="item.user_id"
                  class="transition-colors hover:bg-emerald-50/40 dark:hover:bg-dark-800/40"
                >
                  <td class="px-5 py-4 font-medium text-stone-950 dark:text-white">{{ item.email || '-' }}</td>
                  <td class="px-5 py-4 text-stone-600 dark:text-gray-300">{{ item.username || '-' }}</td>
                  <td class="px-5 py-4 text-right font-mono font-medium text-stone-950 dark:text-white">{{ formatCnyAmount(item.total_recharged_amount) }}</td>
                  <td class="px-5 py-4 text-right font-mono font-semibold text-emerald-700 dark:text-emerald-300">{{ formatCnyAmount(item.total_rebate) }}</td>
                  <td class="px-5 py-4 text-stone-600 dark:text-gray-300">{{ formatDateTime(item.created_at) || '-' }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </section>
      </template>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import userAPI from '@/api/user'
import type { UserAffiliateDetail } from '@/types'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { useClipboard } from '@/composables/useClipboard'
import { formatCurrency, formatDateTime } from '@/utils/format'
import { extractApiErrorMessage } from '@/utils/apiError'
import { buildAffiliateRegisterLink } from '@/utils/invitationRegisterLink'

const { t } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()
const { copyToClipboard } = useClipboard()

const loading = ref(true)
const transferring = ref(false)
const detail = ref<UserAffiliateDetail | null>(null)

const inviteLink = computed(() => {
  if (!detail.value) return ''
  const origin = typeof window === 'undefined' ? '' : window.location.origin
  return buildAffiliateRegisterLink(origin, detail.value.aff_code)
})

// Rebate rate is a percentage in the range [0, 100]; backend already clamps it.
// We trim trailing zeros (e.g. 20.00 → "20", 12.50 → "12.5") for a cleaner UI.
const formattedRebateRate = computed(() => {
  const v = detail.value?.effective_rebate_rate_percent ?? 0
  const rounded = Math.round(v * 100) / 100
  return Number.isInteger(rounded) ? String(rounded) : rounded.toString()
})

function formatCount(value: number): string {
  return value.toLocaleString()
}

function formatCnyAmount(value: number | null | undefined): string {
  return formatCurrency(value ?? 0, 'CNY')
}

async function loadAffiliateDetail(silent = false): Promise<void> {
  if (!silent) {
    loading.value = true
  }
  try {
    detail.value = await userAPI.getAffiliateDetail()
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('affiliate.loadFailed')))
  } finally {
    if (!silent) {
      loading.value = false
    }
  }
}

async function copyCode(): Promise<void> {
  if (!detail.value?.aff_code) return
  await copyToClipboard(detail.value.aff_code, t('affiliate.codeCopied'))
}

async function copyInviteLink(): Promise<void> {
  if (!inviteLink.value) return
  await copyToClipboard(inviteLink.value, t('affiliate.linkCopied'))
}

async function transferQuota(): Promise<void> {
  if (!detail.value || detail.value.aff_quota <= 0 || transferring.value) return
  transferring.value = true
  try {
    const resp = await userAPI.transferAffiliateQuota()
    appStore.showSuccess(t('affiliate.transfer.success', { amount: formatCnyAmount(resp.transferred_quota) }))
    await Promise.all([
      loadAffiliateDetail(true),
      authStore.refreshUser().catch(() => undefined),
    ])
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('affiliate.transferFailed')))
  } finally {
    transferring.value = false
  }
}

onMounted(() => {
  void loadAffiliateDetail()
})
</script>
