import { useEffect, useState } from "react";
import { auth, sessionExpiredEvent } from "../services/auth";

export function useAuth() {
  const [username, setUsername] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  useEffect(() => {
    let active = true;
    const expire = () => { setUsername(null); setError("Сессия завершена. Войдите снова"); };
    window.addEventListener(sessionExpiredEvent, expire);
    auth.session().then(user => { if (active) setUsername(user.username); })
      .catch(() => {})
      .finally(() => { if (active) setLoading(false); });
    return () => { active = false; window.removeEventListener(sessionExpiredEvent, expire); };
  }, []);
  const login = async (loginName: string, password: string) => {
    setError("");
    try { const user = await auth.login(loginName, password); setUsername(user.username); }
    catch (e) { setError(e instanceof Error ? e.message : "Ошибка входа"); }
  };
  const logout = async () => {
    setError("");
    try { await auth.logout(); setUsername(null); }
    catch (e) { setError(e instanceof Error ? e.message : "Ошибка выхода"); }
  };
  return { username, loading, error, login, logout };
}
