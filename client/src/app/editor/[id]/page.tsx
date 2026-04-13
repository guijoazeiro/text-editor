"use client";

import { useEffect, useState, useRef } from "react";
import { useParams, useRouter } from "next/navigation";
import { useAuthStore } from "@/store/authStore";
import { documentsAPI } from "@/lib/api";
import { useWebSocket } from "@/hooks/useWebSocket";
import { useYjsEditor } from "@/hooks/useYjsEditor";
import Navbar from "@/components/Navbar";
import UserPresence from "@/components/UserPresence";

export default function EditorPage() {
  const params = useParams();
  const router = useRouter();
  const documentId = params.id as string;
  const { isAuthenticated, initialize, user, token, isHydrated } =
    useAuthStore();

  const [title, setTitle] = useState("");
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [permission, setPermission] = useState("");

  const [initialContent, setInitialContent] = useState<string | undefined>(
    undefined,
  );
  const documentFetchedRef = useRef(false);

  const { ws, isConnected, onlineUsers } = useWebSocket(
    documentId,
    token || "",
  );

  const { content, updateContent, synced } = useYjsEditor({
    documentId,
    ws: initialContent !== undefined ? ws : null,
    initialContent,
    userId: user?.id,
    userName: user?.name,
  });

  useEffect(() => {
    initialize();
  }, [initialize]);

  useEffect(() => {
    if (!isHydrated) return;
    if (!isAuthenticated) {
      router.push("/login");
      return;
    }
    if (!documentFetchedRef.current) {
      documentFetchedRef.current = true;
      fetchDocument();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isAuthenticated, isHydrated]);

  const fetchDocument = async () => {
    try {
      const response = await documentsAPI.get(documentId);
      const doc = response.data.data.document;
      const perm = response.data.data.permission;

      setTitle(doc.title);
      setInitialContent(doc.content ?? "");
      setPermission(perm);
    } catch {
      router.push("/dashboard");
    } finally {
      setLoading(false);
    }
  };

  const handleSave = async () => {
    if (permission === "viewer") return;
    setSaving(true);
    try {
      await documentsAPI.update(documentId, { title, content });
    } catch (err) {
      console.error("Failed to save document:", err);
    } finally {
      setSaving(false);
    }
  };

  const handleContentChange = (e: React.ChangeEvent<HTMLTextAreaElement>) => {
    updateContent(e.target.value);
  };

  useEffect(() => {
    if (permission === "viewer" || !synced) return;
    const id = setInterval(() => {
      if (content || title) {
        documentsAPI
          .update(documentId, { title, content })
          .catch(console.error);
      }
    }, 30_000);
    return () => clearInterval(id);
  }, [documentId, title, content, permission, synced]);

  if (loading) {
    return (
      <div className="min-h-screen bg-gray-50">
        <Navbar />
        <div className="flex items-center justify-center h-[calc(100vh-4rem)]">
          <div className="text-gray-600">Loading document...</div>
        </div>
      </div>
    );
  }

  const canEdit = permission === "owner" || permission === "editor";

  return (
    <div className="min-h-screen bg-gray-50">
      <Navbar />

      <div className="max-w-5xl mx-auto px-4 sm:px-6 lg:px-8 py-6">
        {/* Header */}
        <div className="flex items-center justify-between mb-6">
          <div className="flex items-center space-x-4">
            <button
              onClick={() => router.push("/dashboard")}
              className="text-gray-600 hover:text-[#1479b0] transition"
            >
              ← Back
            </button>

            {/* WebSocket status */}
            <div className="flex items-center space-x-2">
              <div
                className={`w-2 h-2 rounded-full ${isConnected ? "bg-green-500" : "bg-gray-400"}`}
              />
              <span className="text-sm text-gray-600">
                {isConnected ? "Connected" : "Disconnected"}
              </span>
            </div>

            {/* Yjs sync status */}
            <div className="flex items-center space-x-2">
              <div
                className={`w-2 h-2 rounded-full ${synced ? "bg-blue-500" : "bg-yellow-400"}`}
              />
              <span className="text-sm text-gray-600">
                {synced ? "Synced" : "Syncing…"}
              </span>
            </div>
          </div>

          <div className="flex items-center space-x-4">
            {user && (
              <UserPresence users={onlineUsers} currentUserId={user.id} />
            )}

            {canEdit && (
              <button
                onClick={handleSave}
                disabled={saving}
                className="px-6 py-2 bg-[#1479b0] text-white rounded-lg font-medium hover:bg-[#0f5f8d] transition disabled:opacity-50"
              >
                {saving ? "Saving…" : "Save"}
              </button>
            )}

            {!canEdit && (
              <span className="px-4 py-2 bg-gray-100 text-gray-600 rounded-lg text-sm">
                View Only
              </span>
            )}
          </div>
        </div>

        {/* Document */}
        <div className="bg-white rounded-lg shadow-sm border border-gray-200 overflow-hidden">
          <div className="p-6 border-b border-gray-200">
            <input
              type="text"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              disabled={!canEdit}
              placeholder="Untitled Document"
              className="w-full text-3xl font-bold text-gray-900 outline-none placeholder-gray-400 disabled:bg-transparent"
            />
          </div>

          <div className="p-6">
            <textarea
              value={content}
              onChange={handleContentChange}
              disabled={!canEdit}
              placeholder="Start typing… (CRDT-powered real-time collaboration)"
              className="w-full h-[calc(100vh-20rem)] text-gray-900 outline-none resize-none font-mono text-sm leading-relaxed placeholder-gray-400 disabled:bg-transparent"
            />
          </div>
        </div>

        <div className="mt-4 text-center text-xs text-gray-500 space-y-1">
          {permission && (
            <div>
              You have <span className="font-medium">{permission}</span>{" "}
              permission
            </div>
          )}
          {synced && (
            <div className="text-blue-600">
              ✓ Real-time collaboration powered by Yjs CRDT
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
