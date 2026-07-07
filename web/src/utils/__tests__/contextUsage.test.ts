import { describe, it, expect } from 'vitest'
import { colorState, abbrevK, formatContextLabel } from '@/utils/contextUsage'

describe('contextUsage', () => {
  describe('colorState', () => {
    it('returns normal for ratios below warning threshold', () => {
      expect(colorState(0.0)).toBe('normal')
      expect(colorState(0.5)).toBe('normal')
    })

    it('returns warning for ratios between warning and critical thresholds', () => {
      expect(colorState(0.75)).toBe('warning')
      expect(colorState(0.89)).toBe('warning')
    })

    it('returns critical for ratios at or above auto-compression threshold', () => {
      expect(colorState(0.9)).toBe('critical')
      expect(colorState(1.0)).toBe('critical')
    })
  })

  describe('abbrevK', () => {
    it('formats thousands with one decimal below 100k', () => {
      expect(abbrevK(19930)).toBe('19.9k')
    })

    it('formats round thousands without decimal at or above 100k', () => {
      expect(abbrevK(199300)).toBe('199k')
      expect(abbrevK(1000000)).toBe('1000k')
    })

    it('is monotonic non-decreasing across a sample sequence', () => {
      const values = [0, 500, 999, 1000, 9999, 10000, 99999, 100000, 199300, 1000000]
      const formatted = values.map(abbrevK)
      const numeric = formatted.map((s) => Number.parseFloat(s.replace('k', '')))
      for (let i = 1; i < numeric.length; i++) {
        expect(numeric[i]).toBeGreaterThanOrEqual(numeric[i - 1])
      }
    })
  })

  describe('formatContextLabel', () => {
    it('has "{used} / {max} 已用上下文" shape', () => {
      const label = formatContextLabel(19930, 1000000)
      expect(label).toBe('19.9k / 1000k 已用上下文')
      expect(label).toMatch(/^\S+ \/ \S+ 已用上下文$/)
    })
  })
})
