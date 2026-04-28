"use client";

import { useCallback, useEffect, useState } from "react";
import { collaboratorsAPI, shareLinksAPI } from "@/lib/api";
import { useToastStore } from "@/store/toastStore";
import type { Collaborator } from "@/types";

interface ShareLink {
  token: string;
  permission: "viewer" | "editor";
  url: string;
}

function CopyIcon() {
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
        d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"
      />
    </svg>
  );
}

function TrashIcon() {
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
        d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
      />
    </svg>
  );
}

function ShareIcon() {
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
        d="M8.684 13.342C8.886 12.938 9 12.482 9 12c0-.482-.114-.938-.316-1.342m0 2.684a3 3 0 110-2.684m0 2.684l6.632 3.316m-6.632-6l6.632-3.316m0 0a3 3 0 105.367-2.684 3 3 0 00-5.367 2.684zm0 9.316a3 3 0 105.368 2.684 3 3 0 00-5.368-2.684z"
      />
    </svg>
  );
}

function Avatar({ name }: { name: string }) {
  const initials = name
    .split(" ")
    .map((n) => n[0])
    .slice(0, 2)
    .join("")
    .toUpperCase();
  return (
    <div className="w-8 h-8 rounded-full bg-blue-600 flex items-center justify-center text-white text-xs font-bold shrink-0">
      {initials}
    </div>
  );
}

const PERMISSION_LABELS: Record<string, string> = {
  owner: "Owner",
  editor: "Editor",
  viewer: "Viewer",
};

interface Props {
  open: boolean;
  onClose: () => void;
  documentId: string;
  isOwner: boolean;
}

