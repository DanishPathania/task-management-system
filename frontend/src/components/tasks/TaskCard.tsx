"use client";

import React from "react";
import { Task } from "@/types";
import { Calendar, CheckSquare, AlertTriangle, Lock } from "lucide-react";

interface TaskCardProps {
  task: Task;
  onClick: () => void;
}

export function TaskCard({ task, onClick }: TaskCardProps) {
  const getPriorityBadge = (priority: string) => {
    switch (priority) {
      case "HIGH":
        return <span className="px-2 py-0.5 rounded text-[11px] font-semibold bg-red-500/10 text-red-400 border border-red-500/20">HIGH</span>;
      case "MEDIUM":
        return <span className="px-2 py-0.5 rounded text-[11px] font-semibold bg-amber-500/10 text-amber-400 border border-amber-500/20">MEDIUM</span>;
      default:
        return <span className="px-2 py-0.5 rounded text-[11px] font-semibold bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">LOW</span>;
    }
  };

  const formatDate = (dateStr?: string) => {
    if (!dateStr) return null;
    const date = new Date(dateStr);
    return date.toLocaleDateString("en-US", { month: "short", day: "numeric" });
  };

  const isOverdue =
    task.due_date && task.status !== "DONE" && new Date(task.due_date).getTime() < Date.now();
  const subtaskCount = task.subtask_count || 0;
  const completedSubtasks = task.completed_subtask_count || 0;
  const completionPct = task.completion_percent || 0;

  return (
    <div
      onClick={onClick}
      className="group relative bg-slate-800/80 hover:bg-slate-800 border border-slate-700/60 hover:border-blue-500/50 rounded-xl p-4 cursor-pointer transition-all duration-200 shadow-sm hover:shadow-md hover:shadow-blue-500/5 hover:-translate-y-0.5"
    >
      <div className="flex items-start justify-between gap-2 mb-2">
        <h4 className="font-semibold text-slate-100 text-sm group-hover:text-blue-400 transition leading-snug line-clamp-2">
          {task.title}
        </h4>
        <div className="flex items-center gap-1 shrink-0">
          {task.is_blocked && (
            <span className="px-1.5 py-0.5 rounded text-[10px] font-semibold bg-amber-500/15 text-amber-300 border border-amber-500/30 flex items-center gap-0.5" title="Blocked by incomplete dependency">
              <Lock className="w-2.5 h-2.5" />
              <span>BLOCKED</span>
            </span>
          )}
          {getPriorityBadge(task.priority)}
        </div>
      </div>

      {task.description && (
        <p className="text-xs text-slate-400 line-clamp-2 mb-3 leading-relaxed">
          {task.description}
        </p>
      )}

      {subtaskCount > 0 && (
        <div className="mb-3 space-y-1">
          <div className="flex items-center justify-between text-[11px] text-slate-400">
            <span className="flex items-center gap-1">
              <CheckSquare className="w-3 h-3 text-blue-400" />
              <span>Subtasks {completedSubtasks}/{subtaskCount}</span>
            </span>
            <span>{completionPct}%</span>
          </div>
          <div className="w-full h-1.5 bg-slate-700/70 rounded-full overflow-hidden">
            <div
              className="h-full bg-blue-500 transition-all"
              style={{ width: `${completionPct}%` }}
            />
          </div>
        </div>
      )}

      <div className="flex items-center justify-between pt-2 border-t border-slate-700/40 text-xs text-slate-400">
        <div className="flex items-center gap-1.5">
          {task.due_date ? (
            <div className={`flex items-center gap-1 text-[11px] ${isOverdue ? "text-red-400 font-medium" : "text-slate-400"}`}>
              {isOverdue ? <AlertTriangle className="w-3.5 h-3.5 text-red-400" /> : <Calendar className="w-3.5 h-3.5 text-slate-500" />}
              <span>{formatDate(task.due_date)}</span>
            </div>
          ) : (
            <span className="text-[11px] text-slate-500">No due date</span>
          )}
        </div>

        <div className="flex items-center gap-1.5">
          {task.assignee_name ? (
            <div className="flex items-center gap-1 text-[11px] text-slate-300 bg-slate-700/50 px-2 py-0.5 rounded-full border border-slate-600/40" title={`Assigned to ${task.assignee_name}`}>
              <div className="w-3.5 h-3.5 rounded-full bg-blue-500 text-white font-bold flex items-center justify-center text-[9px]">
                {task.assignee_name.charAt(0).toUpperCase()}
              </div>
              <span className="truncate max-w-[80px]">{task.assignee_name}</span>
            </div>
          ) : (
            <span className="text-[11px] text-slate-500 italic">Unassigned</span>
          )}
        </div>
      </div>
    </div>
  );
}
