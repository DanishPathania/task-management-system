import { RealtimeEvent } from "@/types";
import { api } from "./api";

const WS_BASE_URL = process.env.NEXT_PUBLIC_WS_URL || "ws://localhost:8080/api/v1/ws";

type EventHandler = (event: RealtimeEvent) => void;

class RealtimeClient {
  private ws: WebSocket | null = null;
  private reconnectAttempts = 0;
  private maxReconnectDelay = 30000;
  private reconnectTimeout: NodeJS.Timeout | null = null;
  private eventHandlers = new Set<EventHandler>();
  private activeSubscriptions = new Set<string>();
  private isExplicitlyClosed = false;

  connect() {
    if (typeof window === "undefined") return;
    if (this.ws && (this.ws.readyState === WebSocket.OPEN || this.ws.readyState === WebSocket.CONNECTING)) {
      return;
    }

    this.isExplicitlyClosed = false;
    const token = api.getToken();
    const url = token ? `${WS_BASE_URL}?token=${encodeURIComponent(token)}` : WS_BASE_URL;

    try {
      this.ws = new WebSocket(url);

      this.ws.onopen = () => {
        this.reconnectAttempts = 0;
        // Resubscribe to all active projects after reconnect
        this.activeSubscriptions.forEach((projectId) => {
          this.send({ action: "subscribe", projectId });
        });
      };

      this.ws.onmessage = (event) => {
        try {
          const lines = event.data.split("\n");
          for (const line of lines) {
            if (!line.trim()) continue;
            const parsed: RealtimeEvent = JSON.parse(line);
            this.eventHandlers.forEach((handler) => handler(parsed));
          }
        } catch (err) {
          console.error("Failed to parse incoming WebSocket message:", err);
        }
      };

      this.ws.onclose = () => {
        if (!this.isExplicitlyClosed) {
          this.scheduleReconnect();
        }
      };

      this.ws.onerror = (err) => {
        console.warn("WebSocket error:", err);
        this.ws?.close();
      };
    } catch (err) {
      this.scheduleReconnect();
    }
  }

  private scheduleReconnect() {
    if (this.reconnectTimeout) clearTimeout(this.reconnectTimeout);

    // Exponential backoff: min 1s, doubling up to max 30s + small jitter
    const delay = Math.min(1000 * Math.pow(2, this.reconnectAttempts), this.maxReconnectDelay);
    const jitter = Math.random() * 1000;

    this.reconnectAttempts++;
    this.reconnectTimeout = setTimeout(() => {
      this.connect();
    }, delay + jitter);
  }

  subscribeProject(projectId: string) {
    this.activeSubscriptions.add(projectId);
    this.send({ action: "subscribe", projectId });
  }

  unsubscribeProject(projectId: string) {
    this.activeSubscriptions.delete(projectId);
    this.send({ action: "unsubscribe", projectId });
  }

  onEvent(handler: EventHandler): () => void {
    this.eventHandlers.add(handler);
    return () => {
      this.eventHandlers.delete(handler);
    };
  }

  private send(data: any) {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(data));
    }
  }

  disconnect() {
    this.isExplicitlyClosed = true;
    if (this.reconnectTimeout) clearTimeout(this.reconnectTimeout);
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
  }
}

export const realtime = new RealtimeClient();
