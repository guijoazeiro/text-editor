import { useEffect, useRef, useState, useCallback } from "react";
import * as Y from "yjs";
import { YjsWebSocketProvider } from "@/lib/yjs-provider";
import { WebSocketClient } from "@/lib/websocket";

export interface RemoteUser {
  clientId: number;
  user: {
    id: string;
    name: string;
    color: string;
  };
  cursor?: unknown;
}

interface UseYjsEditorOptions {
  documentId: string;
  ws: WebSocketClient | null;
  userId?: string;
  userName?: string;
  userColor?: string;
}

export interface YjsEditorState {
  ydoc: Y.Doc | null;
  provider: YjsWebSocketProvider | null;
  synced: boolean;
  remoteUsers: RemoteUser[];
}

export const useYjsEditor = ({
  documentId,
  ws,
  userId,
  userName,
  userColor,
}: UseYjsEditorOptions): YjsEditorState => {
  const [synced, setSynced] = useState(false);
  const [remoteUsers, setRemoteUsers] = useState<RemoteUser[]>([]);

  const ydocRef = useRef<Y.Doc | null>(null);
  const providerRef = useRef<YjsWebSocketProvider | null>(null);

  useEffect(() => {
    if (!ws || !documentId) return;

    const ydoc = new Y.Doc();
    ydocRef.current = ydoc;

    const provider = new YjsWebSocketProvider(documentId, ydoc, ws);
    providerRef.current = provider;

    if (userId && userName) {
      provider.setAwarenessField("user", {
        id: userId,
        name: userName,
        color: userColor ?? generateColor(userId),
      });
    }

    provider.on("synced", () => setSynced(true));

    const awarenessObserver = () => {
      const states: RemoteUser[] = [];
      provider.awareness.getStates().forEach((state, clientId) => {
        if (clientId !== provider.awareness.clientID && state.user) {
          states.push({ clientId, user: state.user, cursor: state.cursor });
        }
      });
      setRemoteUsers(states);
    };

    provider.awareness.on("change", awarenessObserver);

    return () => {
      provider.awareness.off("change", awarenessObserver);
      provider.destroy();
      ydoc.destroy();
      ydocRef.current = null;
      providerRef.current = null;
      setSynced(false);
      setRemoteUsers([]);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [documentId, ws]);

  return {
    ydoc: ydocRef.current,
    provider: providerRef.current,
    synced,
    remoteUsers,
  };
};

export const useYdoc = (state: YjsEditorState) => state.ydoc;

function generateColor(userId: string): string {
  const colors = [
    "#3B82F6",
    "#10B981",
    "#F59E0B",
    "#EF4444",
    "#8B5CF6",
    "#EC4899",
    "#14B8A6",
    "#F97316",
  ];
  let hash = 0;
  for (let i = 0; i < userId.length; i++) hash += userId.charCodeAt(i);
  return colors[hash % colors.length];
}
