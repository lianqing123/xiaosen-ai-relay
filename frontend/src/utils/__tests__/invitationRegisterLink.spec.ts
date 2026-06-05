import { describe, expect, it } from 'vitest'

import {
  buildAffiliateRegisterLink,
  buildInvitationRegisterLink,
  resolveInvitationCodeFromQuery
} from '../invitationRegisterLink'

describe('invitation register links', () => {
  it('uses invitation_code as the canonical register link parameter', () => {
    expect(buildInvitationRegisterLink('https://api.zhongzhuan.pro', ' INVITE-123 ')).toBe(
      'https://api.zhongzhuan.pro/register?invitation_code=INVITE-123'
    )
  })

  it('builds affiliate register links that satisfy invitation registration and rebate binding', () => {
    expect(buildAffiliateRegisterLink('https://api.zhongzhuan.pro', ' AFF123 ')).toBe(
      'https://api.zhongzhuan.pro/register?invitation_code=AFF123&aff=AFF123'
    )
    expect(buildAffiliateRegisterLink('', ' AFF123 ')).toBe(
      '/register?invitation_code=AFF123&aff=AFF123'
    )
  })

  it('resolves invitation codes from existing shared link parameter aliases', () => {
    expect(resolveInvitationCodeFromQuery({ invitation_code: ' INVITE-A ' })).toBe('INVITE-A')
    expect(resolveInvitationCodeFromQuery({ invite_code: 'INVITE-B' })).toBe('INVITE-B')
    expect(resolveInvitationCodeFromQuery({ invite: ['INVITE-C'] })).toBe('INVITE-C')
    expect(resolveInvitationCodeFromQuery({ invitation: 'INVITE-D' })).toBe('INVITE-D')
    expect(resolveInvitationCodeFromQuery({ code: 'INVITE-E' })).toBe('INVITE-E')
    expect(resolveInvitationCodeFromQuery({ aff: ' AFF-OLD ' })).toBe('AFF-OLD')
    expect(resolveInvitationCodeFromQuery({ aff_code: 'AFF-CODE' })).toBe('AFF-CODE')
  })

  it('does not treat empty query values as invitation codes', () => {
    expect(resolveInvitationCodeFromQuery({ invitation_code: ' ', invite: null })).toBe('')
  })
})
