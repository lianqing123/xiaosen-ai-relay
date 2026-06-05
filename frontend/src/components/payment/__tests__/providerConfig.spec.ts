import { describe, expect, it } from 'vitest'
import {
  METHOD_ORDER,
  PROVIDER_CONFIG_FIELDS,
  PROVIDER_SUPPORTED_TYPES,
} from '@/components/payment/providerConfig'

function findField(key: string) {
  const fields = PROVIDER_CONFIG_FIELDS.wxpay || []
  return fields.find(field => field.key === key)
}

describe('PROVIDER_CONFIG_FIELDS.wxpay', () => {
  it('keeps admin form validation aligned with backend-required credentials', () => {
    expect(findField('publicKeyId')?.optional).toBeFalsy()
    expect(findField('certSerial')?.optional).toBeFalsy()
  })

  it('only keeps the simplified visible credential set in the admin form', () => {
    expect(findField('mpAppId')).toBeUndefined()
    expect(findField('h5AppName')).toBeUndefined()
    expect(findField('h5AppUrl')).toBeUndefined()
  })
})

describe('payment provider method mapping', () => {
  it('keeps USDC off EasyPay provider instances', () => {
    expect(PROVIDER_SUPPORTED_TYPES.easypay).not.toContain('usdc')
  })

  it('offers USDC only through the standalone USDC provider', () => {
    expect(PROVIDER_SUPPORTED_TYPES.usdc).toEqual(['usdc'])
  })

  it('places USDC in the user-facing method ordering', () => {
    expect(METHOD_ORDER).toContain('usdc')
  })
})
