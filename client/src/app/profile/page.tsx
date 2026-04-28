"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { useAuthStore } from "@/store/authStore";
import { authAPI } from "@/lib/api";
import { useToastStore } from "@/store/toastStore";
import Navbar from "@/components/Navbar";
import type { User } from "@/types";

function Avatar({ name, size = "lg" }: { name: string; size?: "sm" | "lg" }) {
  const initials = name
    .split(" ")
    .map((n) => n[0])
    .slice(0, 2)
    .join("")
    .toUpperCase();

  const cls = size === "lg" ? "w-20 h-20 text-2xl" : "w-10 h-10 text-sm";

  return (
    <div
      className={`${cls} rounded-full bg-blue-600 flex items-center justify-center text-white font-bold`}
    >
      {initials}
    </div>
  );
}

export default function ProfilePage() {
  const router = useRouter();
  const { user, isAuthenticated, isHydrated, initialize, login, logout } =
    useAuthStore();
  const toast = useToastStore();

  const [profile, setProfile] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);
  const [editing, setEditing] = useState(false);
  const [name, setName] = useState("");
  const [saving, setSaving] = useState(false);

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
    fetchProfile();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isAuthenticated, isHydrated]);

  const fetchProfile = async () => {
    try {
      const res = await authAPI.me();
      const u = res.data.data;
      setProfile(u);
      setName(u.name);
    } catch {
      toast.error("Failed to load profile");
    } finally {
      setLoading(false);
    }
  };

  const handleSave = async () => {
    if (!name.trim()) return;
    setSaving(true);
    try {
      if (user) login(useAuthStore.getState().token!, { ...user, name });
      setProfile((p) => (p ? { ...p, name } : p));
      setEditing(false);
      toast.success("Name updated");
    } catch {
      toast.error("Failed to update name");
    } finally {
      setSaving(false);
    }
  };

  const handleLogout = () => {
    logout();
    router.push("/login");
  };

  if (loading) {
    return (
      <div className="min-h-screen">
        <Navbar />
        <div className="flex items-center justify-center h-[calc(100vh-4rem)]">
          <div className="text-[var(--text-secondary)] animate-pulse">
            Loading profile…
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen">
      <Navbar />
      <div className="max-w-2xl mx-auto px-4 sm:px-6 py-12">
        <div className="bg-[var(--bg-card)] rounded-2xl border border-[var(--border)] overflow-hidden shadow-xl shadow-black/10">
          <div className="h-24 bg-blue-600/15" />

          <div className="px-8 pb-8">
            <div className="-mt-10 mb-4">
              <Avatar name={profile?.name ?? "?"} size="lg" />
            </div>

            {editing ? (
              <div className="flex items-center gap-3 mb-1">
                <input
                  type="text"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  className="text-xl font-bold bg-transparent border-b-2 border-blue-500 outline-none text-[var(--text-primary)] w-full max-w-xs pb-0.5"
                  autoFocus
                  onKeyDown={(e) => e.key === "Enter" && handleSave()}
                />
                <button
                  onClick={handleSave}
                  disabled={saving}
                  className="px-3 py-1 bg-blue-600 text-white rounded-lg text-sm font-medium hover:bg-blue-500 transition disabled:opacity-40"
                >
                  {saving ? "Saving…" : "Save"}
                </button>
                <button
                  onClick={() => {
                    setEditing(false);
                    setName(profile?.name ?? "");
                  }}
                  className="px-3 py-1 text-sm text-[var(--text-secondary)] hover:text-[var(--text-primary)] transition"
                >
                  Cancel
                </button>
              </div>
            ) : (
              <div className="flex items-center gap-2 mb-1">
                <h1 className="text-xl font-bold text-[var(--text-primary)]">
                  {profile?.name}
                </h1>
                <button
                  onClick={() => setEditing(true)}
                  className="text-[var(--text-muted)] hover:text-[var(--text-secondary)] transition-colors"
                  title="Edit name"
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
                      d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"
                    />
                  </svg>
                </button>
              </div>
            )}

            <p className="text-sm text-[var(--text-secondary)]">
              {profile?.email}
            </p>

            <hr className="border-[var(--border)] my-6" />

            <div className="space-y-4">
              <div className="flex justify-between items-center py-3 px-4 bg-[var(--bg-base)] rounded-xl">
                <span className="text-sm text-[var(--text-secondary)]">
                  Email
                </span>
                <span className="text-sm font-medium text-[var(--text-primary)]">
                  {profile?.email}
                </span>
              </div>
              <div className="flex justify-between items-center py-3 px-4 bg-[var(--bg-base)] rounded-xl">
                <span className="text-sm text-[var(--text-secondary)]">
                  User ID
                </span>
                <span className="text-xs font-mono text-[var(--text-muted)] truncate max-w-[180px]">
                  {profile?.id}
                </span>
              </div>
            </div>

            <hr className="border-[var(--border)] my-6" />

            <button
              onClick={handleLogout}
              className="w-full py-2.5 rounded-xl border border-red-500/30 text-red-400 text-sm font-medium hover:bg-red-500/10 transition-colors"
            >
              Sign out
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
