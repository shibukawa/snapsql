// Notification type
export interface Notification {
  id: number;
  title: string;
  body: string;
  icon_url?: string;
  important: boolean;
  cancelable: boolean;
  expires_at?: string;
  created_at: string;
  updated_at?: string;
  read_at?: string;
  delivered_at?: string;
}

// SSE Event types
export type SSEEventType = 'notification' | 'read' | 'update';

export interface SSEEvent {
  type: SSEEventType;
  payload: Notification | ReadEvent;
}

export interface ReadEvent {
  notification_id: number;
  user_id: string;
}

// Job type
export type JobType = 'success' | 'error' | 'fix';
