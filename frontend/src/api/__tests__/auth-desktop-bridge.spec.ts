import { describe, expect, it } from 'vitest'

import { getDesktopAuthorizationCallback } from '../auth'

describe('desktop oauth completion', () => {
  it('accepts the server-issued desktop callback URL', () => {
    expect(getDesktopAuthorizationCallback({
      desktop_callback_url: 'ximoai://desktop/callback?code=dca_code&state=state'
    })).toBe('ximoai://desktop/callback?code=dca_code&state=state')
  })

  it('does not allow arbitrary callback URLs', () => {
    expect(getDesktopAuthorizationCallback({
      desktop_callback_url: 'https://evil.example/callback?code=leak'
    })).toBeNull()
  })
})
