import {
  useCallback,
  useEffect,
  useRef,
  useState,
  type FormEvent,
} from "react";
import {
  Link,
  useLocation,
  useParams,
  useSearchParams,
} from "react-router-dom";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  ArrowRight,
  ArrowUpRight,
  BookOpen,
  CalendarDays,
  Check,
  CheckCircle2,
  ChevronRight,
  Circle,
  Clock3,
  FileText,
  GraduationCap,
  MessageSquare,
  Plus,
  Search,
  Send,
  Sparkles,
  Users,
  XCircle,
  LogOut,
  RotateCcw,
  ShieldCheck,
} from "lucide-react";
import { api, safeSource } from "./api";
import type {
  Article,
  Attempt,
  Config,
  Dashboard,
  Exam,
  Intern,
  Lesson,
  Meeting,
  Message,
  RequestItem,
  Task,
  User,
} from "./types";
import {
  DateText,
  directionTitle,
  Empty,
  ErrorBox,
  Modal,
  PageTitle,
  Progress,
  roles,
  Spinner,
  Status,
  TextLink,
} from "./ui";
function useData<T>(key: string, path: string = key) {
  return useQuery<T>({ queryKey: [key], queryFn: () => api<T>("/" + path) });
}
function useAction<T = unknown>(fn: (v: T) => Promise<unknown>) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: fn,
    onSettled: () => qc.invalidateQueries(),
  });
}
function QueryState({
  query,
}: {
  query: { isPending: boolean; error: unknown; refetch: () => unknown };
}) {
  return query.isPending ? (
    <Spinner />
  ) : (
    <ErrorBox error={query.error} retry={() => query.refetch()} />
  );
}
export function Overview({ user }: { user: User }) {
  const q = useData<Dashboard>("dashboard");
  const d = q.data;
  const nextMeeting = d?.meetings
    ?.filter((m) => new Date(m.starts_at).getTime() > Date.now())
    .sort(
      (a, b) =>
        new Date(a.starts_at).getTime() - new Date(b.starts_at).getTime(),
    )[0];
  return (
    <>
      <PageTitle
        eyebrow={
          user.role === "intern"
            ? "АДАПТАЦИЯ · ПЕРВЫЕ ШАГИ"
            : "ПРОСТРАНСТВО НАСТАВНИКА"
        }
        title={`Привет, ${user.name.split(" ")[0]}!`}
        description={
          user.role === "intern"
            ? "Большой путь начинается с небольших шагов. Вот твой план на сегодня."
            : "Помоги команде уверенно пройти первые шаги."
        }
      />
      <QueryState query={q} />
      {d &&
        (user.role === "intern" ? (
          <>
            <section className="journey-hero">
              <div className="journey-copy">
                <span className="eyebrow">ТВОЙ ПРОГРЕСС</span>
                <h2>
                  {d.progress >= 100
                    ? "Отличная работа. Продолжай расти."
                    : "Всё получится. Шаг за шагом."}
                </h2>
                <p>
                  Выполнено {d.completed} из {d.total} задач адаптации
                </p>
                <Progress value={d.progress} />
                <Link className="button primary" to="/tasks">
                  Продолжить адаптацию <ArrowRight size={17} />
                </Link>
              </div>
              <div className="journey-visual" aria-hidden="true">
                <div className="orbit-ring" />
                <span className="orbit-step step-one">
                  <Check size={22} />
                </span>
                <span className="orbit-step step-two">
                  <BookOpen size={23} />
                </span>
                <span className="orbit-step step-three">
                  <Sparkles size={27} />
                </span>
                <div className="progress-circle">
                  <strong>
                    {Math.round(d.progress)}
                    <small>%</small>
                  </strong>
                  <span>уже пройдено</span>
                </div>
              </div>
            </section>
            <div className="dashboard-columns">
              <section className="section-block">
                <div className="section-heading">
                  <h2>Сейчас в фокусе</h2>
                  <TextLink to="/tasks">Все задачи</TextLink>
                </div>
                <TaskList
                  tasks={(d.tasks || [])
                    .filter((t) => t.status !== "completed")
                    .slice(0, 4)}
                  compact
                />
                <div className="mentor-strip">
                  <span className="avatar green">
                    <Users size={19} />
                  </span>
                  <div>
                    <small>ТВОЙ НАСТАВНИК</small>
                    <strong>
                      {user.mentor_name || "Наставник ещё не назначен"}
                    </strong>
                    <p>Рядом, чтобы помочь тебе освоиться</p>
                  </div>
                </div>
              </section>
              <aside className="dashboard-aside">
                <section className="upcoming">
                  <div className="section-heading">
                    <h2>Ближайшая встреча</h2>
                    <CalendarDays size={19} />
                  </div>
                  {nextMeeting ? (
                    <MeetingRow meeting={nextMeeting} />
                  ) : (
                    <p className="muted">Пока нет запланированных встреч</p>
                  )}
                  <TextLink to="/meetings">Календарь встреч</TextLink>
                </section>
                <section className="ask-panel">
                  <span className="spark-icon">
                    <Sparkles size={22} />
                  </span>
                  <h2>
                    Есть вопрос?
                    <br />
                    Давай разберёмся.
                  </h2>
                  <p>
                    Про процессы, Git или первую задачу — спрашивай всё, что
                    важно.
                  </p>
                  <Link className="button secondary" to="/chat">
                    Спросить AI-buddy <ArrowUpRight size={16} />
                  </Link>
                </section>
              </aside>
            </div>
            <RecommendedLesson />
            {d.attempts?.length > 0 && (
              <section className="section-block">
                <div className="section-heading">
                  <h2>Последние результаты</h2>
                  <TextLink to="/learning">К обучению</TextLink>
                </div>
                {d.attempts.slice(0, 3).map((a) => (
                  <div className="result-row" key={a.id}>
                    <GraduationCap size={20} />
                    <span>
                      Проверка знаний · {a.correct} из {a.total} верно
                    </span>
                    <strong>{a.score}%</strong>
                    <DateText value={a.created_at} />
                  </div>
                ))}
              </section>
            )}
          </>
        ) : (
          <>
            <div className="metrics">
              <Metric label="Стажёров рядом" value={d.interns?.length || 0} />
              <Metric
                label="Заявок ожидают решения"
                value={
                  (d.requests || []).filter((r) =>
                    [
                      "pending",
                      "pending_approval",
                      "waiting_approval",
                      "created",
                    ].includes(r.status),
                  ).length
                }
              />
              <Metric
                label="Средний прогресс"
                value={`${Math.round(d.stats?.average_progress || 0)}%`}
              />
            </div>
            <section className="section-block">
              <div className="section-heading">
                <h2>Твои стажёры</h2>
                <TextLink to="/interns">Открыть список</TextLink>
              </div>
              <InternTable interns={d.interns || []} />
            </section>
            <section className="section-block">
              <div className="section-heading">
                <h2>Доступы и согласования</h2>
                <TextLink to="/requests">Все заявки</TextLink>
              </div>
              <RequestList requests={(d.requests || []).slice(0, 5)} mentor />
            </section>
          </>
        ))}
    </>
  );
}
function RecommendedLesson() {
  const query = useData<Lesson[]>("lessons");
  const lesson = query.data?.find((item) => !item.completed);
  if (!lesson) return null;
  return (
    <section className="section-block">
      <div className="section-heading">
        <h2>Следующий шаг в обучении</h2>
        <TextLink to="/learning">Все занятия</TextLink>
      </div>
      <Link className="next-lesson" to={"/knowledge/" + lesson.article_id}>
        <GraduationCap size={24} />
        <div>
          <strong>{lesson.title}</strong>
          <span>{lesson.duration_minutes} минут · Теория и практика</span>
        </div>
        <ArrowRight size={20} />
      </Link>
    </section>
  );
}
export function TasksPage() {
  const q = useData<Task[]>("tasks");
  const [filter, setFilter] = useState("all");
  return (
    <>
      <PageTitle
        eyebrow="ТВОЙ МАРШРУТ"
        title="План адаптации"
        description="Знакомься с командой, изучай процессы и отмечай сделанное."
      />
      <div className="tabs">
        {[
          ["all", "Все задачи"],
          ["active", "В работе"],
          ["completed", "Выполненные"],
        ].map(([id, title]) => (
          <button
            key={id}
            className={filter === id ? "active" : ""}
            onClick={() => setFilter(id)}
          >
            {title}
          </button>
        ))}
      </div>
      <QueryState query={q} />
      {q.data && (
        <TaskList
          tasks={q.data.filter(
            (t) =>
              filter === "all" ||
              (filter === "active"
                ? t.status !== "completed"
                : t.status === "completed"),
          )}
        />
      )}
    </>
  );
}
function TaskList({
  tasks,
  compact = false,
  readonly = false,
}: {
  tasks: Task[];
  compact?: boolean;
  readonly?: boolean;
}) {
  const action = useAction<{ id: string; status: string }>((v) =>
    api("/tasks/" + encodeURIComponent(v.id), {
      method: "PATCH",
      body: JSON.stringify({ status: v.status }),
    }),
  );
  if (!tasks.length)
    return (
      <Empty>
        Здесь пока нет задач. Отличный момент посмотреть базу знаний.
      </Empty>
    );
  return (
    <div className="task-list">
      <ErrorBox error={action.error} />
      {tasks.map((t) => (
        <article
          className={`task-row ${t.status === "completed" ? "done" : ""}`}
          key={t.id}
        >
          <span className="task-check">
            {t.status === "completed" ? (
              <CheckCircle2 size={23} />
            ) : (
              <Circle size={23} />
            )}
          </span>
          <div className="task-content">
            <div className="task-title-line">
              <h3>{t.title}</h3>
              <Status value={t.status} />
            </div>
            {!compact && <p>{t.description}</p>}
            <div className="task-meta">
              {t.priority && (
                <span>
                  {t.priority === "high"
                    ? "Высокий приоритет"
                    : t.priority === "low"
                      ? "Низкий приоритет"
                      : "Обычный приоритет"}
                </span>
              )}
              <span>
                <Clock3 size={13} />
                <DateText value={t.due_at} />
              </span>
              {t.article_id && (
                <Link to={"/knowledge/" + t.article_id}>
                  Материал <ArrowUpRight size={12} />
                </Link>
              )}
            </div>
          </div>
          <div className="task-actions">
            {!readonly &&
              t.status !== "completed" &&
              (t.kind === "manual" ? (
                <button
                  className="button small secondary"
                  disabled={action.isPending}
                  onClick={() =>
                    action.mutate({ id: t.id, status: "completed" })
                  }
                >
                  Готово <Check size={14} />
                </button>
              ) : (
                <Link
                  className="icon-link"
                  aria-label={"Продолжить: " + t.title}
                  to={t.kind === "access" ? "/requests" : "/learning"}
                >
                  <ArrowRight size={19} />
                </Link>
              ))}
          </div>
        </article>
      ))}
    </div>
  );
}
export function KnowledgePage() {
  const [search, setSearch] = useState("");
  const [direction, setDirection] = useState("");
  const [category, setCategory] = useState("Все материалы");
  const q = useQuery({
    queryKey: ["knowledge", search, direction],
    queryFn: () =>
      api<Article[]>(
        "/knowledge?q=" +
          encodeURIComponent(search) +
          "&direction=" +
          encodeURIComponent(direction),
      ),
  });
  const all = useData<Article[]>("knowledge-categories", "knowledge");
  const categories = [
    "Все материалы",
    ...new Set(all.data?.map((a) => a.category) || []),
  ];
  const directions = [
    ...new Set(all.data?.map((a) => a.direction).filter(Boolean) || []),
  ];
  return (
    <>
      <PageTitle
        eyebrow="ЗНАНИЯ ПОД РУКОЙ"
        title="База знаний"
        description="Инструкции, договорённости и ответы на частые вопросы."
      />
      <div className="search-toolbar">
        <label className="search-input">
          <Search size={20} />
          <input
            aria-label="Поиск по базе знаний"
            placeholder="Что хочешь узнать? Например, Git или доступы"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
          {search && (
            <button
              className="text-button"
              onClick={() => setSearch("")}
              aria-label="Очистить поиск"
            >
              ×
            </button>
          )}
        </label>
        <select
          aria-label="Направление"
          value={direction}
          onChange={(e) => setDirection(e.target.value)}
        >
          <option value="">Все направления</option>
          {directions.map((d) => (
            <option key={d} value={d}>
              {directionTitle(d)}
            </option>
          ))}
        </select>
      </div>
      <div className="knowledge-layout">
        <aside className="category-nav" aria-label="Категории">
          {categories.map((c) => (
            <button
              className={category === c ? "active" : ""}
              key={c}
              onClick={() => setCategory(c)}
            >
              {c}
              <span>
                {all.data?.filter(
                  (a) => c === "Все материалы" || a.category === c,
                ).length || 0}
              </span>
            </button>
          ))}
        </aside>
        <section>
          <div className="section-heading">
            <h2>{category}</h2>
            <span className="muted">
              {q.data?.filter(
                (a) => category === "Все материалы" || a.category === category,
              ).length || 0}{" "}
              материалов
            </span>
          </div>
          <QueryState query={q} />
          {q.data &&
            !q.data.filter(
              (a) => category === "Все материалы" || a.category === category,
            ).length && (
              <Empty>
                Ничего не нашли. Попробуй другой запрос или категорию.
              </Empty>
            )}
          <div className="article-list">
            {q.data
              ?.filter(
                (a) => category === "Все материалы" || a.category === category,
              )
              .map((a) => (
                <Link
                  className="article-item"
                  to={"/knowledge/" + a.id}
                  key={a.id}
                >
                  <div className="article-icon">
                    <FileText size={22} />
                  </div>
                  <div>
                    <div className="article-meta">
                      <span>{a.category}</span>
                      <span>·</span>
                      <span>{a.duration_minutes} мин</span>
                      {a.read && (
                        <span className="read-label">
                          <Check size={13} />
                          Изучено
                        </span>
                      )}
                    </div>
                    <h3>{a.title}</h3>
                    <p>{a.summary}</p>
                  </div>
                  <ArrowUpRight size={20} />
                </Link>
              ))}
          </div>
        </section>
      </div>
    </>
  );
}
export function ArticlePage() {
  const params = useParams();
  const id = params.id || params["*"] || "";
  const loc = useLocation();
  const q = useData<Article>("article-" + id, "knowledge/" + id);
  const lessons = useData<Lesson[]>("lessons");
  const read = useAction(() =>
    api("/knowledge/" + id + "/read", { method: "POST", body: "{}" }),
  );
  useEffect(() => {
    if (q.data && loc.hash)
      setTimeout(
        () =>
          document
            .getElementById(decodeURIComponent(loc.hash.slice(1)))
            ?.scrollIntoView({ block: "start" }),
        80,
      );
  }, [q.data, loc.hash]);
  return (
    <>
      <Link className="back-link" to="/knowledge">
        ← База знаний
      </Link>
      <QueryState query={q} />
      {q.data && (
        <>
          <PageTitle
            eyebrow={q.data.category}
            title={q.data.title}
            description={q.data.summary}
          />
          <div className="article-byline">
            <BookOpen size={16} />
            <span>{q.data.duration_minutes} минут на изучение</span>
            <span>·</span>
            <span>{directionTitle(q.data.direction)}</span>
            {q.data.read && <Status value="completed" />}
          </div>
          <div className="article-layout">
            <article className="article-body">
              {q.data.sections.map((s) => (
                <section id={s.id} key={s.id}>
                  <h2>{s.title}</h2>
                  <ContentText content={s.content} />
                </section>
              ))}
              <div className="article-finish">
                <CheckCircle2 size={27} />
                <div>
                  <h3>Теперь ты знаешь чуть больше</h3>
                  <p>Отметь материал изученным, чтобы сохранить прогресс.</p>
                </div>
                <button
                  className="button primary"
                  disabled={read.isPending || q.data.read}
                  onClick={() => read.mutate(undefined)}
                >
                  {q.data.read ? "Материал изучен" : "Отметить изученным"}
                </button>
              </div>
              <ErrorBox error={read.error} />
              {lessons.data
                ?.filter((l) => l.article_id === q.data?.id)
                .map((l) => (
                  <Link
                    key={l.id}
                    className="next-lesson"
                    to={"/learning/" + l.id}
                  >
                    <GraduationCap size={23} />
                    <div>
                      <strong>Проверь себя: {l.title}</strong>
                      <span>Короткий экзамен с обратной связью</span>
                    </div>
                    <ArrowRight size={22} />
                  </Link>
                ))}
            </article>
            <aside className="article-toc">
              <strong>В ЭТОМ МАТЕРИАЛЕ</strong>
              {q.data.sections.map((s) => (
                <a key={s.id} href={"#" + s.id}>
                  {s.title}
                </a>
              ))}
              <Link
                className="button secondary"
                to={
                  "/chat?message=" +
                  encodeURIComponent(
                    "Объясни материал «" + q.data.title + "» простыми словами",
                  ) +
                  (lessons.data?.find((l) => l.article_id === q.data?.id)
                    ? "&lesson=" +
                      lessons.data.find((l) => l.article_id === q.data?.id)!.id
                    : "")
                }
              >
                <Sparkles size={16} /> Разобрать с AI
              </Link>
            </aside>
          </div>
        </>
      )}
    </>
  );
}
function ContentText({ content }: { content: string }) {
  return (
    <>
      {content.split(/\n\n+/).map((p, i) =>
        p.startsWith("```") ? (
          <pre key={i}>
            <code>{p.replace(/^```\w*\n?|```$/g, "")}</code>
          </pre>
        ) : (
          <p className="article-paragraph" key={i}>
            {p}
          </p>
        ),
      )}
    </>
  );
}
export function LessonsPage() {
  const q = useData<Lesson[]>("lessons");
  return (
    <>
      <PageTitle
        eyebrow="УЧИСЬ И ПРИМЕНЯЙ"
        title="Твоё обучение"
        description="Короткие занятия, практика и проверка знаний — в удобном темпе."
      />
      <QueryState query={q} />
      {q.data && !q.data.length && (
        <Empty>Занятия появятся здесь, когда будет готова программа.</Empty>
      )}
      <div className="lesson-list">
        {q.data?.map((l, i) => (
          <article key={l.id} className="lesson-item">
            <div className="lesson-index">{String(i + 1).padStart(2, "0")}</div>
            <div className="lesson-info">
              <div className="article-meta">
                <span>{directionTitle(l.direction)}</span>
                <span>·</span>
                <span>{l.duration_minutes} мин</span>
                {l.completed && <Status value="completed" />}
              </div>
              <h2>{l.title}</h2>
              <p>{l.description}</p>
              <div className="lesson-actions">
                <Link
                  className="button secondary small"
                  to={"/knowledge/" + l.article_id}
                >
                  Изучить материал <BookOpen size={15} />
                </Link>
                <Link className="inline-link" to={"/learning/" + l.id}>
                  {l.completed ? "Пройти ещё раз" : "Проверить знания"}
                  <ArrowRight size={15} />
                </Link>
              </div>
            </div>
            <GraduationCap className="lesson-decoration" size={40} />
          </article>
        ))}
      </div>
    </>
  );
}
export function ExamPage() {
  const { id } = useParams();
  const q = useData<Exam>("exam-" + id, "lessons/" + id + "/exam");
  const [answers, setAnswers] = useState<Record<string, number>>({});
  const [result, setResult] = useState<Attempt>();
  const send = useAction<Record<string, number>>(async (v) => {
    const r = await api<Attempt>("/lessons/" + id + "/exam", {
      method: "POST",
      body: JSON.stringify({ answers: v }),
    });
    setResult(r);
  });
  return (
    <>
      <Link to="/learning" className="back-link">
        ← Обучение
      </Link>
      <PageTitle
        eyebrow="ПРОВЕРКА ЗНАНИЙ"
        title={q.data?.title || "Короткий экзамен"}
        description="Это безопасное место для ошибок. После экзамена разберём каждый ответ."
      />
      <QueryState query={q} />
      {result ? (
        <>
          <section className="exam-result">
            <div className="score-badge">
              {result.score}
              <small>%</small>
            </div>
            <div>
              <h2>
                {result.score >= 80
                  ? "Отличная работа!"
                  : "Хорошее начало — продолжай практиковаться"}
              </h2>
              <p>
                {result.correct} из {result.total} правильных ответов
              </p>
              <p>{result.recommendation}</p>
            </div>
          </section>
          <section className="feedback-list">
            {result.feedback.map((f, i) => (
              <article key={i}>
                <div
                  className={
                    f.correct ? "feedback-icon correct" : "feedback-icon wrong"
                  }
                >
                  {f.correct ? (
                    <CheckCircle2 size={21} />
                  ) : (
                    <XCircle size={21} />
                  )}
                </div>
                <div>
                  <h3>{f.question}</h3>
                  <p>{f.answer}</p>
                  <p className="muted">{f.explanation}</p>
                </div>
              </article>
            ))}
          </section>
          <div className="button-row">
            <button
              className="button secondary"
              onClick={() => {
                setResult(undefined);
                setAnswers({});
                window.scrollTo(0, 0);
              }}
            >
              Попробовать ещё раз
            </button>
            <Link
              className="button primary"
              to={
                "/chat?lesson=" +
                id +
                "&message=" +
                encodeURIComponent(
                  "Помоги разобрать ошибки в экзамене и дай рекомендации",
                )
              }
            >
              Разобрать с AI <Sparkles size={16} />
            </Link>
          </div>
        </>
      ) : (
        q.data && (
          <form
            className="exam-form"
            onSubmit={(e) => {
              e.preventDefault();
              send.mutate(answers);
            }}
          >
            <div className="exam-summary">
              <span>{q.data.questions.length} вопросов</span>
              <span>
                Отвечено {Object.keys(answers).length} из{" "}
                {q.data.questions.length}
              </span>
            </div>
            {q.data.questions.map((question, i) => (
              <fieldset className="question" key={question.id}>
                <legend>
                  <span>{String(i + 1).padStart(2, "0")}</span>
                  {question.prompt}
                </legend>
                {question.options.map((o, index) => (
                  <label
                    className={
                      answers[question.id] === index
                        ? "answer selected"
                        : "answer"
                    }
                    key={index}
                  >
                    <input
                      type="radio"
                      name={question.id}
                      checked={answers[question.id] === index}
                      onChange={() =>
                        setAnswers({ ...answers, [question.id]: index })
                      }
                      required
                    />
                    <span>{o}</span>
                  </label>
                ))}
              </fieldset>
            ))}
            <ErrorBox error={send.error} />
            <button
              className="button primary"
              disabled={
                send.isPending ||
                Object.keys(answers).length !== q.data.questions.length
              }
            >
              {send.isPending
                ? "Проверяем ответы…"
                : "Завершить и узнать результат"}
              <ArrowRight size={18} />
            </button>
          </form>
        )
      )}
    </>
  );
}
export function ChatPage({ config }: { config?: Config }) {
  const [params] = useSearchParams();
  const [text, setText] = useState(params.get("message") || "");
  const [optimistic, setOptimistic] = useState("");
  const [request, setRequest] = useState(false);
  const q = useData<Message[]>("chat");
  const bottom = useRef<HTMLDivElement>(null);
  const send = useAction<string>(async (message) => {
    setOptimistic(message);
    try {
      await api("/chat", {
        method: "POST",
        body: JSON.stringify({
          message,
          lesson_id: params.get("lesson") || undefined,
        }),
      });
      setText("");
    } finally {
      setOptimistic("");
    }
  });
  useEffect(() => {
    bottom.current?.scrollIntoView({ behavior: "smooth", block: "end" });
  }, [q.data, optimistic]);
  return (
    <div className="chat-page">
      <PageTitle
        eyebrow="ТВОЙ AI-НАСТАВНИК"
        title="Давай разберёмся"
        description="Знаю контекст твоей команды и помогу найти нужный ответ."
        action={
          <span className="badge">
            <span className="tiny-dot" />
            {config?.ai_mock_mode ? "AI · Деморежим" : "AI · RouterAI"}
          </span>
        }
      />
      {config?.ai_mock_mode && (
        <div className="notice slim">
          <Sparkles size={16} /> Ответы демонстрационные, на основе учебной базы
          знаний. RouterAI можно подключить в настройках.
        </div>
      )}
      <div className="chat-messages" aria-live="polite">
        <QueryState query={q} />
        {!q.data?.length && !q.isPending && (
          <div className="chat-welcome">
            <span className="chat-avatar">
              <Sparkles size={29} />
            </span>
            <h2>С чего начнём?</h2>
            <p>Задай вопрос или выбери подсказку ниже</p>
            <div className="prompt-grid">
              {[
                "Как получить доступ к репозиторию?",
                "Объясни процесс code review",
                "С чего начать адаптацию?",
                "Помоги подготовиться к встрече",
              ].map((p) => (
                <button onClick={() => setText(p)} key={p}>
                  {p}
                  <ArrowUpRight size={16} />
                </button>
              ))}
            </div>
          </div>
        )}
        {q.data?.map((m) => (
          <div
            className={
              "message " +
              (m.role === "user" ? "user-message" : "assistant-message")
            }
            key={m.id}
          >
            <div className="message-author">
              {m.role === "user" ? (
                "Ты"
              ) : (
                <>
                  <Sparkles size={15} />
                  AI-buddy
                </>
              )}
              <DateText value={m.created_at} />
            </div>
            <div className="message-content">{m.content}</div>
            {!!m.sources?.length && (
              <div className="sources">
                <strong>
                  <BookOpen size={14} /> Источники
                </strong>
                {m.sources.map((s, i) => (
                  <a key={i} href={safeSource(s.url)}>
                    <FileText size={14} />
                    {s.title}
                    <ArrowUpRight size={13} />
                  </a>
                ))}
              </div>
            )}
            {!!m.suggested_actions?.length && (
              <div className="suggested-action">
                <p>
                  Могу помочь с заявкой на доступ. Данные можно проверить перед
                  отправкой.
                </p>
                <button
                  className="button secondary small"
                  onClick={() => setRequest(true)}
                >
                  Оформить заявку <ArrowRight size={14} />
                </button>
              </div>
            )}
          </div>
        ))}
        {optimistic && (
          <>
            <div className="message user-message">
              <div className="message-author">Ты</div>
              <div className="message-content">{optimistic}</div>
            </div>
            <div className="typing" role="status">
              <Sparkles size={16} /> AI-buddy готовит ответ<span>•••</span>
            </div>
          </>
        )}
        <div ref={bottom} />
      </div>
      <div className="chat-composer-wrap">
        <ErrorBox error={send.error} />
        <form
          className="chat-composer"
          onSubmit={(e) => {
            e.preventDefault();
            if (text.trim()) send.mutate(text.trim());
          }}
        >
          <textarea
            aria-label="Сообщение AI-наставнику"
            placeholder="Спроси о том, что важно сейчас…"
            value={text}
            onChange={(e) => setText(e.target.value)}
            maxLength={4000}
            rows={2}
            onKeyDown={(e) => {
              if (e.key === "Enter" && !e.shiftKey) {
                e.preventDefault();
                if (text.trim() && !send.isPending) send.mutate(text.trim());
              }
            }}
          />
          <button
            className="send-button"
            aria-label="Отправить сообщение"
            disabled={!text.trim() || send.isPending}
          >
            <ArrowRight size={21} />
          </button>
        </form>
        <p className="fine-print">
          AI может ошибаться. Важные решения проверяй по источникам. Доступы
          выдаются в учебной среде.
        </p>
      </div>
      {request && (
        <RequestDialog
          onClose={() => setRequest(false)}
          initialReason={
            q.data?.filter((m) => m.role === "user").at(-1)?.content || ""
          }
        />
      )}
    </div>
  );
}
export function RequestsPage({ user }: { user: User }) {
  const q = useData<RequestItem[]>("requests");
  const [open, setOpen] = useState(false);
  return (
    <>
      <PageTitle
        eyebrow="РАБОЧИЕ ИНСТРУМЕНТЫ"
        title={user.role === "mentor" ? "Заявки на доступ" : "Мои заявки"}
        description={
          user.role === "mentor"
            ? "Проверяй запросы своих стажёров и помогай им двигаться дальше."
            : "Здесь можно следить за доступами и их согласованием."
        }
        action={
          user.role === "intern" && (
            <button className="button primary" onClick={() => setOpen(true)}>
              <Plus size={18} />
              Новая заявка
            </button>
          )
        }
      />
      <div className="notice slim">
        <ShieldCheck size={17} /> Учебная система заявок. Реальные права к
        корпоративным ресурсам не выдаются.
      </div>
      <QueryState query={q} />
      {q.data && (
        <RequestList requests={q.data} mentor={user.role === "mentor"} />
      )}{" "}
      {open && <RequestDialog onClose={() => setOpen(false)} />}
    </>
  );
}
function RequestList({
  requests,
  mentor = false,
}: {
  requests: RequestItem[];
  mentor?: boolean;
}) {
  const action = useAction<{ id: string; approve: boolean }>((v) =>
    api("/requests/" + v.id + "/decision", {
      method: "POST",
      body: JSON.stringify({ approve: v.approve }),
    }),
  );
  const [expanded, setExpanded] = useState("");
  if (!requests.length)
    return (
      <Empty>Заявок пока нет. Когда понадобится доступ, создай первую.</Empty>
    );
  return (
    <div className="request-list">
      <ErrorBox error={action.error} />
      {requests.map((r) => (
        <article className="request-item" key={r.id}>
          <div className="request-top">
            <span className="request-icon">
              <ShieldCheck size={22} />
            </span>
            <div>
              <span className="request-id">
                ЗАЯВКА · {r.id.slice(0, 8).toUpperCase()}
              </span>
              <h3>{r.resource}</h3>
              {mentor && <p className="muted">{r.user_name}</p>}
            </div>
            <Status value={r.status} />
          </div>
          <p className="request-reason">{r.reason}</p>
          <div className="request-bottom">
            <DateText value={r.created_at} />
            <button
              className="text-button"
              onClick={() => setExpanded(expanded === r.id ? "" : r.id)}
            >
              {expanded === r.id ? "Скрыть историю" : "История заявки"}
            </button>
            {mentor &&
              [
                "created",
                "pending",
                "pending_approval",
                "waiting_approval",
              ].includes(r.status) && (
                <div className="button-row">
                  <button
                    className="button small secondary"
                    disabled={action.isPending}
                    onClick={() => action.mutate({ id: r.id, approve: false })}
                  >
                    Отклонить
                  </button>
                  <button
                    className="button small primary"
                    disabled={action.isPending}
                    onClick={() => action.mutate({ id: r.id, approve: true })}
                  >
                    Согласовать <Check size={15} />
                  </button>
                </div>
              )}
          </div>
          {expanded === r.id && (
            <ol className="history">
              {r.history?.map((h, i) => (
                <li key={i}>
                  <Status value={h.status} />
                  <DateText value={h.at} />
                  <p>{h.note}</p>
                </li>
              ))}
            </ol>
          )}
        </article>
      ))}
    </div>
  );
}
export function RequestDialog({
  onClose,
  initialReason = "",
}: {
  onClose: () => void;
  initialReason?: string;
}) {
  const [resource, setResource] = useState("backend-repository");
  const [reason, setReason] = useState(initialReason);
  const [confirmed, setConfirmed] = useState(false);
  const action = useAction(async () => {
    await api("/requests", {
      method: "POST",
      body: JSON.stringify({
        resource: resource.trim(),
        reason: reason.trim(),
        confirmed: true,
      }),
    });
    onClose();
  });
  const close = useCallback(() => onClose(), [onClose]);
  return (
    <Modal title="Заявка на доступ" onClose={close}>
      <p className="muted">
        Проверь данные. Заявка отправится твоему наставнику только после
        подтверждения.
      </p>
      <form
        onSubmit={(e) => {
          e.preventDefault();
          if (confirmed) action.mutate(undefined);
        }}
      >
        <label>
          Ресурс
          <input
            required
            value={resource}
            onChange={(e) => setResource(e.target.value)}
            maxLength={150}
          />
        </label>
        <label>
          Для какой задачи нужен доступ
          <textarea
            required
            value={reason}
            onChange={(e) => setReason(e.target.value)}
            placeholder="Например, для первой задачи в команде"
            maxLength={1000}
            rows={3}
          />
        </label>
        <label className="checkbox-label">
          <input
            type="checkbox"
            checked={confirmed}
            onChange={(e) => setConfirmed(e.target.checked)}
          />
          Подтверждаю создание учебной заявки
        </label>
        <ErrorBox error={action.error} />
        <div className="modal-actions">
          <button className="button secondary" type="button" onClick={close}>
            Отмена
          </button>
          <button
            className="button primary"
            disabled={
              !confirmed ||
              !resource.trim() ||
              !reason.trim() ||
              action.isPending
            }
          >
            {action.isPending ? "Отправляем…" : "Подтвердить и отправить"}
          </button>
        </div>
      </form>
    </Modal>
  );
}
export function MeetingsPage() {
  const q = useData<Meeting[]>("meetings");
  const action = useAction(() =>
    api("/meetings/demo", { method: "POST", body: "{}" }),
  );
  return (
    <>
      <PageTitle
        eyebrow="НА СВЯЗИ С КОМАНДОЙ"
        title="Встречи"
        description="Готовься к разговорам заранее. Нужные материалы всегда рядом."
        action={
          <button
            className="button primary"
            disabled={action.isPending}
            onClick={() => action.mutate(undefined)}
          >
            <Plus size={17} />
            Тестовая встреча
          </button>
        }
      />
      <div className="notice">
        <CalendarDays size={20} />
        <div>
          <strong>Проверим, как работают напоминания?</strong>
          <p>
            Тестовая встреча начнётся через 2 минуты. Telegram-напоминание
            придёт, если бот настроен, а уведомления включены в профиле.
          </p>
        </div>
      </div>
      <ErrorBox error={action.error} />
      {action.isSuccess && (
        <div className="success" role="status">
          Встреча создана. Её статус отображается в списке.
        </div>
      )}
      <QueryState query={q} />
      {q.data && !q.data.length && (
        <Empty>Встреч пока нет. Новые события появятся здесь.</Empty>
      )}
      <div className="meeting-list">
        {q.data?.map((m) => (
          <MeetingRow meeting={m} key={m.id} />
        ))}
      </div>
    </>
  );
}
function MeetingRow({ meeting: m }: { meeting: Meeting }) {
  const date = new Date(m.starts_at);
  return (
    <article className="meeting-row">
      <div className="date-block">
        <strong>{isNaN(date.getTime()) ? "—" : date.getDate()}</strong>
        <span>
          {isNaN(date.getTime())
            ? ""
            : date.toLocaleString("ru-RU", { month: "short" })}
        </span>
      </div>
      <div>
        <h3>{m.title}</h3>
        <p>
          <DateText value={m.starts_at} />
        </p>
        {m.notification_status &&
          (m.notification_status === "pending" ? (
            <span className="badge">Ожидает напоминания</span>
          ) : (
            <Status value={m.notification_status} />
          ))}{" "}
        {m.article_id && (
          <Link className="inline-link" to={"/knowledge/" + m.article_id}>
            Подготовиться к встрече <ArrowUpRight size={14} />
          </Link>
        )}
      </div>
    </article>
  );
}
export function ProfilePage({
  user,
  onExit,
}: {
  user: User;
  onExit: () => void;
}) {
  const sharedMentor = user.id.startsWith("demo-mentor-");
  const [notifications, setNotifications] = useState(user.notifications);
  const [minutes, setMinutes] = useState(user.reminder_minutes);
  const [reset, setReset] = useState(false);
  const save = useAction(() =>
    api("/me", {
      method: "PATCH",
      body: JSON.stringify({ notifications, reminder_minutes: minutes }),
    }),
  );
  const logout = useAction(async () => {
    await api("/auth/logout", { method: "POST", body: "{}" });
    onExit();
  });
  const remove = useAction(async () => {
    await api("/me", { method: "DELETE" });
    onExit();
  });
  return (
    <>
      <PageTitle
        eyebrow="ТВОЁ ПРОСТРАНСТВО"
        title="Профиль и настройки"
        description="Команда, уведомления и управление учебным аккаунтом."
      />
      <div className="profile-layout">
        <section className="profile-summary">
          <span className="avatar large">{user.name.slice(0, 1)}</span>
          <h2>{user.name}</h2>
          <span className="badge">{roles[user.role]}</span>
          <dl>
            <dt>Команда</dt>
            <dd>{user.team_name}</dd>
            <dt>Направление</dt>
            <dd>{directionTitle(user.direction)}</dd>
            {user.role === "intern" && (
              <>
                <dt>Наставник</dt>
                <dd>{user.mentor_name || "Не назначен"}</dd>
              </>
            )}
          </dl>
          {user.is_demo && (
            <span className="demo-chip">
              <span />
              Демонстрационный профиль
            </span>
          )}
        </section>
        <div>
          <section className="settings-section">
            <div className="section-heading">
              <h2>Уведомления</h2>
              <CalendarDays size={20} />
            </div>
            <p className="muted">
              Напомним о встрече в Telegram, чтобы осталось время на подготовку.
            </p>
            <form
              onSubmit={(e) => {
                e.preventDefault();
                save.mutate(undefined);
              }}
            >
              <label className="switch-label">
                <span>
                  <strong>Telegram-напоминания</strong>
                  <small>Перед запланированными встречами</small>
                </span>
                <input
                  type="checkbox"
                  role="switch"
                  checked={notifications}
                  onChange={(e) => setNotifications(e.target.checked)}
                />
              </label>
              <label>
                Напомнить за
                <select
                  value={minutes}
                  onChange={(e) => setMinutes(Number(e.target.value))}
                >
                  {[1, 2, 5, 10, 15, 30, 60].map((n) => (
                    <option key={n} value={n}>
                      {n} мин.
                    </option>
                  ))}
                </select>
              </label>
              <ErrorBox error={save.error} />
              {save.isSuccess && (
                <p className="saved" role="status">
                  <Check size={15} />
                  Настройки сохранены
                </p>
              )}
              <button className="button primary" disabled={save.isPending}>
                Сохранить настройки
              </button>
            </form>
          </section>
          <section className="settings-section">
            <h2>Учётная запись</h2>
            <p className="muted">
              {sharedMentor
                ? "Это общий учебный аккаунт наставника. Для смены роли выйди и зарегистрируйся заново."
                : "Для смены роли можно создать новый учебный профиль."}
            </p>
            <div className="button-row">
              <button
                className="button secondary"
                onClick={() => logout.mutate(undefined)}
                disabled={logout.isPending}
              >
                <LogOut size={16} />
                Выйти
              </button>
              {user.is_demo && (
                <button
                  className="button danger-outline"
                  onClick={() => setReset(true)}
                >
                  <RotateCcw size={16} />
                  {sharedMentor
                    ? "Выйти и выбрать другую роль"
                    : "Сбросить демопрофиль"}
                </button>
              )}
            </div>
            <ErrorBox error={logout.error} />
          </section>
        </div>
      </div>
      {reset && (
        <Modal
          title={
            sharedMentor
              ? "Выйти и выбрать другую роль?"
              : "Сбросить демопрофиль?"
          }
          onClose={() => setReset(false)}
        >
          <p>
            {sharedMentor
              ? "Сессия завершится. Данные общего наставника сохранятся для следующих демонстраций."
              : "Учебный прогресс, заявки и встречи этого профиля будут удалены. Затем можно пройти регистрацию заново."}
          </p>
          <ErrorBox error={remove.error} />
          <div className="modal-actions">
            <button
              className="button secondary"
              onClick={() => setReset(false)}
            >
              Отмена
            </button>
            <button
              className="button danger"
              disabled={remove.isPending}
              onClick={() => remove.mutate(undefined)}
            >
              {sharedMentor
                ? "Выйти и выбрать роль"
                : "Удалить и начать заново"}
            </button>
          </div>
        </Modal>
      )}
    </>
  );
}
export function InternsPage() {
  const q = useData<Intern[]>("interns");
  return (
    <>
      <PageTitle
        eyebrow="ПРОСТРАНСТВО НАСТАВНИКА"
        title="Мои стажёры"
        description="Понимай, где нужна поддержка, и помогай сделать следующий шаг."
      />
      <QueryState query={q} />
      {q.data && <InternTable interns={q.data} />}
    </>
  );
}
function InternTable({ interns }: { interns: Intern[] }) {
  if (!interns.length)
    return <Empty>За тобой пока не закреплены стажёры.</Empty>;
  return (
    <div className="table-scroll">
      <table>
        <thead>
          <tr>
            <th>Стажёр</th>
            <th>Прогресс</th>
            <th>Задачи</th>
            <th>Требуют внимания</th>
            <th>
              <span className="sr-only">Профиль</span>
            </th>
          </tr>
        </thead>
        <tbody>
          {interns.map((i) => (
            <tr key={i.id}>
              <td>
                <Link className="person-cell" to={"/interns/" + i.id}>
                  <span className="avatar">{i.name.slice(0, 1)}</span>
                  <span>
                    <strong>{i.name}</strong>
                    <small>{i.team_name}</small>
                  </span>
                </Link>
              </td>
              <td>
                <Progress value={i.progress} label />
              </td>
              <td>
                {i.completed} / {i.total}
              </td>
              <td>
                {i.weak_topics?.length ? (
                  i.weak_topics.join(", ")
                ) : (
                  <span className="muted">Всё по плану</span>
                )}
              </td>
              <td>
                <Link
                  className="icon-link"
                  to={"/interns/" + i.id}
                  aria-label={"Профиль: " + i.name}
                >
                  <ChevronRight size={19} />
                </Link>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
export function InternDetail() {
  const { id } = useParams();
  const q = useData<Dashboard>("intern-" + id, "interns/" + id);
  const [dialog, setDialog] = useState<"task" | "meeting" | null>(null);
  return (
    <>
      <Link className="back-link" to="/interns">
        ← Мои стажёры
      </Link>
      <PageTitle
        eyebrow="АДАПТАЦИЯ СТАЖЁРА"
        title={q.data?.user?.name || "Профиль стажёра"}
        description={q.data?.user?.team_name}
        action={
          <div className="button-row">
            <button
              className="button secondary"
              onClick={() => setDialog("meeting")}
            >
              <CalendarDays size={16} />
              Назначить встречу
            </button>
            <button
              className="button primary"
              onClick={() => setDialog("task")}
            >
              <Plus size={17} />
              Назначить задачу
            </button>
          </div>
        }
      />
      <QueryState query={q} />
      {q.data && (
        <>
          <section className="intern-progress">
            <div>
              <h2>Прогресс адаптации</h2>
              <p>
                {q.data.completed} из {q.data.total} задач выполнено
              </p>
            </div>
            <Progress value={q.data.progress} label />
          </section>
          <section className="section-block">
            <h2>План адаптации</h2>
            <TaskList tasks={q.data.tasks || []} readonly />
          </section>
          <section className="section-block">
            <h2>Заявки на доступ</h2>
            <RequestList requests={q.data.requests || []} mentor />
          </section>
          <section className="section-block">
            <h2>Результаты обучения</h2>
            {q.data.attempts?.length ? (
              q.data.attempts.map((a) => (
                <article className="mentor-attempt" key={a.id}>
                  <div className="section-heading">
                    <h3>
                      Проверка знаний · {a.correct} из {a.total} верно
                    </h3>
                    <strong>{a.score}%</strong>
                  </div>
                  <p className="muted">
                    <DateText value={a.created_at} />
                  </p>
                  <p>{a.recommendation}</p>
                  {a.feedback?.map((f, i) => (
                    <div className="mentor-feedback" key={i}>
                      <Status value={f.correct ? "completed" : "blocked"} />
                      <strong>{f.question}</strong>
                      <p>{f.answer}</p>
                      <p className="muted">{f.explanation}</p>
                    </div>
                  ))}
                </article>
              ))
            ) : (
              <Empty>Стажёр пока не проходил экзамены.</Empty>
            )}
          </section>
          <section className="section-block">
            <h2>Встречи</h2>
            {q.data.meetings?.map((m) => (
              <MeetingRow meeting={m} key={m.id} />
            ))}
          </section>
        </>
      )}
      {dialog && (
        <AssignmentDialog
          kind={dialog}
          internId={id!}
          onClose={() => setDialog(null)}
        />
      )}
    </>
  );
}
function AssignmentDialog({
  kind,
  internId,
  onClose,
}: {
  kind: "task" | "meeting";
  internId: string;
  onClose: () => void;
}) {
  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [at, setAt] = useState("");
  const [article, setArticle] = useState("");
  const articles = useData<Article[]>("knowledge-assignment", "knowledge");
  const action = useAction(async () => {
    await api(
      "/interns/" + internId + (kind === "task" ? "/tasks" : "/meetings"),
      {
        method: "POST",
        body: JSON.stringify({
          title,
          description,
          [kind === "task" ? "due_at" : "starts_at"]: new Date(
            at,
          ).toISOString(),
          article_id: article || undefined,
        }),
      },
    );
    onClose();
  });
  return (
    <Modal
      title={kind === "task" ? "Новая задача стажёру" : "Встреча со стажёром"}
      onClose={onClose}
    >
      <form
        onSubmit={(e) => {
          e.preventDefault();
          action.mutate(undefined);
        }}
      >
        <label>
          Название
          <input
            required
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            maxLength={150}
          />
        </label>
        {kind === "task" && (
          <label>
            Описание
            <textarea
              required
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              rows={3}
            />
          </label>
        )}
        <label>
          {kind === "task" ? "Срок выполнения" : "Дата и время"}
          <input
            type="datetime-local"
            required
            value={at}
            onChange={(e) => setAt(e.target.value)}
            min={new Date(Date.now() - new Date().getTimezoneOffset() * 60000)
              .toISOString()
              .slice(0, 16)}
          />
        </label>
        <label>
          Материал для подготовки
          <select
            value={article}
            onChange={(e) => setArticle(e.target.value)}
            required={kind === "meeting"}
          >
            <option value="">Выбери материал</option>
            {articles.data?.map((a) => (
              <option key={a.id} value={a.id}>
                {a.title}
              </option>
            ))}
          </select>
        </label>
        <ErrorBox error={action.error} />
        <div className="modal-actions">
          <button type="button" className="button secondary" onClick={onClose}>
            Отмена
          </button>
          <button className="button primary" disabled={action.isPending}>
            Назначить
          </button>
        </div>
      </form>
    </Modal>
  );
}
function Metric({ label, value }: { label: string; value: string | number }) {
  return (
    <div className="metric">
      <span>{label}</span>
      <strong>{value}</strong>
    </div>
  );
}
export function AnalyticsPage({ user }: { user: User }) {
  const q = useData<Dashboard>("dashboard");
  const d = q.data;
  return (
    <>
      <PageTitle
        eyebrow="КОМАНДА В ФОКУСЕ"
        title="Как проходит адаптация"
        description={"Общая картина по доступным командам · " + user.team_name}
      />
      <QueryState query={q} />
      {d && (
        <>
          <div className="metrics four">
            <Metric label="Стажёров" value={d.stats?.interns || 0} />
            <Metric label="Наставников" value={d.stats?.mentors || 0} />
            <Metric
              label="Средний прогресс"
              value={`${Math.round(d.stats?.average_progress || 0)}%`}
            />
            <Metric label="Обращений к AI" value={d.stats?.ai_messages || 0} />
          </div>
          <section className="section-block">
            <div className="section-heading">
              <h2>Адаптация по командам</h2>
              <span className="badge">Доступные данные</span>
            </div>
            <div className="table-scroll">
              <table>
                <thead>
                  <tr>
                    <th>Команда</th>
                    <th>Стажёры</th>
                    <th>Средний прогресс</th>
                    <th>Статус</th>
                  </tr>
                </thead>
                <tbody>
                  {d.team_stats?.map((t) => (
                    <tr key={t.team_name}>
                      <td>
                        <strong>{t.team_name}</strong>
                      </td>
                      <td>{t.interns}</td>
                      <td>
                        <Progress value={t.progress} label />
                      </td>
                      <td>
                        <span className="badge">
                          {t.progress >= 80
                            ? "Уверенный старт"
                            : t.progress >= 40
                              ? "В процессе"
                              : "Первые шаги"}
                        </span>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
            {!d.team_stats?.length && (
              <Empty>Данные по командам пока отсутствуют.</Empty>
            )}
          </section>
          <section className="analytics-note">
            <Users size={27} />
            <div>
              <h2>Рядом в нужный момент</h2>
              <p>
                Прогресс складывается из выполненных задач, изученных материалов
                и результатов обучения. Доступы учитываются после согласования
                наставником.
              </p>
            </div>
          </section>
          <section className="section-block">
            <h2>Что измеряем</h2>
            <div className="measurement-grid">
              <div>
                <strong>Освоение программы</strong>
                <p>Доля завершённых шагов адаптации</p>
              </div>
              <div>
                <strong>Поддержка наставника</strong>
                <p>Закреплённые наставники и команды</p>
              </div>
              <div>
                <strong>Самостоятельность</strong>
                <p>Количество обращений к AI-помощнику</p>
              </div>
            </div>
          </section>
        </>
      )}
    </>
  );
}
