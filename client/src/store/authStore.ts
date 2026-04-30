import { create } from "zustand";
import { persist, createJSONStorage } from "zustand/middleware";
import type { User } from "@/types";

interface AuthState {
  user: User | null;
  token: string | null;
  isAuthenticated: boolean;
  isHydrated: boolean;
  login: (token: string, user: User) => void;
  logout: () => void;
  initialize: () => void;
  setHydrated: (hydrated: boolean) => void;
  setToken: (token: string) => void;
}

function getTokenExpiry(token: string): number | null {
  try {
    const payload = token.split(".")[1];
    const decoded = JSON.parse(atob(payload));
    return typeof decoded.exp === "number" ? decoded.exp : null;
  } catch {
    return null;
  }
}

function isTokenValid(token: string | null): boolean {
  if (!token) return false;
  const exp = getTokenExpiry(token);
  if (exp === null) return true;
  return Date.now() / 1000 < exp - 30;
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set, get) => ({
      user: null,
      token: null,
      isAuthenticated: false,
      isHydrated: false,

      login: (token: string, user: User) => {
        set({ token, user, isAuthenticated: true });
      },

      logout: () => {
        set({ token: null, user: null, isAuthenticated: false });
      },

      initialize: () => {
        const { token, user } = get();
        if (token && user && isTokenValid(token)) {
          set({ isAuthenticated: true });
        } else if (token && !isTokenValid(token)) {
          set({ token: null, user: null, isAuthenticated: false });
        }
      },

      setHydrated: (hydrated: boolean) => {
        set({ isHydrated: hydrated });
      },

      setToken: (token: string) => {
        set({ token, isAuthenticated: true });
      },
    }),
    {
      name: "auth-storage",
      storage: createJSONStorage(() => localStorage),
      onRehydrateStorage: () => (state) => {
        state?.setHydrated(true);
        if (state?.token && state?.user && isTokenValid(state.token)) {
          state.isAuthenticated = true;
        } else if (state?.token && !isTokenValid(state.token)) {
          state.token = null;
          state.user = null;
          state.isAuthenticated = false;
        }
      },
    },
  ),
);
