export interface LoginEvent {
  id: number;
  user_id: string | null;
  user_name: string | null;
  email: string;
  method: "password" | "google";
  result: "success" | "failure";
  reason: string | null;
  ip: string;
  user_agent: string;
  created_at: string;
}

export interface AccountActivity {
  id: string;
  email: string;
  name_ar: string | null;
  role: string;
  college_name: string | null;
  last_seen_at: string | null;
  last_login_at: string | null;
  online: boolean;
}

export interface AccountSummary {
  total: number;
  online: number;
  today: number;
  last_30_days: number;
  never_signed_in: number;
}

export interface LoginLogFilters {
  q: string;
  method: string;
  result: string;
  from: string;
  to: string;
}

/** Query string of the login log; empty filters are left out. */
export function loginLogQuery(filters: LoginLogFilters, page: number, limit: number): string {
  const params = new URLSearchParams({ page: String(page), limit: String(limit) });
  for (const [key, value] of Object.entries(filters)) {
    const trimmed = value.trim();
    if (trimmed) params.set(key, trimmed);
  }
  return params.toString();
}

/** A timestamp in Baghdad time, the university's clock whatever the viewer's. */
export function formatBaghdadTime(iso: string | null, locale: string): string {
  if (!iso) return "";
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) return "";
  return new Intl.DateTimeFormat(locale === "ar" ? "ar-IQ" : "en-GB", {
    timeZone: "Asia/Baghdad",
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    numberingSystem: "latn",
  }).format(date);
}
