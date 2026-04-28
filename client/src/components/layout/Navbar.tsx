"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useAuthStore } from "@/store/authStore";
import { useThemeStore } from "@/store/themeStore";

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

export default function Navbar() {
  const router = useRouter();
  const { user, logout } = useAuthStore();
  const { theme, toggle } = useThemeStore();

  const handleLogout = () => {
    logout();
    router.push("/login");
  };

  return (
    <nav className="bg-[var(--bg-nav)] backdrop-blur-md border-b border-[var(--border)] sticky top-0 z-40 transition-colors duration-200">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="flex justify-between items-center h-14">
          <Link
            href="/dashboard"
            className="text-xl font-bold bg-gradient-to-r from-blue-500 to-violet-500 bg-clip-text text-transparent hover:opacity-80 transition-opacity"
          >
            Docs Editor
          </Link>

          <div className="flex items-center gap-2">
            {/* Theme toggle */}
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

            {user && (
              <>
                <span className="text-sm text-[var(--text-secondary)] hidden sm:block px-1">
                  {user.name}
                </span>
                <button
                  onClick={handleLogout}
                  className="text-sm text-[var(--text-secondary)] hover:text-[var(--text-primary)] transition-colors px-3 py-1.5 rounded-lg hover:bg-black/5 dark:hover:bg-white/5"
                >
                  Logout
                </button>
              </>
            )}
          </div>
        </div>
      </div>
    </nav>
  );
}
