import { useEffect, useState, useRef, useCallback } from "react";
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
  cursor?: {
    anchor: number;
    head: number;
  };
}

interface UseYjsEditorOptions {
  documentId: string;
  ws: WebSocketClient | null;
  initialContent?: string;
  userId?: string;
  userName?: string;
  userColor?: string;
}

export const useYjsEditor = ({
  documentId,
  ws,
  initialContent = "",
  userId,
  userName,
  userColor,
}: UseYjsEditorOptions) => {
  const [content, setContent] = useState(initialContent);
  const [synced, setSynced] = useState(false);
  const [remoteUsers, setRemoteUsers] = useState<RemoteUser[]>([]);

  const ydocRef = useRef<Y.Doc | null>(null);
  const providerRef = useRef<YjsWebSocketProvider | null>(null);
  const ytextRef = useRef<Y.Text | null>(null);
  const seededRef = useRef(false);

  useEffect(() => {
    if (!ws || !documentId) return;

    const ydoc = new Y.Doc();
    ydocRef.current = ydoc;

    const ytext = ydoc.getText("content");
    ytextRef.current = ytext;

    if (!seededRef.current && initialContent && ytext.length === 0) {
      seededRef.current = true;
      ytext.insert(0, initialContent);
    }

    setContent(ytext.toString());

    const provider = new YjsWebSocketProvider(documentId, ydoc, ws);
    providerRef.current = provider;

    if (userId && userName) {
      provider.setAwarenessField("user", {
        id: userId,
        name: userName,
        color: userColor ?? generateColor(userId),
      });
    }

    provider.on("synced", () => {
      setSynced(true);
      setContent(ytext.toString());
    });

    const textObserver = () => setContent(ytext.toString());
    ytext.observe(textObserver);

    const awarenessObserver = () => {
      const states: RemoteUser[] = [];
      provider.awareness.getStates().forEach((state, clientId) => {
        if (clientId !== provider.awareness.clientID && state.user) {
          states.push({
            clientId,
            user: state.user,
            cursor: state.cursor,
          });
        }
      });
      setRemoteUsers(states);
    };

    provider.awareness.on("change", awarenessObserver);

    return () => {
      provider.awareness.off("change", awarenessObserver);
      ytext.unobserve(textObserver);
      provider.destroy();
      ydoc.destroy();
      ydocRef.current = null;
      providerRef.current = null;
      ytextRef.current = null;
      seededRef.current = false;
      setSynced(false);
      setRemoteUsers([]);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [documentId, ws]);

  const updateContent = useCallback((newContent: string) => {
    const ytext = ytextRef.current;
    if (!ytext) return;
    const current = ytext.toString();
    if (current === newContent) return;
    applyStringDiff(ytext, current, newContent);
    setContent(newContent);
  }, []);

  const updateCursor = useCallback(
    (selectionStart: number, selectionEnd: number) => {
      providerRef.current?.setAwarenessField("cursor", {
        anchor: selectionStart,
        head: selectionEnd,
      });
    },
    [],
  );

  return {
    content,
    updateContent,
    updateCursor,
    synced,
    remoteUsers,
    ydoc: ydocRef.current,
    provider: providerRef.current,
  };
};

function applyStringDiff(ytext: Y.Text, oldStr: string, newStr: string) {
  let prefixLen = 0;
  const minLen = Math.min(oldStr.length, newStr.length);
  while (prefixLen < minLen && oldStr[prefixLen] === newStr[prefixLen]) {
    prefixLen++;
  }

  let oldSuffixLen = 0;
  let newSuffixLen = 0;
  while (
    oldSuffixLen < oldStr.length - prefixLen &&
    newSuffixLen < newStr.length - prefixLen &&
    oldStr[oldStr.length - 1 - oldSuffixLen] ===
      newStr[newStr.length - 1 - newSuffixLen]
  ) {
    oldSuffixLen++;
    newSuffixLen++;
  }

  const deleteCount = oldStr.length - prefixLen - oldSuffixLen;
  const insertText = newStr.slice(prefixLen, newStr.length - newSuffixLen);

  if (deleteCount > 0) ytext.delete(prefixLen, deleteCount);
  if (insertText.length > 0) ytext.insert(prefixLen, insertText);
}

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
