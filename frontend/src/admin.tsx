import { useState, type FormEvent } from "react";
import { Link } from "react-router-dom";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  Check,
  KeyRound,
  LogOut,
  Save,
  ShieldCheck,
  Unplug,
  Send,
  Sparkles,
  RefreshCw,
} from "lucide-react";
import { api, ApiError, makeSettingsPayload } from "./api";
import type { Settings } from "./types";
import { Brand, DateText, ErrorBox, Modal, PageTitle, Spinner } from "./ui";
type FormSettings = Settings & {
  telegram_token: string;
  routerai_api_key: string;
  clear_telegram_token?: boolean;
  clear_routerai_key?: boolean;
};
export function AdminPage() {
  const qc = useQueryClient();
  const q = useQuery({
    queryKey: ["admin-settings"],
    queryFn: () => api<Settings>("/admin/settings"),
    retry: false,
  });
  const [password, setPassword] = useState("");
  const login = useMutation({
    mutationFn: () =>
      api("/admin/login", {
        method: "POST",
        body: JSON.stringify({ password }),
      }),
    onSuccess: () => {
      setPassword("");
      qc.invalidateQueries({ queryKey: ["admin-settings"] });
    },
  });
  const logout = useMutation({
    mutationFn: () => api("/admin/logout", { method: "POST", body: "{}" }),
    onSuccess: () => {
      qc.removeQueries({ queryKey: ["admin-audit"] });
      qc.resetQueries({ queryKey: ["admin-settings"] });
    },
  });
  const unauthorized = q.error instanceof ApiError && q.error.status === 401;
  return (
    <div className="admin-app">
      <header className="admin-header">
        <Link to="/">
          <Brand />
        </Link>
        <span>
          <ShieldCheck size={16} />
          Администрирование
        </span>
        {q.data && (
          <button
            className="button small secondary"
            disabled={logout.isPending}
            onClick={() => logout.mutate()}
          >
            <LogOut size={15} />
            Выйти
          </button>
        )}
      </header>
      {q.isPending ? (
        <Spinner />
      ) : unauthorized || !q.data ? (
        <main className="admin-login">
          <div className="login-symbol">
            <KeyRound size={27} />
          </div>
          <h1>Вход администратора</h1>
          <p>
            Настройка интеграций и аудит действий.
            <br />
            Отдельный защищённый доступ.
          </p>
          <form
            onSubmit={(e) => {
              e.preventDefault();
              login.mutate();
            }}
          >
            <label>
              Пароль
              <input
                type="password"
                autoComplete="current-password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                required
              />
            </label>
            <ErrorBox error={login.error || (!unauthorized && q.error)} />
            <button className="button primary full" disabled={login.isPending}>
              {login.isPending ? "Проверяем…" : "Войти в настройки"}
              <ShieldCheck size={17} />
            </button>
          </form>
          <Link className="back-link" to="/">
            ← Вернуться в приложение
          </Link>
        </main>
      ) : (
        <main className="admin-content">
          <PageTitle
            eyebrow="УПРАВЛЕНИЕ ПЛАТФОРМОЙ"
            title="Интеграции и настройки"
            description="Секреты хранятся на сервере и никогда не возвращаются в браузер."
          />
          <AdminSettings initial={q.data} />
          <Audit />
        </main>
      )}
    </div>
  );
}
export function AdminSettings({ initial }: { initial: Settings }) {
  const qc = useQueryClient();
  const [form, setForm] = useState<FormSettings>({
    ...initial,
    telegram_token: "",
    routerai_api_key: "",
  });
  const [clear, setClear] = useState<"telegram" | "routerai" | null>(null);
  const [test, setTest] = useState<{
    kind: string;
    ok: boolean;
    message: string;
    models?: { id: string }[];
  }>();
  function update<K extends keyof FormSettings>(
    key: K,
    value: FormSettings[K],
  ) {
    setForm((f) => ({ ...f, [key]: value }));
  }
  const save = useMutation({
    mutationFn: (payload?: Record<string, unknown>) =>
      api<Settings>("/admin/settings", {
        method: "PUT",
        body: JSON.stringify(makeSettingsPayload(payload || form)),
      }),
    onSuccess: (settings) => {
      setForm({ ...settings, telegram_token: "", routerai_api_key: "" });
      setClear(null);
      qc.invalidateQueries({ queryKey: ["admin-settings"] });
      qc.invalidateQueries({ queryKey: ["admin-audit"] });
      qc.invalidateQueries({ queryKey: ["config"] });
    },
  });
  const testing = useMutation({
    mutationFn: async (kind: string) => {
      setTest(undefined);
      const r = await api<{
        ok: boolean;
        message: string;
        models?: { id: string }[];
      }>("/admin/test/" + kind, { method: "POST", body: "{}" });
      setTest({ kind, ...r });
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: ["admin-audit"] }),
  });
  function submit(e: FormEvent) {
    e.preventDefault();
    setTest(undefined);
    save.mutate(undefined);
  }
  return (
    <>
      <div className="notice">
        <ShieldCheck size={20} />
        <div>
          <strong>Пустое поле секрета сохраняет текущее значение</strong>
          <p>
            Чтобы удалить ключ или токен, используй отдельное действие. Проверка
            соединения использует сохранённые настройки.
          </p>
        </div>
      </div>
      <form onSubmit={submit}>
        <section className="admin-section">
          <div className="admin-section-title">
            <span className="integration-icon">
              <Send size={23} />
            </span>
            <div>
              <h2>Telegram</h2>
              <p>Вход, Mini App и напоминания о встречах</p>
            </div>
            <span
              className={
                "badge " +
                (initial.telegram_token_set ? "status-completed" : "")
              }
            >
              {initial.telegram_token_set ? "Токен сохранён" : "Токен не задан"}
            </span>
          </div>
          <div className="form-grid">
            <label>
              Токен бота
              <input
                type="password"
                autoComplete="new-password"
                placeholder={
                  initial.telegram_token_set
                    ? "Сохранён · оставь пустым, чтобы не менять"
                    : "Вставь токен от BotFather"
                }
                value={form.telegram_token}
                onChange={(e) => update("telegram_token", e.target.value)}
              />
            </label>
            <label>
              Username бота
              <input
                placeholder="my_buddy_bot"
                value={form.telegram_username}
                onChange={(e) =>
                  update("telegram_username", e.target.value.replace(/^@/, ""))
                }
              />
            </label>
          </div>
          <label className="checkbox-label">
            <input
              type="checkbox"
              checked={form.telegram_enabled}
              onChange={(e) => update("telegram_enabled", e.target.checked)}
            />
            Включить Telegram-интеграцию
          </label>
          <div className="button-row">
            <button
              type="button"
              className="button secondary small"
              disabled={testing.isPending}
              onClick={() => testing.mutate("telegram")}
            >
              <RefreshCw size={14} />
              Проверить Telegram
            </button>
            {initial.telegram_token_set && (
              <button
                type="button"
                className="text-button danger-text"
                onClick={() => setClear("telegram")}
              >
                <Unplug size={14} />
                Удалить токен
              </button>
            )}
          </div>
        </section>
        <section className="admin-section">
          <div className="admin-section-title">
            <span className="integration-icon green">
              <Sparkles size={24} />
            </span>
            <div>
              <h2>RouterAI</h2>
              <p>AI-наставник и разбор учебных материалов</p>
            </div>
            <span
              className={
                "badge " + (initial.routerai_key_set ? "status-completed" : "")
              }
            >
              {initial.routerai_key_set
                ? "API-ключ сохранён"
                : "API-ключ не задан"}
            </span>
          </div>
          <div className="form-grid">
            <label>
              API-ключ
              <input
                type="password"
                autoComplete="new-password"
                value={form.routerai_api_key}
                onChange={(e) => update("routerai_api_key", e.target.value)}
                placeholder={
                  initial.routerai_key_set
                    ? "Сохранён · оставь пустым, чтобы не менять"
                    : "Вставь ключ RouterAI"
                }
              />
            </label>
            <label>
              Базовый URL
              <input
                type="url"
                required
                value={form.routerai_base_url}
                onChange={(e) => update("routerai_base_url", e.target.value)}
              />
            </label>
            <label>
              Модель
              <input
                value={form.routerai_model}
                onChange={(e) => update("routerai_model", e.target.value)}
                placeholder="ID модели с поддержкой tool calling"
                list="routerai-models"
              />
              <datalist id="routerai-models">
                {test?.models?.map((m) => (
                  <option value={m.id} key={m.id} />
                ))}
              </datalist>
            </label>
            <label>
              Максимум токенов
              <input
                type="number"
                min={100}
                max={16000}
                required
                value={form.ai_max_tokens}
                onChange={(e) =>
                  update("ai_max_tokens", Number(e.target.value))
                }
              />
            </label>
            <label>
              Таймаут запроса, секунд
              <input
                type="number"
                min={5}
                max={180}
                required
                value={form.ai_timeout_seconds}
                onChange={(e) =>
                  update("ai_timeout_seconds", Number(e.target.value))
                }
              />
            </label>
          </div>
          <label className="checkbox-label">
            <input
              type="checkbox"
              checked={form.ai_mock_mode}
              onChange={(e) => update("ai_mock_mode", e.target.checked)}
            />
            Демонстрационный AI-режим (без обращения к провайдеру)
          </label>
          <div className="button-row">
            <button
              type="button"
              className="button secondary small"
              disabled={testing.isPending}
              onClick={() => testing.mutate("routerai")}
            >
              <RefreshCw size={14} />
              Проверить RouterAI
            </button>
            {initial.routerai_key_set && (
              <button
                type="button"
                className="text-button danger-text"
                onClick={() => setClear("routerai")}
              >
                <Unplug size={14} />
                Удалить API-ключ
              </button>
            )}
          </div>
        </section>
        <section className="admin-section">
          <h2>Адрес приложения</h2>
          <label>
            Публичный HTTPS URL
            <input
              type="url"
              value={form.app_url}
              onChange={(e) => update("app_url", e.target.value)}
              required
            />
          </label>
          <p className="muted">
            Используется для ссылок из Telegram и Mini App.
          </p>
        </section>
        <ErrorBox error={save.error || testing.error} />
        {testing.isPending && (
          <div className="notice slim" role="status">
            <RefreshCw className="spin" size={17} />
            Проверяем сохранённую конфигурацию…
          </div>
        )}
        {test && (
          <div className={test.ok ? "success" : "error"} role="status">
            <span>
              {test.kind === "telegram" ? "Telegram" : "RouterAI"}:{" "}
              {test.ok
                ? "соединение установлено"
                : "проверка не пройдена. Проверь конфигурацию и доступность сервиса."}
            </span>
            {test.models?.length && (
              <span> · Доступно моделей: {test.models.length}</span>
            )}
          </div>
        )}
        {save.isSuccess && (
          <div className="success" role="status">
            <Check size={17} />
            Настройки сохранены. Поля секретов очищены.
          </div>
        )}
        <div className="admin-save">
          <button className="button primary" disabled={save.isPending}>
            <Save size={17} />
            {save.isPending ? "Сохраняем…" : "Сохранить настройки"}
          </button>
        </div>
      </form>
      {clear && (
        <Modal
          title={
            clear === "telegram"
              ? "Удалить токен Telegram?"
              : "Удалить ключ RouterAI?"
          }
          onClose={() => setClear(null)}
        >
          <p>
            Секрет будет удалён с сервера. Для повторного подключения
            понадобится ввести его заново.
          </p>
          <p className="muted">
            {clear === "telegram"
              ? "Telegram-интеграция будет выключена."
              : "AI-наставник переключится в демонстрационный режим."}
          </p>
          <ErrorBox error={save.error} />
          <div className="modal-actions">
            <button className="button secondary" onClick={() => setClear(null)}>
              Отмена
            </button>
            <button
              className="button danger"
              disabled={save.isPending}
              onClick={() =>
                save.mutate({
                  ...form,
                  telegram_enabled:
                    clear === "telegram" ? false : form.telegram_enabled,
                  ai_mock_mode: clear === "routerai" ? true : form.ai_mock_mode,
                  [clear === "telegram"
                    ? "clear_telegram_token"
                    : "clear_routerai_key"]: true,
                })
              }
            >
              Удалить секрет
            </button>
          </div>
        </Modal>
      )}
    </>
  );
}
function Audit() {
  const q = useQuery({
    queryKey: ["admin-audit"],
    queryFn: () =>
      api<{ id: string; action: string; created_at: string }[]>("/admin/audit"),
  });
  return (
    <section className="admin-section audit">
      <div className="section-heading">
        <h2>Журнал действий</h2>
        <button className="text-button" onClick={() => q.refetch()}>
          <RefreshCw size={14} />
          Обновить
        </button>
      </div>
      {q.isPending && <Spinner />}
      <ErrorBox error={q.error} />
      {q.data?.length ? (
        <div className="table-scroll">
          <table>
            <thead>
              <tr>
                <th>Время</th>
                <th>Действие</th>
              </tr>
            </thead>
            <tbody>
              {q.data.map((a) => (
                <tr key={a.id}>
                  <td>
                    <DateText value={a.created_at} />
                  </td>
                  <td>{a.action}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : (
        <p className="muted">Изменения настроек и проверки появятся здесь.</p>
      )}
    </section>
  );
}
