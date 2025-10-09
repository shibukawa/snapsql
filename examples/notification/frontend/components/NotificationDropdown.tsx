'use client';

import React, { useEffect, useRef } from 'react';
import type { Notification } from '@/types/notification';

interface NotificationDropdownProps {
  notifications: Notification[];
  isOpen: boolean;
  onClose: () => void;
  onNotificationClick: (notification: Notification) => void;
  error?: string | null;
  isLoading?: boolean;
}

export function NotificationDropdown({
  notifications,
  isOpen,
  onClose,
  onNotificationClick,
  error,
  isLoading,
}: NotificationDropdownProps) {
  const dropdownRef = useRef<HTMLDivElement>(null);

  // Handle click outside to close dropdown
  useEffect(() => {
    if (!isOpen) return;

    const handleClickOutside = (event: MouseEvent) => {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
        onClose();
      }
    };

    document.addEventListener('mousedown', handleClickOutside);
    return () => {
      document.removeEventListener('mousedown', handleClickOutside);
    };
  }, [isOpen, onClose]);

  // Removed: Auto-mark non-important notifications as read
  // Users must explicitly click on notifications to mark them as read

  // Get latest 10 notifications
  const latestNotifications = notifications.slice(0, 10);

  // Format relative time
  const formatRelativeTime = (dateString: string): string => {
    const date = new Date(dateString);
    const now = new Date();
    const diffInSeconds = Math.floor((now.getTime() - date.getTime()) / 1000);

    if (diffInSeconds < 60) {
      return 'Just now';
    } else if (diffInSeconds < 3600) {
      const minutes = Math.floor(diffInSeconds / 60);
      return `${minutes}m ago`;
    } else if (diffInSeconds < 86400) {
      const hours = Math.floor(diffInSeconds / 3600);
      return `${hours}h ago`;
    } else if (diffInSeconds < 604800) {
      const days = Math.floor(diffInSeconds / 86400);
      return `${days}d ago`;
    } else {
      return date.toLocaleDateString('en-US', {
        year: 'numeric',
        month: 'short',
        day: 'numeric',
      });
    }
  };

  return (
    <>
      {/* Backdrop for mobile */}
      {isOpen && (
        <div
          className="fixed inset-0 bg-black bg-opacity-25 z-40 md:hidden"
          onClick={onClose}
          aria-hidden="true"
        />
      )}

      {/* Dropdown */}
      <div
        ref={dropdownRef}
        className={`
          fixed md:absolute 
          left-0 right-0 md:left-auto md:right-0 
          top-16 md:top-full md:mt-2 
          w-full md:w-96 
          max-h-[calc(100vh-5rem)] md:max-h-[80vh]
          bg-white 
          rounded-none md:rounded-lg 
          shadow-2xl md:shadow-lg 
          border-t md:border border-gray-200 
          z-50
          overflow-hidden flex flex-col
          transition-all duration-200 ease-in-out
          ${isOpen ? 'opacity-100 translate-y-0' : 'opacity-0 -translate-y-2 pointer-events-none'}
        `}
      >
        {/* Header */}
        <div className="px-4 py-3 border-b border-gray-200 bg-white flex items-center justify-between">
          <h3 className="text-lg font-semibold text-gray-900">Notifications</h3>
          {/* Close button for mobile */}
          <button
            onClick={onClose}
            className="md:hidden p-1.5 text-gray-400 hover:text-gray-600 hover:bg-gray-100 rounded-lg transition-colors"
            aria-label="Close"
          >
            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

      {/* Notification List */}
      <div className="overflow-y-auto flex-1">
        {/* Error State */}
        {error && (
          <div className="px-4 py-6">
            <div className="bg-red-50 border border-red-200 rounded-lg p-4">
              <div className="flex items-start gap-3">
                <svg
                  className="w-5 h-5 text-red-500 flex-shrink-0 mt-0.5"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
                  />
                </svg>
                <div className="flex-1">
                  <p className="text-sm font-medium text-red-800">An error occurred</p>
                  <p className="text-sm text-red-600 mt-1">{error}</p>
                </div>
              </div>
            </div>
          </div>
        )}

        {/* Loading State */}
        {isLoading && !error && (
          <div className="px-4 py-8 text-center">
            <div className="inline-block animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
            <p className="text-sm text-gray-500 mt-2">Loading...</p>
          </div>
        )}

        {/* Empty State */}
        {!isLoading && !error && latestNotifications.length === 0 && (
          <div className="px-4 py-8 text-center text-gray-500">
            No notifications
          </div>
        )}

        {/* Notification List */}
        {!isLoading && !error && latestNotifications.length > 0 && (
          <ul className="divide-y divide-gray-100">
            {latestNotifications.map((notification) => {
              const isUnread = !notification.read_at;
              const isImportant = notification.important;

              return (
                <li key={notification.id}>
                  <button
                    onClick={() => onNotificationClick(notification)}
                    className={`
                      w-full px-4 py-3.5 md:py-3 text-left 
                      hover:bg-gray-50 active:bg-gray-100
                      transition-colors
                      relative
                      ${isUnread ? 'bg-blue-50 hover:bg-blue-100 border-l-4 border-blue-500' : 'bg-white border-l-4 border-transparent'}
                    `}
                  >
                    <div className="flex items-start gap-3">
                      {/* Unread Indicator Dot */}
                      {isUnread && (
                        <div className="flex-shrink-0 mt-1.5 md:mt-1">
                          <div className="w-2.5 h-2.5 bg-blue-500 rounded-full animate-pulse" />
                        </div>
                      )}

                      {/* Important Indicator */}
                      {!isUnread && isImportant && (
                        <div className="flex-shrink-0 mt-1.5 md:mt-1">
                          <div className="w-2 h-2 bg-red-500 rounded-full" />
                        </div>
                      )}

                      {/* Spacer for alignment when no indicator */}
                      {!isUnread && !isImportant && (
                        <div className="flex-shrink-0 w-2.5" />
                      )}

                      {/* Notification Content */}
                      <div className="flex-1 min-w-0">
                        <div className="flex items-start justify-between gap-2">
                          <p
                            className={`
                              text-sm md:text-sm text-gray-900 
                              line-clamp-2 md:truncate
                              ${isUnread ? 'font-bold' : 'font-medium'}
                            `}
                          >
                            {notification.title}
                          </p>
                          {isUnread && (
                            <span className="flex-shrink-0 px-2 py-0.5 text-xs font-semibold text-blue-700 bg-blue-100 rounded-full">
                              Unread
                            </span>
                          )}
                        </div>
                        <p className={`text-xs mt-1.5 md:mt-1 ${isUnread ? 'text-gray-600' : 'text-gray-500'}`}>
                          {formatRelativeTime(notification.created_at)}
                        </p>
                      </div>
                    </div>
                  </button>
                </li>
              );
            })}
          </ul>
        )}
      </div>
    </div>
    </>
  );
}
