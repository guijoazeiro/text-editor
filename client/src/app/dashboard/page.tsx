"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import { useAuthStore } from "@/store/authStore";
import { documentsAPI } from "@/lib/api";
import { useToastStore } from "@/store/toastStore";
import Navbar from "@/components/Navbar";
import { DocumentCardSkeleton } from "@/components/ui/Skeleton";
import type { Document } from "@/types";

function getPreviewText(doc: Document): string {
  if (!doc.content) return "Empty document";
  if (doc.content_format === "tiptap") {
    try {
      const json = JSON.parse(doc.content);
      const extractText = (node: {
        type?: string;
        text?: string;
        content?: unknown[];
      }): string => {
        if (node.type === "text") return node.text ?? "";
        if (node.content && Array.isArray(node.content)) {
          return (node.content as (typeof node)[]).map(extractText).join(" ");
        }
        return "";
      };
      return extractText(json).slice(0, 120) || "Empty document";
    } catch {
      return "Empty document";
    }
  }
  return doc.content.slice(0, 120);
}

function formatDate(date: string) {
  const d = new Date(date);
  const now = new Date();
  const diff = now.getTime() - d.getTime();
  const days = Math.floor(diff / 86_400_000);

  if (days === 0) return "Today";
  if (days === 1) return "Yesterday";
  if (days < 7) return `${days} days ago`;

  return d.toLocaleDateString("en-US", {
    month: "short",
    day: "numeric",
    year: "numeric",
  });
}

function PlusIcon() {
  return (
    <svg
      className="w-4 h-4"
      fill="none"
      viewBox="0 0 24 24"
      stroke="currentColor"
      strokeWidth={2}
    >
      <path strokeLinecap="round" strokeLinejoin="round" d="M12 4v16m8-8H4" />
    </svg>
  );
}

function DocIcon() {
  return (
    <svg
      className="w-12 h-12 mx-auto"
      fill="none"
      stroke="currentColor"
      viewBox="0 0 24 24"
    >
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth={1.25}
        d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"
      />
    </svg>
  );
}

