"use client";

import { useEffect } from "react";
import { useThemeStore } from "@/store/themeStore";

/**
 * Reads the persisted theme from the store and applies the
 * corresponding class to <html> on every render/change.
 * Must be mounted inside the root layout as a client component.
 */
export function ThemeProvider({ children }: { children: React.ReactNode }) {
  const theme = useThemeStore((s) => s.theme);

  useEffect(() => {
    const root = document.documentElement;
    if (theme === "dark") {
      root.classList.add("dark");
    } else {
      root.classList.remove("dark");
    }
  }, [theme]);

  return <>{children}</>;
}
