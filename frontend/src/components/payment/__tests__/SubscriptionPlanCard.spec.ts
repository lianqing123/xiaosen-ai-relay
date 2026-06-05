import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => {
        const messages: Record<string, string> = {
          'payment.planCard.rate': '倍率',
          'payment.planCard.quota': '额度',
          'payment.planCard.models': '模型支持',
          'payment.days': '天',
          'payment.weeks': '周',
          'payment.perMonth': '月',
          'payment.perYear': '年',
          'payment.subscribeNow': '立即订阅',
          'payment.renewNow': '立即续费',
          'payment.currentPlan': '当前套餐',
        }
        return messages[key] || key
      },
    }),
  }
})

import SubscriptionPlanCard from '../SubscriptionPlanCard.vue'
import type { SubscriptionPlan } from '@/types/payment'

const plan: SubscriptionPlan = {
  id: 3,
  group_id: 12,
  group_platform: 'codex',
  group_name: 'Codex 标准版',
  rate_multiplier: 1,
  daily_limit_usd: 55,
  weekly_limit_usd: 210,
  monthly_limit_usd: 420,
  supported_model_scopes: ['claude', 'gemini_text', 'gemini_image'],
  name: 'Codex 标准版',
  description: '推荐，比入门版月额度 +133%，单位额度省约 13%。',
  price: 99,
  original_price: 0,
  validity_days: 30,
  validity_unit: 'day',
  features: ['30 天有效期', '用量统计与调用日志'],
  for_sale: true,
  sort_order: 30,
}

describe('SubscriptionPlanCard', () => {
  it('shows package description, combined quota, and only GPT/Imagen model labels', () => {
    const wrapper = mount(SubscriptionPlanCard, {
      props: {
        plan,
      },
    })

    expect(wrapper.text()).toContain('推荐，比入门版月额度 +133%，单位额度省约 13%。')
    expect(wrapper.text()).toContain('推荐')
    expect(wrapper.text()).toContain('加量 +133%')
    expect(wrapper.text()).toContain('省约 13%')
    expect(wrapper.text()).toContain('日')
    expect(wrapper.text()).toContain('$55')
    expect(wrapper.text()).toContain('周')
    expect(wrapper.text()).toContain('$210')
    expect(wrapper.text()).toContain('月')
    expect(wrapper.text()).toContain('$420')
    expect(wrapper.text()).toContain('GPT 全系列')
    expect(wrapper.text()).toContain('Imagen')
    expect(wrapper.text()).not.toContain('Claude')
    expect(wrapper.text()).not.toContain('Gemini')
  })
})
