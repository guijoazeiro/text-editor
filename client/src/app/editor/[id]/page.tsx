"use client";

import { useEffect, useState, useRef } from "react";
import { useParams, useRouter } from "next/navigation";
import { useEditor, EditorContent } from "@tiptap/react";
import StarterKit from "@tiptap/starter-kit";
import Collaboration from "@tiptap/extension-collaboration";
import CollaborationCursor from "@tiptap/extension-collaboration-cursor";
import Underline from "@tiptap/extension-underline";
import Placeholder from "@tiptap/extension-placeholder";
import { useAuthStore } from "@/store/authStore";
import { documentsAPI } from "@/lib/api";
import { useWebSocket } from "@/hooks/useWebSocket";
import { useYjsEditor } from "@/hooks/useYjsEditor";
import { YChange } from "@/lib/ychange";
import Navbar from "@/components/Navbar";
import EditorToolbar from "@/components/EditorToolbar";
import UserPresence from "@/components/UserPresence";

interface DocMeta {
  title: string;
  permission: string;
  content_format: string;
  content_plain?: string;
}

export default function EditorPage() {
  const params = useParams();
  const router = useRouter();
  const documentId = params.id as string;
  const { isAuthenticated, initialize, user, token, isHydrated } =
    useAuthStore();

  const [meta, setMeta] = useState<DocMeta | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [title, setTitle] = useState("");
  const documentFetchedRef = useRef(false);
  const seededRef = useRef(false);

  const userColor = generateColor(user?.id ?? "");
  const userName = user?.name ?? "Anonymous";

  const { ws, isConnected, onlineUsers } = useWebSocket(
    documentId,
    token || "",
  );

  const { ydoc, awareness, provider, synced, localSynced, remoteUsers } = useYjsEditor({
    documentId,
    ws: meta !== null ? ws : null,
    userId: user?.id,
    userName: user?.name,
    userColor,
  });

  const editor = useEditor(
    {
      extensions: [
        StarterKit.configure({ history: false }),
        Underline,
        YChange,
        Placeholder.configure({
          placeholder: "Start typing… (CRDT-powered real-time collaboration)",
          emptyEditorClass: "is-editor-empty",
        }),
        Collaboration.configure({
          document: ydoc,
          field: "content",
          ySyncOptions: {
            colors: [{ light: "#6b7280", dark: "#6b7280" }],
          },
        }),
        CollaborationCursor.configure({
          provider: { awareness },
          user: { name: userName, color: userColor },
        }),
      ],
      editable: false,
      editorProps: {
        attributes: { class: "tiptap-editor", spellcheck: "false" },
      },
      immediatelyRender: false,
    },
    [],
  );

  useEffect(() => {
    if (!editor || !meta) return;
    const canEdit = meta.permission === "owner" || meta.permission === "editor";
    // Allow editing as soon as local IndexedDB is synced (offline support)
    editor.setEditable(canEdit && localSynced);
  }, [editor, localSynced, meta]);

  useEffect(() => {
    if (!synced || !editor || !meta || seededRef.current) return;

    const fragment = ydoc.getXmlFragment("content");
    if (fragment.length > 0) {
      seededRef.current = true;
      return;
    }

    if (meta.content_plain && meta.content_plain.trim()) {
      seededRef.current = true;
      if (meta.content_format === "tiptap") {
        try {
          const parsed = JSON.parse(meta.content_plain);
          editor.commands.setContent(parsed, false);
        } catch {
          editor.commands.setContent(`<p>${meta.content_plain}</p>`, false);
        }
      } else {
        editor.commands.setContent(
          `<p>${meta.content_plain.replace(/\n/g, "</p><p>")}</p>`,
          false,
        );
      }
    } else {
      seededRef.current = true;
    }
  }, [synced, editor, meta, ydoc]);

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
      const res = await documentsAPI.get(documentId);
      const doc = res.data.data.document;
      const perm = res.data.data.permission;
      setTitle(doc.title);
      setMeta({
        title: doc.title,
        permission: perm,
        content_format: doc.content_format ?? "text",
        content_plain: doc.content,
      });
    } catch {
      router.push("/dashboard");
    } finally {
      setLoading(false);
    }
  };

  const handleSave = async () => {
    if (!meta || meta.permission === "viewer" || !editor) return;
    setSaving(true);
    try {
      const json = JSON.stringify(editor.getJSON());
      await documentsAPI.update(documentId, {
        title,
        content: json,
        content_format: "tiptap",
      });
    } catch (err) {
      console.error("Failed to save:", err);
    } finally {
      setSaving(false);
    }
  };

  useEffect(() => {
    if (!meta || meta.permission === "viewer" || !synced || !editor) return;
    const id = setInterval(() => {
      const json = JSON.stringify(editor.getJSON());
      documentsAPI
        .update(documentId, { title, content: json, content_format: "tiptap" })
        .catch(console.error);
    }, 30_000);
    return () => clearInterval(id);
  }, [documentId, title, synced, meta, editor]);

  useEffect(() => {
    return () => {
      editor?.destroy();
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

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

  const canEdit = meta?.permission === "owner" || meta?.permission === "editor";

  return (
    <div className="min-h-screen bg-gray-50">
      <Navbar />

      <div className="max-w-5xl mx-auto px-4 sm:px-6 lg:px-8 py-6">
        <div className="flex items-center justify-between mb-6">
          <div className="flex items-center space-x-4">
            <button
              onClick={() => router.push("/dashboard")}
              className="text-gray-600 hover:text-[#1479b0] transition"
            >
              ← Back
            </button>

            <div className="flex items-center space-x-2">
              <div
                className={`w-2 h-2 rounded-full ${isConnected ? "bg-green-500" : "bg-gray-400"}`}
              />
              <span className="text-sm text-gray-600">
                {isConnected ? "Connected" : "Offline (Saved locally)"}
              </span>
            </div>

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
            <UserPresence
              users={onlineUsers}
              remoteUsers={remoteUsers}
              currentUserId={user?.id ?? ""}
            />

            {canEdit && (
              <button
                onClick={handleSave}
                disabled={saving || !synced}
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

          {canEdit && (
            <EditorToolbar editor={editor ?? null} disabled={!synced} />
          )}

          <div className="p-6">
            <EditorContent editor={editor} />
          </div>
        </div>

        <div className="mt-4 text-center text-xs text-gray-500 space-y-1">
          {meta?.permission && (
            <div>
              You have <span className="font-medium">{meta.permission}</span>{" "}
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
