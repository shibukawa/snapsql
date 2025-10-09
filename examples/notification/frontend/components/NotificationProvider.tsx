'use client';

import React, { createContext, useContext, useState, useEffect, useCallback, useRef } from 'react';
import type { Notification, SSEEvent } from '@/types/notification';

interface NotificationContextValue {
  notifications: Notification[];
  unreadCount: number;
  isLoading: boolean;
  error: string | null;
  markAsRead: (notificationId: number) => Promise<void>;
  markNonImportantAsRead: () => Promise<void>;
  refetch: () => Promise<void>;
}

const NotificationContext = createContext<NotificationContextValue | undefined>(undefined);

export function useNotification() {
  const context = useContext(NotificationContext);
  if (!context) {
    throw new Error('useNotification must be used within NotificationProvider');
  }
  return context;
}

interface NotificationProviderProps {
  children: React.ReactNode;
}

export function NotificationProvider({ children }: NotificationProviderProps) {
  const [notifications, setNotifications] = useState<Notification[]>([]);
  const [unreadCount, setUnreadCount] = useState(0);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  
  const pollingIntervalRef = useRef<NodeJS.Timeout | null>(null);
  const lastUpdatedAtRef = useRef<string | null>(null);
  const isInitialLoadRef = useRef(true);

  const userId = process.env.NEXT_PUBLIC_USER_ID || 'EMP001';
  const backendUrl = process.env.BACKEND_API_URL || 'http://localhost:8080';
  const pollingInterval = 5000; // 5 seconds

  // Calculate unread count
  const calculateUnreadCount = useCallback((notificationList: Notification[]) => {
    if (!Array.isArray(notificationList)) {
      console.error('calculateUnreadCount: notificationList is not an array', notificationList);
      return 0;
    }
    return notificationList.filter(n => !n.read_at).length;
  }, []);

  // Merge and sort notifications by created_at (newest first)
  const mergeAndSortNotifications = useCallback((existing: Notification[], incoming: Notification[]) => {
    // Create a map of existing notifications by ID
    const notificationMap = new Map<number, Notification>();
    
    existing.forEach(n => {
      notificationMap.set(n.id, n);
    });
    
    // Add or update with incoming notifications
    incoming.forEach(n => {
      notificationMap.set(n.id, n);
    });
    
    // Convert back to array and sort by created_at (newest first)
    const merged = Array.from(notificationMap.values());
    merged.sort((a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime());
    
    return merged;
  }, []);

  // Fetch notifications with polling logic
  const fetchNotifications = useCallback(async () => {
    const wasInitialLoad = isInitialLoadRef.current;
    
    try {
      if (wasInitialLoad) {
        setIsLoading(true);
      }
      setError(null);
      
      // Build URL based on whether this is initial load or polling
      let url = `${backendUrl}/api/users/${userId}/notifications`;
      const params = new URLSearchParams();
      
      if (wasInitialLoad) {
        // Initial load: get latest 10 notifications (both read and unread)
        params.append('limit', '10');
        console.log('[POLLING] Initial load: fetching latest 10 notifications');
      } else if (lastUpdatedAtRef.current) {
        // Subsequent polls: get only new unread notifications since last update
        params.append('since', lastUpdatedAtRef.current);
        params.append('unread_only', 'true');
        console.log('[POLLING] Polling for updates since:', lastUpdatedAtRef.current);
      }
      
      if (params.toString()) {
        url += `?${params.toString()}`;
      }
      
      const response = await fetch(url);
      
      if (!response.ok) {
        throw new Error('Failed to fetch notifications');
      }
      
      const data = await response.json();
      const notificationArray = Array.isArray(data) ? data : [];
      console.log(`[POLLING] Fetched ${notificationArray.length} notification(s)`);
      
      if (wasInitialLoad) {
        // Initial load: replace all notifications
        setNotifications(notificationArray);
        setUnreadCount(calculateUnreadCount(notificationArray));
        isInitialLoadRef.current = false;
      } else if (notificationArray.length > 0) {
        // Polling: merge new notifications
        setNotifications(prev => {
          const merged = mergeAndSortNotifications(prev, notificationArray);
          setUnreadCount(calculateUnreadCount(merged));
          return merged;
        });
      }
      
      // Update lastUpdatedAt to the latest updated_at from fetched notifications
      if (notificationArray.length > 0) {
        const latestUpdatedAt = notificationArray.reduce((latest: string | null, n: Notification) => {
          const updatedAt = n.updated_at || n.created_at;
          if (!latest || new Date(updatedAt) > new Date(latest)) {
            return updatedAt;
          }
          return latest;
        }, lastUpdatedAtRef.current);
        
        console.log('[POLLING] Updated lastUpdatedAt from', lastUpdatedAtRef.current, 'to', latestUpdatedAt);
        lastUpdatedAtRef.current = latestUpdatedAt;
      }
    } catch (err) {
      console.error('Error fetching notifications:', err);
      setError('Failed to fetch notifications');
    } finally {
      if (wasInitialLoad) {
        setIsLoading(false);
      }
    }
  }, [backendUrl, userId, calculateUnreadCount, mergeAndSortNotifications]);

  // Mark a notification as read
  const markAsRead = useCallback(async (notificationId: number) => {
    try {
      const response = await fetch(
        `${backendUrl}/api/users/${userId}/notifications/${notificationId}/read`,
        {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
        }
      );

      if (!response.ok) {
        throw new Error('Failed to mark notification as read');
      }

      // Update local state
      setNotifications(prev => {
        const updated = prev.map(n =>
          n.id === notificationId ? { ...n, read_at: new Date().toISOString() } : n
        );
        setUnreadCount(calculateUnreadCount(updated));
        return updated;
      });
    } catch (err) {
      console.error('Error marking notification as read:', err);
      throw new Error('Failed to mark as read');
    }
  }, [backendUrl, userId, calculateUnreadCount]);

  // Mark non-important notifications as read
  const markNonImportantAsRead = useCallback(async () => {
    try {
      const response = await fetch(
        `${backendUrl}/api/users/${userId}/notifications/mark-non-important`,
        {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
        }
      );

      if (!response.ok) {
        throw new Error('Failed to mark non-important notifications as read');
      }

      // Update local state
      setNotifications(prev => {
        const updated = prev.map(n =>
          !n.important && !n.read_at ? { ...n, read_at: new Date().toISOString() } : n
        );
        setUnreadCount(calculateUnreadCount(updated));
        return updated;
      });
    } catch (err) {
      console.error('Error marking non-important notifications as read:', err);
      // Don't throw error for this operation as it's automatic
    }
  }, [backendUrl, userId, calculateUnreadCount]);

  // Setup polling
  const setupPolling = useCallback(() => {
    // Clear existing polling interval
    if (pollingIntervalRef.current) {
      clearInterval(pollingIntervalRef.current);
    }

    // Start polling at regular intervals
    pollingIntervalRef.current = setInterval(() => {
      fetchNotifications();
    }, pollingInterval);

    console.log('[POLLING] Polling started with interval:', pollingInterval, 'ms');
  }, [fetchNotifications, pollingInterval]);

  // Initialize: fetch notifications and setup polling
  useEffect(() => {
    fetchNotifications();
    setupPolling();

    // Cleanup on unmount
    return () => {
      if (pollingIntervalRef.current) {
        clearInterval(pollingIntervalRef.current);
      }
    };
  }, [fetchNotifications, setupPolling]);

  const value: NotificationContextValue = {
    notifications,
    unreadCount,
    isLoading,
    error,
    markAsRead,
    markNonImportantAsRead,
    refetch: fetchNotifications,
  };

  return (
    <NotificationContext.Provider value={value}>
      {children}
    </NotificationContext.Provider>
  );
}
