import { useEffect, useRef, type ReactNode } from "react";
import { Link } from "react-router-dom";
import {
  AlertCircle,
  ArrowUpRight,
  Check,
  Circle,
  LoaderCircle,
  X,
} from "lucide-react";
export const roles = {
  intern: "Стажёр",
  mentor: "Наставник",
  manager: "Руководитель",
};
export function directionTitle(value: string) {
  const labels: Record<string, string> = {
    backend: "Backend",
    frontend: "Frontend",
    analytics: "Продуктовая аналитика",
    qa: "QA / Тестирование",
    "data-science": "Data Science",
    devops: "DevOps / Infrastructure",
    product: "Управление продуктом",
    design: "UX/UI-дизайн",
    all: "Общие материалы",
  };
  return labels[value] || value;
}
const labels: Record<string, string> = {
  not_started: "Не начато",
  in_progress: "В работе",
  waiting_approval: "На согласовании",
  retry: "Повторная попытка",
  waiting_configuration: "Telegram не настроен",
  sending: "Отправляется",
  demo_reminder: "Демо: без отправки в Telegram",
  pending: "На согласовании",
  pending_approval: "На согласовании",
  created: "Создана",
  approved: "Одобрена",
  rejected: "Отклонена",
  completed: "Выполнено",
  blocked: "Нужна помощь",
  sent: "Отправлено",
  scheduled: "Запланировано",
  disabled: "Уведомления выключены",
  demo: "Демо",
  failed: "Ошибка отправки",
  not_configured: "Telegram не настроен",
  skipped: "Не отправлено",
};
export function Status({ value }: { value: string }) {
  return (
    <span className={`badge status-${value}`}>
      {value === "completed" && <Check size={12} />} {labels[value] || value}
    </span>
  );
}
export function Spinner() {
  return (
    <div className="loading" role="status">
      <LoaderCircle size={22} className="spin" /> Загружаем данные…
    </div>
  );
}
export function ErrorBox({
  error,
  retry,
}: {
  error: unknown;
  retry?: () => void;
}) {
  if (!error) return null;
  return (
    <div className="error" role="alert">
      <AlertCircle size={18} />
      <span>
        {error instanceof Error
          ? error.message
          : "Не удалось загрузить данные."}
      </span>
      {retry && (
        <button onClick={retry} className="text-button">
          Повторить
        </button>
      )}
    </div>
  );
}
export function Empty({ children }: { children: ReactNode }) {
  return (
    <div className="empty">
      <Circle size={24} />
      <p>{children}</p>
    </div>
  );
}
export function PageTitle({
  eyebrow,
  title,
  description,
  action,
}: {
  eyebrow?: string;
  title: string;
  description?: string;
  action?: ReactNode;
}) {
  return (
    <div className="page-title">
      <div>
        {eyebrow && <p className="eyebrow">{eyebrow}</p>}
        <h1>{title}</h1>
        {description && <p className="subtitle">{description}</p>}
      </div>
      {action}
    </div>
  );
}
export function Progress({
  value,
  label = false,
}: {
  value: number;
  label?: boolean;
}) {
  const n = Math.max(0, Math.min(100, value || 0));
  return (
    <div className="progress-wrap">
      <div
        className="progress"
        role="progressbar"
        aria-valuenow={n}
        aria-valuemin={0}
        aria-valuemax={100}
        aria-label="Прогресс адаптации"
      >
        <span style={{ width: `${n}%` }} />
      </div>
      {label && <strong>{Math.round(n)}%</strong>}
    </div>
  );
}
export function DateText({ value }: { value: string }) {
  if (!value || value.startsWith("0001-")) return <span>Без срока</span>;
  const d = new Date(value);
  return (
    <time dateTime={value}>
      {isNaN(d.getTime())
        ? "Без срока"
        : d.toLocaleString("ru-RU", {
            day: "numeric",
            month: "short",
            hour: "2-digit",
            minute: "2-digit",
          })}
    </time>
  );
}
export function TextLink({
  to,
  children,
}: {
  to: string;
  children: ReactNode;
}) {
  return (
    <Link className="inline-link" to={to}>
      {children}
      <ArrowUpRight size={15} />
    </Link>
  );
}
export function Modal({
  title,
  children,
  onClose,
}: {
  title: string;
  children: ReactNode;
  onClose: () => void;
}) {
  const ref = useRef<HTMLDivElement>(null);
  useEffect(() => {
    const previous = document.activeElement as HTMLElement;
    ref.current?.focus();
    const handler = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
      if (e.key === "Tab") {
        const all = Array.from(
          ref.current?.querySelectorAll<HTMLElement>(
            "button, input, textarea, select, a[href]",
          ) || [],
        ).filter((e) => !e.hasAttribute("disabled"));
        const first = all[0],
          last = all[all.length - 1];
        if (e.shiftKey && document.activeElement === first) {
          e.preventDefault();
          last?.focus();
        } else if (!e.shiftKey && document.activeElement === last) {
          e.preventDefault();
          first?.focus();
        }
      }
    };
    document.addEventListener("keydown", handler);
    const overflow = document.body.style.overflow;
    document.body.style.overflow = "hidden";
    return () => {
      document.removeEventListener("keydown", handler);
      document.body.style.overflow = overflow;
      previous?.focus();
    };
  }, [onClose]);
  return (
    <div
      className="modal-overlay"
      onClick={(e) => {
        if (e.target === e.currentTarget) onClose();
      }}
    >
      <div
        className="modal"
        role="dialog"
        aria-modal="true"
        aria-label={title}
        tabIndex={-1}
        ref={ref}
      >
        <div className="modal-head">
          <h2>{title}</h2>
          <button
            className="icon-button"
            aria-label="Закрыть"
            onClick={onClose}
          >
            <X size={20} />
          </button>
        </div>
        {children}
      </div>
    </div>
  );
}
export function Brand() {
  return (
    <span className="brand">
      <span className="brand-mark">
        <i />
        <i />
        <i />
        <i />
      </span>
      AI-buddy<span className="brand-dot">.</span>
    </span>
  );
}
