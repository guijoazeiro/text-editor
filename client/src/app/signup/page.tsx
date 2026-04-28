"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { authAPI } from "@/lib/api";
import { useAuthStore } from "@/store/authStore";
import { useThemeStore } from "@/store/themeStore";
import { Input } from "@/components/ui/Input";

function SunIcon() {
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
        d="M12 3v1m0 16v1m9-9h-1M4 12H3m15.364-6.364l-.707.707M6.343 17.657l-.707.707m12.728 0l-.707-.707M6.343 6.343l-.707-.707M12 7a5 5 0 100 10A5 5 0 0012 7z"
      />
    </svg>
  );
}

function MoonIcon() {
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
        d="M20.354 15.354A9 9 0 018.646 3.646 9.003 9.003 0 0012 21a9.003 9.003 0 008.354-5.646z"
      />
    </svg>
  );
}

const highlights = [
  "Collaborative editing with Yjs CRDT",
  "Real-time cursors & presence",
  "Version history & one-click restore",
  "Granular permission control",
];

export default function SignupPage() {
  const router = useRouter();
  const login = useAuthStore((s) => s.login);
  const { theme, toggle } = useThemeStore();

  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    setLoading(true);
    try {
      const response = await authAPI.signup({ name, email, password });
      const { token, user } = response.data.data;
      login(token, user);
      router.push("/dashboard");
    } catch (err: unknown) {
      const msg =
        (err as { response?: { data?: { message?: string } } })?.response?.data
          ?.message ?? "Could not create account";
      setError(msg);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center p-4 lg:p-8">
      <div className="w-full max-w-5xl flex rounded-3xl overflow-hidden shadow-2xl shadow-black/20 border border-[var(--border)] min-h-[600px]">
        <div className="hidden lg:flex lg:w-1/2 flex-col justify-between bg-slate-950 p-10 relative overflow-hidden">
          <div
            className="absolute inset-0 opacity-[0.03]"
            style={{
              backgroundImage:
                "radial-gradient(circle, #fff 1px, transparent 1px)",
              backgroundSize: "28px 28px",
            }}
          />

          <div className="relative z-10">
            <span className="text-lg font-bold text-white">Docs Editor</span>
          </div>

          <div className="relative z-10 space-y-10">
            <div>
              <h2 className="text-4xl font-bold text-white leading-snug">
                Start writing
                <br />
                in seconds.
              </h2>
              <p className="mt-3 text-slate-400 text-base leading-relaxed">
                Create your free account and invite your team right away.
              </p>
            </div>

            <ul className="space-y-3">
              {highlights.map((h) => (
                <li
                  key={h}
                  className="flex items-center gap-3 text-sm text-slate-300"
                >
                  <svg
                    className="w-4 h-4 text-blue-400 shrink-0"
                    fill="none"
                    viewBox="0 0 24 24"
                    stroke="currentColor"
                    strokeWidth={2}
                  >
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      d="M5 13l4 4L19 7"
                    />
                  </svg>
                  {h}
                </li>
              ))}
            </ul>
          </div>

          <div className="relative z-10">
            <p className="text-xs text-slate-600">
              Powered by Yjs CRDT · End-to-end sync
            </p>
          </div>
        </div>

        <div className="flex-1 flex flex-col bg-[var(--bg-base)]">
          <div className="flex items-center justify-between px-8 py-4 border-b border-[var(--border)]">
            <span className="lg:hidden text-base font-bold text-blue-500">
              Docs Editor
            </span>
            <span className="hidden lg:block" />

            <div className="flex items-center gap-3">
              <button
                onClick={toggle}
                title={
                  theme === "dark"
                    ? "Switch to light mode"
                    : "Switch to dark mode"
                }
                className="w-8 h-8 flex items-center justify-center rounded-lg text-[var(--text-secondary)] hover:text-[var(--text-primary)] hover:bg-black/5 dark:hover:bg-white/5 transition-all"
              >
                {theme === "dark" ? <SunIcon /> : <MoonIcon />}
              </button>

              <Link
                href="/login"
                className="text-sm font-medium text-[var(--text-secondary)] hover:text-[var(--text-primary)] transition-colors"
              >
                Sign in
              </Link>
            </div>
          </div>

          <div className="flex-1 flex items-center justify-center px-8 py-12">
            <div className="w-full max-w-sm">
              <div className="mb-8">
                <h1 className="text-2xl font-bold text-[var(--text-primary)]">
                  Create account
                </h1>
                <p className="text-sm text-[var(--text-secondary)] mt-1">
                  Free forever. No credit card required.
                </p>
              </div>

              <form onSubmit={handleSubmit} className="space-y-5">
                {error && (
                  <div className="bg-red-500/10 border border-red-500/30 text-red-400 px-4 py-3 rounded-xl text-sm">
                    {error}
                  </div>
                )}

                <Input
                  id="name"
                  label="Full name"
                  type="text"
                  required
                  autoComplete="name"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  placeholder="John Doe"
                />

                <Input
                  id="email"
                  label="Email"
                  type="email"
                  required
                  autoComplete="email"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  placeholder="you@example.com"
                />

                <div>
                  <Input
                    id="password"
                    label="Password"
                    type="password"
                    required
                    minLength={6}
                    autoComplete="new-password"
                    showPasswordToggle
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    placeholder="••••••••"
                  />
                  <p className="text-xs text-[var(--text-muted)] mt-1.5 pl-0.5">
                    Minimum 6 characters
                  </p>
                </div>

                <button
                  type="submit"
                  disabled={loading}
                  className="w-full bg-blue-600 hover:bg-blue-500 text-white py-2.5 rounded-xl font-medium text-sm transition-all disabled:opacity-40 disabled:cursor-not-allowed"
                >
                  {loading ? "Creating account…" : "Create account"}
                </button>
              </form>

              <p className="mt-6 text-sm text-center text-[var(--text-secondary)]">
                Already have an account?{" "}
                <Link
                  href="/login"
                  className="text-blue-500 hover:text-blue-400 font-medium transition-colors"
                >
                  Sign in
                </Link>
              </p>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
