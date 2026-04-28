import { create } from "zustand";
import { persist, createJSONStorage } from "zustand/middleware";

export type Theme = "dark" | "light";

interface ThemeState {
  theme: Theme;
  toggle: () => void;
  setTheme: (theme: Theme) => void;
}

export const useThemeStore = create<ThemeState>()(
  persist(
    (set) => ({
      theme: "dark",

      toggle: () =>
        set((state) => ({ theme: state.theme === "dark" ? "light" : "dark" })),

      setTheme: (theme) => set({ theme }),
    }),
    {
      name: "theme-preference",
      storage: createJSONStorage(() => localStorage),
    }
  )
);
