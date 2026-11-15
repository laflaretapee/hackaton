import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import {
  act,
  render,
  screen,
  waitFor,
  within,
  fireEvent,
} from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter } from "react-router-dom";
import { App } from "./App";
import { AdminSettings } from "./admin";
import type { Settings, User } from "./types";
const intern: User = {
  id: "i1",
  name: "Александр Тестовый",
  role: "intern",
  team_id: "t1",
  team_name: "Platform",
  direction: "Backend",
  mentor_name: "Анна Наставник",
  notifications: true,
  reminder_minutes: 5,
  is_demo: true,
};
const settings: Settings = {
  telegram_token_set: true,
  telegram_username: "test_bot",
  telegram_enabled: true,
  routerai_key_set: true,
  routerai_base_url: "https://routerai.ru/api/v1",
  routerai_model: "test-model",
  ai_mock_mode: false,
  app_url: "https://buddy.test",
  ai_timeout_seconds: 60,
  ai_max_tokens: 1500,
};
const tasks = [
  {
    id: "task1",
    title: "Познакомиться с командой",
    description: "Встретиться с наставником",
    status: "not_started",
    kind: "manual",
    due_at: "2026-11-16T10:00:00Z",
    article_id: "git",
    priority: "high",
  },
  {
    id: "task2",
    title: "Получить доступ",
    description: "Оформить заявку",
    status: "waiting_approval",
    kind: "access",
    due_at: "",
    article_id: "",
    priority: "normal",
  },
  {
    id: "task3",
    title: "Изучить Git",
    description: "Пройти экзамен",
    status: "completed",
    kind: "exam",
    due_at: "",
    article_id: "git",
    priority: "low",
  },
];
const articles = [
  {
    id: "git",
    title: "Git и code review",
    category: "Разработка",
    direction: "Backend",
    summary: "Ветки и договорённости команды",
    duration_minutes: 10,
    read: false,
    sections: [
      {
        id: "branches",
        title: "Работа с ветками",
        content: "Создай ветку от main.\n\nКаждое изменение проходит ревью.",
      },
      {
        id: "review",
        title: "Проверка изменений",
        content: "Проверяй тесты до отправки.",
      },
    ],
  },
];
const lessons = [
  {
    id: "lesson1",
    title: "Git на практике",
    description: "Базовые операции с репозиторием",
    direction: "Backend",
    article_id: "git",
    duration_minutes: 15,
    completed: false,
  },
];
const meetings = [
  {
    id: "meeting1",
    title: "Знакомство с командой",
    starts_at: "2026-11-16T12:00:00Z",
    article_id: "git",
    notification_status: "scheduled",
  },
];
const requests = [
  {
    id: "req1",
    resource: "backend-repository",
    reason: "Первая задача",
    status: "pending",
    created_at: "2026-11-12T12:00:00Z",
    user_name: "Александр Тестовый",
    history: [
      {
        status: "created",
        at: "2026-11-12T12:00:00Z",
        note: "Создана стажёром",
      },
    ],
  },
];
const interns = [
  {
    id: "i1",
    name: "Александр Тестовый",
    team_name: "Platform",
    progress: 25,
    completed: 1,
    total: 4,
    weak_topics: ["Git"],
  },
];
const dashboard = {
  progress: 25,
  completed: 1,
  total: 4,
  tasks,
  meetings,
  requests,
  attempts: [
    {
      id: "attempt1",
      lesson_id: "lesson1",
      score: 60,
      correct: 3,
      total: 5,
      feedback: [],
      recommendation: "Изучи ветки",
      created_at: "2026-11-12T12:00:00Z",
    },
  ],
  interns,
  stats: {
    interns: 1,
    mentors: 1,
    teams: 1,
    average_progress: 25,
    ai_messages: 4,
  },
  team_stats: [{ team_name: "Platform", interns: 1, progress: 25 }],
  user: intern,
};
const exam = {
  id: "exam1",
  title: "Основы Git",
  questions: Array.from({ length: 5 }, (_, i) => ({
    id: "q" + i,
    prompt: "Вопрос " + (i + 1),
    options: ["Правильный ответ " + (i + 1), "Другой ответ " + (i + 1)],
  })),
};
let current: User | null;
let admin: boolean;
let messages: unknown[];
let calls: { path: string; method: string; body: any }[];
let overrides: Record<string, ((body: any) => unknown) | { status: number }>;
let storedSettings: Settings;
function setupFetch() {
  vi.stubGlobal(
    "fetch",
    vi.fn(async (url: string, options: RequestInit = {}) => {
      const path = url.replace("/api", "");
      const method = options.method || "GET";
      const body = options.body
        ? JSON.parse(options.body as string)
        : undefined;
      calls.push({ path, method, body });
      const override = overrides[method + " " + path];
      let data: unknown;
      let status = 200;
      if (override) {
        if (typeof override === "function") data = override(body);
        else status = override.status;
      } else if (path === "/me" && method === "GET") {
        data = current;
        if (!current) status = 401;
      } else if (path === "/public/config")
        data = {
          demo_mode: true,
          ai_mock_mode: true,
          telegram_username: "test_bot",
          app_url: "https://buddy.test",
        };
      else if (path === "/teams")
        data = [{ id: "t1", name: "Platform", direction: "Backend" }];
      else if (path === "/auth/demo") {
        current = {
          ...intern,
          role: body.role,
          name: body.name || intern.name,
        };
        data = current;
      } else if (
        path === "/auth/logout" ||
        (path === "/me" && method === "DELETE")
      ) {
        current = null;
        data = { ok: true };
      } else if (path === "/me" && method === "PATCH") {
        current = { ...current!, ...body };
        data = current;
      } else if (path === "/dashboard" || path === "/interns/i1")
        data = dashboard;
      else if (path === "/tasks") data = tasks;
      else if (path.startsWith("/tasks/") && method === "PATCH")
        data = { ...tasks[0], ...body };
      else if (path.startsWith("/knowledge?") || path === "/knowledge")
        data = articles;
      else if (path === "/knowledge/git") data = articles[0];
      else if (path === "/knowledge/git/read") data = { ok: true };
      else if (path === "/lessons") data = lessons;
      else if (path === "/lessons/lesson1/exam")
        data =
          method === "GET"
            ? exam
            : {
                id: "a2",
                lesson_id: "lesson1",
                score: 80,
                correct: 4,
                total: 5,
                recommendation: "Продолжай практиковаться",
                created_at: "2026-11-12T12:00:00Z",
                feedback: [
                  {
                    question: "Вопрос 1",
                    correct: true,
                    answer: "Верно",
                    explanation: "Потому что ветка нужна для работы",
                  },
                  {
                    question: "Вопрос 2",
                    correct: false,
                    answer: "Неверно",
                    explanation: "Повтори раздел про review",
                  },
                ],
              };
      else if (path === "/chat" && method === "GET") data = messages;
      else if (path === "/chat" && method === "POST") {
        messages = [
          ...messages,
          {
            id: "u1",
            role: "user",
            content: body.message,
            created_at: "2026-11-12T12:00:00Z",
          },
          {
            id: "m1",
            role: "assistant",
            content: "Доступ согласует наставник.",
            sources: [
              { title: "Инструкция Git", url: "/knowledge/git#branches" },
            ],
            suggested_actions: ["create_request"],
            created_at: "2026-11-12T12:00:00Z",
          },
        ];
        data = messages.at(-1);
      } else if (path === "/requests")
        data = method === "GET" ? requests : { ...requests[0], ...body };
      else if (path.includes("/decision"))
        data = {
          ...requests[0],
          status: body.approve ? "approved" : "rejected",
        };
      else if (path === "/meetings") data = meetings;
      else if (path === "/meetings/demo") data = meetings[0];
      else if (path === "/interns") data = interns;
      else if (path.startsWith("/interns/i1/")) data = { id: "new", ...body };
      else if (path === "/auth/login-link")
        data = { url: "https://t.me/test_bot?start=token", token: "token" };
      else if (path === "/auth/exchange") status = 409;
      else if (path === "/admin/settings") {
        if (!admin) status = 401;
        else if (method === "GET") data = storedSettings;
        else {
          storedSettings = {
            ...storedSettings,
            ...body,
            telegram_token_set: body.clear_telegram_token ? false : true,
            routerai_key_set: body.clear_routerai_key ? false : true,
          };
          data = storedSettings;
        }
      } else if (path === "/admin/login") {
        admin = true;
        data = { ok: true };
      } else if (path === "/admin/logout") {
        admin = false;
        data = { ok: true };
      } else if (path === "/admin/audit")
        data = [
          {
            id: "a1",
            action: "Настройки обновлены",
            created_at: "2026-11-12T12:00:00Z",
          },
        ];
      else if (path.startsWith("/admin/test/"))
        data = { ok: true, message: "ok", models: [{ id: "model-two" }] };
      else {
        throw new Error("Unexpected request " + method + " " + path);
      }
      return { ok: status < 400, status, json: async () => data } as Response;
    }),
  );
}
function mount(route = "/", user: User | null = intern, staleTime = 0) {
  current = user;
  const client = new QueryClient({
    defaultOptions: {
      queries: { retry: false, staleTime },
      mutations: { retry: false },
    },
  });
  const view = render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={[route]}>
        <App />
      </MemoryRouter>
    </QueryClientProvider>,
  );
  return { ...view, client };
}
beforeEach(() => {
  current = intern;
  admin = false;
  messages = [];
  calls = [];
  overrides = {};
  storedSettings = { ...settings };
  setupFetch();
  Element.prototype.scrollIntoView = vi.fn();
  window.scrollTo = vi.fn();
});
afterEach(() => {
  vi.unstubAllGlobals();
  delete window.Telegram;
});
const click = async (name: string | RegExp) =>
  userEvent.click(await screen.findByRole("button", { name }));
