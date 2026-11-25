"use client";

import React, { useEffect, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { Project, DashboardAnalytics } from "@/types";
import { api } from "@/lib/api";
import { useAuth } from "@/hooks/useAuth";
import { CreateProjectModal } from "@/components/projects/CreateProjectModal";
import { Plus, FolderKanban, ArrowUpRight, Clock, CheckCircle2, AlertTriangle, ListTodo } from "lucide-react";

export default function DashboardPage() {
  const { user, loading: authLoading } = useAuth();
  const router = useRouter();

  const [projects, setProjects] = useState<Project[]>([]);
  const [analytics, setAnalytics] = useState<DashboardAnalytics | null>(null);
  const [loading, setLoading] = useState(true);
  const [isCreateOpen, setIsCreateOpen] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!authLoading && !user) {
      router.push("/login");
    }
  }, [user, authLoading, router]);

  useEffect(() => {
    if (user) {
      loadProjects();
    }
  }, [user]);

  const loadProjects = async () => {
    setLoading(true);
    setError(null);
    try {
      const [data, dashStats] = await Promise.all([
        api.listProjects(),
        api.getDashboardAnalytics().catch(() => null),
      ]);
      setProjects(data);
      setAnalytics(dashStats);
    } catch (err: any) {
      setError(err.message || "Failed to load projects");
    } finally {
      setLoading(false);
    }
  };

  const handleProjectCreated = (newProject: Project) => {
    setProjects((prev) => [newProject, ...prev]);
  };

  if (authLoading || (loading && projects.length === 0)) {
    return (
      <div className="flex-1 flex items-center justify-center p-8">
        <div className="w-8 h-8 border-2 border-blue-500 border-t-transparent rounded-full animate-spin" />
      </div>
    );
  }

  return (
    <div className="max-w-7xl mx-auto px-6 py-8 w-full space-y-8 flex-1">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold text-white tracking-tight">Projects Dashboard</h1>
          <p className="text-xs text-slate-400 mt-1">
            Manage your collaborative team workspaces and track progress
          </p>
        </div>

        <button
          onClick={() => setIsCreateOpen(true)}
          className="inline-flex items-center gap-2 px-4 py-2 rounded-xl bg-blue-600 hover:bg-blue-500 text-white font-medium text-sm transition shadow-lg shadow-blue-600/20"
        >
          <Plus className="w-4 h-4" />
          <span>New Project</span>
        </button>
      </div>

      {error && (
        <div className="p-4 bg-red-500/10 border border-red-500/20 text-red-400 text-sm rounded-xl">
          {error}
        </div>
      )}

      {/* Overall Dashboard Analytics Summary */}
      {analytics && (
        <div className="grid grid-cols-2 sm:grid-cols-4 gap-4">
          <div className="bg-slate-900/80 border border-slate-800 rounded-2xl p-4 space-y-1">
            <div className="flex items-center justify-between text-xs text-slate-400">
              <span>Total Tasks</span>
              <ListTodo className="w-4 h-4 text-blue-400" />
            </div>
            <p className="text-2xl font-bold text-white">{analytics.total_tasks}</p>
          </div>
          <div className="bg-slate-900/80 border border-slate-800 rounded-2xl p-4 space-y-1">
            <div className="flex items-center justify-between text-xs text-slate-400">
              <span>Completed</span>
              <CheckCircle2 className="w-4 h-4 text-emerald-400" />
            </div>
            <p className="text-2xl font-bold text-emerald-400">{analytics.completed_tasks}</p>
            <span className="text-[11px] text-slate-500">{analytics.completion_percent}% overall</span>
          </div>
          <div className="bg-slate-900/80 border border-slate-800 rounded-2xl p-4 space-y-1">
            <div className="flex items-center justify-between text-xs text-slate-400">
              <span>In Progress</span>
              <Clock className="w-4 h-4 text-blue-400" />
            </div>
            <p className="text-2xl font-bold text-blue-400">{analytics.in_progress_tasks}</p>
          </div>
          <div className="bg-slate-900/80 border border-slate-800 rounded-2xl p-4 space-y-1">
            <div className="flex items-center justify-between text-xs text-slate-400">
              <span>Overdue</span>
              <AlertTriangle className="w-4 h-4 text-red-400" />
            </div>
            <p className="text-2xl font-bold text-red-400">{analytics.overdue_tasks}</p>
          </div>
        </div>
      )}

      {/* Projects Grid */}
      {projects.length === 0 ? (
        <div className="bg-slate-900/60 border border-slate-800 rounded-3xl p-12 text-center max-w-lg mx-auto space-y-4">
          <div className="w-12 h-12 rounded-2xl bg-slate-800 text-slate-400 mx-auto flex items-center justify-center">
            <FolderKanban className="w-6 h-6" />
          </div>
          <div className="space-y-1">
            <h3 className="font-semibold text-slate-200 text-base">No projects found</h3>
            <p className="text-xs text-slate-400 max-w-xs mx-auto">
              Get started by creating your first workspace or ask your team admin to invite you.
            </p>
          </div>
          <button
            onClick={() => setIsCreateOpen(true)}
            className="inline-flex items-center gap-1.5 px-4 py-2 rounded-xl bg-blue-600 hover:bg-blue-500 text-white text-xs font-medium transition"
          >
            <Plus className="w-4 h-4" />
            <span>Create Workspace</span>
          </button>
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {projects.map((project) => (
            <Link
              key={project.id}
              href={`/projects/${project.id}`}
              className="group bg-slate-900/80 hover:bg-slate-800/80 border border-slate-800 hover:border-blue-500/50 rounded-2xl p-6 transition-all duration-200 flex flex-col justify-between hover:shadow-xl hover:shadow-blue-500/5 hover:-translate-y-1"
            >
              <div className="space-y-3">
                <div className="flex items-start justify-between">
                  <div className="w-10 h-10 rounded-xl bg-gradient-to-tr from-blue-600/20 to-indigo-600/20 border border-blue-500/20 flex items-center justify-center text-blue-400">
                    <FolderKanban className="w-5 h-5" />
                  </div>
                  <span
                    className={`text-[11px] font-semibold px-2 py-0.5 rounded-full border ${
                      project.user_role === "admin"
                        ? "bg-purple-500/10 text-purple-400 border-purple-500/20"
                        : "bg-slate-800 text-slate-300 border-slate-700"
                    }`}
                  >
                    {(project.user_role || "member").toUpperCase()}
                  </span>
                </div>

                <div>
                  <h3 className="font-semibold text-slate-100 text-base group-hover:text-blue-400 transition flex items-center justify-between">
                    <span>{project.name}</span>
                    <ArrowUpRight className="w-4 h-4 text-slate-500 group-hover:text-blue-400 transition" />
                  </h3>
                  <p className="text-xs text-slate-400 mt-1 line-clamp-2 leading-relaxed">
                    {project.description || "No description provided."}
                  </p>
                </div>
              </div>

              <div className="pt-6 border-t border-slate-800/80 mt-4 flex items-center justify-between text-xs text-slate-400">
                <div className="flex items-center gap-1.5">
                  <Clock className="w-3.5 h-3.5 text-slate-500" />
                  <span>{new Date(project.created_at).toLocaleDateString()}</span>
                </div>
                <span className="text-[11px] text-blue-400 font-medium group-hover:underline">
                  View Tasks &rarr;
                </span>
              </div>
            </Link>
          ))}
        </div>
      )}

      {/* Modal */}
      <CreateProjectModal
        isOpen={isCreateOpen}
        onClose={() => setIsCreateOpen(false)}
        onProjectCreated={handleProjectCreated}
      />
    </div>
  );
}
