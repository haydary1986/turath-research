const API_BASE = "";

interface APIResponse<T> {
  success: boolean;
  data?: T;
  error?: string;
  meta?: {
    total: number;
    page: number;
    limit: number;
  };
}

function getToken(): string | null {
  if (typeof window === "undefined") return null;
  return localStorage.getItem("turath_token");
}

export function setToken(token: string) {
  localStorage.setItem("turath_token", token);
}

export function clearToken() {
  localStorage.removeItem("turath_token");
}

async function request<T>(
  path: string,
  options?: RequestInit
): Promise<APIResponse<T>> {
  const url = `${API_BASE}${path}`;
  const token = getToken();

  const res = await fetch(url, {
    ...options,
    headers: {
      "Content-Type": "application/json",
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...options?.headers,
    },
  });

  let json: APIResponse<T>;
  try {
    json = await res.json();
  } catch {
    throw new Error(
      `Server returned non-JSON response (HTTP ${res.status})`
    );
  }

  if (!res.ok || !json.success) {
    throw new Error(json.error || `Request failed: ${res.status}`);
  }

  return json;
}

async function authedFetch(path: string, options?: RequestInit): Promise<Response> {
  const token = getToken();
  return fetch(`${API_BASE}${path}`, {
    ...options,
    headers: {
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...options?.headers,
    },
  });
}

/** Uploads one file as multipart/form-data (the browser sets the boundary header). */
async function upload<T>(path: string, file: File): Promise<APIResponse<T>> {
  const form = new FormData();
  form.append("file", file);
  const res = await authedFetch(path, { method: "POST", body: form });

  let json: APIResponse<T>;
  try {
    json = await res.json();
  } catch {
    throw new Error(`Server returned non-JSON response (HTTP ${res.status})`);
  }
  if (!res.ok || !json.success) {
    throw new Error(json.error || `Request failed: ${res.status}`);
  }
  return json;
}

/** Fetches a protected binary file and returns a temporary object URL for it. */
export async function fetchObjectUrl(path: string): Promise<string> {
  const res = await authedFetch(path);
  if (!res.ok) {
    throw new Error(`Request failed: ${res.status}`);
  }
  return URL.createObjectURL(await res.blob());
}

export const api = {
  upload,

  get: <T>(path: string) => request<T>(path),

  post: <T>(path: string, body: unknown) =>
    request<T>(path, {
      method: "POST",
      body: JSON.stringify(body),
    }),

  put: <T>(path: string, body: unknown) =>
    request<T>(path, {
      method: "PUT",
      body: JSON.stringify(body),
    }),

  delete: <T>(path: string) => request<T>(path, { method: "DELETE" }),
};
