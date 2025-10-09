'use client';

import { useState } from 'react';
import type { JobType } from '@/types/notification';

interface JobTriggerProps {
  type: JobType;
  label: string;
}

export default function JobTrigger({ type, label }: JobTriggerProps) {
  const [isExecuting, setIsExecuting] = useState(false);
  const [message, setMessage] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const handleClick = async () => {
    setIsExecuting(true);
    setMessage(null);
    setError(null);

    try {
      const response = await fetch('/api/jobs', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ type }),
      });

      if (!response.ok) {
        throw new Error('Failed to start job');
      }

      setMessage('Job started');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'An error occurred');
    } finally {
      setIsExecuting(false);
    }
  };

  return (
    <div className="flex flex-col gap-2">
      <button
        onClick={handleClick}
        disabled={isExecuting}
        className={`
          w-full sm:w-auto
          px-5 sm:px-6 py-2.5 sm:py-3 
          rounded-lg font-medium text-white text-sm sm:text-base
          transition-all duration-200
          shadow-sm hover:shadow-md
          ${isExecuting 
            ? 'bg-gray-400 cursor-not-allowed' 
            : 'bg-blue-600 hover:bg-blue-700 active:bg-blue-800 active:scale-[0.98]'
          }
        `}
      >
        {isExecuting ? (
          <span className="flex items-center justify-center gap-2">
            <svg className="animate-spin h-4 w-4" fill="none" viewBox="0 0 24 24">
              <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
              <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
            </svg>
            Running...
          </span>
        ) : label}
      </button>
      {message && (
        <div className="flex items-start gap-2 p-2 bg-green-50 border border-green-200 rounded-md">
          <svg className="w-4 h-4 text-green-600 flex-shrink-0 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
          </svg>
          <p className="text-xs sm:text-sm text-green-700">{message}</p>
        </div>
      )}
      {error && (
        <div className="flex items-start gap-2 p-2 bg-red-50 border border-red-200 rounded-md">
          <svg className="w-4 h-4 text-red-600 flex-shrink-0 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
          </svg>
          <p className="text-xs sm:text-sm text-red-700">{error}</p>
        </div>
      )}
    </div>
  );
}
