import { useEffect, useRef, useState } from 'react';
import { WebSocketClient, UserPresence } from '@/lib/websocket';

export const useWebSocket = (documentId: string, token: string) => {
    const wsRef = useRef<WebSocketClient | null>(null);
    const [isConnected, setIsConnected] = useState(false);
    const [onlineUsers, setOnlineUsers] = useState<UserPresence[]>([]);

    useEffect(() => {
        if (!documentId || !token) return;

        const ws = new WebSocketClient(documentId, token);
        wsRef.current = ws;

        ws.onConnect(() => {
            console.log('Connected to WebSocket');
            setIsConnected(true);
        });

        ws.onDisconnect(() => {
            console.log('Disconnected from WebSocket');
            setIsConnected(false);
        });

        ws.on('presence', (message) => {
            if (message.data?.users) {
                setOnlineUsers(message.data.users);
            }
        });

        ws.on('join', (message) => {
            console.log(`${message.user_name} joined`);
        });

        ws.on('leave', (message) => {
            console.log(`${message.user_name} left`);
        });

        ws.connect();

        return () => {
            ws.disconnect();
        };
    }, [documentId, token]);

    return {
        ws: wsRef.current,
        isConnected,
        onlineUsers,
    };
};