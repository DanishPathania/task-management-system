"use client";

import React, { createContext, useContext, useEffect, useState } from "react";
import { User } from "@/types";
import { api } from "@/lib/api";
import { realtime } from "@/lib/websocket";

interface AuthContextType {
  user: User | null;
  loading: boolean;
  login: (email: string, password: string) => Promise<void>;
  register: (name: string, email: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
  refreshUser: () => Promise<void>;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState<boolean>(true);

  const fetchCurrentUser = async () => {
    try {
      const u = await api.getMe();
      setUser(u);
      realtime.connect();
    } catch {
      setUser(null);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchCurrentUser();
  }, []);

  const login = async (email: string, password: string) => {
    const res = await api.login(email, password);
    setUser(res.user);
    realtime.connect();
  };

  const register = async (name: string, email: string, password: string) => {
    await api.register(name, email, password);
  };

  const logout = async () => {
    try {
      await api.logout();
    } finally {
      setUser(null);
      realtime.disconnect();
    }
  };

  const refreshUser = async () => {
    await fetchCurrentUser();
  };

  return (
    <AuthContext.Provider value={{ user, loading, login, register, logout, refreshUser }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error("useAuth must be used within an AuthProvider");
  }
  return context;
}
