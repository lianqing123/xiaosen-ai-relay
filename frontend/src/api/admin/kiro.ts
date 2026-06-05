import { apiClient } from '../client'

export interface KiroCredentialEntry {
  type: string
  path?: string
  enabled?: boolean
  profile_arn?: string
  region?: string
  api_region?: string
  comment?: string
  has_refresh_token?: boolean
  refresh_token_sha256?: string
  refresh_token_present?: boolean
}

export interface KiroOfficialQuotaAccount {
  credential_sha256?: string
  user_id?: string
  available: boolean
  error?: string
  subscription_title?: string
  subscription_type?: string
  overage_status?: string
  next_reset_at?: string
  display_name?: string
  currency?: string
  current_usage: number
  usage_limit: number
  remaining: number
  overage_charges: number
  overage_rate: number
  deduplicated?: boolean
}

export interface KiroOfficialQuota {
  available: boolean
  source: string
  fetched_at: string
  error?: string
  account_count: number
  unique_user_count: number
  subscription_title?: string
  subscription_type?: string
  overage_status?: string
  next_reset_at?: string
  display_name?: string
  currency?: string
  current_usage: number
  usage_limit: number
  remaining: number
  overage_charges: number
  overage_rate: number
  accounts?: KiroOfficialQuotaAccount[]
}

export interface KiroGatewayStatus {
  kiro_home: string
  credentials_file_exists: boolean
  credential_count: number
  credentials: KiroCredentialEntry[]
  official_quota?: KiroOfficialQuota
  activation_script_exists: boolean
  cli_available: boolean
  cli_path?: string
  gateway_healthy: boolean
  gateway_health_message?: string
}

export interface KiroActivationResult {
  message: string
  output?: string
}

export interface KiroCredentialImportResult {
  message: string
  credential_count: number
  credentials: KiroCredentialEntry[]
  two_factor_accepted: boolean
  activation?: KiroActivationResult
}

export interface KiroCredentialImportPayload {
  type: 'refresh_token' | 'json' | 'sqlite'
  refreshToken?: string
  file?: File | null
  profileArn?: string
  region?: string
  apiRegion?: string
  append?: boolean
  activate?: boolean
  twoFactorCode?: string
}

export interface KiroCLILoginInput {
  provider: 'builder' | 'google' | 'github' | 'identity_center'
  identityProviderUrl?: string
  region?: string
  append?: boolean
  activate?: boolean
}

export interface KiroCLILoginSession {
  id: string
  status: 'starting' | 'waiting' | 'importing' | 'imported' | 'activated' | 'failed'
  provider: string
  verification_url?: string
  user_code?: string
  message?: string
  output?: string
  error?: string
  started_at: string
  updated_at: string
  credential_count?: number
}

export async function getStatus(): Promise<KiroGatewayStatus> {
  const { data } = await apiClient.get<KiroGatewayStatus>('/admin/kiro/status')
  return data
}

export async function importCredentials(payload: KiroCredentialImportPayload): Promise<KiroCredentialImportResult> {
  const form = new FormData()
  form.append('type', payload.type)
  if (payload.refreshToken) form.append('refresh_token', payload.refreshToken)
  if (payload.file) form.append('credential_file', payload.file)
  if (payload.profileArn) form.append('profile_arn', payload.profileArn)
  if (payload.region) form.append('region', payload.region)
  if (payload.apiRegion) form.append('api_region', payload.apiRegion)
  if (payload.append) form.append('append', 'true')
  if (payload.activate) form.append('activate', 'true')
  if (payload.twoFactorCode) form.append('two_factor_code', payload.twoFactorCode)

  const { data } = await apiClient.post<KiroCredentialImportResult>('/admin/kiro/credentials', form, {
    headers: { 'Content-Type': 'multipart/form-data' },
    timeout: 180000
  })
  return data
}

export async function activate(): Promise<KiroActivationResult> {
  const { data } = await apiClient.post<KiroActivationResult>('/admin/kiro/activate', undefined, {
    timeout: 180000
  })
  return data
}

export async function startCLILogin(payload: KiroCLILoginInput): Promise<KiroCLILoginSession> {
  const { data } = await apiClient.post<KiroCLILoginSession>('/admin/kiro/login/start', {
    provider: payload.provider,
    identity_provider_url: payload.identityProviderUrl,
    region: payload.region,
    append: payload.append,
    activate: payload.activate
  }, { timeout: 180000 })
  return data
}

export async function getCLILoginSession(id: string): Promise<KiroCLILoginSession> {
  const { data } = await apiClient.get<KiroCLILoginSession>(`/admin/kiro/login/${id}`)
  return data
}

export const kiroAPI = {
  getStatus,
  importCredentials,
  activate,
  startCLILogin,
  getCLILoginSession
}

export default kiroAPI