export function CollaboratorsModal({
  open,
  onClose,
  documentId,
  isOwner,
}: Props) {
  const toast = useToastStore();

  const [collaborators, setCollaborators] = useState<Collaborator[]>([]);
  const [loading, setLoading] = useState(false);
  const [shareLink, setShareLink] = useState<ShareLink | null>(null);

  const [inviteEmail, setInviteEmail] = useState("");
  const [invitePermission, setInvitePermission] = useState<"editor" | "viewer">(
    "editor",
  );
  const [inviting, setInviting] = useState(false);

  const [generatingLink, setGeneratingLink] = useState(false);
  const [linkPermission, setLinkPermission] = useState<"viewer" | "editor">(
    "viewer",
  );

  const fetchCollaborators = useCallback(async () => {
    setLoading(true);
    try {
      const res = await collaboratorsAPI.list(documentId);
      setCollaborators(res.data.data ?? []);
    } catch {
      toast.error("Failed to load collaborators");
    } finally {
      setLoading(false);
    }
  }, [documentId, toast]);

  useEffect(() => {
    if (open) fetchCollaborators();
  }, [open, fetchCollaborators]);

  useEffect(() => {
    if (!open) return;
    const h = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    document.addEventListener("keydown", h);
    return () => document.removeEventListener("keydown", h);
  }, [open, onClose]);

  const handleInvite = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!inviteEmail.trim()) return;
    setInviting(true);
    try {
      await collaboratorsAPI.add(documentId, {
        email: inviteEmail.trim(),
        permission: invitePermission,
      });
      toast.success(`Invite sent to ${inviteEmail.trim()}`);
      setInviteEmail("");
      await fetchCollaborators();
    } catch {
      toast.error("Failed to invite collaborator");
    } finally {
      setInviting(false);
    }
  };

  const handlePermissionChange = async (userId: string, permission: string) => {
    try {
      await collaboratorsAPI.update(documentId, userId, { permission });
      setCollaborators((prev) =>
        prev.map((c) => (c.user_id === userId ? { ...c, permission } : c)),
      );
      toast.success("Permission updated");
    } catch {
      toast.error("Failed to update permission");
    }
  };

  /* remove ----------------------------------------------------------------- */
  const handleRemove = async (userId: string, name: string) => {
    try {
      await collaboratorsAPI.remove(documentId, userId);
      setCollaborators((prev) => prev.filter((c) => c.user_id !== userId));
      toast.success(`Removed ${name}`);
    } catch {
      toast.error("Failed to remove collaborator");
    }
  };

  /* share link ------------------------------------------------------------- */
  const handleGenerateLink = async () => {
    setGeneratingLink(true);
    try {
      const res = await shareLinksAPI.create(documentId, {
        permission: linkPermission,
      });
      const data = res.data.data;
      const url = `${window.location.origin}/shared/${data.token}`;
      setShareLink({ token: data.token, permission: linkPermission, url });
      toast.success("Share link created");
    } catch {
      toast.error("Failed to create share link");
    } finally {
      setGeneratingLink(false);
    }
  };

  const handleCopyLink = () => {
    if (!shareLink) return;
    navigator.clipboard.writeText(shareLink.url);
    toast.success("Link copied to clipboard");
  };

  const handleRevokeLink = async () => {
    try {
      await shareLinksAPI.remove(documentId);
      setShareLink(null);
      toast.success("Share link revoked");
    } catch {
      toast.error("Failed to revoke link");
    }
  };

  if (!open) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50 backdrop-blur-sm">
      <div className="w-full max-w-lg bg-[var(--bg-card)] rounded-2xl border border-[var(--border)] shadow-2xl shadow-black/20 overflow-hidden">
        <div className="flex items-center justify-between px-6 py-5 border-b border-[var(--border)]">
          <div className="flex items-center gap-2 text-[var(--text-primary)]">
            <ShareIcon />
            <h2 className="text-base font-semibold">Share document</h2>
          </div>
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

        <div className="p-6 space-y-6 max-h-[70vh] overflow-y-auto">
          {isOwner && (
            <section>
              <h3 className="text-xs font-semibold uppercase tracking-wider text-[var(--text-muted)] mb-3">
                Invite people
              </h3>
              <form onSubmit={handleInvite} className="flex gap-2">
                <input
                  type="email"
                  required
                  placeholder="colleague@example.com"
                  value={inviteEmail}
                  onChange={(e) => setInviteEmail(e.target.value)}
                  className="flex-1 min-w-0 px-3 py-2 text-sm rounded-lg border border-[var(--border)] bg-[var(--bg-input)] text-[var(--text-primary)] placeholder:text-[var(--text-placeholder)] outline-none focus:border-blue-500 focus:ring-2 focus:ring-blue-500/20 transition-all"
                />
                <select
                  value={invitePermission}
                  onChange={(e) =>
                    setInvitePermission(e.target.value as "editor" | "viewer")
                  }
                  className="px-2 py-2 text-sm rounded-lg border border-[var(--border)] bg-[var(--bg-input)] text-[var(--text-primary)] outline-none focus:border-blue-500 transition-all"
                >
                  <option value="editor">Editor</option>
                  <option value="viewer">Viewer</option>
                </select>
                <button
                  type="submit"
                  disabled={inviting}
                  className="px-4 py-2 bg-blue-600 text-white text-sm font-medium rounded-lg hover:bg-blue-500 transition-colors disabled:opacity-40 shrink-0"
                >
                  {inviting ? "…" : "Invite"}
                </button>
              </form>
            </section>
          )}

          <section>
            <h3 className="text-xs font-semibold uppercase tracking-wider text-[var(--text-muted)] mb-3">
              People with access
            </h3>

            {loading && (
              <div className="space-y-3">
                {[1, 2].map((i) => (
                  <div
                    key={i}
                    className="flex items-center gap-3 animate-pulse"
                  >
                    <div className="w-8 h-8 rounded-full bg-[var(--border)]" />
                    <div className="flex-1 space-y-1.5">
                      <div className="h-3 w-1/3 rounded bg-[var(--border)]" />
                      <div className="h-2.5 w-1/2 rounded bg-[var(--border)]" />
                    </div>
                  </div>
                ))}
              </div>
            )}

            {!loading && collaborators.length === 0 && (
              <p className="text-sm text-[var(--text-muted)] py-2">
                No collaborators yet. Invite someone above.
              </p>
            )}

            {!loading && (
              <ul className="space-y-2">
                {collaborators.map((c) => (
                  <li key={c.user_id} className="flex items-center gap-3 py-1">
                    <Avatar name={c.user?.name ?? "?"} />
                    <div className="flex-1 min-w-0">
                      <p className="text-sm font-medium text-[var(--text-primary)] truncate">
                        {c.user?.name ?? "Unknown"}
                      </p>
                      <p className="text-xs text-[var(--text-muted)] truncate">
                        {c.user?.email}
                      </p>
                    </div>

                    {isOwner && c.permission !== "owner" ? (
                      <select
                        value={c.permission}
                        onChange={(e) =>
                          handlePermissionChange(c.user_id, e.target.value)
                        }
                        className="text-xs px-2 py-1 rounded-lg border border-[var(--border)] bg-[var(--bg-input)] text-[var(--text-secondary)] outline-none focus:border-blue-500 transition-all"
                      >
                        <option value="editor">Editor</option>
                        <option value="viewer">Viewer</option>
                      </select>
                    ) : (
                      <span className="text-xs text-[var(--text-muted)] px-2 py-1 rounded-lg bg-[var(--bg-base)] border border-[var(--border)]">
                        {PERMISSION_LABELS[c.permission] ?? c.permission}
                      </span>
                    )}

                    {isOwner && c.permission !== "owner" && (
                      <button
                        onClick={() =>
                          handleRemove(c.user_id, c.user?.name ?? "user")
                        }
                        title="Remove access"
                        className="text-[var(--text-muted)] hover:text-red-400 transition-colors shrink-0"
                      >
                        <TrashIcon />
                      </button>
                    )}
                  </li>
                ))}
              </ul>
            )}
          </section>

          {isOwner && (
            <section className="border-t border-[var(--border)] pt-5">
              <h3 className="text-xs font-semibold uppercase tracking-wider text-[var(--text-muted)] mb-3">
                Public link
              </h3>

              {shareLink ? (
                <div className="space-y-3">
                  <div className="flex items-center gap-2 px-3 py-2 rounded-lg bg-[var(--bg-base)] border border-[var(--border)]">
                    <span className="flex-1 text-xs text-[var(--text-secondary)] truncate font-mono">
                      {shareLink.url}
                    </span>
                    <button
                      onClick={handleCopyLink}
                      title="Copy link"
                      className="text-[var(--text-muted)] hover:text-blue-500 transition-colors shrink-0"
                    >
                      <CopyIcon />
                    </button>
                  </div>
                  <div className="flex items-center justify-between">
                    <span className="text-xs text-[var(--text-muted)]">
                      Anyone with the link can{" "}
                      <span className="font-medium text-[var(--text-secondary)]">
                        {shareLink.permission}
                      </span>
                    </span>
                    <button
                      onClick={handleRevokeLink}
                      className="text-xs text-red-400 hover:text-red-300 transition-colors flex items-center gap-1"
                    >
                      <TrashIcon />
                      Revoke
                    </button>
                  </div>
                </div>
              ) : (
                <div className="flex items-center gap-2">
                  <select
                    value={linkPermission}
                    onChange={(e) =>
                      setLinkPermission(e.target.value as "viewer" | "editor")
                    }
                    className="text-sm px-3 py-2 rounded-lg border border-[var(--border)] bg-[var(--bg-input)] text-[var(--text-primary)] outline-none focus:border-blue-500 transition-all"
                  >
                    <option value="viewer">Viewer</option>
                    <option value="editor">Editor</option>
                  </select>
                  <button
                    onClick={handleGenerateLink}
                    disabled={generatingLink}
                    className="flex-1 py-2 px-4 border border-[var(--border)] rounded-lg text-sm text-[var(--text-secondary)] hover:border-blue-500/50 hover:text-blue-500 transition-all disabled:opacity-40"
                  >
                    {generatingLink ? "Generating…" : "Generate link"}
                  </button>
                </div>
              )}
            </section>
          )}
        </div>
      </div>
    </div>
  );
}
