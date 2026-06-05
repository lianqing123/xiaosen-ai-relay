const INVITATION_QUERY_KEYS = [
  'invitation_code',
  'invite_code',
  'invite',
  'invitation',
  'code',
  'aff',
  'aff_code'
] as const

type QueryValue = string | null | Array<string | null> | undefined

function firstQueryValue(value: QueryValue): string {
  if (Array.isArray(value)) {
    return firstQueryValue(value[0])
  }
  return typeof value === 'string' ? value.trim() : ''
}

export function resolveInvitationCodeFromQuery(query: Record<string, QueryValue>): string {
  for (const key of INVITATION_QUERY_KEYS) {
    const code = firstQueryValue(query[key])
    if (code) return code
  }
  return ''
}

export function buildInvitationRegisterLink(origin: string, code: string): string {
  const normalizedCode = code.trim()
  const base = origin.trim().replace(/\/+$/, '')
  if (!base) {
    return `/register?invitation_code=${encodeURIComponent(normalizedCode)}`
  }
  const url = new URL('/register', base)
  url.searchParams.set('invitation_code', normalizedCode)
  return url.toString()
}

export function buildAffiliateRegisterLink(origin: string, code: string): string {
  const normalizedCode = code.trim()
  const base = origin.trim().replace(/\/+$/, '')
  if (!base) {
    return `/register?invitation_code=${encodeURIComponent(normalizedCode)}&aff=${encodeURIComponent(normalizedCode)}`
  }
  const url = new URL('/register', base)
  url.searchParams.set('invitation_code', normalizedCode)
  url.searchParams.set('aff', normalizedCode)
  return url.toString()
}
