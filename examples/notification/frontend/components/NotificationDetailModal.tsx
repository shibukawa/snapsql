'use client';

import React, { useEffect, useState } from 'react';
import type { Notification } from '@/types/notification';

interface NotificationDetailModalProps {
  notification: Notification | null;
  isOpen: boolean;
  onClose: () => void;
  onMarkAsRead: (notificationId: number) => Promise<void>;
}

export function NotificationDetailModal({
  notification,
  isOpen,
  onClose,
  onMarkAsRead,
}: NotificationDetailModalProps) {
  const [isMarkingAsRead, setIsMarkingAsRead] = useState(false);

  // Close modal on Escape key
  useEffect(() => {
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && isOpen) {
        handleClose();
      }
    };

    if (isOpen) {
      document.addEventListener('keydown', handleEscape);
      // Prevent body scroll when modal is open
      document.body.style.overflow = 'hidden';
    }

    return () => {
      document.removeEventListener('keydown', handleEscape);
      document.body.style.overflow = 'unset';
    };
  }, [isOpen, notification]);

  if (!isOpen || !notification) {
    return null;
  }

  const handleClose = async () => {
    // Mark as read when closing if unread
    if (!notification.read_at) {
      try {
        setIsMarkingAsRead(true);
        await onMarkAsRead(notification.id);
      } catch (err) {
        console.error('Failed to mark as read:', err);
        // Continue closing even if mark as read fails
      } finally {
        setIsMarkingAsRead(false);
      }
    }
    onClose();
  };

  const handleBackdropClick = (e: React.MouseEvent<HTMLDivElement>) => {
    if (e.target === e.currentTarget) {
      handleClose();
    }
  };

  const formatDate = (dateString: string) => {
    const date = new Date(dateString);
    return date.toLocaleString('en-US', {
      year: 'numeric',
      month: 'long',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    });
  };

  const isUnread = !notification.read_at;

  return (
    <div
      className="fixed inset-0 z-50 flex items-end md:items-center justify-center bg-black bg-opacity-50 p-0 md:p-4"
      onClick={handleBackdropClick}
    >
      <div className="bg-white rounded-t-2xl md:rounded-lg shadow-xl w-full md:max-w-2xl max-h-[90vh] overflow-y-auto">
        {/* Header */}
        <div className="flex items-start justify-between p-4 md:p-6 border-b border-gray-200 sticky top-0 bg-white z-10">
          <div className="flex-1 min-w-0">
            <div className="flex items-start gap-2">
              {notification.important && (
                <span className="inline-block w-2 h-2 bg-red-500 rounded-full mt-1.5 flex-shrink-0" title="Important" />
              )}
              <h2 className="text-lg md:text-xl font-semibold text-gray-900 break-words">
                {notification.title}
              </h2>
            </div>
            <p className="text-xs md:text-sm text-gray-500 mt-1">
              {formatDate(notification.created_at)}
            </p>
          </div>
          <button
            onClick={handleClose}
            className="text-gray-400 hover:text-gray-600 hover:bg-gray-100 rounded-lg p-1.5 transition-colors ml-2 flex-shrink-0"
            aria-label="Close"
          >
            <svg
              className="w-5 h-5 md:w-6 md:h-6"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M6 18L18 6M6 6l12 12"
              />
            </svg>
          </button>
        </div>

        {/* Body */}
        <div className="p-4 md:p-6">
          {/* Icon */}
          {notification.icon_url && (
            <div className="mb-4">
              <img
                src={notification.icon_url}
                alt="Notification icon"
                className="w-12 h-12 md:w-16 md:h-16 rounded-lg object-cover"
              />
            </div>
          )}

          {/* Body text */}
          <div className="prose prose-sm max-w-none">
            <p className="text-sm md:text-base text-gray-700 whitespace-pre-wrap leading-relaxed">
              {notification.body}
            </p>
          </div>

          {/* Metadata */}
          <div className="mt-6 pt-4 border-t border-gray-200">
            <dl className="grid grid-cols-1 sm:grid-cols-2 gap-4 text-sm">
              <div>
                <dt className="font-medium text-gray-500 text-xs md:text-sm">Priority</dt>
                <dd className="mt-1.5">
                  {notification.important ? (
                    <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-red-100 text-red-800">
                      Important
                    </span>
                  ) : (
                    <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-gray-100 text-gray-800">
                      Normal
                    </span>
                  )}
                </dd>
              </div>
              <div>
                <dt className="font-medium text-gray-500 text-xs md:text-sm">Status</dt>
                <dd className="mt-1.5">
                  {isUnread ? (
                    <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-blue-100 text-blue-800">
                      Unread
                    </span>
                  ) : (
                    <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-800">
                      Read
                    </span>
                  )}
                </dd>
              </div>
            </dl>
          </div>

        </div>

        {/* Footer */}
        <div className="flex items-center justify-end gap-2 sm:gap-3 p-4 md:p-6 border-t border-gray-200 bg-gray-50 sticky bottom-0">
          <button
            onClick={handleClose}
            disabled={isMarkingAsRead}
            className="px-4 py-2.5 md:py-2 text-sm font-medium text-white bg-blue-600 rounded-lg hover:bg-blue-700 active:bg-blue-800 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {isMarkingAsRead ? 'Processing...' : 'Close'}
          </button>
        </div>
      </div>
    </div>
  );
}
