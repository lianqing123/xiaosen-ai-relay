<template>
  <div class="card overflow-hidden">
    <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
      <div class="flex items-start justify-between gap-4">
        <div>
          <h2 class="text-lg font-medium text-gray-900 dark:text-white">
            订阅超额处理
          </h2>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
            订阅日、周、月额度用完后，是否允许继续按余额计费使用统一 API。
          </p>
        </div>
        <span
          class="rounded-full border px-3 py-1 text-xs font-medium"
          :class="overagePaygEnabled ? 'border-emerald-200 bg-emerald-50 text-emerald-700 dark:border-emerald-800 dark:bg-emerald-900/20 dark:text-emerald-300' : 'border-gray-200 bg-gray-50 text-gray-500 dark:border-dark-600 dark:bg-dark-800 dark:text-gray-400'"
        >
          {{ overagePaygEnabled ? '已开启' : '已关闭' }}
        </span>
      </div>
    </div>

    <div class="space-y-5 px-6 py-6">
      <div class="rounded-2xl border border-gray-100 bg-gray-50/70 p-4 dark:border-dark-700 dark:bg-dark-800/60">
        <div class="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <p class="font-medium text-gray-900 dark:text-white">超额后按量付费</p>
            <p class="mt-1 text-sm leading-6 text-gray-500 dark:text-gray-400">
              开启后：订阅额度内扣订阅额度；额度超出时自动切到余额扣费。关闭后：达到订阅限额即停止请求。
            </p>
          </div>
          <label class="relative inline-flex cursor-pointer items-center">
            <input
              v-model="overagePaygEnabled"
              type="checkbox"
              class="peer sr-only"
              :disabled="saving"
              @change="handleToggle"
            />
            <div class="h-6 w-11 rounded-full bg-gray-200 transition peer-checked:bg-primary-600 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-primary-300 after:absolute after:left-[2px] after:top-[2px] after:h-5 after:w-5 after:rounded-full after:border after:border-gray-300 after:bg-white after:transition-all after:content-[''] peer-checked:after:translate-x-full peer-checked:after:border-white dark:bg-gray-700 dark:peer-focus:ring-primary-800"></div>
          </label>
        </div>
      </div>

      <p class="text-xs leading-5 text-gray-400 dark:text-gray-500">
        这个开关只影响订阅分组超额后的统一 API 计费映射，不修改套餐额度、到期时间或后台分组配置。
      </p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { userAPI } from '@/api'
import { extractApiErrorMessage } from '@/utils/apiError'

const props = defineProps<{
  enabled: boolean
}>()

const appStore = useAppStore()
const authStore = useAuthStore()
const overagePaygEnabled = ref(props.enabled)
const saving = ref(false)

watch(() => props.enabled, (value) => {
  overagePaygEnabled.value = value
})

const handleToggle = async () => {
  const nextValue = overagePaygEnabled.value
  saving.value = true
  try {
    const updated = await userAPI.updateProfile({
      subscription_overage_payg_enabled: nextValue
    })
    authStore.user = updated
    appStore.showSuccess('设置已保存')
  } catch (err: unknown) {
    overagePaygEnabled.value = !nextValue
    appStore.showError(extractApiErrorMessage(err, '保存失败'))
  } finally {
    saving.value = false
  }
}
</script>
