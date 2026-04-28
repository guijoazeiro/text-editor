"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { useAuthStore } from "@/store/authStore";
import { authAPI } from "@/lib/api";
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

const features = [
  {
    icon: (
      <svg
        className="w-5 h-5"
        fill="none"
        viewBox="0 0 24 24"
        stroke="currentColor"
        strokeWidth={1.5}
      >
        <path
          strokeLinecap="round"
          strokeLinejoin="round"
          d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0z"
        />
      </svg>
    ),
    title: "Real-time collaboration",
    desc: "Edit together with your team, see cursors live.",
  },
  {
    icon: (
      <svg
        className="w-5 h-5"
        fill="none"
        viewBox="0 0 24 24"
        stroke="currentColor"
        strokeWidth={1.5}
      >
        <path
          strokeLinecap="round"
          strokeLinejoin="round"
          d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"
        />
      </svg>
    ),
    title: "Full version history",
    desc: "Restore any previous version with one click.",
  },
  {
    icon: (
      <svg
        className="w-5 h-5"
        fill="none"
        viewBox="0 0 24 24"
        stroke="currentColor"
        strokeWidth={1.5}
      >
        <path
          strokeLinecap="round"
          strokeLinejoin="round"
          d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z"
        />
      </svg>
    ),
    title: "Permission control",
    desc: "Share as viewer or editor, manage access easily.",
  },
];

export default function LoginPage() {
  const router = useRouter();
  const login = useAuthStore((s) => s.login);
  const { theme, toggle } = useThemeStore();

  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    setLoading(true);
    try {
      const response = await authAPI.login({ email, password });
      const { token, user } = response.data.data;
      login(token, user);
      router.push("/dashboard");
    } catch (err: unknown) {
      const msg =
        (err as { response?: { data?: { message?: string } } })?.response?.data
          ?.message ?? "Invalid email or password";
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
                Write together,
                <br />
                think together.
              </h2>
              <p className="mt-3 text-slate-400 text-base leading-relaxed">
                A collaborative editor built for teams who move fast.
              </p>
            </div>

            <ul className="space-y-6">
              {features.map((f) => (
                <li key={f.title} className="flex items-start gap-4">
                  <div className="mt-0.5 shrink-0 w-9 h-9 rounded-lg bg-white/5 border border-white/8 flex items-center justify-center text-blue-400">
                    {f.icon}
                  </div>
                  <div>
                    <p className="text-sm font-semibold text-white">
                      {f.title}
                    </p>
                    <p className="text-sm text-slate-400 mt-0.5">{f.desc}</p>
                  </div>
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
                href="/signup"
                className="text-sm font-medium text-[var(--text-secondary)] hover:text-[var(--text-primary)] transition-colors"
              >
                Sign up
              </Link>
            </div>
          </div>

          <div className="flex-1 flex items-center justify-center px-8 py-12">
            <div className="w-full max-w-sm">
              <div className="mb-8">
                <h1 className="text-2xl font-bold text-[var(--text-primary)]">
                  Sign in
                </h1>
                <p className="text-sm text-[var(--text-secondary)] mt-1">
                  Welcome back. Enter your credentials to continue.
                </p>
              </div>

              <form onSubmit={handleSubmit} className="space-y-5">
                {error && (
                  <div className="bg-red-500/10 border border-red-500/30 text-red-400 px-4 py-3 rounded-xl text-sm">
                    {error}
                  </div>
                )}

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

                <Input
                  id="password"
                  label="Password"
                  type="password"
                  required
                  autoComplete="current-password"
                  showPasswordToggle
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  placeholder="••••••••"
                />

                <div className="flex items-center justify-end">
                  <a
                    href="#"
                    className="text-xs text-blue-500 hover:text-blue-400 transition-colors"
                  >
                    Forgot password?
                  </a>
                </div>

                <button
                  type="submit"
                  disabled={loading}
                  className="w-full bg-blue-600 hover:bg-blue-500 text-white py-2.5 rounded-xl font-medium text-sm transition-all disabled:opacity-40 disabled:cursor-not-allowed"
                >
                  {loading ? "Signing in…" : "Sign in"}
                </button>
              </form>

              <p className="mt-6 text-sm text-center text-[var(--text-secondary)]">
                Don&apos;t have an account?{" "}
                <Link
                  href="/signup"
                  className="text-blue-500 hover:text-blue-400 font-medium transition-colors"
                >
                  Sign up for free
                </Link>
              </p>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
