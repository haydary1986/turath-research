"use client";

import { create } from "zustand";

interface User {
  id: string;
  email: string;
  name_ar?: string | null;
  name_en?: string | null;
  role: string;
  institution_id?: string | null;
  college_id?: string | null;
  department_id?: string | null;
}

interface AuthState {
  token: string | null;
  user: User | null;
  setAuth: (token: string, user: User) => void;
  logout: () => void;
  isAuthenticated: () => boolean;
}

export const useAuth = create<AuthState>((set, get) => ({
  token: typeof window !== "undefined" ? sessionStorage.getItem("token") : null,
  user: typeof window !== "undefined"
    ? JSON.parse(sessionStorage.getItem("user") || "null")
    : null,

  setAuth: (token, user) => {
    sessionStorage.setItem("token", token);
    sessionStorage.setItem("user", JSON.stringify(user));
    set({ token, user });
  },

  logout: () => {
    sessionStorage.removeItem("token");
    sessionStorage.removeItem("user");
    set({ token: null, user: null });
  },

  isAuthenticated: () => get().token !== null,
}));
