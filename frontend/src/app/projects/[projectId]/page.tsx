"use client";

import React, { Suspense, useEffect, useState } from "react";
import { useParams, useRouter, useSearchParams } from "next/navigation";
import { Project, ProjectMember, Task, RealtimeEvent, TaskDependency, ProjectStatus, ProjectLabel } from "@/types";
import { api } from "@/lib/api";
import { realtime } from "@/lib/websocket";
import { useAuth } from "@/hooks/useAuth";
import { TaskCard } from "@/components/tasks/TaskCard";
import { TaskModal } from "@/components/tasks/TaskModal";
import { CreateTaskModal } from "@/components/tasks/CreateTaskModal";
import { MembersModal } from "@/components/projects/MembersModal";
import { ActivityLogPanel } from "@/components/projects/ActivityLogPanel";
import { CalendarView } from "@/components/projects/CalendarView";
import { TimelineView } from "@/components/projects/TimelineView";
import { AnalyticsPanel } from "@/components/projects/AnalyticsPanel";
import { WorkflowSettingsModal } from "@/components/projects/WorkflowSettingsModal";
import {
  Users,
  Plus,
  Trash2,
  Search,
  RotateCcw,
  LayoutGrid,
  List as ListIcon,
  History,
  AlertTriangle,
  Calendar as CalendarIcon,
  GitBranch,
  BarChart3,
  Settings2,
} from "lucide-react";

type ProjectView = "board" | "list" | "calendar" | "timeline" | "activity" | "reports";

