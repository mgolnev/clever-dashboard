export const sessionExpiredEvent = "clever:session-expired";

export async function apiFetch(url: string, init: RequestInit = {}): Promise<Response> {
  const headers = new Headers(init.headers);
  headers.set("X-Requested-With", "CleverDashboard");
  const response = await fetch(url, { ...init, headers, credentials: "same-origin" });
  if (response.status === 401 && !url.startsWith("/api/auth/")) {
    window.dispatchEvent(new Event(sessionExpiredEvent));
  }
  return response;
}

async function authRequest(path: string, init?: RequestInit): Promise<{ username: string }> {
  const res = await apiFetch(`/api/auth/${path}`, init);
  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new Error(body.error || "Не удалось связаться с сервером");
  }
  return res.json();
}

export const auth = {
  session: () => authRequest("session"),
  login: (username: string, password: string) => authRequest("login", {
    method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ username, password }),
  }),
  logout: async () => {
    const res = await apiFetch("/api/auth/logout", { method: "POST" });
    if (!res.ok && res.status !== 401) throw new Error("Не удалось выйти. Попробуйте ещё раз");
  },
};
