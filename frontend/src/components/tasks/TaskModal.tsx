"use client";

import React, { useState, useEffect } from "react";
import { Task, Comment, ProjectMember, TaskStatus, TaskPriority, TaskDependenciesResponse, TaskAttachment, ProjectStatus, ProjectLabel } from "@/types";
import { api } from "@/lib/api";
import { useAuth } from "@/hooks/useAuth";
import { X, Calendar, User, Trash2, Send, Clock, AlertCircle, CheckSquare, Plus, ChevronDown, ChevronRight, GitBranch, Lock, Paperclip, Download, Tag } from "lucide-react";

interface TaskModalProps {
  task: Task;
  projectId: string;
  members: ProjectMember[];
  allTasks?: Task[];
  statuses?: ProjectStatus[];
  projectLabels?: ProjectLabel[];
  isOpen: boolean;
  onClose: () => void;
  onTaskUpdated: (task: Task) => void;
  onTaskDeleted: (taskId: string) => void;
  userRole?: string;
}

export function TaskModal({
  task,
  projectId,
  members,
  allTasks = [],
  statuses = [],
  projectLabels = [],
  isOpen,
  onClose,
  onTaskUpdated,
  onTaskDeleted,
  userRole,
}: TaskModalProps) {
  const { user } = useAuth();
  const [currentTask, setCurrentTask] = useState<Task>(task);
  const [title, setTitle] = useState(task.title);
  const [description, setDescription] = useState(task.description || "");
  const [status, setStatus] = useState<TaskStatus>(task.status);
  const [priority, setPriority] = useState<TaskPriority>(task.priority);
  const [assignedTo, setAssignedTo] = useState<string>(task.assigned_to || "");
  const [startDate, setStartDate] = useState<string>(task.start_date ? task.start_date.split("T")[0] : "");
  const [dueDate, setDueDate] = useState<string>(task.due_date ? task.due_date.split("T")[0] : "");

  const [comments, setComments] = useState<Comment[]>([]);
  const [newComment, setNewComment] = useState("");
  const [isSubmittingComment, setIsSubmittingComment] = useState(false);
  const [isUpdating, setIsUpdating] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Subtasks
  const [subtasks, setSubtasks] = useState<Task[]>([]);
  const [subtasksExpanded, setSubtasksExpanded] = useState(true);
  const [newSubtaskTitle, setNewSubtaskTitle] = useState("");
  const [newSubtaskAssignee, setNewSubtaskAssignee] = useState("");
  const [newSubtaskPriority, setNewSubtaskPriority] = useState<TaskPriority>("MEDIUM");
  const [isCreatingSubtask, setIsCreatingSubtask] = useState(false);

  // Dependencies
  const [deps, setDeps] = useState<TaskDependenciesResponse>({ prerequisites: [], dependents: [] });
  const [selectedPrereqId, setSelectedPrereqId] = useState("");

  // Attachments & Labels
  const [attachments, setAttachments] = useState<TaskAttachment[]>([]);
  const [uploadingFile, setUploadingFile] = useState(false);
  const [taskLabels, setTaskLabels] = useState<ProjectLabel[]>([]);

  useEffect(() => {
    setCurrentTask(task);
    setTitle(task.title);
    setDescription(task.description || "");
    setStatus(task.status);
    setPriority(task.priority);
    setAssignedTo(task.assigned_to || "");
    setStartDate(task.start_date ? task.start_date.split("T")[0] : "");
    setDueDate(task.due_date ? task.due_date.split("T")[0] : "");
    loadComments();
    loadDependencies();
    api.listAttachments(projectId, task.id).then(setAttachments).catch(() => {});
    api.listTaskLabels(projectId, task.id).then(setTaskLabels).catch(() => {});
    if (!task.parent_task_id) {
      loadSubtasks();
    } else {
      setSubtasks([]);
    }
  }, [task]);

  const loadDependencies = async () => {
    try {
      const data = await api.listTaskDependencies(projectId, task.id);
      setDeps(data);
    } catch {
      // ignore
    }
  };

  const loadSubtasks = async () => {
    try {
      const data = await api.listSubtasks(projectId, task.id);
      setSubtasks(data || []);
    } catch (err) {
      console.error("Failed to load subtasks:", err);
    }
  };

  const loadComments = async () => {
    try {
      const data = await api.listComments(projectId, task.id);
      setComments(data);
    } catch (err: any) {
      console.error("Failed to load comments:", err);
    }
  };

  if (!isOpen) return null;

  const handleSaveAttributes = async (e?: React.FormEvent) => {
    if (e) e.preventDefault();
    setIsUpdating(true);
    setError(null);

    try {
      const updates: any = {
        title,
        description,
        status,
        priority,
      };

      if (assignedTo === "") {
        updates.unassign = true;
      } else {
        updates.assigned_to = assignedTo;
      }

      if (startDate === "") {
        updates.clear_start_date = true;
      } else {
        updates.start_date = new Date(startDate).toISOString();
      }

      if (dueDate === "") {
        updates.clear_due_date = true;
      } else {
        updates.due_date = new Date(dueDate).toISOString();
      }

      const updated = await api.updateTask(projectId, task.id, updates);
      setCurrentTask(updated);
      onTaskUpdated(updated);
    } catch (err: any) {
      setError(err.message || "Failed to update task");
    } finally {
      setIsUpdating(false);
    }
  };

  const handleDeleteTask = async () => {
    if (!confirm("Are you sure you want to delete this task? This action cannot be undone.")) {
      return;
    }
    try {
      await api.deleteTask(projectId, task.id);
      onTaskDeleted(task.id);
      onClose();
    } catch (err: any) {
      setError(err.message || "Failed to delete task");
    }
  };

  const handleAddComment = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newComment.trim() || isSubmittingComment) return;

    setIsSubmittingComment(true);
    try {
      const comment = await api.addComment(projectId, task.id, newComment.trim());
      setComments((prev) => [...prev, comment]);
      setNewComment("");
    } catch (err: any) {
      setError(err.message || "Failed to add comment");
    } finally {
      setIsSubmittingComment(false);
    }
  };

  const handleDeleteComment = async (commentId: string) => {
    try {
      await api.deleteComment(projectId, task.id, commentId);
      setComments((prev) => prev.filter((c) => c.id !== commentId));
    } catch (err: any) {
      setError(err.message || "Failed to delete comment");
    }
  };

  const handleCreateSubtask = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newSubtaskTitle.trim() || isCreatingSubtask) return;
    setIsCreatingSubtask(true);
    setError(null);
    try {
      const payload: Partial<Task> = {
        title: newSubtaskTitle.trim(),
        priority: newSubtaskPriority,
      };
      if (newSubtaskAssignee) payload.assigned_to = newSubtaskAssignee;
      const created = await api.createSubtask(projectId, task.id, payload);
      setSubtasks((prev) => [...prev, created]);
      setNewSubtaskTitle("");
      setNewSubtaskAssignee("");
      const updatedParent = await api.getTask(projectId, task.id);
      setCurrentTask(updatedParent);
      onTaskUpdated(updatedParent);
    } catch (err: any) {
      setError(err.message || "Failed to create subtask");
    } finally {
      setIsCreatingSubtask(false);
    }
  };

  const handleToggleSubtaskStatus = async (sub: Task) => {
    const nextStatus: TaskStatus = sub.status === "DONE" ? "TODO" : "DONE";
    try {
      const updated = await api.updateTask(projectId, sub.id, { status: nextStatus });
      setSubtasks((prev) => prev.map((s) => (s.id === sub.id ? updated : s)));
      const updatedParent = await api.getTask(projectId, task.id);
      setCurrentTask(updatedParent);
      onTaskUpdated(updatedParent);
    } catch (err: any) {
      setError(err.message || "Failed to update subtask");
    }
  };

  const handleSubtaskAssign = async (sub: Task, assigneeId: string) => {
    try {
      const updated = await api.updateTask(
        projectId,
        sub.id,
        assigneeId ? { assigned_to: assigneeId } : { unassign: true }
      );
      setSubtasks((prev) => prev.map((s) => (s.id === sub.id ? updated : s)));
    } catch (err: any) {
      setError(err.message || "Failed to assign subtask");
    }
  };

  const handleDeleteSubtask = async (subId: string) => {
    try {
      await api.deleteTask(projectId, subId);
      setSubtasks((prev) => prev.filter((s) => s.id !== subId));
      const updatedParent = await api.getTask(projectId, task.id);
      setCurrentTask(updatedParent);
      onTaskUpdated(updatedParent);
    } catch (err: any) {
      setError(err.message || "Failed to delete subtask");
    }
  };

  const handleAddDependency = async () => {
    if (!selectedPrereqId) return;
    setError(null);
    try {
      await api.addTaskDependency(projectId, task.id, selectedPrereqId);
      setSelectedPrereqId("");
      await loadDependencies();
      const refreshed = await api.getTask(projectId, task.id);
      setCurrentTask(refreshed);
      onTaskUpdated(refreshed);
    } catch (err: any) {
      setError(err.message || "Failed to add dependency");
    }
  };

  const handleRemoveDependency = async (depId: string) => {
    setError(null);
    try {
      await api.removeTaskDependency(projectId, task.id, depId);
      await loadDependencies();
      const refreshed = await api.getTask(projectId, task.id);
      setCurrentTask(refreshed);
      onTaskUpdated(refreshed);
    } catch (err: any) {
      setError(err.message || "Failed to remove dependency");
    }
  };

  const handleFileUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    setUploadingFile(true);
    setError(null);
    try {
      const att = await api.uploadAttachment(projectId, task.id, file);
      setAttachments((prev) => [att, ...prev]);
    } catch (err: any) {
      setError(err.message || "Failed to upload attachment");
    } finally {
      setUploadingFile(false);
      e.target.value = "";
    }
  };

  const handleDeleteAttachment = async (attId: string) => {
    try {
      await api.deleteAttachment(projectId, task.id, attId);
      setAttachments((prev) => prev.filter((a) => a.id !== attId));
    } catch (err: any) {
      setError(err.message || "Failed to delete attachment");
    }
  };

  const handleToggleLabel = async (label: ProjectLabel) => {
    const exists = taskLabels.some((l) => l.id === label.id);
    try {
      if (exists) {
        await api.detachTaskLabel(projectId, task.id, label.id);
        setTaskLabels((prev) => prev.filter((l) => l.id !== label.id));
      } else {
        await api.attachTaskLabel(projectId, task.id, label.id);
        setTaskLabels((prev) => [...prev, label]);
      }
    } catch (err: any) {
      setError(err.message || "Failed to update label");
    }
  };

  const canDeleteTask = userRole === "admin" || user?.id === currentTask.created_by;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/70 backdrop-blur-sm animate-in fade-in duration-200">
      <div className="bg-slate-900 border border-slate-800 rounded-2xl w-full max-w-4xl max-h-[90vh] flex flex-col shadow-2xl overflow-hidden">
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-slate-800">
          <div className="flex items-center gap-2">
            <span className="text-xs font-semibold px-2 py-0.5 rounded bg-blue-500/10 text-blue-400 border border-blue-500/20">
              Task Details
            </span>
            <span className="text-xs text-slate-500">ID: {currentTask.id.substring(0, 8)}...</span>
          </div>
          <div className="flex items-center gap-2">
            {canDeleteTask && (
              <button
                onClick={handleDeleteTask}
                className="p-1.5 text-slate-400 hover:text-red-400 hover:bg-red-500/10 rounded-lg transition"
                title="Delete Task"
              >
                <Trash2 className="w-4 h-4" />
              </button>
            )}
            <button
              onClick={onClose}
              className="p-1.5 text-slate-400 hover:text-slate-200 hover:bg-slate-800 rounded-lg transition"
            >
              <X className="w-5 h-5" />
            </button>
          </div>
        </div>

        {error && (
          <div className="px-6 py-2.5 bg-red-500/10 border-b border-red-500/20 text-red-400 text-xs flex items-center gap-2">
            <AlertCircle className="w-4 h-4 shrink-0" />
            <span>{error}</span>
          </div>
        )}

        {/* Content Body: Split View */}
        <div className="flex-1 grid grid-cols-1 md:grid-cols-3 overflow-hidden">
          {/* Left Column: Attributes & Details */}
          <div className="p-6 md:col-span-2 overflow-y-auto border-b md:border-b-0 md:border-r border-slate-800 space-y-5">
            <div>
              <label className="text-xs font-semibold text-slate-400 block mb-1">Title</label>
              <input
                type="text"
                value={title}
                onChange={(e) => setTitle(e.target.value)}
                onBlur={() => handleSaveAttributes()}
                className="w-full bg-slate-800/80 border border-slate-700 rounded-lg px-3 py-2 text-slate-100 font-medium focus:outline-none focus:border-blue-500 transition"
              />
            </div>

            <div>
              <label className="text-xs font-semibold text-slate-400 block mb-1">Description</label>
              <textarea
                value={description}
                rows={4}
                onChange={(e) => setDescription(e.target.value)}
                onBlur={() => handleSaveAttributes()}
                placeholder="Add a more detailed description..."
                className="w-full bg-slate-800/80 border border-slate-700 rounded-lg px-3 py-2 text-slate-200 text-sm focus:outline-none focus:border-blue-500 transition resize-none"
              />
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="text-xs font-semibold text-slate-400 block mb-1">Status</label>
                <select
                  value={status}
                  onChange={(e) => {
                    setStatus(e.target.value as TaskStatus);
                    setTimeout(() => handleSaveAttributes(), 50);
                  }}
                  className="w-full bg-slate-800/80 border border-slate-700 rounded-lg px-3 py-2 text-sm text-slate-200 focus:outline-none focus:border-blue-500 transition"
                >
                  {statuses.length > 0 ? (
                    statuses.map((s) => (
                      <option key={s.key} value={s.key}>
                        {s.name}
                      </option>
                    ))
                  ) : (
                    <>
                      <option value="TODO">To Do</option>
                      <option value="IN_PROGRESS">In Progress</option>
                      <option value="DONE">Done</option>
                    </>
                  )}
                </select>
              </div>

              <div>
                <label className="text-xs font-semibold text-slate-400 block mb-1">Priority</label>
                <select
                  value={priority}
                  onChange={(e) => {
                    setPriority(e.target.value as TaskPriority);
                    setTimeout(() => handleSaveAttributes(), 50);
                  }}
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
                  onChange={(e) => {
                    setAssignedTo(e.target.value);
                    setTimeout(() => handleSaveAttributes(), 50);
                  }}
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
                <label className="text-xs font-semibold text-slate-400 block mb-1">Start Date</label>
                <input
                  type="date"
                  value={startDate}
                  onChange={(e) => {
                    setStartDate(e.target.value);
                    setTimeout(() => handleSaveAttributes(), 50);
                  }}
                  className="w-full bg-slate-800/80 border border-slate-700 rounded-lg px-3 py-2 text-sm text-slate-200 focus:outline-none focus:border-blue-500 transition"
                />
              </div>

              <div>
                <label className="text-xs font-semibold text-slate-400 block mb-1">Due Date</label>
                <input
                  type="date"
                  value={dueDate}
                  onChange={(e) => {
                    setDueDate(e.target.value);
                    setTimeout(() => handleSaveAttributes(), 50);
                  }}
                  className="w-full bg-slate-800/80 border border-slate-700 rounded-lg px-3 py-2 text-sm text-slate-200 focus:outline-none focus:border-blue-500 transition"
                />
              </div>
            </div>

            {/* Dependencies Section */}
            <div className="pt-4 border-t border-slate-800/80 space-y-3">
              <div className="flex items-center justify-between">
                <h4 className="text-xs font-semibold text-slate-300 flex items-center gap-1.5">
                  <GitBranch className="w-4 h-4 text-blue-400" />
                  <span>Dependencies (Finish-to-Start)</span>
                </h4>
                {currentTask.is_blocked && (
                  <span className="text-[11px] px-2 py-0.5 rounded bg-amber-500/15 text-amber-300 border border-amber-500/30 flex items-center gap-1">
                    <Lock className="w-3 h-3" />
                    <span>Blocked by prerequisite</span>
                  </span>
                )}
              </div>

              {deps.prerequisites.length > 0 && (
                <div className="space-y-1.5">
                  <span className="text-[11px] text-slate-500">Blocked by:</span>
                  {deps.prerequisites.map((d) => (
                    <div key={d.id} className="flex items-center justify-between bg-slate-800/60 border border-slate-700/50 rounded-lg px-3 py-1.5 text-xs">
                      <span className="text-slate-200 truncate">
                        {d.depends_on_title} <span className="text-[10px] text-slate-400">({d.depends_on_status})</span>
                      </span>
                      <button
                        type="button"
                        onClick={() => handleRemoveDependency(d.id)}
                        className="text-slate-500 hover:text-red-400 p-0.5"
                      >
                        <X className="w-3.5 h-3.5" />
                      </button>
                    </div>
                  ))}
                </div>
              )}

              {deps.dependents.length > 0 && (
                <div className="space-y-1">
                  <span className="text-[11px] text-slate-500">Blocking:</span>
                  <div className="flex flex-wrap gap-1.5">
                    {deps.dependents.map((d) => (
                      <span key={d.id} className="text-[11px] px-2 py-0.5 rounded bg-slate-800 border border-slate-700 text-slate-300">
                        {d.task_title}
                      </span>
                    ))}
                  </div>
                </div>
              )}

              <div className="flex gap-2">
                <select
                  value={selectedPrereqId}
                  onChange={(e) => setSelectedPrereqId(e.target.value)}
                  className="flex-1 bg-slate-800/80 border border-slate-700 rounded-lg px-2.5 py-1.5 text-xs text-slate-200"
                >
                  <option value="">Select prerequisite task...</option>
                  {allTasks
                    .filter((t) => t.id !== currentTask.id && !deps.prerequisites.some((p) => p.depends_on_task_id === t.id))
                    .map((t) => (
                      <option key={t.id} value={t.id}>
                        {t.title} ({t.status})
                      </option>
                    ))}
                </select>
                <button
                  type="button"
                  onClick={handleAddDependency}
                  disabled={!selectedPrereqId}
                  className="px-3 py-1.5 rounded-lg bg-slate-700 hover:bg-slate-600 disabled:opacity-40 text-xs text-white font-medium"
                >
                  Link
                </button>
              </div>
            </div>

            {/* Subtasks Section (only on top-level tasks) */}
            {!currentTask.parent_task_id && (
              <div className="pt-4 border-t border-slate-800/80 space-y-3">
                <button
                  type="button"
                  onClick={() => setSubtasksExpanded((v) => !v)}
                  className="w-full flex items-center justify-between text-xs font-semibold text-slate-300 hover:text-white transition"
                >
                  <span className="flex items-center gap-1.5">
                    {subtasksExpanded ? <ChevronDown className="w-4 h-4" /> : <ChevronRight className="w-4 h-4" />}
                    <CheckSquare className="w-4 h-4 text-blue-400" />
                    <span>
                      Subtasks ({subtasks.filter((s) => s.status === "DONE").length}/{subtasks.length})
                    </span>
                  </span>
                  {subtasks.length > 0 && (
                    <span className="text-[11px] text-blue-400 font-medium">
                      {Math.round((subtasks.filter((s) => s.status === "DONE").length / subtasks.length) * 100)}% complete
                    </span>
                  )}
                </button>

                {subtasks.length > 0 && (
                  <div className="w-full h-1.5 bg-slate-800 rounded-full overflow-hidden">
                    <div
                      className="h-full bg-blue-500 transition-all"
                      style={{
                        width: `${Math.round((subtasks.filter((s) => s.status === "DONE").length / subtasks.length) * 100)}%`,
                      }}
                    />
                  </div>
                )}

                {subtasksExpanded && (
                  <div className="space-y-2">
                    {subtasks.map((sub) => (
                      <div
                        key={sub.id}
                        className="flex items-center justify-between gap-2 bg-slate-800/50 border border-slate-700/50 rounded-lg px-3 py-2 text-xs"
                      >
                        <div className="flex items-center gap-2 min-w-0 flex-1">
                          <input
                            type="checkbox"
                            checked={sub.status === "DONE"}
                            onChange={() => handleToggleSubtaskStatus(sub)}
                            className="w-4 h-4 accent-blue-600 rounded shrink-0"
                          />
                          <span className={`truncate ${sub.status === "DONE" ? "line-through text-slate-500" : "text-slate-200"}`}>
                            {sub.title}
                          </span>
                        </div>

                        <div className="flex items-center gap-2 shrink-0">
                          <select
                            value={sub.assigned_to || ""}
                            onChange={(e) => handleSubtaskAssign(sub, e.target.value)}
                            className="bg-slate-800 border border-slate-700 rounded px-1.5 py-0.5 text-[11px] text-slate-300 max-w-[110px]"
                          >
                            <option value="">Unassigned</option>
                            {members.map((m) => (
                              <option key={m.user_id} value={m.user_id}>
                                {m.user_name || m.user_email}
                              </option>
                            ))}
                          </select>
                          {(userRole === "admin" || user?.id === sub.created_by) && (
                            <button
                              type="button"
                              onClick={() => handleDeleteSubtask(sub.id)}
                              className="text-slate-500 hover:text-red-400 p-0.5 transition"
                              title="Delete Subtask"
                            >
                              <Trash2 className="w-3.5 h-3.5" />
                            </button>
                          )}
                        </div>
                      </div>
                    ))}

                    <form onSubmit={handleCreateSubtask} className="flex flex-col sm:flex-row gap-2 pt-1">
                      <input
                        type="text"
                        value={newSubtaskTitle}
                        onChange={(e) => setNewSubtaskTitle(e.target.value)}
                        placeholder="Add a subtask..."
                        className="flex-1 bg-slate-800/80 border border-slate-700 rounded-lg px-2.5 py-1.5 text-xs text-slate-100 placeholder-slate-500 focus:outline-none focus:border-blue-500"
                      />
                      <select
                        value={newSubtaskPriority}
                        onChange={(e) => setNewSubtaskPriority(e.target.value as TaskPriority)}
                        className="bg-slate-800/80 border border-slate-700 rounded-lg px-2 py-1.5 text-xs text-slate-300"
                      >
                        <option value="LOW">Low</option>
                        <option value="MEDIUM">Medium</option>
                        <option value="HIGH">High</option>
                      </select>
                      <button
                        type="submit"
                        disabled={!newSubtaskTitle.trim() || isCreatingSubtask}
                        className="px-3 py-1.5 rounded-lg bg-blue-600 hover:bg-blue-500 disabled:opacity-50 text-white text-xs font-medium flex items-center justify-center gap-1"
                      >
                        <Plus className="w-3.5 h-3.5" />
                        <span>Add</span>
                      </button>
                    </form>
                  </div>
                )}
              </div>
            )}

            {/* Labels Section */}
            {projectLabels.length > 0 && (
              <div className="pt-4 border-t border-slate-800/80 space-y-2">
                <h4 className="text-xs font-semibold text-slate-300 flex items-center gap-1.5">
                  <Tag className="w-4 h-4 text-blue-400" />
                  <span>Labels</span>
                </h4>
                <div className="flex flex-wrap gap-1.5">
                  {projectLabels.map((lbl) => {
                    const active = taskLabels.some((l) => l.id === lbl.id);
                    return (
                      <button
                        key={lbl.id}
                        type="button"
                        onClick={() => handleToggleLabel(lbl)}
                        className={`px-2.5 py-1 rounded-full text-[11px] font-medium border transition ${
                          active
                            ? "bg-blue-600/25 text-blue-300 border-blue-500/40"
                            : "bg-slate-800 text-slate-400 border-slate-700 hover:text-slate-200"
                        }`}
                      >
                        {lbl.name}
                      </button>
                    );
                  })}
                </div>
              </div>
            )}

            {/* File Attachments Section */}
            <div className="pt-4 border-t border-slate-800/80 space-y-2.5">
              <div className="flex items-center justify-between">
                <h4 className="text-xs font-semibold text-slate-300 flex items-center gap-1.5">
                  <Paperclip className="w-4 h-4 text-blue-400" />
                  <span>Attachments ({attachments.length})</span>
                </h4>
                <label className="cursor-pointer px-2.5 py-1 rounded-lg bg-slate-800 hover:bg-slate-700 border border-slate-700 text-[11px] text-slate-200 font-medium">
                  <span>{uploadingFile ? "Uploading..." : "Upload File"}</span>
                  <input type="file" onChange={handleFileUpload} disabled={uploadingFile} className="hidden" />
                </label>
              </div>
              {attachments.length > 0 && (
                <div className="space-y-1.5">
                  {attachments.map((att) => (
                    <div key={att.id} className="flex items-center justify-between bg-slate-800/50 border border-slate-700/50 rounded-lg px-3 py-2 text-xs">
                      <div className="min-w-0 flex-1">
                        <p className="text-slate-200 truncate font-medium">{att.original_name}</p>
                        <p className="text-[10px] text-slate-500">
                          {(att.size_bytes / 1024).toFixed(1)} KB &middot; {new Date(att.created_at).toLocaleDateString()}
                        </p>
                      </div>
                      <div className="flex items-center gap-1.5 shrink-0">
                        <a
                          href={api.getAttachmentDownloadUrl(projectId, task.id, att.id)}
                          target="_blank"
                          rel="noreferrer"
                          className="p-1 text-slate-400 hover:text-blue-400"
                          title="Download"
                        >
                          <Download className="w-3.5 h-3.5" />
                        </a>
                        {(userRole === "admin" || user?.id === att.uploaded_by) && (
                          <button
                            type="button"
                            onClick={() => handleDeleteAttachment(att.id)}
                            className="p-1 text-slate-400 hover:text-red-400"
                            title="Remove"
                          >
                            <Trash2 className="w-3.5 h-3.5" />
                          </button>
                        )}
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>

            <div className="pt-4 border-t border-slate-800/80 text-xs text-slate-400 flex items-center justify-between">
              <span>Created by <strong className="text-slate-300">{currentTask.creator_name || "Unknown"}</strong></span>
              <span>{new Date(currentTask.created_at).toLocaleDateString()}</span>
            </div>
          </div>

          {/* Right Column: Live Comments Feed */}
          <div className="p-6 flex flex-col h-full bg-slate-900/60 overflow-hidden">
            <h4 className="text-xs font-bold text-slate-400 uppercase tracking-wider mb-3">
              Comments & Activity ({comments.length})
            </h4>

            <div className="flex-1 overflow-y-auto space-y-3 pr-1 mb-4">
              {comments.length === 0 ? (
                <div className="text-center py-8 text-slate-500 text-xs">
                  No comments yet. Start the conversation!
                </div>
              ) : (
                comments.map((comment) => {
                  const canDelete = userRole === "admin" || user?.id === comment.user_id;
                  return (
                    <div key={comment.id} className="bg-slate-800/70 border border-slate-700/50 rounded-xl p-3 text-xs space-y-1">
                      <div className="flex items-center justify-between">
                        <div className="flex items-center gap-1.5 font-medium text-slate-300">
                          <div className="w-4 h-4 rounded-full bg-indigo-500/30 text-indigo-300 font-bold flex items-center justify-center text-[9px]">
                            {(comment.user_name || "U").charAt(0).toUpperCase()}
                          </div>
                          <span>{comment.user_name || "Member"}</span>
                        </div>
                        <div className="flex items-center gap-1 text-[10px] text-slate-500">
                          <span>{new Date(comment.created_at).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })}</span>
                          {canDelete && (
                            <button
                              onClick={() => handleDeleteComment(comment.id)}
                              className="text-slate-500 hover:text-red-400 p-0.5 ml-1 transition"
                              title="Delete Comment"
                            >
                              <X className="w-3 h-3" />
                            </button>
                          )}
                        </div>
                      </div>
                      <p className="text-slate-200 leading-relaxed whitespace-pre-wrap pl-5">
                        {comment.content}
                      </p>
                    </div>
                  );
                })
              )}
            </div>

            <form onSubmit={handleAddComment} className="flex gap-2">
              <input
                type="text"
                value={newComment}
                onChange={(e) => setNewComment(e.target.value)}
                placeholder="Write a comment..."
                disabled={isSubmittingComment}
                className="flex-1 bg-slate-800 border border-slate-700 rounded-lg px-3 py-2 text-xs text-slate-100 placeholder-slate-500 focus:outline-none focus:border-blue-500 transition"
              />
              <button
                type="submit"
                disabled={!newComment.trim() || isSubmittingComment}
                className="px-3 py-2 rounded-lg bg-blue-600 hover:bg-blue-500 disabled:opacity-50 text-white transition flex items-center justify-center"
              >
                <Send className="w-3.5 h-3.5" />
              </button>
            </form>
          </div>
        </div>
      </div>
    </div>
  );
}
