import { useEffect, useState, useMemo, useCallback } from "react";
import * as Y from "yjs";
import { YjsWebSocketProvider } from "@/lib/yjs-provider";
import { WebSocketClient } from "@/lib/websocket";
import * as awarenessProtocol from "y-protocols/awareness";

import { IndexeddbPersistence } from "y-indexeddb";

export interface RemoteUser {
  clientId: number;
  user: { id: string; name: string; color: string };
  cursor?: unknown;
}

export interface YjsEditorState {
  ydoc: Y.Doc;
  awareness: awarenessProtocol.Awareness;
  provider: YjsWebSocketProvider | null;
  synced: boolean;
  localSynced: boolean;
  remoteUsers: RemoteUser[];
  applyReset: (snapshot: Uint8Array) => void;
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
  const [localSynced, setLocalSynced] = useState(false);
  const [remoteUsers, setRemoteUsers] = useState<RemoteUser[]>([]);
  const [provider, setProvider] = useState<YjsWebSocketProvider | null>(null);

  const { ydoc, awareness } = useMemo(() => {
    const doc = new Y.Doc();
    const aware = new awarenessProtocol.Awareness(doc);
    return { ydoc: doc, awareness: aware };
  }, []);

  useEffect(() => {
    return () => {
      ydoc.destroy();
    };
  }, [ydoc]);

  useEffect(() => {
    if (!documentId) return;

    setLocalSynced(false);
    const idbProvider = new IndexeddbPersistence(documentId, ydoc);

    idbProvider.on("synced", () => {
      console.log("[Yjs] Local IndexedDB synced");
      setLocalSynced(true);
    });

    return () => {
      idbProvider.destroy();
    };
  }, [documentId, ydoc]);

  useEffect(() => {
    if (!ws || !documentId) return;

    const p = new YjsWebSocketProvider(documentId, ydoc, ws, awareness);
    setProvider(p);

    if (userId && userName) {
      p.setAwarenessField("user", {
        id: userId,
        name: userName,
        color: userColor ?? generateColor(userId),
      });
    }

    p.on("synced", () => {
      setSynced(true);
    });

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

  const applyReset = useCallback(
    (snapshot: Uint8Array) => {
      try {
        console.log("[Yjs] Applying yjs-reset snapshot to ydoc…");

        Y.transact(
          ydoc,
          () => {
            Y.applyUpdate(ydoc, snapshot);
          },
          "yjs-reset",
        );
        console.log("[Yjs] yjs-reset applied successfully");
      } catch (err) {
        console.error("[Yjs] Failed to apply yjs-reset snapshot:", err);
      }
    },
    [ydoc],
  );

  useEffect(() => {
    if (!provider) return;

    const onReset = (snapshot: Uint8Array) => {
      applyReset(snapshot);
    };

    provider.on("reset", onReset);
    return () => {
      provider.off("reset", onReset);
    };
  }, [provider, applyReset]);

  return {
    ydoc,
    awareness,
    provider,
    synced,
    localSynced,
    remoteUsers,
    applyReset,
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
