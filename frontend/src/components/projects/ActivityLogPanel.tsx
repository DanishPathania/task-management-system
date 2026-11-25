"use client";

import React, { useEffect, useState } from "react";
import { ActivityLog } from "@/types";
import { api } from "@/lib/api";
import { realtime } from "@/lib/websocket";
import { History, ArrowRight } from "lucide-react";

interface ActivityLogPanelProps {
  projectId: string;
}

export function ActivityLogPanel({ projectId }: ActivityLogPanelProps) {
  const [activities, setActivities] = useState<ActivityLog[]>([]);
  const [page, setPage] = useState(1);
  const [totalPages, setTotalPages] = useState(1);
  const [loading, setLoading] = useState(true);

  const loadActivity = async (p = page) => {
    setLoading(true);
    try {
      const res = await api.listActivity(projectId, p, 30);
      setActivities(res.activities || []);
      setTotalPages(res.pagination?.total_pages || 1);
    } catch (err) {
      console.error("Failed to load activity:", err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadActivity(page);
  }, [projectId, page]);

  useEffect(() => {
    const unsubscribe = realtime.onEvent((event) => {
      if (event.projectId === projectId && event.event === "activity.created" && event.payload) {
        const entry = event.payload as ActivityLog;
        setActivities((prev) => {
          if (prev.some((a) => a.id === entry.id)) return prev;
          return [entry, ...prev];
        });
      }
    });
    return () => unsubscribe();
  }, [projectId]);

  const formatAction = (action: string) => {
    return action.replace(/\./g, " ").replace(/_/g, " ");
  };

  const formatVal = (val: any) => {
    if (val === null || val === undefined) return "";
    if (typeof val === "object") return JSON.stringify(val);
    return String(val);
  };

  return (
    <div className="bg-slate-900/90 border border-slate-800 rounded-2xl overflow-hidden">
      <div className="px-6 py-4 border-b border-slate-800 flex items-center justify-between">
        <div className="flex items-center gap-2">
          <History className="w-4 h-4 text-blue-400" />
          <h3 className="text-sm font-semibold text-slate-100">Project Activity & Audit History</h3>
        </div>
        <span className="text-xs text-slate-500">Read-only chronological log</span>
      </div>

      {loading ? (
        <div className="p-8 flex justify-center">
          <div className="w-6 h-6 border-2 border-blue-500 border-t-transparent rounded-full animate-spin" />
        </div>
      ) : activities.length === 0 ? (
        <div className="p-10 text-center text-xs text-slate-500">No activity recorded yet.</div>
      ) : (
        <div className="divide-y divide-slate-800/70">
          {activities.map((a) => (
            <div key={a.id} className="px-6 py-3.5 flex flex-col sm:flex-row sm:items-center justify-between gap-2 text-xs">
              <div className="space-y-1">
                <div className="flex flex-wrap items-center gap-1.5">
                  <span className="font-semibold text-slate-200">{a.actor_name || "System"}</span>
                  <span className="px-2 py-0.5 rounded bg-slate-800 text-blue-400 font-medium capitalize">
                    {formatAction(a.action)}
                  </span>
                  {a.entity_title && (
                    <span className="font-medium text-slate-300">&ldquo;{a.entity_title}&rdquo;</span>
                  )}
                </div>
                {(a.old_value !== undefined || a.new_value !== undefined) && (
                  <div className="flex flex-wrap items-center gap-1.5 text-[11px] text-slate-400">
                    {a.old_value !== undefined && a.old_value !== null && (
                      <span className="bg-red-500/10 text-red-300 px-1.5 py-0.5 rounded line-through">
                        {formatVal(a.old_value)}
                      </span>
                    )}
                    {a.old_value !== undefined && a.new_value !== undefined && (
                      <ArrowRight className="w-3 h-3 text-slate-500" />
                    )}
                    {a.new_value !== undefined && a.new_value !== null && (
                      <span className="bg-emerald-500/10 text-emerald-300 px-1.5 py-0.5 rounded">
                        {formatVal(a.new_value)}
                      </span>
                    )}
                  </div>
                )}
              </div>
              <span className="text-[11px] text-slate-500 shrink-0">
                {new Date(a.created_at).toLocaleString()}
              </span>
            </div>
          ))}
        </div>
      )}

      {totalPages > 1 && (
        <div className="flex items-center justify-between px-6 py-3 border-t border-slate-800 text-xs text-slate-400">
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
  );
}
