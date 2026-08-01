import apiClient from '@/api/client'
import type { ApiResponse } from '@/types'

export interface DesktopPublicJWK {
  kty: 'EC'
  crv: 'P-256'
  x: string
  y: string
  alg?: 'ES256'
}

export interface DesktopAuthorizationRequest {
  code_challenge: string
  code_challenge_method: 'S256'
  device_jwk: DesktopPublicJWK
  redirect_uri: string
}

export interface DesktopAuthorizationGrant {
  code: string
  expires_in: number
}

export async function issueDesktopAuthorization(request: DesktopAuthorizationRequest): Promise<DesktopAuthorizationGrant> {
  const response = await apiClient.post<ApiResponse<DesktopAuthorizationGrant>>('/desktop/authorize', request)
  return response.data.data
}
