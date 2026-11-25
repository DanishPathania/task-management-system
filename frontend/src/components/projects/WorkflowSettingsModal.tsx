"use client";

import React, { useEffect, useState } from "react";
import { ProjectLabel, ProjectMember, ProjectStatus, RecurringTask, TaskPriority } from "@/types";
import { api } from "@/lib/api";
import { X, Plus, Trash2, Tag, Repeat, Columns, AlertCircle } from "lucide-react";

interface WorkflowSettingsModalProps {
  projectId: string;
  isOpen: boolean;
  onClose: () => void;
  statuses: ProjectStatus[];
  labels: ProjectLabel[];
  members: ProjectMember[];
  isAdmin: boolean;
  onChanged: () => void;
}

export function WorkflowSettingsModal({
  projectId,
  isOpen,
  onClose,
  statuses,
  labels,
  members,
  isAdmin,
  onChanged,
}: WorkflowSettingsModalProps) {
  const [tab, setTab] = useState<"statuses" | "labels" | "recurring">("statuses");
  const [error, setError] = useState<string | null>(null);

  // Status form
  const [statusKey, setStatusKey] = useState("");
  const [statusName, setStatusName] = useState("");
  const [statusColor, setStatusColor] = useState("amber");

  // Label form
  const [labelName, setLabelName] = useState("");
  const [labelColor, setLabelColor] = useState("blue");

  // Recurring tasks
  const [recurringList, setRecurringList] = useState<RecurringTask[]>([]);
  const [recTitle, setRecTitle] = useState("");
  const [recFreq, setRecFreq] = useState<"DAILY" | "WEEKLY" | "MONTHLY">("WEEKLY");
  const [recInterval, setRecInterval] = useState(1);
  const [recPriority, setRecPriority] = useState<TaskPriority>("MEDIUM");
  const [recStart, setRecStart] = useState(() => new Date().toISOString().split("T")[0]);
  const [recAssignee, setRecAssignee] = useState("");

  useEffect(() => {
    if (isOpen) {
      api.listRecurringTasks(projectId).then(setRecurringList).catch(() => {});
    }
  }, [isOpen, projectId]);

  if (!isOpen) return null;

  const handleAddStatus = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!statusKey.trim() || !statusName.trim()) return;
    setError(null);
    try {
      await api.createStatus(projectId, {
        key: statusKey.trim(),
        name: statusName.trim(),
        color: statusColor,
        position: statuses.length,
        is_done: false,
        allowed_transitions: [],
      });
      setStatusKey("");
      setStatusName("");
      onChanged();
    } catch (err: any) {
      setError(err.message || "Failed to add status");
    }
  };

  const handleDeleteStatus = async (id: string) => {
    setError(null);
    try {
      await api.deleteStatus(projectId, id);
      onChanged();
    } catch (err: any) {
      setError(err.message || "Failed to delete status");
    }
  };

  const handleAddLabel = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!labelName.trim()) return;
    setError(null);
    try {
      await api.createLabel(projectId, labelName.trim(), labelColor);
      setLabelName("");
      onChanged();
    } catch (err: any) {
      setError(err.message || "Failed to create label");
    }
  };

  const handleDeleteLabel = async (id: string) => {
    setError(null);
    try {
      await api.deleteLabel(projectId, id);
      onChanged();
    } catch (err: any) {
      setError(err.message || "Failed to delete label");
    }
  };

  const handleCreateRecurring = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!recTitle.trim()) return;
    setError(null);
    try {
      const created = await api.createRecurringTask(projectId, {
        title: recTitle.trim(),
        frequency: recFreq,
        interval_value: recInterval,
        priority: recPriority,
        start_date: new Date(recStart).toISOString(),
        assigned_to: recAssignee || undefined,
      });
      setRecurringList((prev) => [created, ...prev]);
      setRecTitle("");
    } catch (err: any) {
      setError(err.message || "Failed to create recurring task");
    }
  };

  const handleToggleRecurring = async (rt: RecurringTask) => {
    try {
      const updated = await api.toggleRecurringTask(projectId, rt.id, !rt.is_active);
      setRecurringList((prev) => prev.map((item) => (item.id === rt.id ? updated : item)));
    } catch (err: any) {
      setError(err.message || "Failed to toggle recurring task");
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/70 backdrop-blur-sm">
      <div className="bg-slate-900 border border-slate-800 rounded-2xl w-full max-w-2xl shadow-2xl overflow-hidden flex flex-col max-h-[85vh]">
        <div className="flex items-center justify-between px-6 py-4 border-b border-slate-800">
          <h3 className="font-semibold text-slate-100 text-base">Workflow, Labels &amp; Recurring Tasks</h3>
          <button onClick={onClose} className="p-1.5 text-slate-400 hover:text-white rounded-lg">
            <X className="w-5 h-5" />
          </button>
        </div>

        <div className="flex gap-2 px-6 pt-3 border-b border-slate-800 text-xs">
          {[
            { id: "statuses" as const, label: "Workflow Statuses", icon: Columns },
            { id: "labels" as const, label: "Labels", icon: Tag },
            { id: "recurring" as const, label: "Recurring Tasks", icon: Repeat },
          ].map((t) => {
            const Icon = t.icon;
            return (
              <button
                key={t.id}
                onClick={() => setTab(t.id)}
                className={`px-3 py-2 border-b-2 flex items-center gap-1.5 font-medium transition ${
                  tab === t.id ? "border-blue-500 text-blue-400" : "border-transparent text-slate-400 hover:text-slate-200"
                }`}
              >
                <Icon className="w-3.5 h-3.5" />
                <span>{t.label}</span>
              </button>
            );
          })}
        </div>

        {error && (
          <div className="px-6 py-2.5 bg-red-500/10 border-b border-red-500/20 text-red-400 text-xs flex items-center gap-2">
            <AlertCircle className="w-4 h-4 shrink-0" />
            <span>{error}</span>
          </div>
        )}

        <div className="p-6 overflow-y-auto space-y-4 text-xs">
          {tab === "statuses" && (
            <>
              <div className="space-y-2">
                {statuses.map((s) => (
                  <div key={s.id} className="flex items-center justify-between bg-slate-800/60 border border-slate-700/50 rounded-xl px-4 py-2.5">
                    <div>
                      <span className="font-semibold text-slate-100">{s.name}</span>
                      <span className="ml-2 font-mono text-[11px] text-slate-400">({s.key})</span>
                    </div>
                    {isAdmin && (
                      <button onClick={() => handleDeleteStatus(s.id)} className="text-slate-500 hover:text-red-400 p-1">
                        <Trash2 className="w-3.5 h-3.5" />
                      </button>
                    )}
                  </div>
                ))}
              </div>

              {isAdmin && (
                <form onSubmit={handleAddStatus} className="flex flex-wrap gap-2 pt-2 border-t border-slate-800">
                  <input
                    type="text"
                    placeholder="Key (e.g. BLOCKED)"
                    value={statusKey}
                    onChange={(e) => setStatusKey(e.target.value)}
                    className="bg-slate-800 border border-slate-700 rounded-lg px-3 py-1.5 text-slate-100 flex-1"
                  />
                  <input
                    type="text"
                    placeholder="Label (e.g. Blocked)"
                    value={statusName}
                    onChange={(e) => setStatusName(e.target.value)}
                    className="bg-slate-800 border border-slate-700 rounded-lg px-3 py-1.5 text-slate-100 flex-1"
                  />
                  <select
                    value={statusColor}
                    onChange={(e) => setStatusColor(e.target.value)}
                    className="bg-slate-800 border border-slate-700 rounded-lg px-2 py-1.5 text-slate-200"
                  >
                    <option value="slate">Slate</option>
                    <option value="blue">Blue</option>
                    <option value="amber">Amber</option>
                    <option value="purple">Purple</option>
                    <option value="emerald">Emerald</option>
                  </select>
                  <button type="submit" className="px-3 py-1.5 rounded-lg bg-blue-600 hover:bg-blue-500 text-white font-medium flex items-center gap-1">
                    <Plus className="w-3.5 h-3.5" />
                    <span>Add Status</span>
                  </button>
                </form>
              )}
            </>
          )}

          {tab === "labels" && (
            <>
              <div className="flex flex-wrap gap-2">
                {labels.map((l) => (
                  <div key={l.id} className="flex items-center gap-1.5 bg-slate-800 border border-slate-700 rounded-full px-3 py-1">
                    <span className="text-slate-200 font-medium">{l.name}</span>
                    {isAdmin && (
                      <button onClick={() => handleDeleteLabel(l.id)} className="text-slate-500 hover:text-red-400">
                        <X className="w-3 h-3" />
                      </button>
                    )}
                  </div>
                ))}
              </div>
              <form onSubmit={handleAddLabel} className="flex gap-2 pt-2 border-t border-slate-800">
                <input
                  type="text"
                  placeholder="New label name..."
                  value={labelName}
                  onChange={(e) => setLabelName(e.target.value)}
                  className="flex-1 bg-slate-800 border border-slate-700 rounded-lg px-3 py-1.5 text-slate-100"
                />
                <select
                  value={labelColor}
                  onChange={(e) => setLabelColor(e.target.value)}
                  className="bg-slate-800 border border-slate-700 rounded-lg px-2 py-1.5 text-slate-200"
                >
                  <option value="blue">Blue</option>
                  <option value="emerald">Green</option>
                  <option value="amber">Amber</option>
                  <option value="red">Red</option>
                  <option value="purple">Purple</option>
                </select>
                <button type="submit" className="px-3 py-1.5 rounded-lg bg-blue-600 hover:bg-blue-500 text-white font-medium">
                  Add Label
                </button>
              </form>
            </>
          )}

          {tab === "recurring" && (
            <>
              <div className="space-y-2">
                {recurringList.map((rt) => (
                  <div key={rt.id} className="flex items-center justify-between bg-slate-800/60 border border-slate-700/50 rounded-xl px-4 py-2.5">
                    <div>
                      <p className="font-semibold text-slate-100">{rt.title}</p>
                      <p className="text-[11px] text-slate-400">
                        Every {rt.interval_value} {rt.frequency.toLowerCase()} &middot; Next run:{" "}
                        {new Date(rt.next_run_at).toLocaleDateString()}
                      </p>
                    </div>
                    <button
                      onClick={() => handleToggleRecurring(rt)}
                      className={`px-2.5 py-1 rounded-lg border text-[11px] font-medium ${
                        rt.is_active
                          ? "bg-emerald-500/15 text-emerald-400 border-emerald-500/30"
                          : "bg-slate-800 text-slate-400 border-slate-700"
                      }`}
                    >
                      {rt.is_active ? "Active" : "Paused"}
                    </button>
                  </div>
                ))}
              </div>

              <form onSubmit={handleCreateRecurring} className="space-y-2 pt-3 border-t border-slate-800">
                <input
                  type="text"
                  required
                  placeholder="Recurring task title..."
                  value={recTitle}
                  onChange={(e) => setRecTitle(e.target.value)}
                  className="w-full bg-slate-800 border border-slate-700 rounded-lg px-3 py-1.5 text-slate-100"
                />
                <div className="grid grid-cols-2 sm:grid-cols-4 gap-2">
                  <select
                    value={recFreq}
                    onChange={(e) => setRecFreq(e.target.value as any)}
                    className="bg-slate-800 border border-slate-700 rounded-lg px-2 py-1.5 text-slate-200"
                  >
                    <option value="DAILY">Daily</option>
                    <option value="WEEKLY">Weekly</option>
                    <option value="MONTHLY">Monthly</option>
                  </select>
                  <input
                    type="number"
                    min={1}
                    value={recInterval}
                    onChange={(e) => setRecInterval(Number(e.target.value) || 1)}
                    className="bg-slate-800 border border-slate-700 rounded-lg px-2 py-1.5 text-slate-200"
                    title="Interval"
                  />
                  <select
                    value={recPriority}
                    onChange={(e) => setRecPriority(e.target.value as TaskPriority)}
                    className="bg-slate-800 border border-slate-700 rounded-lg px-2 py-1.5 text-slate-200"
                  >
                    <option value="LOW">Low</option>
                    <option value="MEDIUM">Medium</option>
                    <option value="HIGH">High</option>
                  </select>
                  <input
                    type="date"
                    value={recStart}
                    onChange={(e) => setRecStart(e.target.value)}
                    className="bg-slate-800 border border-slate-700 rounded-lg px-2 py-1.5 text-slate-200"
                  />
                </div>
                <div className="flex gap-2">
                  <select
                    value={recAssignee}
                    onChange={(e) => setRecAssignee(e.target.value)}
                    className="flex-1 bg-slate-800 border border-slate-700 rounded-lg px-2 py-1.5 text-slate-200"
                  >
                    <option value="">Unassigned</option>
                    {members.map((m) => (
                      <option key={m.user_id} value={m.user_id}>
                        {m.user_name || m.user_email}
                      </option>
                    ))}
                  </select>
                  <button type="submit" className="px-4 py-1.5 rounded-lg bg-blue-600 hover:bg-blue-500 text-white font-medium">
                    Create Schedule
                  </button>
                </div>
              </form>
            </>
          )}
        </div>
      </div>
    </div>
  );
}
