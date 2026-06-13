import { describe, expect, it } from 'vitest'
import { displayUnitPrice } from '../modelPricingDisplay'

describe('displayUnitPrice', () => {
  it('prefers per-request image price over legacy image token price', () => {
    expect(displayUnitPrice({
      billing_mode: 'image',
      image_output_price: 0.00006,
      per_request_price: 0.4,
    })).toBe(0.4)
  })

  it('keeps legacy image price as a fallback when no per-request price exists', () => {
    expect(displayUnitPrice({
      billing_mode: 'image',
      image_output_price: 0.00006,
      per_request_price: null,
    })).toBe(0.00006)
  })
})
