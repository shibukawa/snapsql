'use client';

import React, { useState } from 'react';

interface NotificationIconProps {
  unreadCount: number;
  onClick: () => void;
  isLoading: boolean;
}

export function NotificationIcon({ unreadCount, onClick, isLoading }: NotificationIconProps) {
  return (
    <button
      onClick={onClick}
      className="
        relative p-2 sm:p-2.5
        text-gray-600 hover:text-gray-900 
        hover:bg-gray-100 active:bg-gray-200
        focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-1
        rounded-lg 
        transition-all duration-200
        touch-manipulation
      "
      aria-label="Notifications"
    >
      {/* Bell Icon */}
      <svg
        className="w-5 h-5 sm:w-6 sm:h-6"
        fill="none"
        stroke="currentColor"
        viewBox="0 0 24 24"
        xmlns="http://www.w3.org/2000/svg"
      >
        <path
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth={2}
          d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9"
        />
      </svg>

      {/* Loading Indicator */}
      {isLoading && (
        <div className="absolute top-1 right-1">
          <div className="w-2 h-2 bg-blue-500 rounded-full animate-pulse" />
        </div>
      )}

      {/* Unread Badge */}
      {!isLoading && unreadCount > 0 && (
        <span className="
          absolute -top-1 -right-1 sm:top-0 sm:right-0
          inline-flex items-center justify-center 
          px-1.5 py-0.5 sm:px-2 sm:py-1
          text-[10px] sm:text-xs font-bold leading-none 
          text-white 
          bg-red-500 
          rounded-full 
          min-w-[18px] sm:min-w-[20px]
          shadow-sm
          transform sm:translate-x-1/2 sm:-translate-y-1/2
        ">
          {unreadCount > 99 ? '99+' : unreadCount}
        </span>
      )}
    </button>
  );
}
