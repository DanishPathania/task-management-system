"use client";

import React, { useEffect } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useAuth } from "@/hooks/useAuth";
import { CheckCircle2, Zap, Users, Shield, ArrowRight } from "lucide-react";

export default function HomePage() {
  const { user, loading } = useAuth();
  const router = useRouter();

  useEffect(() => {
    if (!loading && user) {
      router.push("/dashboard");
    }
  }, [user, loading, router]);

  if (loading) {
    return (
      <div className="flex-1 flex items-center justify-center">
        <div className="w-8 h-8 border-2 border-blue-500 border-t-transparent rounded-full animate-spin" />
      </div>
    );
  }

  return (
    <div className="flex-1 flex flex-col justify-center items-center px-6 py-20 text-center relative overflow-hidden">
      {/* Background glow effects */}
      <div className="absolute top-1/4 left-1/2 -translate-x-1/2 -translate-y-1/2 w-96 h-96 bg-blue-600/15 rounded-full blur-3xl pointer-events-none" />
      <div className="absolute top-1/3 left-1/3 -translate-x-1/2 -translate-y-1/2 w-72 h-72 bg-indigo-600/10 rounded-full blur-3xl pointer-events-none" />

      <div className="max-w-3xl mx-auto space-y-6 relative z-10">
        <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full bg-blue-500/10 border border-blue-500/20 text-blue-400 text-xs font-semibold uppercase tracking-wider">
          <Zap className="w-3.5 h-3.5" />
          <span>Real-Time Collaborative Platform</span>
        </div>

        <h1 className="text-4xl sm:text-6xl font-extrabold text-white tracking-tight leading-tight">
          Supercharge Your Team with{" "}
          <span className="text-transparent bg-clip-text bg-gradient-to-r from-blue-400 via-indigo-300 to-purple-400">
            Real-Time
          </span>{" "}
          Task Management
        </h1>

        <p className="text-lg text-slate-400 max-w-2xl mx-auto leading-relaxed">
          Coordinate projects, track live presence, update tasks seamlessly across connected browsers, and collaborate with zero delay.
        </p>

        <div className="flex flex-col sm:flex-row items-center justify-center gap-4 pt-4">
          <Link
            href="/register"
            className="w-full sm:w-auto px-6 py-3 rounded-xl bg-blue-600 hover:bg-blue-500 text-white font-medium text-sm transition shadow-lg shadow-blue-600/25 flex items-center justify-center gap-2"
          >
            <span>Get Started for Free</span>
            <ArrowRight className="w-4 h-4" />
          </Link>
          <Link
            href="/login"
            className="w-full sm:w-auto px-6 py-3 rounded-xl bg-slate-800 hover:bg-slate-700 border border-slate-700 text-slate-200 font-medium text-sm transition"
          >
            Sign In to Existing Account
          </Link>
        </div>

        {/* Feature Highlights */}
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-6 pt-16 text-left">
          <div className="bg-slate-900/60 border border-slate-800 rounded-2xl p-5 space-y-2">
            <div className="w-9 h-9 rounded-lg bg-blue-500/10 text-blue-400 flex items-center justify-center">
              <Zap className="w-5 h-5" />
            </div>
            <h3 className="font-semibold text-slate-100 text-sm">Instant Synchronization</h3>
            <p className="text-xs text-slate-400 leading-relaxed">
              Bi-directional WebSocket streaming reflects task changes, comments, and assignments immediately.
            </p>
          </div>

          <div className="bg-slate-900/60 border border-slate-800 rounded-2xl p-5 space-y-2">
            <div className="w-9 h-9 rounded-lg bg-purple-500/10 text-purple-400 flex items-center justify-center">
              <Users className="w-5 h-5" />
            </div>
            <h3 className="font-semibold text-slate-100 text-sm">Live Presence Tracking</h3>
            <p className="text-xs text-slate-400 leading-relaxed">
              Redis-backed leases show which team members are currently online in the workspace.
            </p>
          </div>

          <div className="bg-slate-900/60 border border-slate-800 rounded-2xl p-5 space-y-2">
            <div className="w-9 h-9 rounded-lg bg-emerald-500/10 text-emerald-400 flex items-center justify-center">
              <Shield className="w-5 h-5" />
            </div>
            <h3 className="font-semibold text-slate-100 text-sm">Project-Scoped RBAC</h3>
            <p className="text-xs text-slate-400 leading-relaxed">
              Fine-grained admin and member roles enforce strict security boundaries on the backend.
            </p>
          </div>
        </div>
      </div>
    </div>
  );
}