export default function DashboardPage() {
  const router = useRouter();
  const { isAuthenticated, isHydrated, initialize } = useAuthStore();
  const toast = useToastStore();

  const [documents, setDocuments] = useState<Document[]>([]);
  const [loading, setLoading] = useState(true);
  const [creating, setCreating] = useState(false);
  const [search, setSearch] = useState("");
  const [trashOpen, setTrashOpen] = useState(false);
  const [trashDocs, setTrashDocs] = useState<Document[]>([]);
  const [trashLoading, setTrashLoading] = useState(false);
  const searchTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    initialize();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    if (!isHydrated) return;
    if (!isAuthenticated) {
      router.push("/login");
      return;
    }
    fetchDocuments();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isAuthenticated, isHydrated]);

  const fetchDocuments = useCallback(
    async (q?: string) => {
      try {
        const response = await documentsAPI.list(q ? { q } : undefined);
        const data = response.data.data;
        if (data && typeof data === "object" && "documents" in data) {
          setDocuments(data.documents ?? []);
        } else {
          setDocuments(Array.isArray(data) ? data : []);
        }
      } catch {
        toast.error("Failed to load documents");
      } finally {
        setLoading(false);
      }
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [],
  );

  useEffect(() => {
    if (searchTimerRef.current) clearTimeout(searchTimerRef.current);
    searchTimerRef.current = setTimeout(() => {
      fetchDocuments(search || undefined);
    }, 350);
    return () => {
      if (searchTimerRef.current) clearTimeout(searchTimerRef.current);
    };
  }, [search, fetchDocuments]);

  const openTrash = async () => {
    setTrashOpen(true);
    setTrashLoading(true);
    try {
      const res = await documentsAPI.trash();
      setTrashDocs(res.data.data ?? []);
    } catch {
      toast.error("Failed to load trash");
    } finally {
      setTrashLoading(false);
    }
  };

  const handleRestore = async (id: string) => {
    try {
      await documentsAPI.restore(id);
      setTrashDocs((prev) => prev.filter((d) => d.id !== id));
      toast.success("Document restored");
      fetchDocuments(search || undefined);
    } catch {
      toast.error("Failed to restore document");
    }
  };

  const [deletingId, setDeletingId] = useState<string | null>(null);

  const handleDelete = async (id: string, e: React.MouseEvent) => {
    e.stopPropagation();
    setDeletingId(id);
    try {
      await documentsAPI.delete(id);
      setDocuments((prev) => prev.filter((d) => d.id !== id));
      toast.success("Document moved to trash");
    } catch {
      toast.error("Failed to delete document");
    } finally {
      setDeletingId(null);
    }
  };

  const createDocument = async () => {
    setCreating(true);
    try {
      const response = await documentsAPI.create({
        title: "Untitled Document",
        content: "",
        content_format: "tiptap",
      });
      const newDoc = response.data.data;
      router.push(`/editor/${newDoc.id}`);
    } catch {
      toast.error("Failed to create document");
      setCreating(false);
    }
  };

  return (
    <div className="min-h-screen">
      <Navbar />

      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {/* Header row */}
        <div className="flex flex-col sm:flex-row sm:justify-between sm:items-center gap-4 mb-8">
          <div>
            <h1 className="text-2xl font-bold text-[var(--text-primary)]">
              My Documents
            </h1>
            {!loading && (
              <p className="text-sm text-[var(--text-muted)] mt-0.5">
                {documents.length} document{documents.length !== 1 ? "s" : ""}
              </p>
            )}
          </div>

          <div className="flex items-center gap-2">
            {/* Search bar */}
            <div className="relative">
              <svg
                className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-[var(--text-muted)] pointer-events-none"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
                strokeWidth={2}
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  d="M21 21l-4.35-4.35M17 11A6 6 0 105 11a6 6 0 0012 0z"
                />
              </svg>
              <input
                id="dashboard-search"
                type="text"
                placeholder="Search documents…"
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                className="pl-9 pr-3 py-2 text-sm rounded-lg border border-[var(--border)] bg-[var(--bg-card)] text-[var(--text-primary)] placeholder:text-[var(--text-muted)] focus:outline-none focus:ring-2 focus:ring-blue-500/40 w-52 transition-all"
              />
            </div>

            {/* Trash toggle */}
            <button
              onClick={trashOpen ? () => setTrashOpen(false) : openTrash}
              title={trashOpen ? "Back to documents" : "View trash"}
              className={`w-9 h-9 flex items-center justify-center rounded-lg border transition-all ${
                trashOpen
                  ? "border-red-500/50 text-red-400 bg-red-500/10"
                  : "border-[var(--border)] text-[var(--text-secondary)] hover:text-red-400 hover:border-red-500/40"
              }`}
            >
              <svg
                className="w-4 h-4"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
                strokeWidth={2}
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
                />
              </svg>
            </button>

            <button
              onClick={createDocument}
              disabled={creating}
              className="inline-flex items-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-lg text-sm font-medium hover:bg-blue-500 transition-all disabled:opacity-40 shadow-sm shadow-blue-900/30 disabled:cursor-not-allowed"
            >
              {creating ? (
                <svg
                  className="w-4 h-4 animate-spin"
                  fill="none"
                  viewBox="0 0 24 24"
                >
                  <circle
                    className="opacity-25"
                    cx="12"
                    cy="12"
                    r="10"
                    stroke="currentColor"
                    strokeWidth="4"
                  />
                  <path
                    className="opacity-75"
                    fill="currentColor"
                    d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"
                  />
                </svg>
              ) : (
                <PlusIcon />
              )}
              {creating ? "Creating…" : "New Document"}
            </button>
          </div>
        </div>

        {loading && (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-5">
            {Array.from({ length: 6 }).map((_, i) => (
              <DocumentCardSkeleton key={i} />
            ))}
          </div>
        )}

        {!loading && documents.length === 0 && (
          <div className="text-center py-28">
            <div className="text-[var(--text-muted)] mb-5">
              <DocIcon />
            </div>
            <p className="text-[var(--text-secondary)] text-base font-medium mb-1">
              No documents yet
            </p>
            <p className="text-sm text-[var(--text-muted)] mb-6">
              Get started by creating your first document.
            </p>
            <button
              onClick={createDocument}
              className="inline-flex items-center gap-2 px-5 py-2 bg-blue-600 text-white rounded-lg text-sm font-medium hover:bg-blue-500 transition shadow-sm shadow-blue-900/30"
            >
              <PlusIcon />
              Create document
            </button>
          </div>
        )}

        {!loading && documents.length > 0 && (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-5">
            {documents.map((doc) => (
              <div
                key={doc.id}
                onClick={() => router.push(`/editor/${doc.id}`)}
                className="bg-[var(--bg-card)] p-5 rounded-xl border border-[var(--border)] hover:border-blue-500/40 hover:shadow-lg hover:shadow-blue-900/10 transition-all duration-200 cursor-pointer group flex flex-col"
              >
                <h2 className="text-base font-semibold text-[var(--text-primary)] mb-2 truncate group-hover:text-blue-500 transition-colors">
                  {doc.title}
                </h2>

                <p className="text-sm text-[var(--text-secondary)] mb-auto line-clamp-2 leading-relaxed min-h-[2.5rem]">
                  {getPreviewText(doc)}
                </p>

                <div className="flex items-center justify-between text-xs text-[var(--text-muted)] mt-4 pt-4 border-t border-[var(--border)]">
                  <div className="flex items-center gap-2 min-w-0">
                    <span className="truncate max-w-[120px]">
                      {doc.user.name}
                    </span>
                    <span className="shrink-0">
                      {formatDate(doc.updated_at)}
                    </span>
                  </div>

                  <button
                    onClick={(e) => handleDelete(doc.id, e)}
                    disabled={deletingId === doc.id}
                    title="Move to trash"
                    className="shrink-0 ml-2 flex items-center justify-center w-6 h-6 rounded text-[var(--text-muted)] hover:text-red-400 hover:bg-red-500/10 opacity-0 group-hover:opacity-100 transition-all disabled:opacity-40"
                  >
                    {deletingId === doc.id ? (
                      <svg
                        className="w-3 h-3 animate-spin"
                        fill="none"
                        viewBox="0 0 24 24"
                      >
                        <circle
                          className="opacity-25"
                          cx="12"
                          cy="12"
                          r="10"
                          stroke="currentColor"
                          strokeWidth="4"
                        />
                        <path
                          className="opacity-75"
                          fill="currentColor"
                          d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"
                        />
                      </svg>
                    ) : (
                      <svg
                        className="w-3.5 h-3.5"
                        fill="none"
                        viewBox="0 0 24 24"
                        stroke="currentColor"
                        strokeWidth={2}
                      >
                        <path
                          strokeLinecap="round"
                          strokeLinejoin="round"
                          d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
                        />
                      </svg>
                    )}
                  </button>
                </div>
              </div>
            ))}
          </div>
        )}

        {/* ── Trash panel ─────────────────────────────────────────── */}
        {trashOpen && (
          <div>
            <h2 className="text-lg font-semibold text-[var(--text-primary)] mb-4 flex items-center gap-2">
              <svg
                className="w-5 h-5 text-red-400"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
                strokeWidth={2}
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
                />
              </svg>
              Trash
              <span className="text-sm font-normal text-[var(--text-muted)]">
                — documents are permanently deleted after 30 days
              </span>
            </h2>

            {trashLoading && (
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-5">
                {Array.from({ length: 3 }).map((_, i) => (
                  <DocumentCardSkeleton key={i} />
                ))}
              </div>
            )}

            {!trashLoading && trashDocs.length === 0 && (
              <p className="text-sm text-[var(--text-muted)] py-12 text-center">
                Trash is empty.
              </p>
            )}

            {!trashLoading && trashDocs.length > 0 && (
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-5">
                {trashDocs.map((doc) => (
                  <div
                    key={doc.id}
                    className="bg-[var(--bg-card)] p-5 rounded-xl border border-red-500/20 flex flex-col opacity-75"
                  >
                    <h3 className="text-base font-semibold text-[var(--text-primary)] mb-2 truncate line-through decoration-[var(--text-muted)]">
                      {doc.title}
                    </h3>
                    <p className="text-sm text-[var(--text-secondary)] mb-auto line-clamp-2 leading-relaxed min-h-[2.5rem]">
                      {getPreviewText(doc)}
                    </p>
                    <div className="flex items-center justify-between mt-4 pt-4 border-t border-[var(--border)]">
                      <span className="text-xs text-[var(--text-muted)]">
                        {formatDate(doc.updated_at)}
                      </span>
                      <button
                        onClick={() => handleRestore(doc.id)}
                        className="text-xs text-blue-500 hover:text-blue-400 font-medium transition-colors px-2 py-1 rounded hover:bg-blue-500/10"
                      >
                        Restore
                      </button>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
}
