"use client";

import { create } from "zustand";
import { setToken, clearToken } from "./api";

export interface AuthUser {
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
  user: AuthUser | null;
  setAuth: (token: string, user: AuthUser) => void;
  logout: () => void;
  isAuthenticated: () => boolean;
}

export const useAuth = create<AuthState>((set, get) => ({
  token: typeof window !== "undefined" ? localStorage.getItem("turath_token") : null,
  user: typeof window !== "undefined"
    ? JSON.parse(localStorage.getItem("turath_user") || "null")
    : null,

  setAuth: (token, user) => {
    setToken(token);
    localStorage.setItem("turath_user", JSON.stringify(user));
    set({ token, user });
  },

  logout: () => {
    clearToken();
    localStorage.removeItem("turath_user");
    set({ token: null, user: null });
  },

  isAuthenticated: () => get().token !== null,
}));
