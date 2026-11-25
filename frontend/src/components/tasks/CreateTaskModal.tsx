"use client";

import React, { useState } from "react";
import { ProjectMember, Task, TaskPriority, TaskStatus } from "@/types";
import { api } from "@/lib/api";
import { X, Plus, AlertCircle } from "lucide-react";

interface CreateTaskModalProps {
  projectId: string;
  members: ProjectMember[];
  isOpen: boolean;
  onClose: () => void;
  onTaskCreated: (task: Task) => void;
}

export function CreateTaskModal({
  projectId,
  members,
  isOpen,
  onClose,
  onTaskCreated,
}: CreateTaskModalProps) {
  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [status, setStatus] = useState<TaskStatus>("TODO");
  const [priority, setPriority] = useState<TaskPriority>("MEDIUM");
  const [assignedTo, setAssignedTo] = useState<string>("");
  const [dueDate, setDueDate] = useState<string>("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  if (!isOpen) return null;

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!title.trim() || loading) return;

    setLoading(true);
    setError(null);

    try {
      const taskData: any = {
        title: title.trim(),
        description: description.trim() || undefined,
        status,
        priority,
      };

      if (assignedTo) {
        taskData.assigned_to = assignedTo;
      }
      if (dueDate) {
        taskData.due_date = new Date(dueDate).toISOString();
      }

      const created = await api.createTask(projectId, taskData);
      onTaskCreated(created);
      onClose();
      // Reset
      setTitle("");
      setDescription("");
      setStatus("TODO");
      setPriority("MEDIUM");
      setAssignedTo("");
      setDueDate("");
    } catch (err: any) {
      setError(err.message || "Failed to create task");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/70 backdrop-blur-sm animate-in fade-in duration-200">
      <div className="bg-slate-900 border border-slate-800 rounded-2xl w-full max-w-lg shadow-2xl overflow-hidden">
        <div className="flex items-center justify-between px-6 py-4 border-b border-slate-800">
          <h3 className="font-semibold text-slate-100 text-base">Create New Task</h3>
          <button
            onClick={onClose}
            className="p-1.5 text-slate-400 hover:text-slate-200 hover:bg-slate-800 rounded-lg transition"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        {error && (
          <div className="px-6 py-2.5 bg-red-500/10 border-b border-red-500/20 text-red-400 text-xs flex items-center gap-2">
            <AlertCircle className="w-4 h-4 shrink-0" />
            <span>{error}</span>
          </div>
        )}

        <form onSubmit={handleSubmit} className="p-6 space-y-4">
          <div>
            <label className="text-xs font-semibold text-slate-400 block mb-1">
              Title <span className="text-red-400">*</span>
            </label>
            <input
              type="text"
              required
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              placeholder="e.g. Implement WebSocket reconnection"
              className="w-full bg-slate-800/80 border border-slate-700 rounded-lg px-3 py-2 text-slate-100 text-sm focus:outline-none focus:border-blue-500 transition"
            />
          </div>

          <div>
            <label className="text-xs font-semibold text-slate-400 block mb-1">Description</label>
            <textarea
              rows={3}
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="Add details, criteria or context..."
              className="w-full bg-slate-800/80 border border-slate-700 rounded-lg px-3 py-2 text-slate-100 text-sm focus:outline-none focus:border-blue-500 transition resize-none"
            />
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="text-xs font-semibold text-slate-400 block mb-1">Status</label>
              <select
                value={status}
                onChange={(e) => setStatus(e.target.value as TaskStatus)}
                className="w-full bg-slate-800/80 border border-slate-700 rounded-lg px-3 py-2 text-sm text-slate-200 focus:outline-none focus:border-blue-500 transition"
              >
                <option value="TODO">To Do</option>
                <option value="IN_PROGRESS">In Progress</option>
                <option value="DONE">Done</option>
              </select>
            </div>

            <div>
              <label className="text-xs font-semibold text-slate-400 block mb-1">Priority</label>
              <select
                value={priority}
                onChange={(e) => setPriority(e.target.value as TaskPriority)}
                className="w-full bg-slate-800/80 border border-slate-700 rounded-lg px-3 py-2 text-sm text-slate-200 focus:outline-none focus:border-blue-500 transition"
              >
                <option value="LOW">Low</option>
                <option value="MEDIUM">Medium</option>
                <option value="HIGH">High</option>
              </select>
            </div>

            <div>
              <label className="text-xs font-semibold text-slate-400 block mb-1">Assignee</label>
              <select
                value={assignedTo}
                onChange={(e) => setAssignedTo(e.target.value)}
                className="w-full bg-slate-800/80 border border-slate-700 rounded-lg px-3 py-2 text-sm text-slate-200 focus:outline-none focus:border-blue-500 transition"
              >
                <option value="">Unassigned</option>
                {members.map((m) => (
                  <option key={m.user_id} value={m.user_id}>
                    {m.user_name || m.user_email || m.user_id}
                  </option>
                ))}
              </select>
            </div>

            <div>
              <label className="text-xs font-semibold text-slate-400 block mb-1">Due Date</label>
              <input
                type="date"
                value={dueDate}
                onChange={(e) => setDueDate(e.target.value)}
                className="w-full bg-slate-800/80 border border-slate-700 rounded-lg px-3 py-2 text-sm text-slate-200 focus:outline-none focus:border-blue-500 transition"
              />
            </div>
          </div>

          <div className="flex items-center justify-end gap-3 pt-4 border-t border-slate-800">
            <button
              type="button"
              onClick={onClose}
              className="px-4 py-2 text-sm text-slate-400 hover:text-slate-200 transition"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={loading || !title.trim()}
              className="px-4 py-2 rounded-lg bg-blue-600 hover:bg-blue-500 disabled:opacity-50 text-white font-medium text-sm transition flex items-center gap-1.5 shadow-md shadow-blue-600/20"
            >
              <Plus className="w-4 h-4" />
              <span>{loading ? "Creating..." : "Create Task"}</span>
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
