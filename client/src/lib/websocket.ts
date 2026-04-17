export type MessageType =
  | "join"
  | "leave"
  | "edit"
  | "cursor"
  | "presence"
  | "sync"
  | "awareness"
  | "yjs-sync"
  | "yjs-awareness"
  | "yjs-init";

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
  private messageHandlers: Map<MessageType, Set<(data: any) => void>> =
    new Map();
  private connectionHandlers: Set<() => void> = new Set();
  private disconnectionHandlers: Set<() => void> = new Set();

  constructor(
    private documentId: string,
    private token: string,
  ) {}

  connect() {
    const WS_URL = process.env.NEXT_PUBLIC_WS_URL || "ws://localhost:8080";
    const url = `${WS_URL}/ws/documents/${this.documentId}`;

    try {
      this.ws = new WebSocket(url, ["access_token", this.token]);
    } catch (error) {
      console.error("Failed to create WebSocket:", error);
      this.attemptReconnect();
      return;
    }

    this.ws.onopen = () => {
      console.log("WebSocket connected");
      this.reconnectAttempts = 0;
      this.connectionHandlers.forEach((handler) => handler());
    };

    this.ws.onmessage = (event) => {
      try {
        if (typeof event.data !== "string") {
          console.warn("Received non-string message:", event.data);
          return;
        }

        const messages = event.data
          .trim()
          .split("\n")
          .filter((msg) => msg.trim().length > 0);

        messages.forEach((msgStr) => {
          try {
            const trimmedMsg = msgStr.trim();
            if (!trimmedMsg) return;

            const message: WSMessage = JSON.parse(trimmedMsg);
            const handlers = this.messageHandlers.get(message.type);
            if (handlers) {
              handlers.forEach((handler) => handler(message));
            }
          } catch (parseError) {
            console.error("Error parsing WebSocket message:", parseError);
            console.error("Problematic message:", msgStr);
          }
        });
      } catch (error) {
        console.error("Error processing WebSocket message:", error);
      }
    };

    this.ws.onerror = (error) => {
      console.warn("WebSocket connection error occurred");
      if (this.ws) {
        console.warn("WebSocket state:", {
          readyState: this.ws.readyState,
          url: this.ws.url,
        });
      }
    };

    this.ws.onclose = (event) => {
      console.log("WebSocket disconnected", event.code, event.reason);
      this.disconnectionHandlers.forEach((handler) => handler());

      if (event.code !== 1000 && event.code !== 1001) {
        this.attemptReconnect();
      }
    };
  }

  private attemptReconnect() {
    if (this.reconnectAttempts < this.maxReconnectAttempts) {
      this.reconnectAttempts++;
      console.log(`Reconnecting... Attempt ${this.reconnectAttempts}`);
      setTimeout(() => {
        this.connect();
      }, this.reconnectDelay * this.reconnectAttempts);
    } else {
      console.error("Max reconnection attempts reached");
    }
  }

  send(message: WSMessage) {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      try {
        this.ws.send(JSON.stringify(message));
      } catch (error) {
        console.error("Error sending WebSocket message:", error);
      }
    } else {
      console.warn(
        "WebSocket is not connected, message not sent:",
        message.type,
      );
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
      this.ws.close(1000, "Client disconnect");
      this.ws = null;
    }
  }

  isConnected(): boolean {
    return this.ws !== null && this.ws.readyState === WebSocket.OPEN;
  }
}
