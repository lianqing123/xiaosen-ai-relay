<template>
  <section
    class="overflow-hidden rounded-3xl border border-gray-200 bg-white shadow-sm dark:border-dark-700 dark:bg-dark-900"
  >
    <div class="border-b border-gray-100 px-6 py-5 dark:border-dark-700 md:px-8">
      <div class="flex flex-col gap-3 md:flex-row md:items-end md:justify-between">
        <div>
          <p class="text-xs font-semibold uppercase tracking-[0.18em] text-gray-400 dark:text-gray-500">
            Usage Dashboard
          </p>
          <h2 class="mt-2 text-xl font-semibold tracking-tight text-gray-950 dark:text-white">
            {{ t('userSubscriptions.usageDashboard.title') }}
          </h2>
          <p class="mt-2 max-w-2xl text-sm leading-6 text-gray-500 dark:text-gray-400">
            {{ t('userSubscriptions.usageDashboard.description') }}
          </p>
        </div>
        <div class="rounded-full border border-gray-200 px-3 py-1 text-xs font-medium text-gray-500 dark:border-dark-600 dark:text-gray-400">
          {{ t('userSubscriptions.usageDashboard.activeSubscriptions', { count: activeSubscriptionsCount }) }}
        </div>
      </div>
    </div>

    <div v-if="activeSubscriptionsCount === 0" class="px-6 py-8 md:px-8">
      <div class="rounded-2xl border border-gray-200 bg-gray-50 p-5 dark:border-dark-700 dark:bg-dark-800">
        <p class="font-medium text-gray-800 dark:text-gray-200">
          {{ t('userSubscriptions.usageDashboard.noActive') }}
        </p>
        <p class="mt-2 text-sm leading-6 text-gray-500 dark:text-gray-400">
          {{ t('userSubscriptions.usageDashboard.noActiveDescription') }}
        </p>
      </div>
    </div>

    <div v-else-if="metrics.length > 0" class="grid gap-px bg-gray-100 dark:bg-dark-700 md:grid-cols-3">
      <article
        v-for="metric in metrics"
        :key="metric.key"
        class="bg-white p-6 dark:bg-dark-900 md:p-7"
      >
        <div class="flex items-start justify-between gap-4">
          <div>
            <p class="text-sm font-medium text-gray-500 dark:text-gray-400">
              {{ metric.label }}
            </p>
            <p class="mt-2 font-mono text-3xl font-semibold tracking-tight text-gray-950 dark:text-white">
              {{ metric.percentageText }}
            </p>
          </div>
          <span
            class="rounded-full px-2.5 py-1 text-xs font-semibold"
            :class="metric.badgeClass"
          >
            {{ metric.badgeText }}
          </span>
        </div>

        <div class="mt-5 h-2 overflow-hidden rounded-full bg-gray-100 dark:bg-dark-700">
          <div
            class="h-full rounded-full transition-all duration-500"
            :class="metric.barClass"
            :style="{ width: metric.progressWidth }"
          />
        </div>

        <div class="mt-4 flex items-center justify-between gap-4 text-sm">
          <span class="text-gray-500 dark:text-gray-400">
            {{ metric.amountText }}
          </span>
          <span class="font-medium text-gray-700 dark:text-gray-300">
            {{ metric.remainingText }}
          </span>
        </div>
      </article>
    </div>

    <div v-else class="px-6 py-8 md:px-8">
      <div class="rounded-2xl border border-emerald-100 bg-emerald-50/70 p-5 dark:border-emerald-900/50 dark:bg-emerald-900/20">
        <p class="font-medium text-emerald-800 dark:text-emerald-200">
          {{ t('userSubscriptions.usageDashboard.unlimited') }}
        </p>
        <p class="mt-2 text-sm leading-6 text-emerald-700/80 dark:text-emerald-200/70">
          {{ t('userSubscriptions.usageDashboard.unlimitedDescription') }}
        </p>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { UserSubscription } from '@/types'

type UsageWindow = 'daily' | 'weekly' | 'monthly'

interface UsageMetric {
  key: UsageWindow
  label: string
  used: number
  limit: number
  percentage: number
  percentageText: string
  progressWidth: string
  amountText: string
  remainingText: string
  badgeText: string
  badgeClass: string
  barClass: string
}

const props = defineProps<{
  subscriptions: UserSubscription[]
}>()

const { t } = useI18n()

const activeSubscriptions = computed(() =>
  props.subscriptions.filter((subscription) => subscription.status === 'active')
)

const activeSubscriptionsCount = computed(() => activeSubscriptions.value.length)

const formatCurrency = (value: number): string => `$${value.toFixed(2)}`

const usageConfig: Array<{
  key: UsageWindow
  labelKey: string
  usageField: keyof Pick<UserSubscription, 'daily_usage_usd' | 'weekly_usage_usd' | 'monthly_usage_usd'>
  limitField: 'daily_limit_usd' | 'weekly_limit_usd' | 'monthly_limit_usd'
}> = [
  {
    key: 'daily',
    labelKey: 'userSubscriptions.daily',
    usageField: 'daily_usage_usd',
    limitField: 'daily_limit_usd'
  },
  {
    key: 'weekly',
    labelKey: 'userSubscriptions.weekly',
    usageField: 'weekly_usage_usd',
    limitField: 'weekly_limit_usd'
  },
  {
    key: 'monthly',
    labelKey: 'userSubscriptions.monthly',
    usageField: 'monthly_usage_usd',
    limitField: 'monthly_limit_usd'
  }
]

const metrics = computed<UsageMetric[]>(() =>
  usageConfig.flatMap((config) => {
    const scopedSubscriptions = activeSubscriptions.value.filter((subscription) => {
      const limit = subscription.group?.[config.limitField]
      return typeof limit === 'number' && limit > 0
    })

    if (scopedSubscriptions.length === 0) {
      return []
    }

    const used = scopedSubscriptions.reduce(
      (sum, subscription) => sum + Number(subscription[config.usageField] || 0),
      0
    )
    const limit = scopedSubscriptions.reduce(
      (sum, subscription) => sum + Number(subscription.group?.[config.limitField] || 0),
      0
    )
    const rawPercentage = limit > 0 ? (used / limit) * 100 : 0
    const percentage = Math.min(Math.max(rawPercentage, 0), 100)
    const remaining = Math.max(limit - used, 0)

    let badgeText = t('userSubscriptions.usageDashboard.normal')
    let badgeClass = 'bg-emerald-50 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300'
    let barClass = 'bg-emerald-500'

    if (rawPercentage >= 90) {
      badgeText = t('userSubscriptions.usageDashboard.nearLimit')
      badgeClass = 'bg-red-50 text-red-700 dark:bg-red-900/30 dark:text-red-300'
      barClass = 'bg-red-500'
    } else if (rawPercentage >= 70) {
      badgeText = t('userSubscriptions.usageDashboard.watch')
      badgeClass = 'bg-amber-50 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300'
      barClass = 'bg-amber-500'
    }

    return [
      {
        key: config.key,
        label: t(config.labelKey),
        used,
        limit,
        percentage,
        percentageText: `${Math.round(percentage)}%`,
        progressWidth: `${percentage}%`,
        amountText: `${formatCurrency(used)} / ${formatCurrency(limit)}`,
        remainingText: t('userSubscriptions.usageDashboard.remaining', {
          amount: formatCurrency(remaining)
        }),
        badgeText,
        badgeClass,
        barClass
      }
    ]
  })
)
</script>
