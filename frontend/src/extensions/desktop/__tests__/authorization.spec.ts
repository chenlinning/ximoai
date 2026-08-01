import { describe, expect, it, vi } from 'vitest'

import {
  buildDesktopCallbackURL,
  completeDesktopAuthorization,
  parseDesktopAuthorizationQuery,
} from '../authorization'

const publicJWK = { kty: 'EC', crv: 'P-256', x: 'public-x', y: 'public-y', alg: 'ES256' }

function encodeBase64URL(value: string): string {
  const bytes = new TextEncoder().encode(value)
  let binary = ''
  for (const byte of bytes) binary += String.fromCharCode(byte)
  return btoa(binary).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/g, '')
}

describe('desktop browser authorization', () => {
  it('parses only PKCE, public device key, callback and state', () => {
    const result = parseDesktopAuthorizationQuery({
      code_challenge: 'a'.repeat(43),
      code_challenge_method: 'S256',
      device_jwk: encodeBase64URL(JSON.stringify(publicJWK)),
      redirect_uri: 'ximoai://desktop/callback',
      state: 'desktop-state',
    })

    expect(result.request).toEqual({
      code_challenge: 'a'.repeat(43),
      code_challenge_method: 'S256',
      device_jwk: publicJWK,
      redirect_uri: 'ximoai://desktop/callback',
    })
    expect(result.state).toBe('desktop-state')
  })

  it('rejects private JWK material before calling the server', () => {
    expect(() => parseDesktopAuthorizationQuery({
      code_challenge: 'a'.repeat(43),
      code_challenge_method: 'S256',
      device_jwk: encodeBase64URL(JSON.stringify({ ...publicJWK, d: 'private' })),
      redirect_uri: 'ximoai://desktop/callback',
    })).toThrow('invalid desktop authorization request')
  })

  it('returns only the one-time code and original state to the callback', async () => {
    const issue = vi.fn().mockResolvedValue({ code: 'dca_one_time', expires_in: 300 })
    const navigate = vi.fn()

    await completeDesktopAuthorization({
      code_challenge: 'a'.repeat(43),
      code_challenge_method: 'S256',
      device_jwk: encodeBase64URL(JSON.stringify(publicJWK)),
      redirect_uri: 'http://127.0.0.1:49152/callback',
      state: 'opaque-state',
    }, issue, navigate)

    expect(issue).toHaveBeenCalledWith(expect.objectContaining({ redirect_uri: 'http://127.0.0.1:49152/callback' }))
    expect(navigate).toHaveBeenCalledWith('http://127.0.0.1:49152/callback?code=dca_one_time&state=opaque-state')
    expect(navigate.mock.calls[0][0]).not.toContain('access_token')
    expect(navigate.mock.calls[0][0]).not.toContain('refresh_token')
  })

  it('rejects callbacks outside the desktop custom scheme and loopback listener', () => {
    expect(() => parseDesktopAuthorizationQuery({
      code_challenge: 'a'.repeat(43),
      code_challenge_method: 'S256',
      device_jwk: encodeBase64URL(JSON.stringify(publicJWK)),
      redirect_uri: 'https://evil.example/callback',
    })).toThrow('invalid desktop authorization request')
    expect(() => buildDesktopCallbackURL('ximoai://desktop/callback?source=browser', 'dca_code', 'state')).toThrow()
  })
})
