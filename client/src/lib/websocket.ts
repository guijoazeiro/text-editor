export type MessageType = 'join' | 'leave' | 'edit' | 'cursor' | 'presence' | 'sync' | 'awareness';

export interface WSMessage {
  type: MessageType;
  document_id?: string;
  user_id?: string;
  user_name?: string;
  data?: any;
  timestamp?: number;
}

export interface UserPresence {
  user_id: string;
  user_name: string;
  color: string;
  cursor: { line: number; column: number };
  online: boolean;
}

export class WebSocketClient {
  private ws: WebSocket | null = null;
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 5;
  private reconnectDelay = 1000;
  private messageHandlers: Map<MessageType, Set<(data: any) => void>> = new Map();
  private connectionHandlers: Set<() => void> = new Set();
  private disconnectionHandlers: Set<() => void> = new Set();

  constructor(
    private documentId: string,
    private token: string
  ) {}

  connect() {
    const WS_URL = process.env.NEXT_PUBLIC_WS_URL || 'ws://localhost:8080';
    const url = `${WS_URL}/ws/documents/${this.documentId}`;

    this.ws = new WebSocket(url);

    this.ws.onopen = () => {
      console.log('WebSocket connected');
      this.reconnectAttempts = 0;
      this.connectionHandlers.forEach(handler => handler());
    };

    this.ws.onmessage = (event) => {
      try {
        const message: WSMessage = JSON.parse(event.data);
        const handlers = this.messageHandlers.get(message.type);
        if (handlers) {
          handlers.forEach(handler => handler(message));
        }
      } catch (error) {
        console.error('Error parsing WebSocket message:', error);
      }
    };

    this.ws.onerror = (error) => {
      console.error('WebSocket error:', error);
    };

    this.ws.onclose = () => {
      console.log('WebSocket disconnected');
      this.disconnectionHandlers.forEach(handler => handler());
      this.attemptReconnect();
    };
  }

  private attemptReconnect() {
    if (this.reconnectAttempts < this.maxReconnectAttempts) {
      this.reconnectAttempts++;
      console.log(`Reconnecting... Attempt ${this.reconnectAttempts}`);
      setTimeout(() => {
        this.connect();
      }, this.reconnectDelay * this.reconnectAttempts);
    }
  }

  send(message: WSMessage) {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(message));
    } else {
      console.error('WebSocket is not connected');
    }
  }

  on(type: MessageType, handler: (data: any) => void) {
    if (!this.messageHandlers.has(type)) {
      this.messageHandlers.set(type, new Set());
    }
    this.messageHandlers.get(type)!.add(handler);
  }

  off(type: MessageType, handler: (data: any) => void) {
    const handlers = this.messageHandlers.get(type);
    if (handlers) {
      handlers.delete(handler);
    }
  }

  onConnect(handler: () => void) {
    this.connectionHandlers.add(handler);
  }

  onDisconnect(handler: () => void) {
    this.disconnectionHandlers.add(handler);
  }

  disconnect() {
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
  }
}