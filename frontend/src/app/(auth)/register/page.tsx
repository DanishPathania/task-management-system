"use client";

import React, { useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useAuth } from "@/hooks/useAuth";
import { Lock, Mail, User as UserIcon, ArrowRight, AlertCircle } from "lucide-react";

export default function RegisterPage() {
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  const { register } = useAuth();
  const router = useRouter();

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim() || !email.trim() || !password || loading) return;

    if (password.length < 8) {
      setError("Password must be at least 8 characters long");
      return;
    }

    setError(null);
    setLoading(true);

    try {
      await register(name.trim(), email.trim(), password);
      router.push("/login?registered=true");
    } catch (err: any) {
      setError(err.message || "Failed to create account. Please try again.");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="flex-1 flex items-center justify-center px-4 py-12">
      <div className="w-full max-w-md bg-slate-900 border border-slate-800 rounded-2xl p-8 shadow-2xl space-y-6 relative">
        <div className="text-center space-y-1.5">
          <h2 className="text-2xl font-bold text-white tracking-tight">Create an Account</h2>
          <p className="text-xs text-slate-400">Join your team on CollabTask</p>
        </div>

        {error && (
          <div className="p-3 bg-red-500/10 border border-red-500/20 text-red-400 text-xs rounded-xl flex items-center gap-2">
            <AlertCircle className="w-4 h-4 shrink-0" />
            <span>{error}</span>
          </div>
        )}

        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label className="text-xs font-semibold text-slate-400 block mb-1">Full Name</label>
            <div className="relative">
              <UserIcon className="w-4 h-4 text-slate-500 absolute left-3 top-1/2 -translate-y-1/2" />
              <input
                type="text"
                required
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="Sarah Connor"
                className="w-full bg-slate-800/80 border border-slate-700 rounded-xl pl-9 pr-3 py-2.5 text-sm text-slate-100 placeholder-slate-500 focus:outline-none focus:border-blue-500 transition"
              />
            </div>
          </div>

          <div>
            <label className="text-xs font-semibold text-slate-400 block mb-1">Email Address</label>
            <div className="relative">
              <Mail className="w-4 h-4 text-slate-500 absolute left-3 top-1/2 -translate-y-1/2" />
              <input
                type="email"
                required
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                placeholder="you@company.com"
                className="w-full bg-slate-800/80 border border-slate-700 rounded-xl pl-9 pr-3 py-2.5 text-sm text-slate-100 placeholder-slate-500 focus:outline-none focus:border-blue-500 transition"
              />
            </div>
          </div>

          <div>
            <label className="text-xs font-semibold text-slate-400 block mb-1">Password (min 8 characters)</label>
            <div className="relative">
              <Lock className="w-4 h-4 text-slate-500 absolute left-3 top-1/2 -translate-y-1/2" />
              <input
                type="password"
                required
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                placeholder="••••••••"
                className="w-full bg-slate-800/80 border border-slate-700 rounded-xl pl-9 pr-3 py-2.5 text-sm text-slate-100 placeholder-slate-500 focus:outline-none focus:border-blue-500 transition"
              />
            </div>
          </div>

          <button
            type="submit"
            disabled={loading}
            className="w-full py-2.5 rounded-xl bg-blue-600 hover:bg-blue-500 disabled:opacity-50 text-white font-medium text-sm transition shadow-lg shadow-blue-600/20 flex items-center justify-center gap-2"
          >
            <span>{loading ? "Creating Account..." : "Sign Up"}</span>
            <ArrowRight className="w-4 h-4" />
          </button>
        </form>

        <div className="text-center text-xs text-slate-400 pt-2 border-t border-slate-800">
          Already have an account?{" "}
          <Link href="/login" className="text-blue-400 hover:underline font-medium">
            Sign in
          </Link>
        </div>
      </div>
    </div>
  );
}
