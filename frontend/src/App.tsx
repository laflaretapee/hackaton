import { useEffect, useState } from "react";
import {
  Link,
  NavLink,
  Navigate,
  Route,
  Routes,
  useLocation,
  useNavigate,
} from "react-router-dom";
import { useQuery, useQueryClient, type Query } from "@tanstack/react-query";
import {
  BookOpen,
  CalendarDays,
  ChevronRight,
  ClipboardList,
  LayoutDashboard,
  LifeBuoy,
  Menu,
  MessageSquare,
  Settings2,
  Sparkles,
  Users,
  X,
  GraduationCap,
  ShieldCheck,
  ArrowRight,
  Send,
} from "lucide-react";
import { api, ApiError } from "./api";
import type { Config, Team, User } from "./types";
import { Brand, directionTitle, ErrorBox, roles, Spinner } from "./ui";
import {
  Overview,
  TasksPage,
  KnowledgePage,
  ArticlePage,
  LessonsPage,
  ExamPage,
  ChatPage,
  RequestsPage,
  MeetingsPage,
  ProfilePage,
  InternsPage,
  InternDetail,
  AnalyticsPage,
} from "./pages";
import { AdminPage } from "./admin";
declare global {
  interface Window {
    Telegram?: {
      WebApp?: { initData?: string; ready?: () => void; expand?: () => void };
    };
  }
}
export function App() {
  const location = useLocation();
  if (location.pathname.startsWith("/admin")) return <AdminPage />;
  return <UserApp />;
}
function UserApp() {
  const client = useQueryClient();
  const [switchingAccount, setSwitchingAccount] = useState(false);
  const me = useQuery({ queryKey: ["me"], queryFn: () => api<User>("/me") });
  const config = useQuery({
    queryKey: ["config"],
    queryFn: () => api<Config>("/public/config"),
  });
  const sessionExpired =
    me.error instanceof ApiError && me.error.status === 401;
  const accountQuery = (query: Query) =>
    !["me", "config", "teams", "admin-settings", "admin-audit"].includes(
      String(query.queryKey[0]),
    );
  async function clearAccountData() {
    await client.cancelQueries({ predicate: accountQuery });
    client.removeQueries({ predicate: accountQuery });
  }
  useEffect(() => {
    if (sessionExpired) void clearAccountData();
  }, [sessionExpired, client]);
  async function finishLogin() {
    setSwitchingAccount(true);
    try {
      await clearAccountData();
      await me.refetch();
    } finally {
      setSwitchingAccount(false);
    }
  }
  if (me.isPending || switchingAccount)
    return (
      <div className="boot">
        <Brand />
        <Spinner />
      </div>
    );
  if (
    !me.data &&
    me.error &&
    !(me.error instanceof ApiError && me.error.status === 401)
  )
    return (
      <div className="boot">
        <Brand />
        <ErrorBox error={me.error} retry={() => me.refetch()} />
      </div>
    );
  if (!me.data || sessionExpired)
    return (
      <Login
        config={config.data}
        configError={config.error}
        retryConfig={() => config.refetch()}
        onLogin={finishLogin}
      />
    );
  return <Shell user={me.data} config={config.data} />;
}
function Login({
  config,
  configError,
  retryConfig,
  onLogin,
}: {
  config?: Config;
  configError?: unknown;
  retryConfig: () => void;
  onLogin: () => void;
}) {
  const [role, setRole] = useState("intern");
  const [name, setName] = useState("");
  const [team, setTeam] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<unknown>();
  const [loginLink, setLoginLink] = useState<{ url: string; token: string }>();
  const teams = useQuery({
    queryKey: ["teams"],
    queryFn: () => api<Team[]>("/teams"),
  });
  useEffect(() => {
    const app = window.Telegram?.WebApp;
    if (!app?.initData) return;
    app.ready?.();
    app.expand?.();
    setBusy(true);
    api("/auth/telegram", {
      method: "POST",
      body: JSON.stringify({ init_data: app.initData }),
    })
      .then(onLogin)
      .catch(setError)
      .finally(() => setBusy(false));
  }, []);
  useEffect(() => {
    if (!loginLink) return;
    let stopped = false;
    const id = setInterval(() => {
      api("/auth/exchange", {
        method: "POST",
        body: JSON.stringify({ token: loginLink.token }),
      })
        .then(() => {
          if (!stopped) {
            clearInterval(id);
            onLogin();
          }
        })
        .catch((e) => {
          if (e.status !== 409) {
            setError(e);
            clearInterval(id);
          }
        });
    }, 2500);
    return () => {
      stopped = true;
      clearInterval(id);
    };
  }, [loginLink]);
  async function demo(e: React.FormEvent) {
    e.preventDefault();
    setError(undefined);
    setBusy(true);
    try {
      await api("/auth/demo", {
        method: "POST",
        body: JSON.stringify({
          role,
          name: name.trim() || undefined,
          team_id: team || undefined,
        }),
      });
      onLogin();
    } catch (e) {
      setError(e);
    } finally {
      setBusy(false);
    }
  }
  async function telegram() {
    setBusy(true);
    setError(undefined);
    try {
      setLoginLink(await api("/auth/login-link"));
    } catch (e) {
      setError(e);
    } finally {
      setBusy(false);
    }
  }
  return (
    <div className="login">
      <header>
        <Brand />
        <span>Hack&Change · 2026</span>
      </header>
      <main className="login-main">
        <section className="login-story">
          <div className="eyebrow">
            <span className="tiny-dot" /> ТВОЙ УВЕРЕННЫЙ СТАРТ
          </div>
          <h1>
            Новая команда.
            <br />
            Понятный путь.
            <br />
            <span>Ты справишься.</span>
          </h1>
          <p>
            Всё для первых шагов на новом месте: знания, обучение и
            AI-наставник, который всегда рядом.
          </p>
          <div className="story-steps">
            <div>
              <span>01</span>
              <strong>Разберись в процессах</strong>
              <p>Нужные знания в одном месте</p>
            </div>
            <div>
              <span>02</span>
              <strong>Учись на практике</strong>
              <p>Короткие занятия и обратная связь</p>
            </div>
            <div>
              <span>03</span>
              <strong>Двигайся в своём темпе</strong>
              <p>С понятным планом и поддержкой</p>
            </div>
          </div>
          <div className="login-caption">
            <ShieldCheck size={17} /> Демонстрационный проект. Без корпоративных
            данных.
          </div>
        </section>
        <section className="login-panel">
          <div className="login-symbol">
            <Sparkles size={28} />
          </div>
          <h2>Начнём знакомство</h2>
          <p className="muted">
            Войди через Telegram или исследуй продукт в деморежиме.
          </p>
          <ErrorBox error={configError} retry={retryConfig} />
          <button
            className="button telegram-button"
            onClick={telegram}
            disabled={busy || !config?.telegram_username}
          >
            <Send size={17} /> Войти через Telegram
          </button>
          {!config?.telegram_username && (
            <p className="fine-print">
              Telegram пока не настроен. Используй демонстрационный вход.
            </p>
          )}
          {loginLink && (
            <div className="notice">
              <a href={loginLink.url} target="_blank" rel="noreferrer">
                Открыть бота и подтвердить вход <ArrowRight size={14} />
              </a>
              <p>
                Ожидаем подтверждение. После входа это окно обновится
                автоматически.
              </p>
            </div>
          )}
          <ErrorBox error={error} />
          {config?.demo_mode && (
            <>
              <div className="divider">
                <span>Демонстрационный вход</span>
              </div>
              <form onSubmit={demo}>
                <fieldset className="role-picker">
                  <legend>Выбери роль</legend>
                  {(["intern", "mentor", "manager"] as const).map((r) => (
                    <button
                      key={r}
                      type="button"
                      className={role === r ? "selected" : ""}
                      onClick={() => setRole(r)}
                    >
                      {roles[r]}
                    </button>
                  ))}
                </fieldset>
                <label>
                  Как тебя зовут
                  <input
                    value={name}
                    onChange={(e) => setName(e.target.value)}
                    placeholder="Например, Александр"
                    maxLength={80}
                  />
                </label>
                <label>
                  Команда
                  <select
                    value={team}
                    onChange={(e) => setTeam(e.target.value)}
                  >
                    <option value="">Команда по умолчанию</option>
                    {teams.data?.map((t) => (
                      <option key={t.id} value={t.id}>
                        {directionTitle(t.direction)} · {t.name}
                      </option>
                    ))}
                  </select>
                </label>
                <button className="button primary full" disabled={busy}>
                  {busy ? "Подключаемся…" : "Попробовать демо"}
                  <ArrowRight size={18} />
                </button>
                <p className="fine-print">
                  Создадим учебный профиль с синтетическими данными.
                </p>
              </form>
            </>
          )}
          <Link className="admin-entry" to="/admin">
            Вход администратора
          </Link>
        </section>
      </main>
      <footer>AI-buddy · Умная адаптация с человеческим подходом</footer>
    </div>
  );
}
function Shell({ user, config }: { user: User; config?: Config }) {
  const [open, setOpen] = useState(false);
  const location = useLocation();
  const navigate = useNavigate();
  const qc = useQueryClient();
  useEffect(() => {
    setOpen(false);
    document.getElementById("main-content")?.focus();
  }, [location.pathname]);
  const intern = user.role === "intern";
  const links = intern
    ? [
        ["/", "Обзор", LayoutDashboard],
        ["/tasks", "План адаптации", ClipboardList],
        ["/knowledge", "База знаний", BookOpen],
        ["/learning", "Обучение", GraduationCap],
        ["/chat", "AI-наставник", MessageSquare],
        ["/requests", "Мои заявки", LifeBuoy],
        ["/meetings", "Встречи", CalendarDays],
      ]
    : user.role === "mentor"
      ? [
          ["/", "Обзор", LayoutDashboard],
          ["/interns", "Мои стажёры", Users],
          ["/requests", "Заявки на доступ", LifeBuoy],
          ["/knowledge", "База знаний", BookOpen],
          ["/meetings", "Встречи", CalendarDays],
          ["/chat", "AI-наставник", MessageSquare],
        ]
      : [
          ["/", "Обзор команды", LayoutDashboard],
          ["/analytics", "Аналитика", Users],
          ["/knowledge", "База знаний", BookOpen],
        ];
  return (
    <div className="app-shell">
      <a className="skip-link" href="#main-content">
        К содержимому
      </a>
      <aside className={`sidebar ${open ? "open" : ""}`}>
        <div className="sidebar-brand">
          <Link to="/">
            <Brand />
          </Link>
          <button
            className="icon-button mobile-only"
            aria-label="Закрыть меню"
            onClick={() => setOpen(false)}
          >
            <X size={20} />
          </button>
        </div>
        <div className="workspace">
          <span className="workspace-icon">
            {user.team_name?.slice(0, 1) || "P"}
          </span>
          <div>
            <strong>{user.team_name}</strong>
            <span>{directionTitle(user.direction)}</span>
          </div>
          <ChevronRight size={15} />
        </div>
        <nav aria-label="Основная навигация">
          {links.map(([to, title, Icon]) => {
            const C = Icon as typeof Users;
            return (
              <NavLink end={to === "/"} key={to as string} to={to as string}>
                <C size={19} />
                <span>{title as string}</span>
                {to === "/chat" && <span className="ai-label">AI</span>}
              </NavLink>
            );
          })}
        </nav>
        <div className="sidebar-bottom">
          <div className="support-note">
            <Sparkles size={19} />
            <strong>Любой вопрос — это начало</strong>
            <p>AI-buddy поможет разобраться и найти следующий шаг.</p>
            <Link to="/chat">
              Спросить наставника <ArrowRight size={14} />
            </Link>
          </div>
          <NavLink to="/profile" className="profile-link">
            <span className="avatar">{user.name?.slice(0, 1)}</span>
            <span>
              <strong>{user.name}</strong>
              <small>{roles[user.role]}</small>
            </span>
            <Settings2 size={17} />
          </NavLink>
        </div>
      </aside>
      {open && (
        <div className="sidebar-overlay" onClick={() => setOpen(false)} />
      )}
      <div className="main-shell">
        <header className="topbar">
          <button
            className="icon-button mobile-only"
            aria-label="Открыть меню"
            onClick={() => setOpen(true)}
          >
            <Menu size={23} />
          </button>
          <div className="breadcrumb">
            Твоё рабочее пространство <ChevronRight size={14} />
            <span>{roles[user.role]}</span>
          </div>
          <div className="topbar-right">
            {user.is_demo && (
              <span className="demo-chip">
                <span />
                Деморежим
              </span>
            )}
            <Link
              className="topbar-avatar"
              to="/profile"
              aria-label="Открыть профиль"
            >
              {user.name?.slice(0, 1)}
            </Link>
          </div>
        </header>
        <main id="main-content" className="content" tabIndex={-1}>
          <Routes>
            <Route
              path="/"
              element={
                user.role === "manager" ? (
                  <AnalyticsPage user={user} />
                ) : (
                  <Overview user={user} />
                )
              }
            />
            <Route
              path="/tasks"
              element={intern ? <TasksPage /> : <Navigate to="/" />}
            />
            <Route path="/knowledge" element={<KnowledgePage />} />
            <Route path="/knowledge/:id" element={<ArticlePage />} />
            <Route path="/knowledge/*" element={<ArticlePage />} />
            <Route
              path="/learning"
              element={intern ? <LessonsPage /> : <Navigate to="/" />}
            />
            <Route
              path="/learning/:id"
              element={intern ? <ExamPage /> : <Navigate to="/" />}
            />
            <Route path="/chat" element={<ChatPage config={config} />} />
            <Route path="/requests" element={<RequestsPage user={user} />} />
            <Route path="/meetings" element={<MeetingsPage />} />
            <Route
              path="/profile"
              element={
                <ProfilePage
                  user={user}
                  onExit={() => {
                    qc.clear();
                    navigate("/");
                    location.pathname === "/" && window.location.reload();
                  }}
                />
              }
            />
            <Route
              path="/interns"
              element={
                user.role === "mentor" ? <InternsPage /> : <Navigate to="/" />
              }
            />
            <Route
              path="/interns/:id"
              element={
                user.role === "mentor" ? <InternDetail /> : <Navigate to="/" />
              }
            />
            <Route
              path="/analytics"
              element={
                user.role === "manager" ? (
                  <AnalyticsPage user={user} />
                ) : (
                  <Navigate to="/" />
                )
              }
            />
            <Route
              path="*"
              element={
                <div>
                  <h1>Страница не найдена</h1>
                  <Link to="/">Вернуться к обзору</Link>
                </div>
              }
            />
          </Routes>
        </main>
        <footer className="app-footer">
          <span>AI-buddy</span>
          <span>Учебная среда · Hack&Change 2026</span>
        </footer>
      </div>
    </div>
  );
}
