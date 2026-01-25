import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { getUploadUrl, getShareThumbSmallUrl, getShareThumbLargeUrl, clearThumbCache } from '../api'

describe('API utilities', () => {
  describe('getUploadUrl', () => {
    const originalEnv = import.meta.env.VITE_API_URL

    afterEach(() => {
      // Reset env
      if (originalEnv !== undefined) {
        import.meta.env.VITE_API_URL = originalEnv
      } else {
        delete import.meta.env.VITE_API_URL
      }
    })

    it('returns empty string when VITE_API_URL is not set', () => {
      delete import.meta.env.VITE_API_URL
      expect(getUploadUrl()).toBe('')
    })

    it('returns VITE_API_URL when set', () => {
      import.meta.env.VITE_API_URL = 'http://localhost:8080'
      expect(getUploadUrl()).toBe('http://localhost:8080')
    })
  })

  describe('getShareThumbSmallUrl', () => {
    it('constructs correct URL', () => {
      const url = getShareThumbSmallUrl('abc123', 42)
      expect(url).toContain('/api/share/abc123/photo/42/thumb/small')
    })

    it('handles special characters in token', () => {
      const url = getShareThumbSmallUrl('test-token_123', 1)
      expect(url).toContain('/api/share/test-token_123/photo/1/thumb/small')
    })
  })

  describe('getShareThumbLargeUrl', () => {
    it('constructs correct URL', () => {
      const url = getShareThumbLargeUrl('xyz789', 100)
      expect(url).toContain('/api/share/xyz789/photo/100/thumb/large')
    })
  })

  describe('clearThumbCache', () => {
    beforeEach(() => {
      // Mock URL.revokeObjectURL
      vi.stubGlobal('URL', {
        ...URL,
        revokeObjectURL: vi.fn(),
        createObjectURL: vi.fn(() => 'blob:test')
      })
    })

    afterEach(() => {
      vi.unstubAllGlobals()
    })

    it('clears cache without error', () => {
      expect(() => clearThumbCache()).not.toThrow()
    })
  })
})
