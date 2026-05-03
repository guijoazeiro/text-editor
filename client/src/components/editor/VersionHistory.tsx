"use client";

import { useEffect, useRef, useState } from "react";
import type { DocumentVersion } from "@/types";

interface Props {
  open: boolean;
  onClose: () => void;
  versions: DocumentVersion[];
  loading: boolean;
  restoring: boolean;
  currentVersionNumber?: number;
  onRestore: (versionNumber: number) => void;
  pagination?: {
    page: number;
    pages: number;
    total: number;
  };
  onPageChange?: (page: number) => void;
}

function timeAgo(dateStr: string) {
  const diff = Date.now() - new Date(dateStr).getTime();
  const m = Math.floor(diff / 60_000);
  if (m < 1) return "just now";
  if (m < 60) return `${m}m ago`;
  const h = Math.floor(m / 60);
  if (h < 24) return `${h}h ago`;
  const d = Math.floor(h / 24);
  if (d < 30) return `${d}d ago`;
  return new Date(dateStr).toLocaleDateString("en-US", {
    month: "short",
    day: "numeric",
    year: "numeric",
  });
}

function HistoryIcon() {
  return (
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
  );
}

export function VersionHistory({
  open,
  onClose,
  versions,
  loading,
  restoring,
  currentVersionNumber,
  onRestore,
  pagination,
  onPageChange,
}: Props) {
  const [confirmVersion, setConfirmVersion] = useState<DocumentVersion | null>(
    null,
  );
  const panelRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) return;
    const handler = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        if (confirmVersion) setConfirmVersion(null);
        else onClose();
      }
    };
    document.addEventListener("keydown", handler);
    return () => document.removeEventListener("keydown", handler);
  }, [open, onClose, confirmVersion]);

  const handleRestoreConfirm = async () => {
    if (!confirmVersion) return;
    onRestore(confirmVersion.version_number);
    setConfirmVersion(null);
  };

  return (
    <>
      {open && (
        <div
          className="fixed inset-0 z-40 bg-black/30 backdrop-blur-[2px] transition-opacity duration-200"
          onClick={onClose}
        />
      )}

      <div
        ref={panelRef}
        className={`
          fixed top-0 right-0 h-full z-50 w-80
          bg-[var(--bg-card)] border-l border-[var(--border)]
          flex flex-col shadow-2xl shadow-black/20
          transition-transform duration-300 ease-out
          ${open ? "translate-x-0" : "translate-x-full"}
        `}
      >
        <div className="flex items-center justify-between px-5 py-4 border-b border-[var(--border)] shrink-0">
          <div className="flex items-center gap-2 text-[var(--text-primary)]">
            <HistoryIcon />
            <span className="font-semibold text-sm">Version History</span>
          </div>
          <div className="flex items-center gap-2">
            {versions.length > 0 && (
              <span className="text-xs text-[var(--text-muted)] bg-[var(--bg-base)] px-2 py-0.5 rounded-full">
                showing {versions.length}
              </span>
            )}
            <button
              onClick={onClose}
              className="text-[var(--text-muted)] hover:text-[var(--text-primary)] transition-colors rounded-lg p-1 hover:bg-black/5 dark:hover:bg-white/5"
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
                  d="M6 18L18 6M6 6l12 12"
                />
              </svg>
            </button>
          </div>
        </div>

        <div className="flex-1 overflow-y-auto">
          {loading && (
            <div className="flex flex-col gap-3 p-4">
              {Array.from({ length: 5 }).map((_, i) => (
                <div
                  key={i}
                  className="animate-pulse space-y-2 p-3 rounded-xl bg-[var(--bg-base)]"
                >
                  <div className="h-3.5 w-1/3 rounded bg-[var(--border)]" />
                  <div className="h-3 w-2/3 rounded bg-[var(--border)]" />
                </div>
              ))}
            </div>
          )}

          {!loading && versions.length === 0 && (
            <div className="flex flex-col items-center justify-center py-20 px-6 text-center">
              <HistoryIcon />
              <p className="text-sm text-[var(--text-muted)] mt-3">
                No versions yet
              </p>
              <p className="text-xs text-[var(--text-muted)] mt-1">
                Versions are saved automatically as you edit.
              </p>
            </div>
          )}

          {!loading &&
            versions.map((v, index) => {
              const isCurrent =
                index === 0 || v.version_number === currentVersionNumber;

              return (
                <div
                  key={v.id}
                  className={`
                  group flex items-start justify-between gap-3 px-5 py-4
                  border-b border-[var(--border)] last:border-0
                  transition-colors hover:bg-[var(--bg-base)]
                  ${isCurrent ? "bg-blue-500/5" : ""}
                `}
                >
                  <div className="min-w-0 flex-1">
                    <div className="flex items-center gap-2 mb-0.5">
                      <span className="text-xs font-semibold text-[var(--text-primary)]">
                        v{v.version_number}
                      </span>
                      {isCurrent && (
                        <span className="text-[10px] font-medium px-1.5 py-0.5 rounded-full bg-blue-600 text-white">
                          current
                        </span>
                      )}
                    </div>
                    <p className="text-xs text-[var(--text-muted)] truncate">
                      {v.created_by_user?.name ?? "Unknown"}
                    </p>
                    <p className="text-xs text-[var(--text-muted)] mt-0.5">
                      {timeAgo(v.created_at)}
                    </p>
                  </div>

                  {!isCurrent && (
                    <button
                      onClick={() => setConfirmVersion(v)}
                      disabled={restoring}
                      className="shrink-0 opacity-0 group-hover:opacity-100 text-xs font-medium px-2.5 py-1 rounded-lg border border-[var(--border)] text-[var(--text-secondary)] hover:border-blue-500/50 hover:text-blue-500 transition-all disabled:opacity-40 disabled:cursor-not-allowed"
                    >
                      Restore
                    </button>
                  )}
                </div>
              );
            })}
        </div>

        <div className="px-5 py-3 border-t border-[var(--border)] shrink-0">
          {pagination && pagination.pages > 1 ? (
            <div className="flex items-center justify-between">
              <button
                onClick={() => onPageChange?.(pagination.page - 1)}
                disabled={pagination.page <= 1 || loading}
                className="text-xs px-2.5 py-1 rounded-lg border border-[var(--border)] text-[var(--text-secondary)] hover:bg-[var(--bg-base)] transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
              >
                ← Prev
              </button>
              <span className="text-xs text-[var(--text-muted)]">
                {pagination.page} / {pagination.pages}
              </span>
              <button
                onClick={() => onPageChange?.(pagination.page + 1)}
                disabled={pagination.page >= pagination.pages || loading}
                className="text-xs px-2.5 py-1 rounded-lg border border-[var(--border)] text-[var(--text-secondary)] hover:bg-[var(--bg-base)] transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
              >
                Next →
              </button>
            </div>
          ) : (
            <p className="text-xs text-[var(--text-muted)]">
              {pagination?.total ?? versions.length} version{(pagination?.total ?? versions.length) !== 1 ? "s" : ""} total
            </p>
          )}
        </div>
      </div>

      {confirmVersion && (
        <div className="fixed inset-0 z-[60] flex items-center justify-center p-4 bg-black/50 backdrop-blur-sm">
          <div className="w-full max-w-sm bg-[var(--bg-card)] rounded-2xl border border-[var(--border)] p-6 shadow-2xl">
            <h3 className="text-base font-semibold text-[var(--text-primary)] mb-2">
              Restore version {confirmVersion.version_number}?
            </h3>
            <p className="text-sm text-[var(--text-secondary)] mb-6 leading-relaxed">
              The document will revert to this version. All collaborators will
              see the change immediately. This cannot be undone directly, but
              the current version will be preserved in the history.
            </p>
            <div className="flex gap-3">
              <button
                onClick={() => setConfirmVersion(null)}
                className="flex-1 py-2 rounded-xl border border-[var(--border)] text-sm text-[var(--text-secondary)] hover:text-[var(--text-primary)] hover:bg-[var(--bg-base)] transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={handleRestoreConfirm}
                disabled={restoring}
                className="flex-1 py-2 rounded-xl bg-blue-600 text-white text-sm font-medium hover:bg-blue-500 transition-colors disabled:opacity-40"
              >
                {restoring ? "Restoring…" : "Restore"}
              </button>
            </div>
          </div>
        </div>
      )}
    </>
  );
}
