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
      <div className="min-h-screen bg-gray-50">
        <Navbar />
        <div className="flex items-center justify-center h-[calc(100vh-4rem)]">
          <div className="text-gray-600">Loading...</div>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gray-50">
      <Navbar />
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <div className="flex justify-between items-center mb-8">
          <h2 className="text-3xl font-bold text-gray-900">My Documents</h2>
          <button
            onClick={createDocument}
            disabled={creating}
            className="px-6 py-3 bg-[#1479b0] text-white rounded-lg font-medium hover:bg-[#0f5f8d] transition disabled:opacity-50"
          >
            {creating ? "Creating..." : "+ New Document"}
          </button>
        </div>

        {documents.length === 0 ? (
          <div className="text-center py-16">
            <div className="text-gray-400 mb-4">
              <svg
                className="w-16 h-16 mx-auto"
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
            <p className="text-gray-600 text-lg mb-4">No documents yet</p>
            <button
              onClick={createDocument}
              className="text-[#1479b0] hover:underline font-medium"
            >
              Create your first document
            </button>
          </div>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            {documents.map((doc) => (
              <div
                key={doc.id}
                onClick={() => router.push(`/editor/${doc.id}`)}
                className="bg-white p-6 rounded-lg shadow-sm border border-gray-200 hover:shadow-md hover:border-[#1479b0] transition cursor-pointer"
              >
                <h3 className="text-lg font-semibold text-gray-900 mb-2 truncate">
                  {doc.title}
                </h3>
                <p className="text-sm text-gray-600 mb-4 line-clamp-2">
                  {getPreviewText(doc)}
                </p>
                <div className="flex items-center justify-between text-xs text-gray-500">
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
