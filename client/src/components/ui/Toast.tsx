"use client";

import { useToastStore } from "@/store/toastStore";
import { useEffect, useState } from "react";

const ICONS = {
  success: (
    <svg
      className="w-5 h-5"
      fill="none"
      viewBox="0 0 24 24"
      stroke="currentColor"
    >
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth={2}
        d="M5 13l4 4L19 7"
      />
    </svg>
  ),
  error: (
    <svg
      className="w-5 h-5"
      fill="none"
      viewBox="0 0 24 24"
      stroke="currentColor"
    >
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth={2}
        d="M6 18L18 6M6 6l12 12"
      />
    </svg>
  ),
  info: (
    <svg
      className="w-5 h-5"
      fill="none"
      viewBox="0 0 24 24"
      stroke="currentColor"
    >
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth={2}
        d="M13 16h-1v-4h-1m1-4h.01M12 2a10 10 0 110 20A10 10 0 0112 2z"
      />
    </svg>
  ),
  warning: (
    <svg
      className="w-5 h-5"
      fill="none"
      viewBox="0 0 24 24"
      stroke="currentColor"
    >
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth={2}
        d="M12 9v2m0 4h.01M10.29 3.86L1.82 18a2 2 0 001.71 3h16.94a2 2 0 001.71-3L13.71 3.86a2 2 0 00-3.42 0z"
      />
    </svg>
  ),
};

const STYLES = {
  success: "bg-emerald-500/10 border-emerald-500/30 text-emerald-400",
  error: "bg-red-500/10 border-red-500/30 text-red-400",
  info: "bg-blue-500/10 border-blue-500/30 text-blue-400",
  warning: "bg-amber-500/10 border-amber-500/30 text-amber-400",
};

function ToastItem({
  id,
  message,
  variant,
}: {
  id: string;
  message: string;
  variant: keyof typeof STYLES;
}) {
  const remove = useToastStore((s) => s.remove);
  const [visible, setVisible] = useState(false);

  useEffect(() => {
    requestAnimationFrame(() => setVisible(true));
  }, []);

  const handleRemove = () => {
    setVisible(false);
    setTimeout(() => remove(id), 300);
  };

  return (
    <div
      className={`
        flex items-start gap-3 px-4 py-3 rounded-xl border backdrop-blur-sm
        shadow-lg shadow-black/20 transition-all duration-300
        ${STYLES[variant]}
        ${visible ? "opacity-100 translate-y-0" : "opacity-0 translate-y-2"}
      `}
    >
      <span className="mt-0.5 shrink-0">{ICONS[variant]}</span>
      <p className="text-sm font-medium leading-snug flex-1">{message}</p>
      <button
        onClick={handleRemove}
        className="shrink-0 opacity-60 hover:opacity-100 transition-opacity"
        aria-label="Dismiss"
      >
        <svg
          className="w-4 h-4"
          fill="none"
          viewBox="0 0 24 24"
          stroke="currentColor"
        >
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M6 18L18 6M6 6l12 12"
          />
        </svg>
      </button>
    </div>
  );
}

export function ToastContainer() {
  const toasts = useToastStore((s) => s.toasts);

  return (
    <div className="fixed bottom-6 right-6 z-[9999] flex flex-col gap-2 w-80 pointer-events-none">
      {toasts.map((t) => (
        <div key={t.id} className="pointer-events-auto">
          <ToastItem id={t.id} message={t.message} variant={t.variant} />
        </div>
      ))}
    </div>
  );
}
