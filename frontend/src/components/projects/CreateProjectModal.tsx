"use client";

import React, { useState } from "react";
import { Project } from "@/types";
import { api } from "@/lib/api";
import { X, Plus, AlertCircle } from "lucide-react";

interface CreateProjectModalProps {
  isOpen: boolean;
  onClose: () => void;
  onProjectCreated: (project: Project) => void;
}

export function CreateProjectModal({
  isOpen,
  onClose,
  onProjectCreated,
}: CreateProjectModalProps) {
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  if (!isOpen) return null;

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim() || loading) return;

    setLoading(true);
    setError(null);

    try {
      const created = await api.createProject(name.trim(), description.trim() || undefined);
      onProjectCreated(created);
      onClose();
      setName("");
      setDescription("");
    } catch (err: any) {
      setError(err.message || "Failed to create project");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/70 backdrop-blur-sm animate-in fade-in duration-200">
      <div className="bg-slate-900 border border-slate-800 rounded-2xl w-full max-w-md shadow-2xl overflow-hidden">
        <div className="flex items-center justify-between px-6 py-4 border-b border-slate-800">
          <h3 className="font-semibold text-slate-100 text-base">Create Project</h3>
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
              Project Name <span className="text-red-400">*</span>
            </label>
            <input
              type="text"
              required
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="e.g. Core Infrastructure Revamp"
              className="w-full bg-slate-800/80 border border-slate-700 rounded-lg px-3 py-2 text-slate-100 text-sm focus:outline-none focus:border-blue-500 transition"
            />
          </div>

          <div>
            <label className="text-xs font-semibold text-slate-400 block mb-1">Description</label>
            <textarea
              rows={3}
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="What is the goal of this project?"
              className="w-full bg-slate-800/80 border border-slate-700 rounded-lg px-3 py-2 text-slate-100 text-sm focus:outline-none focus:border-blue-500 transition resize-none"
            />
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
              disabled={loading || !name.trim()}
              className="px-4 py-2 rounded-lg bg-blue-600 hover:bg-blue-500 disabled:opacity-50 text-white font-medium text-sm transition flex items-center gap-1.5 shadow-md shadow-blue-600/20"
            >
              <Plus className="w-4 h-4" />
              <span>{loading ? "Creating..." : "Create Project"}</span>
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
