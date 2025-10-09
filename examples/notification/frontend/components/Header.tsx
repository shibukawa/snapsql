'use client';

import React, { useState } from 'react';
import { useNotification } from './NotificationProvider';
import { NotificationIcon } from './NotificationIcon';
import { NotificationDropdown } from './NotificationDropdown';
import { NotificationDetailModal } from './NotificationDetailModal';
import type { Notification } from '@/types/notification';

export function Header() {
  const { notifications, unreadCount, isLoading, error, markAsRead } = useNotification();
  const [isDropdownOpen, setIsDropdownOpen] = useState(false);
  const [selectedNotification, setSelectedNotification] = useState<Notification | null>(null);
  const [isModalOpen, setIsModalOpen] = useState(false);

  const handleIconClick = () => {
    setIsDropdownOpen(!isDropdownOpen);
  };

  const handleCloseDropdown = () => {
    setIsDropdownOpen(false);
  };

  const handleNotificationClick = (notification: Notification) => {
    setSelectedNotification(notification);
    setIsModalOpen(true);
    setIsDropdownOpen(false);
  };

  const handleCloseModal = () => {
    setIsModalOpen(false);
    setSelectedNotification(null);
  };

  return (
    <header className="bg-white shadow-sm border-b border-gray-200 sticky top-0 z-30">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="flex justify-between items-center h-14 sm:h-16">
          {/* Logo/Title */}
          <div className="flex items-center min-w-0">
            <h1 className="text-base sm:text-lg md:text-xl font-semibold text-gray-900 truncate">
              Notification Demo
            </h1>
          </div>

          {/* Notification Icon */}
          <div className="flex items-center relative flex-shrink-0">
            <NotificationIcon
              unreadCount={unreadCount}
              onClick={handleIconClick}
              isLoading={isLoading}
            />
            
            {/* Notification Dropdown */}
            <NotificationDropdown
              notifications={notifications}
              isOpen={isDropdownOpen}
              onClose={handleCloseDropdown}
              onNotificationClick={handleNotificationClick}
              error={error}
              isLoading={isLoading}
            />
          </div>
        </div>
      </div>

      {/* Notification Detail Modal */}
      <NotificationDetailModal
        notification={selectedNotification}
        isOpen={isModalOpen}
        onClose={handleCloseModal}
        onMarkAsRead={markAsRead}
      />
    </header>
  );
}
