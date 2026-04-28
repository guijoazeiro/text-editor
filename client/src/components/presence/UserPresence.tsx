"use client";

import { useMemo } from "react";
import type { UserPresence as UserPresenceType } from "@/lib/websocket";
import type { RemoteUser } from "@/types";

interface Props {
  users: UserPresenceType[];
  remoteUsers?: RemoteUser[];
  currentUserId: string;
}

export default function UserPresence({
  users,
  remoteUsers = [],
  currentUserId,
}: Props) {
  const displayUsers = useMemo(() => {
    const map = new Map<
      string,
      { id: string; name: string; color: string; isTyping: boolean }
    >();

    users.forEach((u) => {
      if (u.user_id && u.user_id !== currentUserId && u.online !== false) {
        map.set(u.user_id, {
          id: u.user_id,
          name: u.user_name,
          color: u.color,
          isTyping: false,
        });
      }
    });

    remoteUsers.forEach((ru) => {
      const id = ru.user?.id;
      if (!id || id === currentUserId) return;
      map.set(id, {
        id,
        name: ru.user.name,
        color: ru.user.color,
        isTyping: ru.cursor !== undefined,
      });
    });

    return Array.from(map.values());
  }, [users, remoteUsers, currentUserId]);

  if (displayUsers.length === 0) {
    return (
      <div className="flex items-center space-x-2">
        <span className="text-sm text-slate-500">Only you</span>
      </div>
    );
  }

  const visible = displayUsers.slice(0, 5);
  const overflow = displayUsers.length - 5;

  return (
    <div className="flex items-center space-x-2">
      <div className="flex -space-x-2">
        {visible.map((u) => (
          <div key={u.id} className="relative group" style={{ zIndex: 10 }}>
            <div
              className="w-8 h-8 rounded-full border-2 border-slate-800 flex items-center justify-center text-xs font-medium text-white select-none cursor-default transition-transform group-hover:scale-110"
              style={{ backgroundColor: u.color }}
            >
              {u.name.charAt(0).toUpperCase()}
            </div>

            {u.isTyping && (
              <span
                className="absolute -bottom-0.5 -right-0.5 w-2.5 h-2.5 rounded-full border border-slate-800"
                style={{
                  backgroundColor: u.color,
                  animation: "yjsPulse 1.4s ease-in-out infinite",
                }}
              />
            )}

            <div
              className="absolute bottom-full mb-2 left-1/2 -translate-x-1/2 px-2 py-1 rounded text-white text-xs whitespace-nowrap opacity-0 group-hover:opacity-100 transition-opacity pointer-events-none"
              style={{ backgroundColor: u.color }}
            >
              {u.name}
              {u.isTyping && <span className="ml-1 opacity-75">editing</span>}
              <span
                className="absolute top-full left-1/2 -translate-x-1/2 border-4 border-transparent"
                style={{ borderTopColor: u.color }}
              />
            </div>
          </div>
        ))}

        {overflow > 0 && (
          <div
            className="w-8 h-8 rounded-full border-2 border-slate-800 bg-slate-600 flex items-center justify-center text-xs font-medium text-white"
            title={`${overflow} more user${overflow > 1 ? "s" : ""}`}
          >
            +{overflow}
          </div>
        )}
      </div>
      <style>{`@keyframes yjsPulse { 0%, 100% { opacity: 1; transform: scale(1); } 50% { opacity: 0.5; transform: scale(1.4); } }`}</style>
    </div>
  );
}
