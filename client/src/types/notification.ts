export interface Notification {
  id: string;
  user_id: string;
  type: string;
  title: string;
  message: string;
  read: boolean;
  created_at: string;
  data?: Record<string, unknown>;
}

export interface NotificationList {
  notifications: Notification[];
  unread_count: number;
}
