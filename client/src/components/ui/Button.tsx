import type { ButtonHTMLAttributes } from "react";
import { cn } from "@/lib/utils";

type Variant = "primary" | "secondary" | "ghost" | "danger";
type Size = "sm" | "md" | "lg";

interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: Variant;
  size?: Size;
  loading?: boolean;
}

const variantStyles: Record<Variant, string> = {
  primary:
    "bg-blue-600 text-white hover:bg-blue-500 focus-visible:ring-blue-500 shadow-sm shadow-blue-900/30",
  secondary:
    "bg-white/5 text-slate-200 border border-white/10 hover:bg-white/10 focus-visible:ring-slate-400",
  ghost:
    "text-slate-400 hover:text-slate-200 hover:bg-white/5 focus-visible:ring-slate-400",
  danger:
    "bg-red-600/20 text-red-400 border border-red-500/30 hover:bg-red-600/30 focus-visible:ring-red-500",
};

const sizeStyles: Record<Size, string> = {
  sm: "px-3 py-1.5 text-xs gap-1.5",
  md: "px-4 py-2 text-sm gap-2",
  lg: "px-5 py-2.5 text-base gap-2",
};

export function Button({
  variant = "primary",
  size = "md",
  loading = false,
  disabled,
  className,
  children,
  ...props
}: ButtonProps) {
  return (
    <button
      disabled={disabled || loading}
      className={cn(
        "inline-flex items-center justify-center rounded-lg font-medium",
        "transition-all duration-150 outline-none focus-visible:ring-2 focus-visible:ring-offset-2 focus-visible:ring-offset-slate-900",
        "disabled:opacity-40 disabled:cursor-not-allowed",
        variantStyles[variant],
        sizeStyles[size],
        className,
      )}
      {...props}
    >
      {loading && (
        <svg className="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
          <circle
            className="opacity-25"
            cx="12"
            cy="12"
            r="10"
            stroke="currentColor"
            strokeWidth="4"
          />
          <path
            className="opacity-75"
            fill="currentColor"
            d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"
          />
        </svg>
      )}
      {children}
    </button>
  );
}
