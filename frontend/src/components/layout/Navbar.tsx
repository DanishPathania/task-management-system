"use client";

import React, { useEffect, useState, useRef } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useAuth } from "@/hooks/useAuth";
import { api } from "@/lib/api";
import { realtime } from "@/lib/websocket";
import { Notification, RealtimeEvent } from "@/types";
import { CheckSquare, LogOut, Bell, Check, CheckCheck, ExternalLink } from "lucide-react";

export function Navbar() {
  const { user, logout } = useAuth();
  const router = useRouter();
  const [notifications, setNotifications] = useState<Notification[]>([]);
  const [unreadCount, setUnreadCount] = useState(0);
  const [dropdownOpen, setDropdownOpen] = useState(false);
  const dropdownRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!user) {
      setNotifications([]);
      setUnreadCount(0);
      return;
    }

    const loadNotifications = async () => {
      try {
        const res = await api.listNotifications(false, 1, 8);
        setNotifications(res.notifications || []);
        setUnreadCount(res.unread_count || 0);
      } catch {
        // Ignore transient errors
      }
    };

    loadNotifications();

    const unsubscribe = realtime.onEvent((event: RealtimeEvent) => {
      if (event.event === "notification.created" && event.payload) {
        const notif = event.payload as Notification;
        setNotifications((prev) => {
          if (prev.some((n) => n.id === notif.id)) return prev;
          return [notif, ...prev].slice(0, 8);
        });
        setUnreadCount((c) => c + 1);
      }
    });

    return () => {
      unsubscribe();
    };
  }, [user]);

  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (dropdownRef.current && !dropdownRef.current.contains(e.target as Node)) {
        setDropdownOpen(false);
      }
    };
    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, []);

  const handleLogout = async () => {
    await logout();
    router.push("/login");
  };

  const handleNotificationClick = async (n: Notification) => {
    if (!n.is_read) {
      try {
        await api.markNotificationRead(n.id);
        setNotifications((prev) => prev.map((item) => (item.id === n.id ? { ...item, is_read: true } : item)));
        setUnreadCount((c) => Math.max(0, c - 1));
      } catch {
        // ignore
      }
    }
    setDropdownOpen(false);
    if (n.project_id) {
      const target = n.task_id ? `/projects/${n.project_id}?taskId=${n.task_id}` : `/projects/${n.project_id}`;
      router.push(target);
    }
  };

  const handleMarkAllRead = async () => {
    try {
      await api.markAllNotificationsRead();
      setNotifications((prev) => prev.map((n) => ({ ...n, is_read: true })));
      setUnreadCount(0);
    } catch {
      // ignore
    }
  };

  return (
    <nav className="border-b border-slate-800 bg-slate-900/90 backdrop-blur-md sticky top-0 z-40 px-4 sm:px-6 py-3">
      <div className="max-w-7xl mx-auto flex items-center justify-between">
        <div className="flex items-center gap-6">
          <Link href={user ? "/dashboard" : "/"} className="flex items-center gap-2 text-white font-bold text-xl tracking-tight hover:opacity-90 transition">
            <div className="w-8 h-8 rounded-lg bg-gradient-to-tr from-blue-600 to-indigo-500 flex items-center justify-center text-white shadow-lg shadow-blue-500/20">
              <CheckSquare className="w-5 h-5" />
            </div>
            <span>CollabTask</span>
          </Link>

          {user && (
            <div className="hidden sm:flex items-center gap-1 text-sm">
              <Link
                href="/dashboard"
                className="px-3 py-1.5 rounded-lg text-slate-300 hover:text-white hover:bg-slate-800/80 transition font-medium"
              >
                Dashboard
              </Link>
              <Link
                href="/notifications"
                className="px-3 py-1.5 rounded-lg text-slate-300 hover:text-white hover:bg-slate-800/80 transition font-medium"
              >
                Notifications
              </Link>
            </div>
          )}
        </div>

        {user ? (
          <div className="flex items-center gap-3">
            {/* Notification Bell */}
            <div className="relative" ref={dropdownRef}>
              <button
                onClick={() => setDropdownOpen((o) => !o)}
                className="relative p-2 rounded-lg text-slate-300 hover:text-white hover:bg-slate-800 transition"
                title="Notifications"
                aria-label="Notifications"
              >
                <Bell className="w-5 h-5" />
                {unreadCount > 0 && (
                  <span className="absolute -top-0.5 -right-0.5 min-w-[18px] h-[18px] px-1 rounded-full bg-blue-600 text-white text-[10px] font-bold flex items-center justify-center">
                    {unreadCount > 99 ? "99+" : unreadCount}
                  </span>
                )}
              </button>

              {dropdownOpen && (
                <div className="absolute right-0 mt-2 w-80 sm:w-96 bg-slate-900 border border-slate-800 rounded-2xl shadow-2xl overflow-hidden z-50">
                  <div className="flex items-center justify-between px-4 py-3 border-b border-slate-800">
                    <span className="text-xs font-semibold text-slate-200 uppercase tracking-wider">
                      Notifications ({unreadCount} unread)
                    </span>
                    {unreadCount > 0 && (
                      <button
                        onClick={handleMarkAllRead}
                        className="text-[11px] text-blue-400 hover:text-blue-300 flex items-center gap-1 font-medium"
                      >
                        <CheckCheck className="w-3.5 h-3.5" />
                        <span>Mark all read</span>
                      </button>
                    )}
                  </div>

                  <div className="max-h-80 overflow-y-auto divide-y divide-slate-800/70">
                    {notifications.length === 0 ? (
                      <div className="p-6 text-center text-xs text-slate-500">
                        You are all caught up!
                      </div>
                    ) : (
                      notifications.map((n) => (
                        <button
                          key={n.id}
                          onClick={() => handleNotificationClick(n)}
                          className={`w-full text-left px-4 py-3 hover:bg-slate-800/70 transition flex items-start gap-2.5 ${
                            n.is_read ? "opacity-65" : "bg-blue-500/5"
                          }`}
                        >
                          <div className={`w-2 h-2 rounded-full mt-1.5 shrink-0 ${n.is_read ? "bg-slate-600" : "bg-blue-500"}`} />
                          <div className="flex-1 min-w-0">
                            <p className="text-xs font-semibold text-slate-200 truncate">{n.title}</p>
                            <p className="text-[11px] text-slate-400 line-clamp-2 mt-0.5">{n.message}</p>
                            <span className="text-[10px] text-slate-500 mt-1 block">
                              {new Date(n.created_at).toLocaleString()}
                            </span>
                          </div>
                          {!n.is_read && <Check className="w-3.5 h-3.5 text-slate-500 shrink-0 mt-1" />}
                        </button>
                      ))
                    )}
                  </div>

                  <Link
                    href="/notifications"
                    onClick={() => setDropdownOpen(false)}
                    className="block px-4 py-2.5 text-center text-xs font-medium text-blue-400 hover:bg-slate-800/80 border-t border-slate-800 transition"
                  >
                    <span className="inline-flex items-center gap-1">
                      <span>View all notifications & preferences</span>
                      <ExternalLink className="w-3 h-3" />
                    </span>
                  </Link>
                </div>
              )}
            </div>

            <div className="flex items-center gap-2 bg-slate-800/80 border border-slate-700/60 px-3 py-1.5 rounded-full text-sm text-slate-200">
              <div className="w-6 h-6 rounded-full bg-blue-600/40 text-blue-400 font-semibold flex items-center justify-center text-xs">
                {user.name.charAt(0).toUpperCase()}
              </div>
              <span className="font-medium hidden sm:inline">{user.name}</span>
            </div>

            <button
              onClick={handleLogout}
              className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm font-medium text-slate-400 hover:text-red-400 hover:bg-red-500/10 transition"
              title="Logout"
            >
              <LogOut className="w-4 h-4" />
              <span className="hidden sm:inline">Logout</span>
            </button>
          </div>
        ) : (
          <div className="flex items-center gap-3">
            <Link
              href="/login"
              className="px-4 py-1.5 text-sm font-medium text-slate-300 hover:text-white transition"
            >
              Sign In
            </Link>
            <Link
              href="/register"
              className="px-4 py-1.5 text-sm font-medium rounded-lg bg-blue-600 hover:bg-blue-500 text-white transition shadow-md shadow-blue-600/20"
            >
              Get Started
            </Link>
          </div>
        )}
      </div>
    </nav>
  );
}
