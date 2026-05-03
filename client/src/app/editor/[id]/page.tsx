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
import { useVersionHistory } from "@/hooks/useVersionHistory";
import { YChange } from "@/lib/ychange";
import Navbar from "@/components/Navbar";
import EditorToolbar from "@/components/EditorToolbar";
import UserPresence from "@/components/UserPresence";
import { VersionHistory } from "@/components/editor/VersionHistory";
import { CollaboratorsModal } from "@/components/editor/CollaboratorsModal";

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
  const [historyOpen, setHistoryOpen] = useState(false);

  const {
    versions,
    loading: versionsLoading,
    restoring,
    pagination: versionPagination,
    fetchVersions,
    restoreVersion,
  } = useVersionHistory({
    documentId,
    onRestoreComplete: () => setHistoryOpen(false),
  });

  const openHistory = () => {
    setHistoryOpen(true);
    fetchVersions(1, 20);
  };
  const [shareOpen, setShareOpen] = useState(false);
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

  const {
    ydoc,
    awareness,
    provider,
    synced,
    localSynced,
    remoteUsers,
    applyReset,
  } = useYjsEditor({
    documentId,
    ws: meta !== null ? ws : null,
    userId: user?.id,
    userName: user?.name,
    userColor,
    token: token || undefined,
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
    editor.setEditable(canEdit && localSynced);
  }, [editor, localSynced, meta]);

  useEffect(() => {
    if (!provider || !editor) return;

    const onReset = (snapshot: Uint8Array) => {
      const canEdit =
        meta?.permission === "owner" || meta?.permission === "editor";
      editor.setEditable(false);
      applyReset(snapshot);
      Promise.resolve().then(() => {
        editor.setEditable(canEdit && localSynced);
      });
    };

    provider.on("reset", onReset);
    return () => {
      provider.off("reset", onReset);
    };
  }, [provider, editor, meta, applyReset, localSynced]);

  useEffect(() => {
    if (!ws || !editor) return;

    const handleContentReset = (
      message: import("@/lib/websocket").WSMessage,
    ) => {
      const content = message?.data?.content;
      if (!content) return;

      const canEdit =
        meta?.permission === "owner" || meta?.permission === "editor";
      editor.setEditable(false);

      try {
        const parsed = JSON.parse(content);
        editor.commands.setContent(parsed, true);
      } catch {
        editor.commands.setContent(`<p>${content}</p>`, true);
      }

      Promise.resolve().then(() => {
        editor.setEditable(canEdit && localSynced);
      });

      console.log(
        "[Editor] document-content-reset applied (legacy version restore)",
      );
    };

    ws.on("document-content-reset", handleContentReset);
    return () => {
      ws.off("document-content-reset", handleContentReset);
    };
  }, [ws, editor, meta, localSynced]);

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
      <div className="min-h-screen">
        <Navbar />
        <div className="flex items-center justify-center h-[calc(100vh-4rem)]">
          <div className="text-[var(--text-secondary)] animate-pulse">
            Loading document...
          </div>
        </div>
      </div>
    );
  }

  const canEdit = meta?.permission === "owner" || meta?.permission === "editor";

  return (
    <div className="min-h-screen">
      <Navbar />

      <div className="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 py-6">
        <div className="flex items-center justify-between mb-5">
          <div className="flex items-center gap-4">
            <button
              onClick={() => router.push("/dashboard")}
              className="text-[var(--text-secondary)] hover:text-[var(--text-primary)] transition text-sm flex items-center gap-1.5"
            >
              ← Back
            </button>

            <div className="flex items-center gap-1.5">
              <div
                className={`w-1.5 h-1.5 rounded-full ${isConnected ? "bg-emerald-400" : "bg-slate-500"}`}
              />
              <span className="text-xs text-[var(--text-secondary)]">
                {isConnected ? "Connected" : "Offline"}
              </span>
            </div>

            <div className="flex items-center gap-1.5">
              <div
                className={`w-1.5 h-1.5 rounded-full ${synced ? "bg-blue-400" : "bg-amber-400"}`}
              />
              <span className="text-xs text-[var(--text-secondary)]">
                {synced ? "Synced" : "Syncing…"}
              </span>
            </div>
          </div>

          <div className="flex items-center gap-3">
            <UserPresence
              users={onlineUsers}
              remoteUsers={remoteUsers}
              currentUserId={user?.id ?? ""}
            />

            {canEdit && (
              <button
                onClick={handleSave}
                disabled={saving || !synced}
                className="px-4 py-1.5 bg-blue-600 text-white rounded-lg text-sm font-medium hover:bg-blue-500 transition disabled:opacity-40 disabled:cursor-not-allowed shadow-sm shadow-blue-900/30"
              >
                {saving ? "Saving…" : "Save"}
              </button>
            )}

            <button
              onClick={openHistory}
              title="Version history"
              className="w-8 h-8 flex items-center justify-center rounded-lg text-[var(--text-secondary)] hover:text-[var(--text-primary)] hover:bg-black/5 dark:hover:bg-white/5 transition-all"
            >
              <svg
                className="w-4 h-4"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
                strokeWidth={1.75}
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"
                />
              </svg>
            </button>

            <button
              onClick={() => setShareOpen(true)}
              title="Share document"
              className="w-8 h-8 flex items-center justify-center rounded-lg text-[var(--text-secondary)] hover:text-[var(--text-primary)] hover:bg-black/5 dark:hover:bg-white/5 transition-all"
            >
              <svg
                className="w-4 h-4"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
                strokeWidth={1.75}
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  d="M8.684 13.342C8.886 12.938 9 12.482 9 12c0-.482-.114-.938-.316-1.342m0 2.684a3 3 0 110-2.684m0 2.684l6.632 3.316m-6.632-6l6.632-3.316m0 0a3 3 0 105.367-2.684 3 3 0 00-5.367 2.684zm0 9.316a3 3 0 105.368 2.684 3 3 0 00-5.368-2.684z"
                />
              </svg>
            </button>

            {!canEdit && (
              <span className="px-3 py-1.5 bg-[var(--bg-card)] border border-[var(--border)] text-[var(--text-secondary)] rounded-lg text-xs">
                View Only
              </span>
            )}
          </div>
        </div>

        <div className="bg-[var(--bg-card)] rounded-xl border border-[var(--border)] overflow-hidden shadow-xl shadow-black/10">
          <div className="px-8 pt-8 pb-5 border-b border-[var(--border)]">
            <input
              type="text"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              disabled={!canEdit}
              placeholder="Untitled Document"
              className="w-full text-3xl font-bold text-[var(--text-primary)] bg-transparent outline-none placeholder-[var(--text-placeholder)] disabled:cursor-default"
            />
          </div>

          {canEdit && (
            <EditorToolbar editor={editor ?? null} disabled={!synced} />
          )}

          <div className="px-8 py-6">
            <EditorContent editor={editor} />
          </div>
        </div>

        <div className="mt-4 text-center text-xs text-[var(--text-muted)] space-y-1">
          {meta?.permission && (
            <div>
              You have{" "}
              <span className="font-medium text-[var(--text-secondary)]">
                {meta.permission}
              </span>{" "}
              permission
            </div>
          )}
          {synced && (
            <div className="text-blue-500/50">
              ✓ Real-time collaboration powered by Yjs CRDT
            </div>
          )}
        </div>
      </div>

      <VersionHistory
        open={historyOpen}
        onClose={() => setHistoryOpen(false)}
        versions={versions}
        loading={versionsLoading}
        restoring={restoring}
        onRestore={restoreVersion}
        pagination={versionPagination}
        onPageChange={(page) => fetchVersions(page, versionPagination.limit)}
      />

      <CollaboratorsModal
        open={shareOpen}
        onClose={() => setShareOpen(false)}
        documentId={documentId}
        isOwner={meta?.permission === "owner"}
      />
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
