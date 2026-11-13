import { FormEvent, useState } from "react";
import App from "../App";
import { useAuth } from "../hooks/useAuth";

export default function AuthGate() {
  const auth = useAuth();
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [busy, setBusy] = useState(false);
  const submit = async (event: FormEvent) => {
    event.preventDefault();
    setBusy(true);
    try { await auth.login(username, password); setPassword(""); }
    finally { setBusy(false); }
  };
  if (auth.loading) return <div className="p-8 text-center text-slate-500" role="status">Проверяем вход…</div>;
  if (auth.username) return <>
    {auth.error && <div role="alert" className="mx-auto max-w-7xl px-4 pt-4 text-rose-700">{auth.error}</div>}
    <App username={auth.username} onLogout={auth.logout} />
  </>;
  return (
    <main className="flex min-h-screen items-center justify-center px-4 py-12">
      <form onSubmit={submit} className="w-full max-w-sm space-y-5 rounded-xl bg-white p-8 shadow-sm ring-1 ring-slate-200">
        <div>
          <h1 className="text-xl font-bold tracking-tight text-ink">CLEVER DASHBOARD</h1>
          <p className="mt-2 text-sm text-slate-500">Войдите, чтобы открыть аналитику</p>
        </div>
        <label className="block text-sm font-medium text-slate-700">Логин
          <input autoFocus required autoComplete="username" value={username} onChange={e => setUsername(e.target.value)} disabled={busy}
            className="mt-2 block w-full rounded-lg border border-slate-300 px-3 py-2 focus:border-brand focus:outline-none focus:ring-1 focus:ring-brand" />
        </label>
        <label className="block text-sm font-medium text-slate-700">Пароль
          <input required type="password" autoComplete="current-password" value={password} onChange={e => setPassword(e.target.value)} disabled={busy}
            className="mt-2 block w-full rounded-lg border border-slate-300 px-3 py-2 focus:border-brand focus:outline-none focus:ring-1 focus:ring-brand" />
        </label>
        {auth.error && <p role="alert" className="text-sm text-rose-700">{auth.error}</p>}
        <button disabled={busy} className="w-full rounded-lg bg-brand px-4 py-2.5 text-sm font-semibold text-white disabled:opacity-50">{busy ? "Входим…" : "Войти"}</button>
      </form>
    </main>
  );
}
