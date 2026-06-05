<template>
  <div
    :class="[
      'group relative flex min-h-[470px] flex-col overflow-hidden rounded-[1.75rem] border transition-all duration-300',
      'shadow-[0_20px_55px_-34px_rgba(15,23,42,0.32)] hover:-translate-y-1 hover:shadow-[0_26px_70px_-36px_rgba(15,23,42,0.42)]',
      borderClass,
      isRenewal
        ? 'bg-gradient-to-b from-emerald-50/90 via-white to-white dark:from-emerald-950/20 dark:via-dark-800 dark:to-dark-800'
        : 'bg-white/95 dark:bg-dark-800',
    ]"
  >
    <div :class="['h-1.5', accentClass]" />

    <div class="flex flex-1 flex-col p-5 sm:p-6">
      <div class="mb-4 flex flex-wrap items-center gap-2">
        <span :class="['rounded-full px-2.5 py-1 text-[11px] font-semibold', badgeLightClass]">
          {{ pLabel }}
        </span>
        <span
          v-for="tag in benefitTags"
          :key="tag"
          class="rounded-full border border-gray-200 bg-gray-50 px-2.5 py-1 text-[11px] font-semibold text-gray-600 dark:border-dark-600 dark:bg-dark-700 dark:text-gray-300"
        >
          {{ tag }}
        </span>
        <span v-if="isRenewal" class="rounded-full bg-emerald-100 px-2.5 py-1 text-[11px] font-semibold text-emerald-700 dark:bg-emerald-900/40 dark:text-emerald-300">
          {{ t('payment.currentPlan') }}
        </span>
      </div>

      <div class="mb-5 grid gap-4 sm:grid-cols-[1fr_auto] sm:items-start">
        <div class="min-w-0">
          <h3 class="text-[1.35rem] font-bold leading-tight tracking-tight text-gray-950 dark:text-white">
            {{ plan.name }}
          </h3>
          <p v-if="plan.description" class="mt-2 min-h-[52px] text-sm leading-6 text-gray-600 dark:text-gray-300">
            {{ plan.description }}
          </p>
        </div>

        <div class="min-w-[118px] rounded-2xl border border-gray-100 bg-gray-50/80 px-4 py-3 text-right dark:border-dark-700 dark:bg-dark-700/40">
          <div class="flex items-baseline justify-end gap-1 whitespace-nowrap">
            <span class="text-xs font-semibold text-gray-400 dark:text-dark-500">¥</span>
            <span :class="['text-[2rem] font-extrabold leading-none tracking-tight', textClass]">{{ plan.price }}</span>
          </div>
          <span class="mt-1 block text-xs font-medium text-gray-400 dark:text-dark-500">/ {{ validitySuffix }}</span>
          <div v-if="plan.original_price" class="mt-2 flex items-center justify-end gap-1.5">
            <span class="text-xs text-gray-400 line-through dark:text-dark-500">¥{{ plan.original_price }}</span>
            <span :class="['rounded px-1.5 py-0.5 text-[10px] font-semibold', discountClass]">{{ discountText }}</span>
          </div>
        </div>
      </div>

      <div class="mb-5 rounded-3xl border border-gray-100 bg-gray-50/80 p-4 text-sm dark:border-dark-700 dark:bg-dark-700/40">
        <div class="mb-3 flex items-center justify-between gap-3">
          <span class="text-xs font-semibold uppercase tracking-[0.16em] text-gray-400 dark:text-dark-500">
            {{ t('payment.planCard.quota') }}
          </span>
          <span class="rounded-full bg-white px-2.5 py-1 text-xs font-semibold text-gray-600 ring-1 ring-gray-200 dark:bg-dark-800 dark:text-gray-300 dark:ring-dark-600">
            {{ rateDisplay }}
          </span>
        </div>

        <div v-if="quotaItems.length > 0" class="grid grid-cols-3 gap-2">
          <div
            v-for="item in quotaItems"
            :key="item.label"
            class="rounded-2xl bg-white px-3 py-3 ring-1 ring-gray-100 dark:bg-dark-800 dark:ring-dark-600"
          >
            <div class="text-[11px] font-medium text-gray-400 dark:text-dark-500">{{ item.label }}</div>
            <div class="mt-1 whitespace-nowrap text-base font-bold leading-none text-gray-950 dark:text-white">
              ${{ item.value }}
            </div>
          </div>
        </div>
        <div v-else class="rounded-2xl bg-white px-3 py-3 text-sm font-semibold text-gray-800 ring-1 ring-gray-100 dark:bg-dark-800 dark:text-gray-200 dark:ring-dark-600">
          {{ quotaDisplay }}
        </div>

        <div v-if="modelScopeLabels.length > 0" class="mt-3 flex items-center justify-between gap-3 border-t border-gray-200/70 pt-3 dark:border-dark-600">
          <span class="shrink-0 text-xs font-medium text-gray-400 dark:text-dark-500">{{ t('payment.planCard.models') }}</span>
          <span class="text-right text-sm font-semibold text-gray-800 dark:text-gray-100">{{ modelScopeLabels.join(' · ') }}</span>
        </div>
        <div v-if="includedRouteLabels.length > 1" class="mt-3 flex items-start justify-between gap-3 border-t border-gray-200/70 pt-3 dark:border-dark-600">
          <span class="shrink-0 text-xs font-medium text-gray-400 dark:text-dark-500">{{ t('payment.planCard.routes') }}</span>
          <span class="text-right text-sm font-semibold text-gray-800 dark:text-gray-100">{{ includedRouteLabels.join(' · ') }}</span>
        </div>
      </div>

      <div v-if="plan.features.length > 0" class="mb-5 space-y-2.5">
        <div v-for="feature in plan.features" :key="feature" class="flex items-start gap-2">
          <svg :class="['mt-0.5 h-4 w-4 flex-shrink-0', iconClass]" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5">
            <path stroke-linecap="round" stroke-linejoin="round" d="M4.5 12.75l6 6 9-13.5" />
          </svg>
          <span class="text-sm leading-5 text-gray-700 dark:text-gray-300">{{ feature }}</span>
        </div>
      </div>

      <div class="flex-1" />

      <!-- Subscribe Button -->
      <button
        type="button"
        :class="['w-full rounded-xl py-3 text-sm font-semibold transition-all active:scale-[0.98]', btnClass]"
        @click="emit('select', plan)"
      >
        {{ isRenewal ? t('payment.renewNow') : t('payment.subscribeNow') }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { SubscriptionPlan } from '@/types/payment'
import type { UserSubscription } from '@/types'
import { normalizePlanValidityUnit } from '@/utils/subscriptionPlan'
import {
  platformAccentBarClass,
  platformBadgeLightClass,
  platformBorderClass,
  platformTextClass,
  platformIconClass,
  platformButtonClass,
  platformDiscountClass,
  platformLabel,
} from '@/utils/platformColors'

const props = defineProps<{ plan: SubscriptionPlan; activeSubscriptions?: UserSubscription[] }>()
const emit = defineEmits<{ select: [plan: SubscriptionPlan] }>()
const { t } = useI18n()

const platform = computed(() => props.plan.group_platform || '')
const isRenewal = computed(() =>
  props.activeSubscriptions?.some(s => s.group_id === props.plan.group_id && s.status === 'active') ?? false
)

// Derived color classes from central config
const accentClass = computed(() => platformAccentBarClass(platform.value))
const borderClass = computed(() => platformBorderClass(platform.value))
const badgeLightClass = computed(() => platformBadgeLightClass(platform.value))
const textClass = computed(() => platformTextClass(platform.value))
const iconClass = computed(() => platformIconClass(platform.value))
const btnClass = computed(() => platformButtonClass(platform.value))
const discountClass = computed(() => platformDiscountClass(platform.value))
const pLabel = computed(() => platformLabel(platform.value))

const discountText = computed(() => {
  if (!props.plan.original_price || props.plan.original_price <= 0) return ''
  const pct = Math.round((1 - props.plan.price / props.plan.original_price) * 100)
  return pct > 0 ? `-${pct}%` : ''
})

const rateDisplay = computed(() => {
  const rate = props.plan.rate_multiplier ?? 1
  return `×${Number(rate.toPrecision(10))}`
})

const formatQuotaAmount = (value: number | null | undefined) => {
  if (value == null) return ''
  const n = Number(value)
  return Number.isInteger(n) ? String(n) : n.toFixed(2).replace(/\.?0+$/, '')
}

const quotaItems = computed(() => [
  props.plan.daily_limit_usd != null ? { label: '日', value: formatQuotaAmount(props.plan.daily_limit_usd) } : null,
  props.plan.weekly_limit_usd != null ? { label: '周', value: formatQuotaAmount(props.plan.weekly_limit_usd) } : null,
  props.plan.monthly_limit_usd != null ? { label: '月', value: formatQuotaAmount(props.plan.monthly_limit_usd) } : null,
].filter((item): item is { label: string; value: string } => item !== null))

const quotaDisplay = computed(() => {
  return quotaItems.value.length > 0
    ? quotaItems.value.map(item => `${item.label} $${item.value}`).join(' · ')
    : t('payment.planCard.unlimited')
})

const benefitTags = computed(() => {
  const text = props.plan.description || ''
  const tags: string[] = []

  if (text.includes('推荐')) tags.push('推荐')
  else if (text.includes('高频使用')) tags.push('高频使用')
  else if (text.includes('团队/重度')) tags.push('团队/重度')
  else if (text.includes('基础月付')) tags.push('基础月付')
  else if (text.includes('低门槛体验')) tags.push('低门槛体验')

  const more = text.match(/\+(\d+)%/)
  if (more) tags.push(`加量 +${more[1]}%`)

  const save = text.match(/省约\s*(\d+)%/)
  if (save) tags.push(`省约 ${save[1]}%`)

  return tags
})

const modelScopeLabels = computed(() => {
  const scopes = props.plan.supported_model_scopes
  if (!scopes || scopes.length === 0) return []
  const labels = ['GPT 全系列']
  if (scopes.includes('gemini_image')) labels.push('Imagen')
  return labels
})

const includedRouteLabels = computed(() => {
  const groups = props.plan.included_groups || []
  if (groups.length > 0) {
    return groups.map(group => group.name).filter(Boolean)
  }
  if (props.plan.group_name) return [props.plan.group_name]
  return []
})

const validitySuffix = computed(() => {
  const u = normalizePlanValidityUnit(props.plan.validity_unit)
  if (u === 'week') return `${props.plan.validity_days}${t('payment.weeks')}`
  if (u === 'month') return t('payment.perMonth')
  if (u === 'year') return t('payment.perYear')
  return `${props.plan.validity_days}${t('payment.days')}`
})
</script>
