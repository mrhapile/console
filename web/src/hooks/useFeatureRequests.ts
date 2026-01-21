import { useState, useEffect, useCallback, useRef } from 'react'
import { api } from '../lib/api'

// Types
export type RequestType = 'bug' | 'feature'
export type RequestStatus = 'submitted' | 'open' | 'in_progress' | 'pr_ready' | 'preview_available' | 'closed'
export type FeedbackType = 'positive' | 'negative'
export type NotificationType = 'issue_created' | 'pr_created' | 'preview_ready' | 'pr_merged' | 'pr_closed' | 'feedback_received'

export interface FeatureRequest {
  id: string
  user_id: string
  title: string
  description: string
  request_type: RequestType
  github_issue_number?: number
  github_issue_url?: string
  status: RequestStatus
  pr_number?: number
  pr_url?: string
  netlify_preview_url?: string
  created_at: string
  updated_at?: string
}

export interface PRFeedback {
  id: string
  feature_request_id: string
  user_id: string
  feedback_type: FeedbackType
  comment?: string
  created_at: string
}

export interface Notification {
  id: string
  user_id: string
  feature_request_id?: string
  notification_type: NotificationType
  title: string
  message: string
  read: boolean
  created_at: string
}

export interface CreateFeatureRequestInput {
  title: string
  description: string
  request_type: RequestType
}

export interface SubmitFeedbackInput {
  feedback_type: FeedbackType
  comment?: string
}

// Status display helpers
export const STATUS_LABELS: Record<RequestStatus, string> = {
  submitted: 'Submitted',
  open: 'Open',
  in_progress: 'In Progress',
  pr_ready: 'PR Ready',
  preview_available: 'Preview Available',
  closed: 'Closed',
}

export const STATUS_COLORS: Record<RequestStatus, string> = {
  submitted: 'bg-gray-500',
  open: 'bg-blue-500',
  in_progress: 'bg-yellow-500',
  pr_ready: 'bg-purple-500',
  preview_available: 'bg-green-500',
  closed: 'bg-gray-400',
}

// Feature Requests Hook
export function useFeatureRequests() {
  const [requests, setRequests] = useState<FeatureRequest[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [isSubmitting, setIsSubmitting] = useState(false)

  const loadRequests = useCallback(async () => {
    try {
      setIsLoading(true)
      const { data } = await api.get<FeatureRequest[]>('/api/feedback/requests')
      setRequests(data || [])
      setError(null)
    } catch (err) {
      console.error('Failed to load feature requests:', err)
      setError('Failed to load requests')
    } finally {
      setIsLoading(false)
    }
  }, [])

  useEffect(() => {
    loadRequests()
  }, [loadRequests])

  // Polling for status updates (every 30 seconds)
  useEffect(() => {
    const interval = setInterval(() => {
      // Only poll if there are pending requests
      const hasPending = requests.some(r =>
        r.status !== 'closed' && r.status !== 'preview_available'
      )
      if (hasPending) {
        loadRequests()
      }
    }, 30000)

    return () => clearInterval(interval)
  }, [requests, loadRequests])

  const createRequest = useCallback(async (input: CreateFeatureRequestInput) => {
    try {
      setIsSubmitting(true)
      const { data } = await api.post<FeatureRequest>('/api/feedback/requests', input)
      setRequests(prev => [data, ...prev])
      return data
    } catch (err) {
      console.error('Failed to create feature request:', err)
      throw err
    } finally {
      setIsSubmitting(false)
    }
  }, [])

  const getRequest = useCallback(async (id: string) => {
    try {
      const { data } = await api.get<FeatureRequest>(`/api/feedback/requests/${id}`)
      return data
    } catch (err) {
      console.error('Failed to get feature request:', err)
      throw err
    }
  }, [])

  const submitFeedback = useCallback(async (requestId: string, input: SubmitFeedbackInput) => {
    try {
      const { data } = await api.post<PRFeedback>(`/api/feedback/requests/${requestId}/feedback`, input)
      return data
    } catch (err) {
      console.error('Failed to submit feedback:', err)
      throw err
    }
  }, [])

  return {
    requests,
    isLoading,
    error,
    isSubmitting,
    loadRequests,
    createRequest,
    getRequest,
    submitFeedback,
  }
}

// Notifications Hook
export function useNotifications() {
  const [notifications, setNotifications] = useState<Notification[]>([])
  const [unreadCount, setUnreadCount] = useState(0)
  const [isLoading, setIsLoading] = useState(true)
  const pollingRef = useRef<number | null>(null)

  const loadNotifications = useCallback(async () => {
    try {
      const { data } = await api.get<Notification[]>('/api/notifications')
      setNotifications(data || [])
    } catch (err) {
      console.error('Failed to load notifications:', err)
    }
  }, [])

  const loadUnreadCount = useCallback(async () => {
    try {
      const { data } = await api.get<{ count: number }>('/api/notifications/unread-count')
      setUnreadCount(data.count)
    } catch (err) {
      console.error('Failed to load unread count:', err)
    }
  }, [])

  const loadAll = useCallback(async () => {
    setIsLoading(true)
    await Promise.all([loadNotifications(), loadUnreadCount()])
    setIsLoading(false)
  }, [loadNotifications, loadUnreadCount])

  useEffect(() => {
    loadAll()
  }, [loadAll])

  // Poll for new notifications every 30 seconds
  useEffect(() => {
    pollingRef.current = window.setInterval(() => {
      loadUnreadCount()
    }, 30000)

    return () => {
      if (pollingRef.current) {
        clearInterval(pollingRef.current)
      }
    }
  }, [loadUnreadCount])

  const markAsRead = useCallback(async (id: string) => {
    try {
      await api.post(`/api/notifications/${id}/read`)
      setNotifications(prev =>
        prev.map(n => (n.id === id ? { ...n, read: true } : n))
      )
      setUnreadCount(prev => Math.max(0, prev - 1))
    } catch (err) {
      console.error('Failed to mark notification as read:', err)
      throw err
    }
  }, [])

  const markAllAsRead = useCallback(async () => {
    try {
      await api.post('/api/notifications/read-all')
      setNotifications(prev => prev.map(n => ({ ...n, read: true })))
      setUnreadCount(0)
    } catch (err) {
      console.error('Failed to mark all notifications as read:', err)
      throw err
    }
  }, [])

  return {
    notifications,
    unreadCount,
    isLoading,
    loadNotifications,
    loadUnreadCount,
    markAsRead,
    markAllAsRead,
    refresh: loadAll,
  }
}

// Combined hook for convenience
export function useFeedback() {
  const featureRequests = useFeatureRequests()
  const notifications = useNotifications()

  return {
    ...featureRequests,
    notifications: notifications.notifications,
    unreadCount: notifications.unreadCount,
    notificationsLoading: notifications.isLoading,
    markNotificationAsRead: notifications.markAsRead,
    markAllNotificationsAsRead: notifications.markAllAsRead,
    refreshNotifications: notifications.refresh,
  }
}
