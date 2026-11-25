"use client";

import React, { useMemo } from "react";
import { Task, TaskDependency } from "@/types";
import { GitBranch, Lock } from "lucide-react";

interface TimelineViewProps {
  tasks: Task[];
  dependencies: TaskDependency[];
  onSelectTask: (task: Task) => void;
}

export function TimelineView({ tasks, dependencies, onSelectTask }: TimelineViewProps) {
  const datedTasks = useMemo(() => {
    return tasks.filter((t) => t.start_date || t.due_date);
  }, [tasks]);

  const { minDate, totalDays } = useMemo(() => {
    if (datedTasks.length === 0) {
      const today = new Date();
      return { minDate: today, totalDays: 14 };
    }
    let minMs = Infinity;
    let maxMs = -Infinity;
    for (const t of datedTasks) {
      const s = t.start_date ? new Date(t.start_date).getTime() : new Date(t.due_date!).getTime();
      const e = t.due_date ? new Date(t.due_date).getTime() : s;
      if (s < minMs) minMs = s;
      if (e > maxMs) maxMs = e;
    }
    const minD = new Date(minMs);
    minD.setDate(minD.getDate() - 1);
    const diffDays = Math.max(14, Math.ceil((maxMs - minD.getTime()) / (1000 * 60 * 60 * 24)) + 2);
    return { minDate: minD, totalDays: Math.min(diffDays, 60) };
  }, [datedTasks]);

  const dayHeaders = useMemo(() => {
    const arr: Date[] = [];
    for (let i = 0; i < totalDays; i++) {
      const d = new Date(minDate);
      d.setDate(minDate.getDate() + i);
      arr.push(d);
    }
    return arr;
  }, [minDate, totalDays]);

  return (
    <div className="bg-slate-900/90 border border-slate-800 rounded-2xl overflow-hidden space-y-4">
      <div className="px-6 py-4 border-b border-slate-800 flex items-center justify-between">
        <div className="flex items-center gap-2">
          <GitBranch className="w-4 h-4 text-blue-400" />
          <h3 className="text-sm font-semibold text-slate-100">Project Timeline & Dependencies</h3>
        </div>
        <span className="text-xs text-slate-400">{datedTasks.length} scheduled task(s)</span>
      </div>

      {datedTasks.length === 0 ? (
        <div className="p-10 text-center text-xs text-slate-500">
          Add start dates or due dates to tasks to visualize them on the timeline.
        </div>
      ) : (
        <div className="overflow-x-auto px-6 pb-6">
          <div className="min-w-[800px] space-y-2">
            {/* Day Header */}
            <div className="grid grid-cols-12 gap-2 text-[10px] text-slate-500 border-b border-slate-800 pb-2">
              <div className="col-span-3 font-semibold uppercase tracking-wider">Task</div>
              <div className="col-span-9 flex justify-between">
                {dayHeaders.filter((_, idx) => idx % Math.max(1, Math.floor(totalDays / 7)) === 0).map((d) => (
                  <span key={d.toISOString()}>{d.toLocaleDateString("en-US", { month: "short", day: "numeric" })}</span>
                ))}
              </div>
            </div>

            {datedTasks.map((t) => {
              const sMs = t.start_date ? new Date(t.start_date).getTime() : new Date(t.due_date!).getTime();
              const eMs = t.due_date ? new Date(t.due_date).getTime() : sMs + 86400000;
              const startOffsetDays = Math.max(0, (sMs - minDate.getTime()) / 86400000);
              const durationDays = Math.max(1, (eMs - sMs) / 86400000);

              const leftPct = Math.min(95, (startOffsetDays / totalDays) * 100);
              const widthPct = Math.max(4, Math.min(100 - leftPct, (durationDays / totalDays) * 100));

              const taskDeps = dependencies.filter((d) => d.task_id === t.id);

              return (
                <div
                  key={t.id}
                  onClick={() => onSelectTask(t)}
                  className="grid grid-cols-12 gap-2 items-center py-2 hover:bg-slate-800/40 rounded-lg px-2 cursor-pointer transition"
                >
                  <div className="col-span-3 min-w-0">
                    <div className="flex items-center gap-1.5">
                      {t.is_blocked && <Lock className="w-3 h-3 text-amber-400 shrink-0" />}
                      <span className="text-xs font-medium text-slate-200 truncate">{t.title}</span>
                    </div>
                    {taskDeps.length > 0 && (
                      <div className="text-[10px] text-slate-500 truncate">
                        Depends on: {taskDeps.map((d) => d.depends_on_title).join(", ")}
                      </div>
                    )}
                  </div>

                  <div className="col-span-9 relative h-6 bg-slate-800/60 rounded-md overflow-hidden">
                    <div
                      className={`absolute top-1 bottom-1 rounded text-[10px] font-medium px-2 flex items-center truncate ${
                        t.status === "DONE"
                          ? "bg-emerald-600/80 text-white"
                          : t.is_blocked
                          ? "bg-amber-600/80 text-white"
                          : "bg-blue-600/80 text-white"
                      }`}
                      style={{ left: `${leftPct}%`, width: `${widthPct}%` }}
                    >
                      {t.title}
                    </div>
                  </div>
                </div>
              );
            })}
          </div>
        </div>
      )}
    </div>
  );
}
