package main

import (
	"encoding/json"
	"net/http"
	"strings"
)

func (a *App) getSettings(w http.ResponseWriter, r *http.Request) error {
	s, _, _, e := a.settings(r.Context())
	if e != nil {
		return e
	}
	return respond(w, s)
}
func (a *App) putSettings(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		Settings
		TelegramToken      string `json:"telegram_token"`
		RouterAIAPIKey     string `json:"routerai_api_key"`
		ClearTelegramToken bool   `json:"clear_telegram_token"`
		ClearRouterAIKey   bool   `json:"clear_routerai_key"`
	}
	if e := decode(r, &in); e != nil {
		return e
	}
	_, token, key, e := a.settings(r.Context())
	if e != nil {
		return e
	}
	if strings.TrimSpace(in.TelegramToken) != "" {
		token = strings.TrimSpace(in.TelegramToken)
	}
	if strings.TrimSpace(in.RouterAIAPIKey) != "" {
		key = strings.TrimSpace(in.RouterAIAPIKey)
	}
	if in.ClearTelegramToken {
		token = ""
	}
	if in.ClearRouterAIKey {
		key = ""
	}
	in.TelegramUsername = strings.TrimPrefix(strings.TrimSpace(in.TelegramUsername), "@")
	if strings.ContainsAny(in.TelegramUsername, " /?&#") {
		return problem(400, "Некорректный username бота")
	}
	if e = validateSettings(in.Settings, token, key); e != nil {
		return problem(400, e.Error())
	}
	te, e := a.cipher.seal(token)
	if e != nil {
		return e
	}
	ke, e := a.cipher.seal(key)
	if e != nil {
		return e
	}
	in.TelegramTokenSet = token != ""
	in.RouterAIKeySet = key != ""
	b, e := json.Marshal(in.Settings)
	if e != nil {
		return e
	}
	_, e = a.db.Exec(r.Context(), "UPDATE settings SET body=$1,telegram_secret=$2,routerai_secret=$3 WHERE id=1", b, te, ke)
	if e != nil {
		return e
	}
	a.audit(r.Context(), "settings.updated (secrets omitted)")
	return respond(w, in.Settings)
}
func (a *App) testTelegram(w http.ResponseWriter, r *http.Request) error {
	_, token, _, e := a.settings(r.Context())
	if e != nil {
		return e
	}
	if token == "" {
		return respond(w, map[string]any{"ok": false, "message": "Сначала сохраните Telegram bot token"})
	}
	var me struct {
		Username string `json:"username"`
	}
	e = a.telegramAPI(r.Context(), token, "getMe", nil, &me)
	a.audit(r.Context(), "integration.telegram.test")
	if e != nil {
		return respond(w, map[string]any{"ok": false, "message": "Telegram не подтвердил токен: " + e.Error()})
	}
	return respond(w, map[string]any{"ok": true, "message": "Telegram Bot API доступен", "username": me.Username})
}
func (a *App) testRouterAI(w http.ResponseWriter, r *http.Request) error {
	s, _, key, e := a.settings(r.Context())
	if e != nil {
		return e
	}
	if key == "" {
		return respond(w, map[string]any{"ok": false, "message": "Сначала сохраните RouterAI API key"})
	}
	var models struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	e = a.routerCall(r.Context(), s, key, "GET", "/models", nil, &models)
	if e != nil {
		return respond(w, map[string]any{"ok": false, "message": e.Error()})
	}
	if s.RouterAIModel != "" {
		var out aiResponse
		e = a.routerCall(r.Context(), s, key, "POST", "/chat/completions", map[string]any{"model": s.RouterAIModel, "messages": []map[string]string{{"role": "user", "content": "Call get_user_profile to test tool calling. Do not answer without using this tool."}}, "max_tokens": 128, "tools": []any{tool("get_user_profile", "Return user's profile", map[string]any{})}, "tool_choice": map[string]any{"type": "function", "function": map[string]string{"name": "get_user_profile"}}}, &out)
		if e != nil {
			return respond(w, map[string]any{"ok": false, "message": "Модель не прошла проверку tool calling: " + e.Error(), "models": models.Data})
		}
		if len(out.Choices) == 0 || len(out.Choices[0].Message.ToolCalls) == 0 {
			return respond(w, map[string]any{"ok": false, "message": "Модель ответила без tool call. Выберите модель с поддержкой инструментов.", "models": models.Data})
		}
	}
	a.audit(r.Context(), "integration.routerai.test")
	return respond(w, map[string]any{"ok": true, "message": "RouterAI доступен. Выбранная модель проверена вызовом инструмента; без выбранной модели проверен только каталог.", "models": models.Data})
}
func (a *App) auditList(w http.ResponseWriter, r *http.Request) error {
	rows, e := a.db.Query(r.Context(), "SELECT id::text,action,created_at FROM audit ORDER BY created_at DESC LIMIT 200")
	if e != nil {
		return e
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var key, action string
		var at any
		if e = rows.Scan(&key, &action, &at); e != nil {
			return e
		}
		out = append(out, map[string]any{"id": key, "action": action, "created_at": at})
	}
	if e = rows.Err(); e != nil {
		return e
	}
	return respond(w, out)
}
