import { useEffect, useRef, useState, useCallback } from 'react';

export interface WebSocketEvent {
  type: string;
  project_id: string;
  timestamp: string;
  data: any;
}

export interface UseWebSocketOptions {
  projectId: string;
  onEvent?: (event: WebSocketEvent) => void;
  onConnect?: () => void;
  onDisconnect?: () => void;
  onError?: (error: Event) => void;
  reconnectInterval?: number;
  maxReconnectAttempts?: number;
  filters?: string[];
}

export function useWebSocket(options: UseWebSocketOptions) {
  const {
    projectId,
    onEvent,
    onConnect,
    onDisconnect,
    onError,
    reconnectInterval = 3000,
    maxReconnectAttempts = 10,
    filters = [],
  } = options;

  const [isConnected, setIsConnected] = useState(false);
  const [reconnectAttempt, setReconnectAttempt] = useState(0);
  const wsRef = useRef<WebSocket | null>(null);
  const reconnectTimeoutRef = useRef<NodeJS.Timeout | null>(null);
  const shouldReconnectRef = useRef(true);

  const connect = useCallback(() => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      return;
    }

    try {
      // Determine WebSocket URL
      const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
      const host = import.meta.env.VITE_WS_URL || window.location.host;
      const wsUrl = `${protocol}//${host}/ws?project_id=${projectId}`;

      const ws = new WebSocket(wsUrl);

      ws.onopen = () => {
        console.log('WebSocket connected');
        setIsConnected(true);
        setReconnectAttempt(0);

        // Subscribe to filters if any
        if (filters.length > 0) {
          ws.send(JSON.stringify({
            type: 'subscribe',
            filters,
          }));
        }

        onConnect?.();
      };

      ws.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data);

          // Handle pong messages
          if (data.type === 'pong') {
            return;
          }

          onEvent?.(data);
        } catch (error) {
          console.error('Failed to parse WebSocket message:', error);
        }
      };

      ws.onerror = (error) => {
        console.error('WebSocket error:', error);
        onError?.(error);
      };

      ws.onclose = () => {
        console.log('WebSocket disconnected');
        setIsConnected(false);
        onDisconnect?.();

        // Attempt to reconnect
        if (shouldReconnectRef.current && reconnectAttempt < maxReconnectAttempts) {
          const delay = reconnectInterval * Math.pow(1.5, reconnectAttempt);
          console.log(`Reconnecting in ${delay}ms (attempt ${reconnectAttempt + 1}/${maxReconnectAttempts})`);

          reconnectTimeoutRef.current = setTimeout(() => {
            setReconnectAttempt((prev) => prev + 1);
            connect();
          }, delay);
        }
      };

      wsRef.current = ws;
    } catch (error) {
      console.error('Failed to create WebSocket connection:', error);
    }
  }, [
    projectId,
    filters,
    onConnect,
    onDisconnect,
    onError,
    onEvent,
    reconnectAttempt,
    reconnectInterval,
    maxReconnectAttempts,
  ]);

  const disconnect = useCallback(() => {
    shouldReconnectRef.current = false;

    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current);
      reconnectTimeoutRef.current = null;
    }

    if (wsRef.current) {
      wsRef.current.close();
      wsRef.current = null;
    }

    setIsConnected(false);
  }, []);

  const send = useCallback((data: any) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify(data));
    } else {
      console.warn('WebSocket is not connected');
    }
  }, []);

  const subscribe = useCallback((filters: string[]) => {
    send({
      type: 'subscribe',
      filters,
    });
  }, [send]);

  const unsubscribe = useCallback((filters: string[]) => {
    send({
      type: 'unsubscribe',
      filters,
    });
  }, [send]);

  // Setup connection on mount
  useEffect(() => {
    shouldReconnectRef.current = true;
    connect();

    // Setup ping interval
    const pingInterval = setInterval(() => {
      if (wsRef.current?.readyState === WebSocket.OPEN) {
        send({ type: 'ping' });
      }
    }, 30000); // Ping every 30 seconds

    return () => {
      clearInterval(pingInterval);
      disconnect();
    };
  }, [projectId]); // Reconnect when project changes

  return {
    isConnected,
    reconnectAttempt,
    send,
    subscribe,
    unsubscribe,
    connect,
    disconnect,
  };
}
