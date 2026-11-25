"use client";

import React, { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { useAuth } from "@/hooks/useAuth";
import { api } from "@/lib/api";
import { Notification, NotificationPreferences } from "@/types";
import { Bell, CheckCheck, Check, Sliders, ArrowRight } from "lucide-react";

export default function NotificationsPage() {
  const { user, loading: authLoading } = useAuth();
  const router = useRouter();

  const [notifications, setNotifications] = useState<Notification[]>([]);
  const [unreadCount, setUnreadCount] = useState(0);
  const [unreadOnly, setUnreadOnly] = useState(false);
  const [page, setPage] = useState(1);
  const [totalPages, setTotalPages] = useState(1);
  const [loading, setLoading] = useState(true);

  const [prefs, setPrefs] = useState<NotificationPreferences | null>(null);
  const [savingPrefs, setSavingPrefs] = useState(false);

  useEffect(() => {
    if (!authLoading && !user) {
      router.push("/login");
    }
  }, [user, authLoading, router]);

  const loadData = async () => {
    setLoading(true);
    try {
      const [notifRes, prefRes] = await Promise.all([
        api.listNotifications(unreadOnly, page, 20),
        api.getNotificationPreferences(),
      ]);
      setNotifications(notifRes.notifications || []);
      setUnreadCount(notifRes.unread_count || 0);
      setTotalPages(notifRes.pagination?.total_pages || 1);
      setPrefs(prefRes);
    } catch (err) {
      console.error("Failed to load notifications:", err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (user) {
      loadData();
    }
  }, [user, unreadOnly, page]);

  const handleMarkRead = async (n: Notification) => {
    if (!n.is_read) {
      await api.markNotificationRead(n.id);
      setNotifications((prev) => prev.map((item) => (item.id === n.id ? { ...item, is_read: true } : item)));
      setUnreadCount((c) => Math.max(0, c - 1));
    }
    if (n.project_id) {
      const target = n.task_id ? `/projects/${n.project_id}?taskId=${n.task_id}` : `/projects/${n.project_id}`;
      router.push(target);
    }
  };

  const handleMarkAllRead = async () => {
    await api.markAllNotificationsRead();
    setNotifications((prev) => prev.map((n) => ({ ...n, is_read: true })));
    setUnreadCount(0);
  };

  const handleTogglePref = async (key: keyof NotificationPreferences) => {
    if (!prefs || savingPrefs) return;
    setSavingPrefs(true);
    const updated = { ...prefs, [key]: !prefs[key] };
    setPrefs(updated);
    try {
      const saved = await api.updateNotificationPreferences(updated);
      setPrefs(saved);
    } finally {
      setSavingPrefs(false);
    }
  };

  if (authLoading || loading) {
    return (
      <div className="flex-1 flex items-center justify-center p-8">
        <div className="w-8 h-8 border-2 border-blue-500 border-t-transparent rounded-full animate-spin" />
      </div>
    );
  }

  return (
    <div className="max-w-5xl mx-auto px-4 sm:px-6 py-8 w-full space-y-6 flex-1">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold text-white tracking-tight flex items-center gap-2">
            <Bell className="w-6 h-6 text-blue-400" />
            <span>Notification Center</span>
          </h1>
          <p className="text-xs text-slate-400 mt-1">
            Stay updated on task assignments, mentions, comments, and deadlines ({unreadCount} unread)
          </p>
        </div>

        <div className="flex items-center gap-2">
          <button
            onClick={() => {
              setUnreadOnly((v) => !v);
              setPage(1);
            }}
            className={`px-3 py-1.5 rounded-xl text-xs font-medium border transition ${
              unreadOnly
                ? "bg-blue-600/20 text-blue-400 border-blue-500/40"
                : "bg-slate-800 text-slate-300 border-slate-700 hover:bg-slate-700"
            }`}
          >
            {unreadOnly ? "Showing Unread" : "Show Unread Only"}
          </button>

          {unreadCount > 0 && (
            <button
              onClick={handleMarkAllRead}
              className="px-3 py-1.5 rounded-xl bg-blue-600 hover:bg-blue-500 text-white text-xs font-medium flex items-center gap-1.5 transition"
            >
              <CheckCheck className="w-4 h-4" />
              <span>Mark All Read</span>
            </button>
          )}
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Notification List */}
        <div className="lg:col-span-2 bg-slate-900/90 border border-slate-800 rounded-2xl overflow-hidden">
          {notifications.length === 0 ? (
            <div className="p-12 text-center text-sm text-slate-500">
              No notifications found.
            </div>
          ) : (
            <div className="divide-y divide-slate-800/80">
              {notifications.map((n) => (
                <div
                  key={n.id}
                  onClick={() => handleMarkRead(n)}
                  className={`p-4 hover:bg-slate-800/60 transition cursor-pointer flex items-start justify-between gap-4 ${
                    n.is_read ? "opacity-70" : "bg-blue-500/5"
                  }`}
                >
                  <div className="flex items-start gap-3">
                    <div className={`w-2.5 h-2.5 rounded-full mt-1.5 shrink-0 ${n.is_read ? "bg-slate-600" : "bg-blue-500"}`} />
                    <div>
                      <p className="text-sm font-semibold text-slate-100">{n.title}</p>
                      <p className="text-xs text-slate-400 mt-0.5">{n.message}</p>
                      <span className="text-[11px] text-slate-500 mt-1.5 block">
                        {new Date(n.created_at).toLocaleString()}
                      </span>
                    </div>
                  </div>

                  <div className="flex items-center gap-2 shrink-0">
                    {!n.is_read && (
                      <span className="text-[10px] px-2 py-0.5 rounded-full bg-blue-500/15 text-blue-400 border border-blue-500/20">
                        New
                      </span>
                    )}
                    {n.project_id && <ArrowRight className="w-4 h-4 text-slate-500" />}
                  </div>
                </div>
              ))}
            </div>
          )}

          {totalPages > 1 && (
            <div className="flex items-center justify-between px-4 py-3 border-t border-slate-800 text-xs text-slate-400">
              <button
                disabled={page <= 1}
                onClick={() => setPage((p) => Math.max(1, p - 1))}
                className="px-3 py-1 rounded bg-slate-800 disabled:opacity-40"
              >
                Previous
              </button>
              <span>Page {page} of {totalPages}</span>
              <button
                disabled={page >= totalPages}
                onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                className="px-3 py-1 rounded bg-slate-800 disabled:opacity-40"
              >
                Next
              </button>
            </div>
          )}
        </div>

        {/* Preferences Panel */}
        <div className="bg-slate-900/90 border border-slate-800 rounded-2xl p-5 space-y-4 h-fit">
          <div className="flex items-center gap-2 border-b border-slate-800 pb-3">
            <Sliders className="w-4 h-4 text-blue-400" />
            <h2 className="text-sm font-semibold text-slate-100">Notification Preferences</h2>
          </div>

          {prefs && (
            <div className="space-y-3 text-xs">
              {[
                { key: "task_assigned" as const, label: "Task assignments" },
                { key: "task_due" as const, label: "Due date & overdue reminders" },
                { key: "mentions" as const, label: "@Mentions in comments" },
                { key: "comments" as const, label: "New comments on my tasks" },
                { key: "status_changes" as const, label: "Task status changes" },
                { key: "member_added" as const, label: "Added to a project" },
              ].map((item) => (
                <label key={item.key} className="flex items-center justify-between cursor-pointer py-1">
                  <span className="text-slate-300">{item.label}</span>
                  <input
                    type="checkbox"
                    checked={Boolean(prefs[item.key])}
                    onChange={() => handleTogglePref(item.key)}
                    className="w-4 h-4 accent-blue-600 rounded"
                  />
                </label>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
