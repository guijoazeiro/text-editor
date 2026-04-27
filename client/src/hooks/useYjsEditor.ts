import { useEffect, useState, useMemo, useCallback } from "react";
import * as Y from "yjs";
import { YjsWebSocketProvider } from "@/lib/yjs-provider";
import { WebSocketClient } from "@/lib/websocket";
import * as awarenessProtocol from "y-protocols/awareness";

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
  token?: string;
}

export const useYjsEditor = ({
  documentId,
  ws,
  userId,
  userName,
  userColor,
  token,
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

    const clearAndSync = async () => {
      try {
        await new Promise<void>((resolve) => {
          const req = indexedDB.deleteDatabase(documentId);
          req.onsuccess = () => resolve();
          req.onerror = () => resolve();
          req.onblocked = () => resolve();
        });
        console.log(
          "[Yjs] IndexedDB cleared — loading fresh state from server",
        );
      } catch {}
      setLocalSynced(true);
    };

    clearAndSync();
  }, [documentId, ydoc]);

  useEffect(() => {
    if (!ws || !documentId) return;

    const p = new YjsWebSocketProvider(documentId, ydoc, ws, awareness, token);
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
    async (snapshot: Uint8Array) => {
      try {
        console.log("[Yjs] Applying yjs-reset snapshot to ydoc…");

        // Apply the authoritative snapshot. We use transact with origin
        // "yjs-reset" so the provider's doc.on("update") listener ignores
        // this update and does NOT re-broadcast it to the server.
        Y.transact(
          ydoc,
          () => {
            Y.applyUpdate(ydoc, snapshot);
          },
          "yjs-reset",
        );

        // Clear the IndexedDB cache so the next page load doesn't restore
        // the pre-reset state from local storage.
        try {
          const { clearDocument } = await import("y-indexeddb");
          await clearDocument(documentId);
          console.log("[Yjs] IndexedDB cleared after reset");
        } catch {
          // y-indexeddb may not export clearDocument — fall back to manual clear
          // The IndexeddbPersistence uses documentId as the DB name directly.
          indexedDB.deleteDatabase(documentId);
          console.log("[Yjs] IndexedDB deleted after reset (fallback)");
        }

        console.log("[Yjs] yjs-reset applied successfully");
      } catch (err) {
        console.error("[Yjs] Failed to apply yjs-reset snapshot:", err);
      }
    },
    [ydoc, documentId],
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
