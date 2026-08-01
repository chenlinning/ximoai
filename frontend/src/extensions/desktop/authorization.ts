import type { DesktopAuthorizationGrant, DesktopAuthorizationRequest, DesktopPublicJWK } from './api'

type QueryValue = string | null | undefined | Array<string | null>
type DesktopAuthorizationQuery = Record<string, QueryValue>
type IssueDesktopAuthorization = (request: DesktopAuthorizationRequest) => Promise<DesktopAuthorizationGrant>
type NavigateToCallback = (url: string) => void

export interface ParsedDesktopAuthorization {
  request: DesktopAuthorizationRequest
  state: string
}

export function parseDesktopAuthorizationQuery(query: DesktopAuthorizationQuery): ParsedDesktopAuthorization {
  const codeChallenge = singleQueryValue(query.code_challenge)
  const method = singleQueryValue(query.code_challenge_method)
  const encodedJWK = singleQueryValue(query.device_jwk)
  const redirectURI = singleQueryValue(query.redirect_uri)
  const state = singleQueryValue(query.state, true)

  if (!codeChallenge || codeChallenge.length < 43 || codeChallenge.length > 128 || method !== 'S256') {
    throw invalidDesktopAuthorizationRequest()
  }
  if (!encodedJWK || encodedJWK.length > 2048 || !redirectURI || redirectURI.length > 512 || state.length > 2048) {
    throw invalidDesktopAuthorizationRequest()
  }
  validateDesktopRedirectURI(redirectURI)

  const deviceJWK = decodePublicJWK(encodedJWK)
  return {
    request: {
      code_challenge: codeChallenge,
      code_challenge_method: 'S256',
      device_jwk: deviceJWK,
      redirect_uri: redirectURI,
    },
    state,
  }
}

export function buildDesktopCallbackURL(redirectURI: string, code: string, state: string): string {
  validateDesktopRedirectURI(redirectURI)
  const callback = new URL(redirectURI)
  callback.searchParams.set('code', code)
  if (state) callback.searchParams.set('state', state)
  return callback.toString()
}

function validateDesktopRedirectURI(raw: string): void {
  try {
    const callback = new URL(raw)
    if (callback.search || callback.hash || callback.username || callback.password) {
      throw invalidDesktopAuthorizationRequest()
    }
    if (callback.protocol === 'ximoai:' && callback.hostname === 'desktop' && callback.pathname === '/callback') {
      return
    }
    const loopbackHost = callback.hostname === '127.0.0.1' || callback.hostname === '[::1]'
    if (callback.protocol === 'http:' && loopbackHost && callback.port && callback.pathname === '/callback') {
      return
    }
  } catch {
    throw invalidDesktopAuthorizationRequest()
  }
  throw invalidDesktopAuthorizationRequest()
}

export async function completeDesktopAuthorization(
  query: DesktopAuthorizationQuery,
  issue: IssueDesktopAuthorization,
  navigate: NavigateToCallback,
): Promise<void> {
  const parsed = parseDesktopAuthorizationQuery(query)
  const grant = await issue(parsed.request)
  if (!grant?.code) throw invalidDesktopAuthorizationRequest()
  navigate(buildDesktopCallbackURL(parsed.request.redirect_uri, grant.code, parsed.state))
}

function decodePublicJWK(encoded: string): DesktopPublicJWK {
  try {
    const normalized = encoded.replace(/-/g, '+').replace(/_/g, '/')
    const padded = normalized.padEnd(Math.ceil(normalized.length / 4) * 4, '=')
    const binary = atob(padded)
    const bytes = Uint8Array.from(binary, (character) => character.charCodeAt(0))
    const value = JSON.parse(new TextDecoder('utf-8', { fatal: true }).decode(bytes)) as Record<string, unknown>
    const allowedKeys = new Set(['kty', 'crv', 'x', 'y', 'alg'])
    if (
      Object.keys(value).some((key) => !allowedKeys.has(key)) ||
      value.kty !== 'EC' || value.crv !== 'P-256' ||
      typeof value.x !== 'string' || !value.x || typeof value.y !== 'string' || !value.y ||
      (value.alg !== undefined && value.alg !== 'ES256')
    ) {
      throw invalidDesktopAuthorizationRequest()
    }
    return value as unknown as DesktopPublicJWK
  } catch {
    throw invalidDesktopAuthorizationRequest()
  }
}

function singleQueryValue(value: QueryValue, optional = false): string {
  if (Array.isArray(value)) throw invalidDesktopAuthorizationRequest()
  if (typeof value === 'string') return value
  if (optional && (value === undefined || value === null)) return ''
  throw invalidDesktopAuthorizationRequest()
}

function invalidDesktopAuthorizationRequest(): Error {
  return new Error('invalid desktop authorization request')
}
