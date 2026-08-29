import { describe, expect, it } from 'vitest'

import { durationSeverity, firstTokenSeverity, outputTokenRate } from '../latencyHealth'

describe('latencyHealth', () => {
  it('classifies first-token latency at 10s/30s/60s boundaries', () => {
    expect(firstTokenSeverity(0)).toBe('good')
    expect(firstTokenSeverity(9_999)).toBe('good')
    expect(firstTokenSeverity(10_000)).toBe('warn')
    expect(firstTokenSeverity(29_999)).toBe('warn')
    expect(firstTokenSeverity(30_000)).toBe('slow')
    expect(firstTokenSeverity(59_999)).toBe('slow')
    expect(firstTokenSeverity(60_000)).toBe('critical')
  })

  it('classifies total duration at 1min/3min/5min boundaries', () => {
    expect(durationSeverity(0)).toBe('good')
    expect(durationSeverity(59_999)).toBe('good')
    expect(durationSeverity(60_000)).toBe('warn')
    expect(durationSeverity(179_999)).toBe('warn')
    expect(durationSeverity(180_000)).toBe('slow')
    expect(durationSeverity(299_999)).toBe('slow')
    expect(durationSeverity(300_000)).toBe('critical')
  })

  it('calculates rounded output token rate after the first token', () => {
    expect(outputTokenRate(257, 3_480, 10_430)).toBe(37)
    expect(outputTokenRate(10, 3_100, 3_680)).toBe(17)
  })

  it('calculates synchronous output token rate over the total duration', () => {
    expect(outputTokenRate(10, null, 2_000)).toBe(5)
  })

  it('does not report a rate without a valid duration or output', () => {
    expect(outputTokenRate(10, 3_680, null)).toBeNull()
    expect(outputTokenRate(10, 3_680, 3_680)).toBeNull()
    expect(outputTokenRate(10, null, 0)).toBeNull()
    expect(outputTokenRate(0, 3_100, 3_680)).toBeNull()
  })
})