function ProjectDetailsContent() {
  const { projectId } = useParams() as { projectId: string };
  const searchParams = useSearchParams();
  const { user, loading: authLoading } = useAuth();
  const router = useRouter();

  const [project, setProject] = useState<Project | null>(null);
  const [members, setMembers] = useState<ProjectMember[]>([]);
  const [tasks, setTasks] = useState<Task[]>([]);
  const [dependencies, setDependencies] = useState<TaskDependency[]>([]);
  const [statuses, setStatuses] = useState<ProjectStatus[]>([]);
  const [labels, setLabels] = useState<ProjectLabel[]>([]);
  const [totalTasks, setTotalTasks] = useState(0);
  const [page, setPage] = useState(1);
  const [totalPages, setTotalPages] = useState(1);
  const [onlineUserIds, setOnlineUserIds] = useState<Set<string>>(new Set());

  const [loading, setLoading] = useState(true);
  const [activeView, setActiveView] = useState<ProjectView>("board");
  const [selectedTask, setSelectedTask] = useState<Task | null>(null);
  const [isCreateTaskOpen, setIsCreateTaskOpen] = useState(false);
  const [isMembersOpen, setIsMembersOpen] = useState(false);
  const [isWorkflowOpen, setIsWorkflowOpen] = useState(false);

  // Backend Filters
  const [searchQuery, setSearchQuery] = useState("");
  const [statusFilter, setStatusFilter] = useState<string>("ALL");
  const [priorityFilter, setPriorityFilter] = useState<string>("ALL");
  const [assigneeFilter, setAssigneeFilter] = useState<string>("ALL");
  const [labelFilter, setLabelFilter] = useState<string>("ALL");
  const [overdueOnly, setOverdueOnly] = useState(false);
  const [dueBefore, setDueBefore] = useState("");
  const [sortBy, setSortBy] = useState("newest");

  useEffect(() => {
    if (!authLoading && !user) {
      router.push("/login");
    }
  }, [user, authLoading, router]);

  const fetchTasks = async (targetPage = page) => {
    try {
      const taskRes = await api.listTasks(projectId, {
        search: searchQuery.trim() || undefined,
        status: statusFilter !== "ALL" ? statusFilter : undefined,
        priority: priorityFilter !== "ALL" ? priorityFilter : undefined,
        assignee: assigneeFilter !== "ALL" && assigneeFilter !== "UNASSIGNED" ? assigneeFilter : undefined,
        label: labelFilter !== "ALL" ? labelFilter : undefined,
        unassigned: assigneeFilter === "UNASSIGNED",
        overdue: overdueOnly,
        due_before: dueBefore || undefined,
        sort_by: sortBy,
        page: targetPage,
        limit: 50,
      });
      setTasks(taskRes.tasks || []);
      setTotalTasks(taskRes.pagination?.total_items || 0);
      setTotalPages(taskRes.pagination?.total_pages || 1);
    } catch (err) {
      console.error("Failed to fetch tasks:", err);
    }
  };

  // Load project, members, tasks, and presence
  const loadProjectData = async () => {
    try {
      const [proj, mems, presenceRes, depsRes, statusRes, labelRes] = await Promise.all([
        api.getProject(projectId),
        api.listMembers(projectId),
        api.getPresence(projectId).catch(() => ({ project_id: projectId, online_user_ids: [] })),
        api.listProjectDependencies(projectId).catch(() => []),
        api.listStatuses(projectId).catch(() => []),
        api.listLabels(projectId).catch(() => []),
      ]);

      setProject(proj);
      setMembers(mems);
      setOnlineUserIds(new Set(presenceRes.online_user_ids || []));
      setDependencies(depsRes || []);
      setStatuses(statusRes || []);
      setLabels(labelRes || []);
      await fetchTasks(1);
    } catch (err: any) {
      console.error("Failed to load project:", err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (user && projectId && !loading) {
      const timer = setTimeout(() => {
        fetchTasks(page);
      }, 200);
      return () => clearTimeout(timer);
    }
  }, [searchQuery, statusFilter, priorityFilter, assigneeFilter, labelFilter, overdueOnly, dueBefore, sortBy, page]);

  useEffect(() => {
    const taskIdParam = searchParams.get("taskId");
    if (taskIdParam && user && projectId) {
      api.getTask(projectId, taskIdParam).then((t) => setSelectedTask(t)).catch(() => {});
    }
  }, [searchParams, user, projectId]);

  useEffect(() => {
    if (user && projectId) {
      loadProjectData();

      // Subscribe to real-time events for this project
      realtime.subscribeProject(projectId);

      const unsubscribe = realtime.onEvent((event: RealtimeEvent) => {
        if (event.projectId !== projectId) return;

        switch (event.event) {
          case "task.created":
            setTasks((prev) => {
              if (prev.some((t) => t.id === event.payload.id)) return prev;
              return [event.payload, ...prev];
            });
            break;

          case "task.updated":
          case "task.status_changed":
          case "task.assigned":
            const updatedTask = event.payload.task || event.payload;
            setTasks((prev) => prev.map((t) => (t.id === updatedTask.id ? updatedTask : t)));
            setSelectedTask((prev) => (prev && prev.id === updatedTask.id ? updatedTask : prev));
            break;

          case "task.deleted":
            const deletedId = event.payload.task_id;
            setTasks((prev) => prev.filter((t) => t.id !== deletedId));
            setSelectedTask((prev) => (prev && prev.id === deletedId ? null : prev));
            break;

          case "project.member_added":
            setMembers((prev) => {
              if (prev.some((m) => m.user_id === event.payload.user_id)) return prev;
              return [...prev, event.payload];
            });
            break;

          case "project.member_removed":
            setMembers((prev) => prev.filter((m) => m.user_id !== event.payload.user_id));
            break;

          case "project.role_changed":
            setMembers((prev) =>
              prev.map((m) => (m.user_id === event.payload.user_id ? event.payload : m))
            );
            if (user?.id === event.payload.user_id) {
              setProject((prev) => (prev ? { ...prev, user_role: event.payload.role } : prev));
            }
            break;

          case "presence.updated":
            const { user_id, is_online } = event.payload;
            setOnlineUserIds((prev) => {
              const next = new Set(prev);
              if (is_online) {
                next.add(user_id);
              } else {
                next.delete(user_id);
              }
              return next;
            });
            break;
        }
      });

      return () => {
        realtime.unsubscribeProject(projectId);
        unsubscribe();
      };
    }
  }, [projectId, user]);

  const resetFilters = () => {
    setSearchQuery("");
    setStatusFilter("ALL");
    setPriorityFilter("ALL");
    setAssigneeFilter("ALL");
    setLabelFilter("ALL");
    setOverdueOnly(false);
    setDueBefore("");
    setSortBy("newest");
    setPage(1);
  };

  const effectiveStatuses =
    statuses.length > 0
      ? statuses
      : [
          { id: "todo", project_id: projectId, key: "TODO", name: "To Do", color: "slate", position: 0, is_done: false, allowed_transitions: [], created_at: "" },
          { id: "in_progress", project_id: projectId, key: "IN_PROGRESS", name: "In Progress", color: "blue", position: 1, is_done: false, allowed_transitions: [], created_at: "" },
          { id: "done", project_id: projectId, key: "DONE", name: "Done", color: "emerald", position: 2, is_done: true, allowed_transitions: [], created_at: "" },
        ];

  const handleDeleteProject = async () => {
    if (!confirm("Are you sure you want to permanently delete this project and all associated tasks?")) {
      return;
    }
    try {
      await api.deleteProject(projectId);
      router.push("/dashboard");
    } catch (err: any) {
      alert(err.message || "Failed to delete project");
    }
  };

  if (loading || !project) {
    return (
      <div className="flex-1 flex items-center justify-center p-8">
        <div className="w-8 h-8 border-2 border-blue-500 border-t-transparent rounded-full animate-spin" />
      </div>
    );
  }

  const isAdmin = project.user_role === "admin";

  return (
    <div className="max-w-7xl mx-auto px-6 py-6 w-full space-y-6 flex-1 flex flex-col">
      {/* Project Header */}
      <div className="bg-slate-900/90 border border-slate-800 rounded-2xl p-6 shadow-sm">
        <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
          <div className="space-y-1.5">
            <div className="flex items-center gap-3">
              <h1 className="text-2xl font-bold text-white tracking-tight">{project.name}</h1>
              <span
                className={`text-[11px] font-semibold px-2 py-0.5 rounded-full border ${
                  isAdmin
                    ? "bg-purple-500/10 text-purple-400 border-purple-500/20"
                    : "bg-slate-800 text-slate-300 border-slate-700"
                }`}
              >
                {project.user_role?.toUpperCase()}
              </span>
            </div>
            {project.description && (
              <p className="text-xs text-slate-400 leading-relaxed max-w-3xl">
                {project.description}
              </p>
            )}
          </div>

          <div className="flex items-center gap-3 shrink-0">
            {/* Live Online Presence Bar */}
            <div className="flex items-center gap-1.5 bg-slate-800/80 border border-slate-700 px-3 py-1.5 rounded-xl text-xs text-slate-300">
              <div className="flex -space-x-2 overflow-hidden">
                {members.slice(0, 5).map((m) => {
                  const isOnline = onlineUserIds.has(m.user_id) || m.user_id === user?.id;
                  return (
                    <div
                      key={m.user_id}
                      className="relative inline-block"
                      title={`${m.user_name || "Member"} (${isOnline ? "Online" : "Offline"})`}
                    >
                      <div className="w-7 h-7 rounded-full bg-slate-700 border-2 border-slate-900 text-slate-200 font-bold flex items-center justify-center text-[10px]">
                        {(m.user_name || m.user_email || "U").charAt(0).toUpperCase()}
                      </div>
                      {isOnline && (
                        <span className="absolute bottom-0 right-0 w-2.5 h-2.5 bg-emerald-500 border-2 border-slate-900 rounded-full animate-pulse" />
                      )}
                    </div>
                  );
                })}
              </div>
              <span className="text-[11px] text-slate-400 ml-1">
                {onlineUserIds.size + 1} online
              </span>
            </div>

            <button
              onClick={() => setIsWorkflowOpen(true)}
              className="px-3 py-1.5 rounded-xl bg-slate-800 hover:bg-slate-700 border border-slate-700 text-slate-200 text-xs font-medium transition flex items-center gap-1.5"
              title="Workflow, Labels & Recurring Tasks"
            >
              <Settings2 className="w-4 h-4 text-blue-400" />
              <span className="hidden sm:inline">Workflow</span>
            </button>

            <button
              onClick={() => setIsMembersOpen(true)}
              className="px-3.5 py-1.5 rounded-xl bg-slate-800 hover:bg-slate-700 border border-slate-700 text-slate-200 text-xs font-medium transition flex items-center gap-1.5"
            >
              <Users className="w-4 h-4 text-blue-400" />
              <span>Members ({members.length})</span>
            </button>

            {isAdmin && (
              <button
                onClick={handleDeleteProject}
                className="p-2 text-slate-400 hover:text-red-400 hover:bg-red-500/10 rounded-xl transition border border-transparent hover:border-red-500/20"
                title="Delete Project"
              >
                <Trash2 className="w-4 h-4" />
              </button>
            )}

            <button
              onClick={() => setIsCreateTaskOpen(true)}
              className="px-4 py-1.5 rounded-xl bg-blue-600 hover:bg-blue-500 text-white text-xs font-medium transition shadow-md shadow-blue-600/20 flex items-center gap-1.5"
            >
              <Plus className="w-4 h-4" />
              <span>Add Task</span>
            </button>
          </div>
        </div>

        {/* View Switcher Tabs */}
        <div className="flex flex-wrap items-center gap-2 pt-4 mt-4 border-t border-slate-800/80">
          {[
            { id: "board" as const, label: "Board", icon: LayoutGrid },
            { id: "list" as const, label: "List", icon: ListIcon },
            { id: "calendar" as const, label: "Calendar", icon: CalendarIcon },
            { id: "timeline" as const, label: "Timeline", icon: GitBranch },
            { id: "activity" as const, label: "Activity", icon: History },
            { id: "reports" as const, label: "Reports", icon: BarChart3 },
          ].map((tab) => {
            const Icon = tab.icon;
            const active = activeView === tab.id;
            return (
              <button
                key={tab.id}
                onClick={() => setActiveView(tab.id)}
                className={`px-3 py-1.5 rounded-xl text-xs font-medium flex items-center gap-1.5 transition ${
                  active
                    ? "bg-blue-600 text-white shadow-sm shadow-blue-600/20"
                    : "bg-slate-800/80 text-slate-300 hover:bg-slate-700 border border-slate-700/60"
                }`}
              >
                <Icon className="w-3.5 h-3.5" />
                <span>{tab.label}</span>
              </button>
            );
          })}
        </div>

        {/* Advanced Search & Filter Controls Bar */}
        {activeView !== "activity" && (
          <div className="flex flex-wrap items-center gap-2.5 pt-4 mt-4 border-t border-slate-800/80 text-xs">
            <div className="relative flex-1 min-w-[200px]">
              <Search className="w-3.5 h-3.5 text-slate-500 absolute left-2.5 top-1/2 -translate-y-1/2" />
              <input
                type="text"
                value={searchQuery}
                onChange={(e) => {
                  setSearchQuery(e.target.value);
                  setPage(1);
                }}
                placeholder="Search tasks by title or description..."
                className="w-full bg-slate-800 border border-slate-700 rounded-lg pl-8 pr-3 py-1.5 text-slate-200 placeholder-slate-500 focus:outline-none focus:border-blue-500"
              />
            </div>

            <select
              value={statusFilter}
              onChange={(e) => {
                setStatusFilter(e.target.value);
                setPage(1);
              }}
              className="bg-slate-800 border border-slate-700 rounded-lg px-2.5 py-1.5 text-slate-300 focus:outline-none focus:border-blue-500"
            >
              <option value="ALL">All Statuses</option>
              {effectiveStatuses.map((s) => (
                <option key={s.key} value={s.key}>
                  {s.name}
                </option>
              ))}
            </select>

            {labels.length > 0 && (
              <select
                value={labelFilter}
                onChange={(e) => {
                  setLabelFilter(e.target.value);
                  setPage(1);
                }}
                className="bg-slate-800 border border-slate-700 rounded-lg px-2.5 py-1.5 text-slate-300 focus:outline-none focus:border-blue-500"
              >
                <option value="ALL">All Labels</option>
                {labels.map((l) => (
                  <option key={l.id} value={l.id}>
                    {l.name}
                  </option>
                ))}
              </select>
            )}

            <select
              value={priorityFilter}
              onChange={(e) => {
                setPriorityFilter(e.target.value);
                setPage(1);
              }}
              className="bg-slate-800 border border-slate-700 rounded-lg px-2.5 py-1.5 text-slate-300 focus:outline-none focus:border-blue-500"
            >
              <option value="ALL">All Priorities</option>
              <option value="LOW">Low</option>
              <option value="MEDIUM">Medium</option>
              <option value="HIGH">High</option>
            </select>

            <select
              value={assigneeFilter}
              onChange={(e) => {
                setAssigneeFilter(e.target.value);
                setPage(1);
              }}
              className="bg-slate-800 border border-slate-700 rounded-lg px-2.5 py-1.5 text-slate-300 focus:outline-none focus:border-blue-500"
            >
              <option value="ALL">All Assignees</option>
              <option value="UNASSIGNED">Unassigned</option>
              {members.map((m) => (
                <option key={m.user_id} value={m.user_id}>
                  {m.user_name || m.user_email}
                </option>
              ))}
            </select>

            <input
              type="date"
              value={dueBefore}
              onChange={(e) => {
                setDueBefore(e.target.value);
                setPage(1);
              }}
              title="Due on or before"
              className="bg-slate-800 border border-slate-700 rounded-lg px-2.5 py-1.5 text-slate-300 focus:outline-none focus:border-blue-500"
            />

            <button
              type="button"
              onClick={() => {
                setOverdueOnly((v) => !v);
                setPage(1);
              }}
              className={`px-2.5 py-1.5 rounded-lg border flex items-center gap-1 transition ${
                overdueOnly
                  ? "bg-red-500/15 text-red-400 border-red-500/30"
                  : "bg-slate-800 text-slate-300 border-slate-700 hover:bg-slate-700"
              }`}
            >
              <AlertTriangle className="w-3.5 h-3.5" />
              <span>Overdue</span>
            </button>

            <select
              value={sortBy}
              onChange={(e) => {
                setSortBy(e.target.value);
                setPage(1);
              }}
              className="bg-slate-800 border border-slate-700 rounded-lg px-2.5 py-1.5 text-slate-300 focus:outline-none focus:border-blue-500"
            >
              <option value="newest">Sort: Newest</option>
              <option value="oldest">Sort: Oldest</option>
              <option value="priority">Sort: Priority</option>
              <option value="due_date">Sort: Due Date</option>
            </select>

            <button
              type="button"
              onClick={resetFilters}
              className="px-2.5 py-1.5 rounded-lg bg-slate-800 hover:bg-slate-700 border border-slate-700 text-slate-300 flex items-center gap-1 transition"
              title="Reset all filters"
            >
              <RotateCcw className="w-3.5 h-3.5" />
              <span>Reset</span>
            </button>

            <span className="text-slate-500 text-[11px] ml-auto">
              {totalTasks} task{totalTasks === 1 ? "" : "s"}
            </span>
          </div>
        )}
      </div>

      {activeView === "activity" ? (
        <ActivityLogPanel projectId={projectId} />
      ) : activeView === "calendar" ? (
        <CalendarView tasks={tasks} onSelectTask={(t) => setSelectedTask(t)} />
      ) : activeView === "timeline" ? (
        <TimelineView tasks={tasks} dependencies={dependencies} onSelectTask={(t) => setSelectedTask(t)} />
      ) : activeView === "reports" ? (
        <AnalyticsPanel projectId={projectId} />
      ) : activeView === "list" ? (
        <div className="bg-slate-900/90 border border-slate-800 rounded-2xl overflow-hidden">
          <div className="overflow-x-auto">
            <table className="w-full text-left text-xs">
              <thead className="bg-slate-800/60 text-slate-400 uppercase tracking-wider border-b border-slate-800">
                <tr>
                  <th className="px-4 py-3">Task</th>
                  <th className="px-4 py-3">Status</th>
                  <th className="px-4 py-3">Priority</th>
                  <th className="px-4 py-3">Assignee</th>
                  <th className="px-4 py-3">Subtasks</th>
                  <th className="px-4 py-3">Due Date</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-800/70">
                {tasks.length === 0 ? (
                  <tr>
                    <td colSpan={6} className="px-4 py-10 text-center text-slate-500">
                      No tasks match your filters.
                    </td>
                  </tr>
                ) : (
                  tasks.map((task) => {
                    const isOverdue = task.due_date && task.status !== "DONE" && new Date(task.due_date).getTime() < Date.now();
                    return (
                      <tr
                        key={task.id}
                        onClick={() => setSelectedTask(task)}
                        className="hover:bg-slate-800/50 cursor-pointer transition"
                      >
                        <td className="px-4 py-3 font-medium text-slate-100">{task.title}</td>
                        <td className="px-4 py-3">
                          <span className="px-2 py-0.5 rounded bg-slate-800 border border-slate-700 text-slate-300">
                            {task.status}
                          </span>
                        </td>
                        <td className="px-4 py-3 text-slate-300">{task.priority}</td>
                        <td className="px-4 py-3 text-slate-300">{task.assignee_name || "Unassigned"}</td>
                        <td className="px-4 py-3 text-slate-400">
                          {task.subtask_count ? `${task.completed_subtask_count || 0}/${task.subtask_count} (${task.completion_percent || 0}%)` : "—"}
                        </td>
                        <td className={`px-4 py-3 ${isOverdue ? "text-red-400 font-medium" : "text-slate-400"}`}>
                          {task.due_date ? new Date(task.due_date).toLocaleDateString() : "—"}
                        </td>
                      </tr>
                    );
                  })
                )}
              </tbody>
            </table>
          </div>
        </div>
      ) : (
      /* Dynamic Kanban Board Columns */
      <div className="grid grid-cols-1 md:grid-cols-3 lg:grid-cols-4 gap-6 flex-1 items-start overflow-x-auto">
        {effectiveStatuses.map((col) => {
          const colTasks = tasks.filter((t) => t.status === col.key);
          return (
            <div key={col.key} className="bg-slate-900/60 border border-slate-800/80 rounded-2xl p-4 flex flex-col min-h-[500px]">
              <div className="flex items-center justify-between pb-3 mb-3 border-b border-slate-800">
                <div className="flex items-center gap-2">
                  <div className={`w-2.5 h-2.5 rounded-full ${col.is_done ? "bg-emerald-500" : col.key === "IN_PROGRESS" ? "bg-blue-500" : "bg-slate-400"}`} />
                  <h3 className="font-semibold text-slate-200 text-sm">{col.name}</h3>
                </div>
                <span className="text-xs bg-slate-800 text-slate-400 px-2 py-0.5 rounded-full font-medium">
                  {colTasks.length}
                </span>
              </div>

              <div className="space-y-3 flex-1">
                {colTasks.map((task) => (
                  <TaskCard
                    key={task.id}
                    task={task}
                    onClick={() => setSelectedTask(task)}
                  />
                ))}
                {colTasks.length === 0 && (
                  <div className="text-center py-12 text-slate-600 text-xs border border-dashed border-slate-800/80 rounded-xl">
                    No tasks in {col.name}
                  </div>
                )}
              </div>
            </div>
          );
        })}
      </div>
      )}

      {activeView !== "activity" && totalPages > 1 && (
        <div className="flex items-center justify-between bg-slate-900/80 border border-slate-800 rounded-xl px-4 py-2.5 text-xs text-slate-400">
          <button
            disabled={page <= 1}
            onClick={() => setPage((p) => Math.max(1, p - 1))}
            className="px-3 py-1 rounded bg-slate-800 hover:bg-slate-700 disabled:opacity-40"
          >
            Previous
          </button>
          <span>Page {page} of {totalPages}</span>
          <button
            disabled={page >= totalPages}
            onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
            className="px-3 py-1 rounded bg-slate-800 hover:bg-slate-700 disabled:opacity-40"
          >
            Next
          </button>
        </div>
      )}

      {/* Modals */}
      {selectedTask && (
        <TaskModal
          task={selectedTask}
          projectId={projectId}
          members={members}
          allTasks={tasks}
          statuses={effectiveStatuses}
          projectLabels={labels}
          isOpen={!!selectedTask}
          onClose={() => setSelectedTask(null)}
          onTaskUpdated={(updated) => {
            setTasks((prev) => prev.map((t) => (t.id === updated.id ? updated : t)));
            setSelectedTask(updated);
          }}
          onTaskDeleted={(deletedId) => {
            setTasks((prev) => prev.filter((t) => t.id !== deletedId));
            setSelectedTask(null);
          }}
          userRole={project.user_role}
        />
      )}

      <CreateTaskModal
        projectId={projectId}
        members={members}
        isOpen={isCreateTaskOpen}
        onClose={() => setIsCreateTaskOpen(false)}
        onTaskCreated={(created) => {
          setTasks((prev) => [created, ...prev]);
        }}
      />

      <MembersModal
        projectId={projectId}
        members={members}
        isOpen={isMembersOpen}
        onClose={() => setIsMembersOpen(false)}
        onMemberAdded={(added) => {
          setMembers((prev) =>
            prev.some((m) => m.user_id === added.user_id) ? prev : [...prev, added]
          );
        }}
        onMemberUpdated={(updated) => {
          setMembers((prev) => prev.map((m) => (m.user_id === updated.user_id ? updated : m)));
        }}
        onMemberRemoved={(removedId) => {
          setMembers((prev) => prev.filter((m) => m.user_id !== removedId));
        }}
        userRole={project.user_role}
      />

      <WorkflowSettingsModal
        projectId={projectId}
        isOpen={isWorkflowOpen}
        onClose={() => setIsWorkflowOpen(false)}
        statuses={effectiveStatuses}
        labels={labels}
        members={members}
        isAdmin={isAdmin}
        onChanged={() => {
          api.listStatuses(projectId).then(setStatuses).catch(() => {});
          api.listLabels(projectId).then(setLabels).catch(() => {});
        }}
      />
    </div>
  );
}

export default function ProjectDetailsPage() {
  return (
    <Suspense
      fallback={
        <div className="flex-1 flex items-center justify-center p-8">
          <div className="w-8 h-8 border-2 border-blue-500 border-t-transparent rounded-full animate-spin" />
        </div>
      }
    >
      <ProjectDetailsContent />
    </Suspense>
  );
}
