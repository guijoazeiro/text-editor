"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { useAuthStore } from "@/store/authStore";
import { documentsAPI } from "@/lib/api";
import Navbar from "@/components/Navbar";

interface Document {
  id: string;
  title: string;
  content: string;
  content_format: string;
  created_at: string;
  updated_at: string;
  user: { name: string };
}

function getPreviewText(doc: Document): string {
  if (!doc.content) return "No content";

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
          return node.content.map(extractText).join(" ");
        }
        return "";
      };
      return extractText(json).slice(0, 120) || "No content";
    } catch {
      return "No content";
    }
  }

  return doc.content.slice(0, 120);
}

export default function DashboardPage() {
  const router = useRouter();
  const { isAuthenticated, initialize } = useAuthStore();
  const [documents, setDocuments] = useState<Document[]>([]);
  const [loading, setLoading] = useState(true);
  const [creating, setCreating] = useState(false);

  useEffect(() => {
    initialize();
    if (!isAuthenticated) {
      router.push("/login");
      return;
    }
    fetchDocuments();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isAuthenticated]);

  const fetchDocuments = async () => {
    try {
      const response = await documentsAPI.list();
      setDocuments(response.data.data);
    } catch (err) {
      console.error("Failed to fetch documents:", err);
    } finally {
      setLoading(false);
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
    } catch (err) {
      console.error("Failed to create document:", err);
      setCreating(false);
    }
  };

  const formatDate = (date: string) =>
    new Date(date).toLocaleDateString("en-US", {
      month: "short",
      day: "numeric",
      year: "numeric",
    });

  if (loading) {
    return (
      <div className="min-h-screen">
        <Navbar />
        <div className="flex items-center justify-center h-[calc(100vh-4rem)]">
          <div className="text-[var(--text-secondary)] animate-pulse">Loading...</div>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen">
      <Navbar />
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <div className="flex justify-between items-center mb-8">
          <h2 className="text-2xl font-bold text-[var(--text-primary)]">My Documents</h2>
          <button
            onClick={createDocument}
            disabled={creating}
            className="px-5 py-2 bg-blue-600 text-white rounded-lg text-sm font-medium hover:bg-blue-500 transition disabled:opacity-40 shadow-sm shadow-blue-900/30"
          >
            {creating ? "Creating..." : "+ New Document"}
          </button>
        </div>

        {documents.length === 0 ? (
          <div className="text-center py-24">
            <div className="text-slate-700 mb-4">
              <svg
                className="w-14 h-14 mx-auto"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={1.5}
                  d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"
                />
              </svg>
            </div>
            <p className="text-[var(--text-secondary)] text-base mb-4">No documents yet</p>
            <button
              onClick={createDocument}
              className="text-blue-400 hover:text-blue-300 font-medium text-sm transition"
            >
              Create your first document
            </button>
          </div>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-5">
            {documents.map((doc) => (
              <div
                key={doc.id}
                onClick={() => router.push(`/editor/${doc.id}`)}
                className="bg-[var(--bg-card)] p-5 rounded-xl border border-[var(--border)] hover:border-blue-500/40 hover:shadow-lg hover:shadow-blue-900/10 transition-all duration-200 cursor-pointer group"
              >
                <h3 className="text-base font-semibold text-[var(--text-primary)] mb-2 truncate group-hover:text-blue-500 transition-colors">
                  {doc.title}
                </h3>
                <p className="text-sm text-[var(--text-secondary)] mb-4 line-clamp-2 leading-relaxed">
                  {getPreviewText(doc)}
                </p>
                <div className="flex items-center justify-between text-xs text-[var(--text-muted)]">
                  <span>{doc.user.name}</span>
                  <span>{formatDate(doc.updated_at)}</span>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
