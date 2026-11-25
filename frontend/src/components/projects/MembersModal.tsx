"use client";

import React, { useState } from "react";
import { ProjectMember, ProjectRole } from "@/types";
import { api } from "@/lib/api";
import { useAuth } from "@/hooks/useAuth";
import { X, UserPlus, Shield, User, Trash2, AlertCircle, CheckCircle2 } from "lucide-react";

interface MembersModalProps {
  projectId: string;
  members: ProjectMember[];
  isOpen: boolean;
  onClose: () => void;
  onMemberAdded: (member: ProjectMember) => void;
  onMemberUpdated: (member: ProjectMember) => void;
  onMemberRemoved: (userId: string) => void;
  userRole?: string;
}

export function MembersModal({
  projectId,
  members,
  isOpen,
  onClose,
  onMemberAdded,
  onMemberUpdated,
  onMemberRemoved,
  userRole,
}: MembersModalProps) {
  const { user } = useAuth();
  const [email, setEmail] = useState("");
  const [role, setRole] = useState<ProjectRole>("member");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

  if (!isOpen) return null;

  const isAdmin = userRole === "admin";

  const handleAddMember = async (e: React.FormEvent) => {
    e.preventDefault();
    const trimmedEmail = email.trim();
    if (!trimmedEmail || loading) return;

    setLoading(true);
    setError(null);
    setSuccess(null);

    try {
      const added = await api.addMember(projectId, trimmedEmail, role);
      onMemberAdded(added);
      setEmail("");
      setSuccess(`Added ${added.user_name || added.user_email || trimmedEmail} as ${added.role}.`);
    } catch (err: any) {
      setError(err.message || "Failed to add member");
    } finally {
      setLoading(false);
    }
  };

  const handleRoleChange = async (userId: string, newRole: ProjectRole) => {
    setError(null);
    setSuccess(null);
    try {
      const updated = await api.updateMemberRole(projectId, userId, newRole);
      onMemberUpdated(updated);
    } catch (err: any) {
      setError(err.message || "Failed to update role");
    }
  };

  const handleRemoveMember = async (userId: string) => {
    if (!confirm("Are you sure you want to remove this member from the project?")) return;

    setError(null);
    setSuccess(null);
    try {
      await api.removeMember(projectId, userId);
      onMemberRemoved(userId);
    } catch (err: any) {
      setError(err.message || "Failed to remove member");
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/70 backdrop-blur-sm animate-in fade-in duration-200">
      <div className="bg-slate-900 border border-slate-800 rounded-2xl w-full max-w-lg shadow-2xl overflow-hidden flex flex-col max-h-[85vh]">
        <div className="flex items-center justify-between px-6 py-4 border-b border-slate-800">
          <div className="flex items-center gap-2">
            <h3 className="font-semibold text-slate-100 text-base">Project Members</h3>
            <span className="text-xs bg-slate-800 text-slate-400 px-2 py-0.5 rounded-full font-medium">
              {members.length}
            </span>
          </div>
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

        {success && (
          <div className="px-6 py-2.5 bg-emerald-500/10 border-b border-emerald-500/20 text-emerald-400 text-xs flex items-center gap-2">
            <CheckCircle2 className="w-4 h-4 shrink-0" />
            <span>{success}</span>
          </div>
        )}

        <div className="p-6 space-y-6 overflow-y-auto">
          {/* Add member section (Admins only) */}
          {isAdmin && (
            <form onSubmit={handleAddMember} className="bg-slate-800/40 border border-slate-700/60 rounded-xl p-4 space-y-3">
              <h4 className="text-xs font-semibold text-slate-300 uppercase tracking-wider flex items-center gap-1.5">
                <UserPlus className="w-3.5 h-3.5 text-blue-400" />
                <span>Add Member by Email</span>
              </h4>
              <div className="flex flex-col sm:flex-row gap-2">
                <input
                  type="email"
                  required
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  placeholder="colleague@company.com"
                  className="flex-1 bg-slate-800 border border-slate-700 rounded-lg px-3 py-2 text-xs text-slate-100 placeholder-slate-500 focus:outline-none focus:border-blue-500 transition"
                />
                <div className="flex gap-2">
                  <select
                    value={role}
                    onChange={(e) => setRole(e.target.value as ProjectRole)}
                    className="bg-slate-800 border border-slate-700 rounded-lg px-2.5 py-2 text-xs text-slate-200 focus:outline-none focus:border-blue-500 transition"
                  >
                    <option value="member">Member</option>
                    <option value="admin">Admin</option>
                  </select>
                  <button
                    type="submit"
                    disabled={loading || !email.trim()}
                    className="px-3.5 py-2 rounded-lg bg-blue-600 hover:bg-blue-500 disabled:opacity-50 text-white text-xs font-medium transition shrink-0"
                  >
                    {loading ? "Adding..." : "Add"}
                  </button>
                </div>
              </div>
            </form>
          )}

          {/* Members list */}
          <div className="space-y-2">
            <h4 className="text-xs font-semibold text-slate-400 uppercase tracking-wider">
              Current Members
            </h4>
            <div className="space-y-2">
              {members.map((m) => {
                const isSelf = user?.id === m.user_id;
                return (
                  <div
                    key={m.user_id}
                    className="flex items-center justify-between bg-slate-800/60 border border-slate-700/40 rounded-xl px-4 py-3 text-xs"
                  >
                    <div className="flex items-center gap-2.5">
                      <div className="w-8 h-8 rounded-full bg-blue-600/30 text-blue-400 font-bold flex items-center justify-center text-xs border border-blue-500/20">
                        {(m.user_name || m.user_email || "U").charAt(0).toUpperCase()}
                      </div>
                      <div>
                        <div className="font-medium text-slate-200 flex items-center gap-1.5">
                          <span>{m.user_name || "Team Member"}</span>
                          {isSelf && (
                            <span className="text-[10px] bg-slate-700 text-slate-300 px-1.5 py-0.2 rounded">You</span>
                          )}
                        </div>
                        <div className="text-slate-400 text-[11px] font-mono">
                          {m.user_email || m.user_id}
                        </div>
                      </div>
                    </div>

                    <div className="flex items-center gap-2">
                      {isAdmin && !isSelf ? (
                        <>
                          <select
                            value={m.role}
                            onChange={(e) => handleRoleChange(m.user_id, e.target.value as ProjectRole)}
                            className="bg-slate-700/80 border border-slate-600/60 rounded px-2 py-1 text-slate-200 focus:outline-none focus:border-blue-500"
                          >
                            <option value="member">Member</option>
                            <option value="admin">Admin</option>
                          </select>
                          <button
                            onClick={() => handleRemoveMember(m.user_id)}
                            className="p-1.5 text-slate-400 hover:text-red-400 hover:bg-red-500/10 rounded transition"
                            title="Remove Member"
                          >
                            <Trash2 className="w-3.5 h-3.5" />
                          </button>
                        </>
                      ) : (
                        <span className={`px-2 py-0.5 rounded text-[11px] font-semibold border ${
                          m.role === "admin"
                            ? "bg-purple-500/10 text-purple-400 border-purple-500/20"
                            : "bg-slate-700 text-slate-300 border-slate-600"
                        }`}>
                          {m.role.toUpperCase()}
                        </span>
                      )}
                    </div>
                  </div>
                );
              })}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
