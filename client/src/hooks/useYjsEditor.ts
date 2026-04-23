import { useEffect, useState, useMemo } from "react";
import * as Y from "yjs";
import { YjsWebSocketProvider } from "@/lib/yjs-provider";
import { WebSocketClient } from "@/lib/websocket";

export interface RemoteUser {
  clientId: number;
  user: { id: string; name: string; color: string };
  cursor?: unknown;
}

export interface YjsEditorState {
  ydoc: Y.Doc;
  provider: YjsWebSocketProvider | null;
  synced: boolean;
  remoteUsers: RemoteUser[];
}

interface UseYjsEditorOptions {
  documentId: string;
  ws: WebSocketClient | null;
  userId?: string;
  userName?: string;
  userColor?: string;
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
  const [provider, setProvider] = useState<YjsWebSocketProvider | null>(null);

  const ydoc = useMemo(() => new Y.Doc(), []);

  useEffect(() => {
    return () => {
      ydoc.destroy();
    };
  }, [ydoc]);

  useEffect(() => {
    if (!ws || !documentId) return;

    const p = new YjsWebSocketProvider(documentId, ydoc, ws);
    setProvider(p);

    if (userId && userName) {
      p.setAwarenessField("user", {
        id: userId,
        name: userName,
        color: userColor ?? generateColor(userId),
      });
    }

    p.on("synced", () => setSynced(true));

    const awarenessObserver = () => {
      const states: RemoteUser[] = [];
      p.awareness.getStates().forEach((state, clientId) => {
        if (clientId !== p.awareness.clientID && state.user) {
          states.push({ clientId, user: state.user, cursor: state.cursor });
        }
      });
      setRemoteUsers(states);
    };
    p.awareness.on("change", awarenessObserver);

    return () => {
      p.awareness.off("change", awarenessObserver);
      p.destroy();
      setProvider(null);
      setSynced(false);
      setRemoteUsers([]);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [documentId, ws]);

  return {
    ydoc,
    provider,
    synced,
    remoteUsers,
  };
};

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
