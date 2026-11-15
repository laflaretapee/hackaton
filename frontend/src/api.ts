export class ApiError extends Error {
  constructor(
    public status: number,
    message: string,
  ) {
    super(message);
  }
}
export async function api<T>(
  path: string,
  options: RequestInit = {},
): Promise<T> {
  let response: Response;
  try {
    response = await fetch("/api" + path, {
      ...options,
      credentials: "same-origin",
      headers: { "Content-Type": "application/json", ...options.headers },
    });
  } catch {
    throw new ApiError(
      0,
      "Не удалось подключиться. Проверь соединение и попробуй ещё раз.",
    );
  }
  if (!response.ok) {
    const messages: Record<number, string> = {
      400: "Проверь заполненные поля и настройки интеграции.",
      401: "Войди в аккаунт, чтобы продолжить.",
      403: "Для этого действия недостаточно прав.",
      404: "Данные не найдены.",
      409: "Действие пока недоступно или ожидает подтверждения.",
      422: "Проверь заполненные поля.",
      429: "Слишком много запросов. Попробуй чуть позже.",
    };
    throw new ApiError(
      response.status,
      messages[response.status] ||
        "Сервис временно недоступен. Попробуй ещё раз.",
    );
  }
  if (response.status === 204) return undefined as T;
  return response.json() as Promise<T>;
}
export function safeSource(url: string) {
  try {
    if (url.startsWith("/") && !url.startsWith("//")) {
      const resolved = new URL(url, window.location.origin);
      return resolved.origin === window.location.origin ? url : "#";
    }
    if (new URL(url).protocol === "https:") return url;
    return "#";
  } catch {
    return "#";
  }
}
export function makeSettingsPayload(value: Record<string, unknown>) {
  const result = { ...value };
  for (const name of ["telegram_token", "routerai_api_key"])
    if (typeof result[name] === "string" && !(result[name] as string).trim())
      delete result[name];
  delete result.telegram_token_set;
  delete result.routerai_key_set;
  return result;
}
