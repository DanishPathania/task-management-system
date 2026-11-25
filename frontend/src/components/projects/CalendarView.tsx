"use client";

import React, { useMemo, useState } from "react";
import { Task } from "@/types";
import { ChevronLeft, ChevronRight, Calendar as CalendarIcon } from "lucide-react";

interface CalendarViewProps {
  tasks: Task[];
  onSelectTask: (task: Task) => void;
}

export function CalendarView({ tasks, onSelectTask }: CalendarViewProps) {
  const [mode, setMode] = useState<"month" | "week">("month");
  const [cursor, setCursor] = useState(() => new Date());

  const tasksByDate = useMemo(() => {
    const map = new Map<string, Task[]>();
    for (const t of tasks) {
      if (!t.due_date) continue;
      const key = t.due_date.split("T")[0];
      const existing = map.get(key) || [];
      existing.push(t);
      map.set(key, existing);
    }
    return map;
  }, [tasks]);

  const days = useMemo(() => {
    const list: Date[] = [];
    if (mode === "month") {
      const year = cursor.getFullYear();
      const month = cursor.getMonth();
      const firstDay = new Date(year, month, 1);
      const startOffset = firstDay.getDay();
      const startDate = new Date(year, month, 1 - startOffset);
      for (let i = 0; i < 42; i++) {
        const d = new Date(startDate);
        d.setDate(startDate.getDate() + i);
        list.push(d);
      }
    } else {
      const dayOfWeek = cursor.getDay();
      const start = new Date(cursor);
      start.setDate(cursor.getDate() - dayOfWeek);
      for (let i = 0; i < 7; i++) {
        const d = new Date(start);
        d.setDate(start.getDate() + i);
        list.push(d);
      }
    }
    return list;
  }, [cursor, mode]);

  const navigate = (dir: -1 | 1) => {
    const next = new Date(cursor);
    if (mode === "month") {
      next.setMonth(cursor.getMonth() + dir);
    } else {
      next.setDate(cursor.getDate() + dir * 7);
    }
    setCursor(next);
  };

  const toDateKey = (d: Date) => {
    const y = d.getFullYear();
    const m = String(d.getMonth() + 1).padStart(2, "0");
    const day = String(d.getDate()).padStart(2, "0");
    return `${y}-${m}-${day}`;
  };

  const todayKey = toDateKey(new Date());

  return (
    <div className="bg-slate-900/90 border border-slate-800 rounded-2xl overflow-hidden">
      <div className="px-5 py-4 border-b border-slate-800 flex flex-wrap items-center justify-between gap-3">
        <div className="flex items-center gap-3">
          <CalendarIcon className="w-4 h-4 text-blue-400" />
          <h3 className="text-sm font-semibold text-slate-100">
            {cursor.toLocaleDateString("en-US", { month: "long", year: "numeric" })}
          </h3>
          <div className="flex items-center gap-1">
            <button
              onClick={() => navigate(-1)}
              className="p-1.5 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-300"
              aria-label="Previous"
            >
              <ChevronLeft className="w-4 h-4" />
            </button>
            <button
              onClick={() => setCursor(new Date())}
              className="px-2.5 py-1 rounded-lg bg-slate-800 hover:bg-slate-700 text-xs text-slate-300"
            >
              Today
            </button>
            <button
              onClick={() => navigate(1)}
              className="p-1.5 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-300"
              aria-label="Next"
            >
              <ChevronRight className="w-4 h-4" />
            </button>
          </div>
        </div>

        <div className="flex items-center gap-1 bg-slate-800 p-1 rounded-lg text-xs">
          <button
            onClick={() => setMode("month")}
            className={`px-3 py-1 rounded-md transition ${mode === "month" ? "bg-blue-600 text-white" : "text-slate-300"}`}
          >
            Month
          </button>
          <button
            onClick={() => setMode("week")}
            className={`px-3 py-1 rounded-md transition ${mode === "week" ? "bg-blue-600 text-white" : "text-slate-300"}`}
          >
            Week
          </button>
        </div>
      </div>

      <div className="grid grid-cols-7 border-b border-slate-800 bg-slate-800/40 text-[11px] font-semibold text-slate-400 text-center">
        {["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"].map((d) => (
          <div key={d} className="py-2">{d}</div>
        ))}
      </div>

      <div className="grid grid-cols-7 divide-x divide-y divide-slate-800/80">
        {days.map((day) => {
          const key = toDateKey(day);
          const dayTasks = tasksByDate.get(key) || [];
          const isCurrentMonth = day.getMonth() === cursor.getMonth();
          const isToday = key === todayKey;

          return (
            <div
              key={key}
              className={`min-h-[110px] p-2 space-y-1.5 ${
                mode === "month" && !isCurrentMonth ? "bg-slate-950/50 opacity-50" : ""
              } ${isToday ? "bg-blue-500/5" : ""}`}
            >
              <div className="flex items-center justify-between">
                <span
                  className={`text-[11px] font-semibold px-1.5 py-0.5 rounded ${
                    isToday ? "bg-blue-600 text-white" : "text-slate-400"
                  }`}
                >
                  {day.getDate()}
                </span>
                {dayTasks.length > 0 && (
                  <span className="text-[10px] text-slate-500">{dayTasks.length}</span>
                )}
              </div>

              <div className="space-y-1">
                {dayTasks.map((t) => {
                  const isDone = t.status === "DONE";
                  const isOverdue = !isDone && new Date(t.due_date!).getTime() < Date.now();
                  return (
                    <button
                      key={t.id}
                      onClick={() => onSelectTask(t)}
                      className={`w-full text-left px-2 py-1 rounded text-[11px] truncate border transition ${
                        isDone
                          ? "bg-emerald-500/10 text-emerald-300 border-emerald-500/20 line-through"
                          : isOverdue
                          ? "bg-red-500/15 text-red-300 border-red-500/30"
                          : "bg-slate-800 text-slate-200 border-slate-700 hover:border-blue-500/50"
                      }`}
                      title={`${t.title} (${t.priority})`}
                    >
                      <span className="font-medium">{t.title}</span>
                    </button>
                  );
                })}
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}
