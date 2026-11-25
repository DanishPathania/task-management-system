"use client";

import React, { useEffect, useState } from "react";
import { ProjectAnalytics } from "@/types";
import { api } from "@/lib/api";
import { BarChart3, CheckCircle2, Clock, AlertTriangle, ListTodo } from "lucide-react";

interface AnalyticsPanelProps {
  projectId: string;
}

export function AnalyticsPanel({ projectId }: AnalyticsPanelProps) {
  const [stats, setStats] = useState<ProjectAnalytics | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    api
      .getProjectAnalytics(projectId)
      .then(setStats)
      .catch((err) => console.error("Failed to load analytics:", err))
      .finally(() => setLoading(false));
  }, [projectId]);

  if (loading || !stats) {
    return (
      <div className="p-10 flex justify-center">
        <div className="w-7 h-7 border-2 border-blue-500 border-t-transparent rounded-full animate-spin" />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* KPI Cards */}
      <div className="grid grid-cols-2 sm:grid-cols-5 gap-4">
        <div className="bg-slate-900/90 border border-slate-800 rounded-2xl p-4 space-y-1">
          <div className="flex items-center justify-between text-xs text-slate-400">
            <span>Total Tasks</span>
            <ListTodo className="w-4 h-4 text-blue-400" />
          </div>
          <p className="text-2xl font-bold text-white">{stats.total_tasks}</p>
        </div>

        <div className="bg-slate-900/90 border border-slate-800 rounded-2xl p-4 space-y-1">
          <div className="flex items-center justify-between text-xs text-slate-400">
            <span>Completed</span>
            <CheckCircle2 className="w-4 h-4 text-emerald-400" />
          </div>
          <p className="text-2xl font-bold text-emerald-400">{stats.completed_tasks}</p>
          <span className="text-[11px] text-slate-500">{stats.completion_percent}% complete</span>
        </div>

        <div className="bg-slate-900/90 border border-slate-800 rounded-2xl p-4 space-y-1">
          <div className="flex items-center justify-between text-xs text-slate-400">
            <span>In Progress</span>
            <Clock className="w-4 h-4 text-blue-400" />
          </div>
          <p className="text-2xl font-bold text-blue-400">{stats.in_progress_tasks}</p>
        </div>

        <div className="bg-slate-900/90 border border-slate-800 rounded-2xl p-4 space-y-1">
          <div className="flex items-center justify-between text-xs text-slate-400">
            <span>Pending</span>
            <BarChart3 className="w-4 h-4 text-slate-400" />
          </div>
          <p className="text-2xl font-bold text-slate-200">{stats.pending_tasks}</p>
        </div>

        <div className="bg-slate-900/90 border border-slate-800 rounded-2xl p-4 space-y-1">
          <div className="flex items-center justify-between text-xs text-slate-400">
            <span>Overdue</span>
            <AlertTriangle className="w-4 h-4 text-red-400" />
          </div>
          <p className="text-2xl font-bold text-red-400">{stats.overdue_tasks}</p>
        </div>
      </div>

      {/* Distribution & Assignee Breakdown */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div className="bg-slate-900/90 border border-slate-800 rounded-2xl p-5 space-y-4">
          <h4 className="text-sm font-semibold text-slate-100">Priority Distribution</h4>
          <div className="space-y-3">
            {["HIGH", "MEDIUM", "LOW"].map((prio) => {
              const found = stats.priority_distribution.find((p) => p.priority === prio);
              const count = found ? found.count : 0;
              const pct = stats.total_tasks > 0 ? Math.round((count / stats.total_tasks) * 100) : 0;
              const barColor = prio === "HIGH" ? "bg-red-500" : prio === "MEDIUM" ? "bg-amber-500" : "bg-emerald-500";
              return (
                <div key={prio} className="space-y-1 text-xs">
                  <div className="flex justify-between text-slate-300">
                    <span>{prio}</span>
                    <span>{count} ({pct}%)</span>
                  </div>
                  <div className="w-full h-2 bg-slate-800 rounded-full overflow-hidden">
                    <div className={`h-full ${barColor}`} style={{ width: `${pct}%` }} />
                  </div>
                </div>
              );
            })}
          </div>
        </div>

        <div className="bg-slate-900/90 border border-slate-800 rounded-2xl p-5 space-y-4">
          <h4 className="text-sm font-semibold text-slate-100">Tasks by Team Member</h4>
          {stats.assignee_breakdown.length === 0 ? (
            <p className="text-xs text-slate-500">No tasks assigned yet.</p>
          ) : (
            <div className="space-y-2.5 text-xs">
              {stats.assignee_breakdown.map((m) => (
                <div key={m.user_name} className="flex items-center justify-between bg-slate-800/50 px-3 py-2 rounded-lg">
                  <span className="font-medium text-slate-200">{m.user_name}</span>
                  <span className="text-slate-400">
                    {m.done} / {m.total} done
                  </span>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>

      {/* Upcoming & Recently Completed */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div className="bg-slate-900/90 border border-slate-800 rounded-2xl p-5 space-y-3">
          <h4 className="text-sm font-semibold text-slate-100">Upcoming Deadlines</h4>
          {stats.upcoming_deadlines.length === 0 ? (
            <p className="text-xs text-slate-500">No upcoming deadlines.</p>
          ) : (
            <div className="space-y-2 text-xs">
              {stats.upcoming_deadlines.map((t) => (
                <div key={t.id} className="flex items-center justify-between bg-slate-800/50 px-3 py-2 rounded-lg">
                  <span className="text-slate-200 truncate">{t.title}</span>
                  <span className="text-slate-400 shrink-0">
                    {t.due_date ? new Date(t.due_date).toLocaleDateString() : ""}
                  </span>
                </div>
              ))}
            </div>
          )}
        </div>

        <div className="bg-slate-900/90 border border-slate-800 rounded-2xl p-5 space-y-3">
          <h4 className="text-sm font-semibold text-slate-100">14-Day Completion Trend</h4>
          {stats.completion_trend.length === 0 ? (
            <p className="text-xs text-slate-500">No completions recorded in the last 14 days.</p>
          ) : (
            <div className="flex items-end gap-2 h-28 pt-2">
              {stats.completion_trend.map((pt) => (
                <div key={pt.date} className="flex-1 flex flex-col items-center gap-1">
                  <div
                    className="w-full bg-blue-500/80 rounded-t"
                    style={{ height: `${Math.max(12, Math.min(100, pt.count * 20))}%` }}
                    title={`${pt.date}: ${pt.count} completed`}
                  />
                  <span className="text-[9px] text-slate-500">{pt.date.slice(5)}</span>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
