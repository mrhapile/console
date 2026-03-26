/**
 * Tests for useLastRoute utility functions.
 *
 * Tests the exported utility functions (getLastRoute, clearLastRoute,
 * getRememberPosition, setRememberPosition) which do not require
 * React Router context.
 *
 * The useLastRoute hook itself requires react-router-dom's useLocation
 * and useNavigate, plus DOM scroll containers, so we test the utilities
 * that can be validated in isolation.
 */

import { describe, it, expect, beforeEach } from 'vitest'
import { getLastRoute, clearLastRoute, getRememberPosition, setRememberPosition } from '../useLastRoute'

const LAST_ROUTE_KEY = 'kubestellar-last-route'
const SCROLL_POSITIONS_KEY = 'kubestellar-scroll-positions'
const REMEMBER_POSITION_KEY = 'kubestellar-remember-position'

describe('useLastRoute utilities', () => {
  beforeEach(() => {
    localStorage.clear()
  })

  describe('getLastRoute', () => {
    it('should return null when no route is stored', () => {
      expect(getLastRoute()).toBeNull()
    })

    it('should return stored route', () => {
      localStorage.setItem(LAST_ROUTE_KEY, '/clusters')
      expect(getLastRoute()).toBe('/clusters')
    })

    it('should return route with query params', () => {
      localStorage.setItem(LAST_ROUTE_KEY, '/workloads?mission=test')
      expect(getLastRoute()).toBe('/workloads?mission=test')
    })

    it('should return root path when stored', () => {
      localStorage.setItem(LAST_ROUTE_KEY, '/')
      expect(getLastRoute()).toBe('/')
    })

    it('should handle localStorage errors gracefully', () => {
      // Store something, then make getItem throw
      const originalGetItem = localStorage.getItem
      localStorage.getItem = () => { throw new Error('quota exceeded') }

      expect(getLastRoute()).toBeNull()

      localStorage.getItem = originalGetItem
    })
  })

  describe('clearLastRoute', () => {
    it('should clear last route from localStorage', () => {
      localStorage.setItem(LAST_ROUTE_KEY, '/clusters')
      clearLastRoute()
      expect(localStorage.getItem(LAST_ROUTE_KEY)).toBeNull()
    })

    it('should clear scroll positions from localStorage', () => {
      localStorage.setItem(SCROLL_POSITIONS_KEY, JSON.stringify({ '/clusters': 100 }))
      clearLastRoute()
      expect(localStorage.getItem(SCROLL_POSITIONS_KEY)).toBeNull()
    })

    it('should clear both route and scroll positions', () => {
      localStorage.setItem(LAST_ROUTE_KEY, '/events')
      localStorage.setItem(SCROLL_POSITIONS_KEY, JSON.stringify({ '/events': 200 }))

      clearLastRoute()

      expect(localStorage.getItem(LAST_ROUTE_KEY)).toBeNull()
      expect(localStorage.getItem(SCROLL_POSITIONS_KEY)).toBeNull()
    })

    it('should not throw when nothing is stored', () => {
      expect(() => clearLastRoute()).not.toThrow()
    })

    it('should handle localStorage errors gracefully', () => {
      const originalRemoveItem = localStorage.removeItem
      localStorage.removeItem = () => { throw new Error('storage error') }

      expect(() => clearLastRoute()).not.toThrow()

      localStorage.removeItem = originalRemoveItem
    })
  })

  describe('getRememberPosition', () => {
    it('should return false by default (off)', () => {
      expect(getRememberPosition('/clusters')).toBe(false)
    })

    it('should return true when preference is set to true', () => {
      localStorage.setItem(REMEMBER_POSITION_KEY, JSON.stringify({ '/clusters': true }))
      expect(getRememberPosition('/clusters')).toBe(true)
    })

    it('should return false when preference is set to false', () => {
      localStorage.setItem(REMEMBER_POSITION_KEY, JSON.stringify({ '/clusters': false }))
      expect(getRememberPosition('/clusters')).toBe(false)
    })

    it('should return false for unknown paths', () => {
      localStorage.setItem(REMEMBER_POSITION_KEY, JSON.stringify({ '/clusters': true }))
      expect(getRememberPosition('/events')).toBe(false)
    })

    it('should handle corrupted JSON gracefully', () => {
      localStorage.setItem(REMEMBER_POSITION_KEY, 'not-json')
      expect(getRememberPosition('/clusters')).toBe(false)
    })
  })

  describe('setRememberPosition', () => {
    it('should save preference for a path', () => {
      setRememberPosition('/clusters', true)
      expect(getRememberPosition('/clusters')).toBe(true)
    })

    it('should update existing preference', () => {
      setRememberPosition('/clusters', true)
      expect(getRememberPosition('/clusters')).toBe(true)

      setRememberPosition('/clusters', false)
      expect(getRememberPosition('/clusters')).toBe(false)
    })

    it('should store preferences for multiple paths independently', () => {
      setRememberPosition('/clusters', true)
      setRememberPosition('/events', false)
      setRememberPosition('/workloads', true)

      expect(getRememberPosition('/clusters')).toBe(true)
      expect(getRememberPosition('/events')).toBe(false)
      expect(getRememberPosition('/workloads')).toBe(true)
    })

    it('should handle localStorage errors gracefully', () => {
      const originalSetItem = localStorage.setItem
      localStorage.setItem = () => { throw new Error('quota exceeded') }

      expect(() => setRememberPosition('/clusters', true)).not.toThrow()

      localStorage.setItem = originalSetItem
    })

    it('should persist data in localStorage as JSON', () => {
      setRememberPosition('/clusters', true)
      const raw = localStorage.getItem(REMEMBER_POSITION_KEY)
      expect(raw).not.toBeNull()
      const parsed = JSON.parse(raw!)
      expect(parsed).toEqual({ '/clusters': true })
    })
  })
})
