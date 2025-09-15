export type Channel = 'telegram' | 'email';

export type Status = 'pending' | 'sent' | 'cancelled' | 'failed';

export interface Notification {
  id: string;
  message: string;
  send_at: string; // "YYYY-MM-DD HH:mm:ss"
  retries: number;
  to: string;
  channel: Channel;
  status: Status;
  created_at?: string;
  updated_at?: string;
}