describe("critical regression cases", () => {
  it("never reuses the prior account's fresh chat after expiry and a new login", async () => {
    const privateMessage = (content: string) => [
      {
        id: content,
        role: "assistant",
        content,
        created_at: "2026-11-12T12:00:00Z",
      },
    ];
    overrides["GET /chat"] = () =>
      privateMessage(
        current?.id === "i2"
          ? "Только переписка Бориса"
          : "Личная переписка Александра",
      );
    const { client } = mount("/chat", intern, 15000);
    await screen.findByText("Личная переписка Александра");
    client.setQueryData(["tasks"], tasks);
    client.setQueryData(["dashboard"], dashboard);
    let completeOldRequest!: (value: typeof requests) => void;
    const oldRequest = client
      .fetchQuery({
        queryKey: ["requests"],
        queryFn: () =>
          new Promise<typeof requests>((resolve) => {
            completeOldRequest = resolve;
          }),
      })
      .catch(() => undefined);
    current = null;
    await act(() => client.invalidateQueries({ queryKey: ["me"] }));
    await screen.findByText("Начнём знакомство");
    overrides["POST /auth/demo"] = () => {
      current = { ...intern, id: "i2", name: "Борис Новый" };
      return current;
    };
    let completeNewProfile: ((value: User) => void) | undefined;
    overrides["GET /me"] = () =>
      new Promise<User>((resolve) => {
        completeNewProfile = resolve;
      });
    const displayedDuringLogin: string[] = [];
    const observer = new MutationObserver(() =>
      displayedDuringLogin.push(document.body.textContent || ""),
    );
    observer.observe(document.body, {
      subtree: true,
      childList: true,
      characterData: true,
    });
    await click("Попробовать демо");
    await waitFor(() => expect(completeNewProfile).toBeTypeOf("function"));
    expect(screen.queryByRole("navigation")).not.toBeInTheDocument();
    expect(screen.queryByText(intern.name)).not.toBeInTheDocument();
    await act(async () => {
      completeNewProfile!(current!);
    });
    await screen.findByText("Только переписка Бориса");
    await act(async () => {
      completeOldRequest(requests);
      await oldRequest;
    });
    observer.disconnect();
    expect(
      screen.queryByText("Личная переписка Александра"),
    ).not.toBeInTheDocument();
    expect(
      displayedDuringLogin.some(
        (text) =>
          text.includes("Личная переписка Александра") ||
          text.includes(intern.name),
      ),
    ).toBe(false);
    expect(
      calls.filter((call) => call.path === "/chat" && call.method === "GET"),
    ).toHaveLength(2);
    expect(client.getQueryData(["tasks"])).toBeUndefined();
    expect(client.getQueryData(["dashboard"])).toBeUndefined();
    expect(client.getQueryData(["requests"])).toBeUndefined();
  });
  it("expires cached session when me refetch returns 401", async () => {
    const { client } = mount("/tasks");
    await screen.findByRole("heading", { name: "Познакомиться с командой" });
    current = null;
    await act(() => client.invalidateQueries({ queryKey: ["me"] }));
    await screen.findByText("Начнём знакомство");
    expect(screen.queryByRole("navigation")).not.toBeInTheDocument();
  });
  it("shows mentor exam scores and recommendation in intern profile", async () => {
    mount("/interns/i1", { ...intern, role: "mentor" });
    await screen.findByRole("heading", { name: "Результаты обучения" });
    expect(screen.getByText("60%")).toBeInTheDocument();
    expect(screen.getByText("Изучи ветки")).toBeInTheDocument();
  });
  it("stops typing and keeps a retryable input after chat failure", async () => {
    overrides["POST /chat"] = { status: 503 };
    mount("/chat");
    await userEvent.type(
      await screen.findByRole("textbox", { name: "Сообщение AI-наставнику" }),
      "Помоги с Git",
    );
    await click("Отправить сообщение");
    await screen.findByRole("alert");
    expect(
      screen.queryByText("AI-buddy готовит ответ"),
    ).not.toBeInTheDocument();
    expect(
      screen.getByRole("textbox", { name: "Сообщение AI-наставнику" }),
    ).toHaveValue("Помоги с Git");
  });
  it("synchronizes form switches after deleting saved Telegram token", async () => {
    admin = true;
    mount("/admin");
    await screen.findByText("Интеграции и настройки");
    await click("Удалить токен");
    await click("Удалить секрет");
    await waitFor(() =>
      expect(screen.queryByRole("dialog")).not.toBeInTheDocument(),
    );
    expect(
      screen.getByRole("checkbox", { name: "Включить Telegram-интеграцию" }),
    ).not.toBeChecked();
    await click("Сохранить настройки");
    await waitFor(() =>
      expect(
        calls
          .filter((c) => c.path === "/admin/settings" && c.method === "PUT")
          .at(-1)?.body.telegram_enabled,
      ).toBe(false),
    );
  });
  it("disables unavailable Telegram login with explanation", async () => {
    overrides["GET /public/config"] = () => ({
      demo_mode: true,
      ai_mock_mode: true,
      telegram_username: "",
      app_url: "",
    });
    mount("/", null);
    expect(
      await screen.findByRole("button", { name: "Войти через Telegram" }),
    ).toBeDisabled();
    expect(screen.getByText(/Telegram пока не настроен/)).toBeInTheDocument();
  });
  it("describes shared mentor reset as a role switch", async () => {
    mount("/profile", {
      ...intern,
      id: "demo-mentor-platform",
      role: "mentor",
    });
    expect(
      await screen.findByRole("button", {
        name: "Выйти и выбрать другую роль",
      }),
    ).toBeInTheDocument();
  });
});
describe("intern workflow", () => {
  it("uses real dashboard progress and links", async () => {
    mount();
    await screen.findByText("Привет, Александр!");
    expect(
      await screen.findByText("Выполнено 1 из 4 задач адаптации"),
    ).toBeInTheDocument();
    expect(screen.getByRole("progressbar")).toHaveAttribute(
      "aria-valuenow",
      "25",
    );
    expect(
      screen.getByRole("link", { name: /Продолжить адаптацию/ }),
    ).toHaveAttribute("href", "/tasks");
    expect(screen.getByText("Анна Наставник")).toBeInTheDocument();
  });
  it("completes only manual tasks and filters completed steps", async () => {
    mount("/tasks");
    await click(/Готово/);
    await waitFor(() =>
      expect(
        calls.some(
          (c) => c.path === "/tasks/task1" && c.body?.status === "completed",
        ),
      ).toBe(true),
    );
    expect(screen.getAllByRole("button", { name: /Готово/ })).toHaveLength(1);
    await click("Выполненные");
    expect(
      screen.queryByText("Познакомиться с командой"),
    ).not.toBeInTheDocument();
    expect(
      screen.getByRole("heading", { name: "Изучить Git" }),
    ).toBeInTheDocument();
    await click("В работе");
    expect(
      screen.queryByRole("heading", { name: "Изучить Git" }),
    ).not.toBeInTheDocument();
    await click("Все задачи");
  });
  it("searches and filters knowledge with encoded server query", async () => {
    mount("/knowledge");
    await screen.findByText("Git и code review");
    await userEvent.type(
      screen.getByRole("textbox", { name: "Поиск по базе знаний" }),
      "ветки",
    );
    await waitFor(() =>
      expect(
        calls.some((c) => c.path.includes("q=" + encodeURIComponent("ветки"))),
      ).toBe(true),
    );
    await userEvent.selectOptions(
      screen.getByRole("combobox", { name: "Направление" }),
      "Backend",
    );
    await click(/Разработка/);
    expect(
      screen.getByRole("link", { name: /Git и code review/ }),
    ).toHaveAttribute("href", "/knowledge/git");
    await click("Очистить поиск");
    expect(
      screen.getByRole("textbox", { name: "Поиск по базе знаний" }),
    ).toHaveValue("");
  });
  it("marks anchored article read and opens related exam", async () => {
    mount("/knowledge/git#branches");
    await screen.findByRole("heading", { name: "Работа с ветками" });
    expect(
      screen.getByRole("link", { name: "Работа с ветками" }),
    ).toHaveAttribute("href", "#branches");
    await click("Отметить изученным");
    await waitFor(() =>
      expect(
        calls.some(
          (c) => c.path === "/knowledge/git/read" && c.method === "POST",
        ),
      ).toBe(true),
    );
    expect(screen.getByRole("link", { name: /Проверь себя/ })).toHaveAttribute(
      "href",
      "/learning/lesson1",
    );
    expect(
      screen.getByRole("link", { name: "Разобрать с AI" }).getAttribute("href"),
    ).toContain("lesson=lesson1");
  });
  it("lists lessons and submits five selected exam answers", async () => {
    mount("/learning");
    const link = await screen.findByRole("link", { name: /Проверить знания/ });
    await userEvent.click(link);
    await screen.findByRole("heading", { name: "Основы Git" });
    expect(screen.getByRole("button", { name: /Завершить/ })).toBeDisabled();
    for (let i = 1; i <= 5; i++)
      await userEvent.click(
        screen.getByRole("radio", { name: "Правильный ответ " + i }),
      );
    await click(/Завершить/);
    await screen.findByText("4 из 5 правильных ответов");
    expect(
      calls.find(
        (c) => c.path === "/lessons/lesson1/exam" && c.method === "POST",
      )?.body.answers,
    ).toEqual({ q0: 0, q1: 0, q2: 0, q3: 0, q4: 0 });
    expect(screen.getByText("Повтори раздел про review")).toBeInTheDocument();
    await click("Попробовать ещё раз");
    expect(screen.getByRole("button", { name: /Завершить/ })).toBeDisabled();
  });
  it("sends chat, displays sources and requires explicit request confirmation", async () => {
    mount("/chat");
    await click("Как получить доступ к репозиторию?");
    await click("Отправить сообщение");
    await screen.findByText("Доступ согласует наставник.");
    expect(
      screen.getByRole("link", { name: "Инструкция Git" }),
    ).toHaveAttribute("href", "/knowledge/git#branches");
    await click(/Оформить заявку/);
    expect(
      calls.some((c) => c.path === "/requests" && c.method === "POST"),
    ).toBe(false);
    const dialog = screen.getByRole("dialog");
    expect(
      within(dialog).getByRole("button", { name: "Подтвердить и отправить" }),
    ).toBeDisabled();
    await userEvent.click(within(dialog).getByRole("checkbox"));
    await click("Подтвердить и отправить");
    await waitFor(() =>
      expect(screen.queryByRole("dialog")).not.toBeInTheDocument(),
    );
    expect(
      calls.find((c) => c.path === "/requests" && c.method === "POST")?.body,
    ).toEqual({
      resource: "backend-repository",
      reason: "Как получить доступ к репозиторию?",
      confirmed: true,
    });
  });
  it("opens new request, validates fields, preserves reason and shows history", async () => {
    mount("/requests");
    await screen.findByText("backend-repository");
    await click("История заявки");
    expect(screen.getByText("Создана стажёром")).toBeInTheDocument();
    await click("Скрыть историю");
    await click("Новая заявка");
    const dialog = screen.getByRole("dialog");
    await userEvent.type(
      within(dialog).getByRole("textbox", {
        name: "Для какой задачи нужен доступ",
      }),
      "Нужен для onboarding",
    );
    await userEvent.click(within(dialog).getByRole("checkbox"));
    await click("Подтвердить и отправить");
    await waitFor(() =>
      expect(calls.some((c) => c.body?.reason === "Нужен для onboarding")).toBe(
        true,
      ),
    );
  });
  it("creates a two minute meeting through API", async () => {
    mount("/meetings");
    await click("Тестовая встреча");
    await screen.findByText(/Встреча создана/);
    expect(
      calls.some((c) => c.path === "/meetings/demo" && c.method === "POST"),
    ).toBe(true);
    expect(
      screen.getByRole("link", { name: "Подготовиться к встрече" }),
    ).toHaveAttribute("href", "/knowledge/git");
  });
  it("updates notification preferences and logs out to registration", async () => {
    mount("/profile");
    await screen.findByRole("heading", { name: "Профиль и настройки" });
    await userEvent.click(screen.getByRole("switch"));
    await userEvent.selectOptions(
      screen.getByRole("combobox", { name: "Напомнить за" }),
      "15",
    );
    await click("Сохранить настройки");
    await screen.findByText("Настройки сохранены");
    expect(
      calls.find((c) => c.path === "/me" && c.method === "PATCH")?.body,
    ).toEqual({ notifications: false, reminder_minutes: 15 });
    await click("Выйти");
    await screen.findByText("Начнём знакомство");
  });
  it("confirms destructive profile reset before deleting", async () => {
    mount("/profile");
    await click("Сбросить демопрофиль");
    expect(calls.some((c) => c.path === "/me" && c.method === "DELETE")).toBe(
      false,
    );
    await click("Отмена");
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
    await click("Сбросить демопрофиль");
    await click("Удалить и начать заново");
    await screen.findByText("Начнём знакомство");
    expect(calls.some((c) => c.path === "/me" && c.method === "DELETE")).toBe(
      true,
    );
  });
});
describe("roles and protected admin", () => {
  it("passes signed Mini App initData unchanged to backend verification", async () => {
    const ready = vi.fn();
    const expand = vi.fn();
    window.Telegram = {
      WebApp: {
        initData: "signed=telegram-data&hash=signature",
        ready,
        expand,
      },
    };
    overrides["POST /auth/telegram"] = () => {
      current = intern;
      return intern;
    };
    mount("/", null);
    await screen.findByText("Привет, Александр!");
    expect(calls.find((c) => c.path === "/auth/telegram")?.body).toEqual({
      init_data: "signed=telegram-data&hash=signature",
    });
    expect(ready).toHaveBeenCalled();
    expect(expand).toHaveBeenCalled();
  });

  it("creates selected synthetic role and team from login", async () => {
    mount("/", null);
    await screen.findByText("Начнём знакомство");
    await click("Руководитель");
    await userEvent.type(
      screen.getByRole("textbox", { name: "Как тебя зовут" }),
      "Мария",
    );
    await userEvent.selectOptions(
      screen.getByRole("combobox", { name: "Команда" }),
      "t1",
    );
    await click("Попробовать демо");
    await screen.findByRole("heading", { name: "Как проходит адаптация" });
    expect(calls.find((c) => c.path === "/auth/demo")?.body).toEqual({
      role: "manager",
      name: "Мария",
      team_id: "t1",
    });
    expect(
      screen.queryByRole("link", { name: "Мои стажёры" }),
    ).not.toBeInTheDocument();
  });
  it("shows Telegram browser confirmation link", async () => {
    mount("/", null);
    await click("Войти через Telegram");
    expect(
      await screen.findByRole("link", { name: /Открыть бота/ }),
    ).toHaveAttribute("href", "https://t.me/test_bot?start=token");
    expect(screen.getByText(/Ожидаем подтверждение/)).toBeInTheDocument();
  });
  it("mentor sees assigned interns and makes approval decisions", async () => {
    mount("/", { ...intern, role: "mentor" });
    await screen.findByText("Твои стажёры");
    await click("Согласовать");
    await waitFor(() =>
      expect(
        calls.some(
          (c) =>
            c.path === "/requests/req1/decision" && c.body?.approve === true,
        ),
      ).toBe(true),
    );
    await click("Отклонить");
    await waitFor(() =>
      expect(
        calls.some(
          (c) =>
            c.path === "/requests/req1/decision" && c.body?.approve === false,
        ),
      ).toBe(true),
    );
    await userEvent.click(screen.getByRole("link", { name: "Открыть список" }));
    expect(
      await screen.findByRole("heading", { name: "Мои стажёры" }),
    ).toBeInTheDocument();
  });
  it("mentor assigns task and meeting with selected material", async () => {
    mount("/interns/i1", { ...intern, role: "mentor" });
    await screen.findByRole("heading", { name: "Александр Тестовый" });
    await click("Назначить задачу");
    let dialog = screen.getByRole("dialog");
    await userEvent.type(
      within(dialog).getByRole("textbox", { name: "Название" }),
      "Моя задача",
    );
    await userEvent.type(
      within(dialog).getByRole("textbox", { name: "Описание" }),
      "Описание задачи",
    );
    fireEvent.change(within(dialog).getByLabelText("Срок выполнения"), {
      target: { value: "2026-12-01T10:00" },
    });
    await click("Назначить");
    await waitFor(() =>
      expect(screen.queryByRole("dialog")).not.toBeInTheDocument(),
    );
    expect(calls.find((c) => c.path === "/interns/i1/tasks")?.body.title).toBe(
      "Моя задача",
    );
    await click("Назначить встречу");
    dialog = screen.getByRole("dialog");
    await userEvent.type(
      within(dialog).getByRole("textbox", { name: "Название" }),
      "Разбор Git",
    );
    fireEvent.change(within(dialog).getByLabelText("Дата и время"), {
      target: { value: "2026-12-01T11:00" },
    });
    await userEvent.selectOptions(within(dialog).getByRole("combobox"), "git");
    await click("Назначить");
    await waitFor(() =>
      expect(
        calls.find((c) => c.path === "/interns/i1/meetings")?.body.article_id,
      ).toBe("git"),
    );
  });
  it("manager renders scoped aggregate metrics", async () => {
    mount("/analytics", { ...intern, role: "manager" });
    await screen.findByRole("heading", { name: "Как проходит адаптация" });
    expect(
      await screen.findByText("Адаптация по командам"),
    ).toBeInTheDocument();
    expect(
      screen.getByText("Platform", { selector: "td strong" }),
    ).toBeInTheDocument();
    expect(screen.getByRole("progressbar")).toHaveAttribute(
      "aria-valuenow",
      "25",
    );
    expect(
      screen.queryByRole("link", { name: "План адаптации" }),
    ).not.toBeInTheDocument();
  });
  it("admin requires separate password and never prepopulates secrets", async () => {
    mount("/admin");
    await userEvent.type(
      await screen.findByLabelText("Пароль"),
      "admin-test-password",
    );
    await click("Войти в настройки");
    await screen.findByText("Интеграции и настройки");
    expect(screen.getByLabelText("Токен бота")).toHaveValue("");
    expect(screen.getByLabelText("API-ключ")).toHaveValue("");
    expect(screen.getByLabelText("Токен бота")).toHaveAttribute(
      "type",
      "password",
    );
    await userEvent.clear(screen.getByLabelText("Модель"));
    await userEvent.type(screen.getByLabelText("Модель"), "new-model");
    await click("Сохранить настройки");
    await screen.findByText(/Настройки сохранены. Поля секретов очищены/);
    const payload = calls.find(
      (c) => c.path === "/admin/settings" && c.method === "PUT",
    )?.body;
    expect(payload.routerai_model).toBe("new-model");
    expect(payload).not.toHaveProperty("telegram_token");
    expect(payload).not.toHaveProperty("routerai_api_key");
    await click("Проверить RouterAI");
    await screen.findByText(/RouterAI: соединение установлено/);
    await click("Проверить Telegram");
    await screen.findByText(/Telegram: соединение установлено/);
    await click("Обновить");
    await click("Выйти");
    await screen.findByRole("heading", { name: "Вход администратора" });
  });
  it("saves new secrets then clears fields and deletes AI key explicitly", async () => {
    const storageWrite = vi.fn();
    vi.stubGlobal("localStorage", {
      setItem: storageWrite,
      getItem: vi.fn(),
      removeItem: vi.fn(),
      clear: vi.fn(),
      length: 0,
    });
    admin = true;
    mount("/admin");
    await screen.findByText("Интеграции и настройки");
    await userEvent.type(screen.getByLabelText("API-ключ"), "new-private-key");
    await userEvent.type(
      screen.getByLabelText("Токен бота"),
      "new-private-token",
    );
    await click("Сохранить настройки");
    await waitFor(() =>
      expect(screen.getByLabelText("API-ключ")).toHaveValue(""),
    );
    expect(
      calls.find((c) => c.path === "/admin/settings" && c.method === "PUT")
        ?.body.routerai_api_key,
    ).toBe("new-private-key");
    expect(storageWrite).not.toHaveBeenCalled();
    await click("Удалить API-ключ");
    await click("Удалить секрет");
    await waitFor(() =>
      expect(screen.queryByRole("dialog")).not.toBeInTheDocument(),
    );
    expect(
      screen.getByRole("checkbox", { name: /Демонстрационный AI-режим/ }),
    ).toBeChecked();
  });
});
describe("initial connection recovery", () => {
  it("shows retry for failed session lookup instead of an unexplained login", async () => {
    overrides["GET /me"] = { status: 503 };
    mount("/");
    expect(await screen.findByRole("alert")).toHaveTextContent(
      "Сервис временно недоступен",
    );
    delete overrides["GET /me"];
    await click("Повторить");
    await screen.findByText("Привет, Александр!");
  });
});
describe("errors, empty states and accessibility", () => {
  it("shows API error with retry that reloads tasks", async () => {
    overrides["GET /tasks"] = { status: 503 };
    mount("/tasks");
    await screen.findByRole("alert");
    delete overrides["GET /tasks"];
    await click("Повторить");
    await screen.findByRole("heading", { name: "Познакомиться с командой" });
  });
  it("shows permission errors without displaying server secrets", async () => {
    overrides["GET /requests"] = { status: 403 };
    mount("/requests");
    expect(await screen.findByRole("alert")).toHaveTextContent(
      "Для этого действия недостаточно прав.",
    );
  });
  it("shows empty knowledge and task states", async () => {
    overrides["GET /knowledge?q=&direction="] = () => [];
    mount("/knowledge");
    await screen.findByText(/Ничего не нашли/);
  });
  it("opens and closes responsive nav and keyboard modal", async () => {
    mount("/requests");
    await click("Открыть меню");
    expect(screen.getByRole("complementary")).toHaveClass("open");
    await click("Закрыть меню");
    expect(screen.getByRole("complementary")).not.toHaveClass("open");
    await click("Новая заявка");
    fireEvent.keyDown(document, { key: "Escape" });
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });
  it("returns unknown route to overview", async () => {
    mount("/missing");
    await screen.findByRole("heading", { name: "Страница не найдена" });
    await userEvent.click(
      screen.getByRole("link", { name: "Вернуться к обзору" }),
    );
    await screen.findByText("Привет, Александр!");
  });
});
