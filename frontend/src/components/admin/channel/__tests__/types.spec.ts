import { describe, expect, it } from 'vitest'
import { pricingFormEntryToAPI, validateIntervals, type IntervalFormEntry, type PricingFormEntry } from '../types'

function makeInterval(over: Partial<IntervalFormEntry>): IntervalFormEntry {
  return {
    min_tokens: 0,
    max_tokens: null,
    tier_label: '',
    input_price: null,
    output_price: null,
    cache_write_price: null,
    cache_read_price: null,
    per_request_price: null,
    sort_order: 0,
    ...over,
  }
}

function makePricingEntry(over: Partial<PricingFormEntry>): PricingFormEntry {
  return {
    models: ['model-a'],
    billing_mode: 'token',
    input_price: null,
    output_price: null,
    cache_write_price: null,
    cache_read_price: null,
    image_output_price: null,
    per_request_price: null,
    intervals: [],
    ...over,
  }
}

function t(key: string, params?: Record<string, unknown>): string {
  return `${key}${params ? ` ${JSON.stringify(params)}` : ''}`
}

describe('validateIntervals', () => {
  describe('token mode', () => {
    it('rejects unbounded interval that is not last', () => {
      const intervals: IntervalFormEntry[] = [
        makeInterval({ min_tokens: 0, max_tokens: null, input_price: 1, output_price: 1 }),
        makeInterval({ min_tokens: 200000, max_tokens: 500000, input_price: 2, output_price: 2 }),
      ]
      expect(validateIntervals(intervals, 'token', t)).toContain('unboundedLast')
    })

    it('accepts unbounded interval at the end', () => {
      const intervals: IntervalFormEntry[] = [
        makeInterval({ min_tokens: 0, max_tokens: 200000, input_price: 1, output_price: 1 }),
        makeInterval({ min_tokens: 200000, max_tokens: null, input_price: 2, output_price: 2 }),
      ]
      expect(validateIntervals(intervals, 'token', t)).toBeNull()
    })

    it('rejects overlapping intervals', () => {
      const intervals: IntervalFormEntry[] = [
        makeInterval({ min_tokens: 0, max_tokens: 250000, input_price: 1, output_price: 1 }),
        makeInterval({ min_tokens: 200000, max_tokens: 500000, input_price: 2, output_price: 2 }),
      ]
      expect(validateIntervals(intervals, 'token', t)).toContain('overlap')
    })

    it('rejects unbounded interval in token mode', () => {
      const intervals: IntervalFormEntry[] = [
        makeInterval({ min_tokens: 0, max_tokens: null, input_price: 1, output_price: 1 }),
        makeInterval({ min_tokens: 100, max_tokens: 200, input_price: 2, output_price: 2 }),
      ]
      expect(validateIntervals(intervals, 'token', t)).toContain('unboundedLast')
    })
  })

  describe('image / per_request mode', () => {
    it('allows multiple unbounded tiers identified by label', () => {
      const intervals: IntervalFormEntry[] = [
        makeInterval({ tier_label: '1K', per_request_price: 0.04 }),
        makeInterval({ tier_label: '2K', per_request_price: 0.06 }),
        makeInterval({ tier_label: '4K', per_request_price: 0.08 }),
      ]
      expect(validateIntervals(intervals, 'image', t)).toBeNull()
      expect(validateIntervals(intervals, 'per_request', t)).toBeNull()
    })

    it('still rejects negative prices', () => {
      const intervals: IntervalFormEntry[] = [
        makeInterval({ tier_label: '1K', per_request_price: -1 }),
      ]
      expect(validateIntervals(intervals, 'image', t)).toContain('negativePrice')
    })

    it('still rejects max <= min on a single tier', () => {
      const intervals: IntervalFormEntry[] = [
        makeInterval({ tier_label: '1K', min_tokens: 100, max_tokens: 50, per_request_price: 0.04 }),
      ]
      expect(validateIntervals(intervals, 'image', t)).toContain('maxGreaterThanMin')
    })
  })
})

describe('pricingFormEntryToAPI', () => {
  it('clears legacy token image price when saving image mode pricing', () => {
    const payload = pricingFormEntryToAPI('gemini', makePricingEntry({
      billing_mode: 'image',
      image_output_price: 60,
      per_request_price: 0.4,
      input_price: 1,
      output_price: 2,
      intervals: [
        makeInterval({ tier_label: '1K', input_price: 1, per_request_price: 0.2 }),
      ],
    }))

    expect(payload.image_output_price).toBeNull()
    expect(payload.input_price).toBeNull()
    expect(payload.output_price).toBeNull()
    expect(payload.per_request_price).toBe(0.4)
    expect(payload.intervals[0].input_price).toBeNull()
    expect(payload.intervals[0].per_request_price).toBe(0.2)
  })

  it('clears per-request prices when saving token mode pricing', () => {
    const payload = pricingFormEntryToAPI('gemini', makePricingEntry({
      billing_mode: 'token',
      input_price: 1,
      image_output_price: 60,
      per_request_price: 0.4,
      intervals: [
        makeInterval({ input_price: 1, per_request_price: 0.2 }),
      ],
    }))

    expect(payload.input_price).toBe(0.000001)
    expect(payload.image_output_price).toBe(0.00006)
    expect(payload.per_request_price).toBeNull()
    expect(payload.intervals[0].input_price).toBe(0.000001)
    expect(payload.intervals[0].per_request_price).toBeNull()
  })
})
