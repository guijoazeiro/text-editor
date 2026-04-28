import { create } from "zustand";

export type ToastVariant = "success" | "error" | "info" | "warning";

export interface Toast {
  id: string;
  message: string;
  variant: ToastVariant;
  duration?: number;
}

interface ToastState {
  toasts: Toast[];
  add: (toast: Omit<Toast, "id">) => void;
  remove: (id: string) => void;
  success: (message: string, duration?: number) => void;
  error: (message: string, duration?: number) => void;
  info: (message: string, duration?: number) => void;
  warning: (message: string, duration?: number) => void;
}

export const useToastStore = create<ToastState>((set) => ({
  toasts: [],

  add: (toast) => {
    const id = `toast-${Date.now()}-${Math.random().toString(36).slice(2)}`;
    set((state) => ({ toasts: [...state.toasts, { ...toast, id }] }));
    setTimeout(() => {
      set((state) => ({ toasts: state.toasts.filter((t) => t.id !== id) }));
    }, toast.duration ?? 4000);
  },

  remove: (id) =>
    set((state) => ({ toasts: state.toasts.filter((t) => t.id !== id) })),

  success: (message, duration) =>
    useToastStore.getState().add({ message, variant: "success", duration }),

  error: (message, duration) =>
    useToastStore.getState().add({ message, variant: "error", duration }),

  info: (message, duration) =>
    useToastStore.getState().add({ message, variant: "info", duration }),

  warning: (message, duration) =>
    useToastStore.getState().add({ message, variant: "warning", duration }),
}));
