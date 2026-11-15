package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type apiError struct {
	code    int
	message string
}

func (e apiError) Error() string         { return e.message }
func problem(code int, msg string) error { return apiError{code, msg} }

type endpoint func(http.ResponseWriter, *http.Request) error
type contextKey string

const userKey contextKey = "user"

func current(r *http.Request) User { return r.Context().Value(userKey).(User) }
func respond(w http.ResponseWriter, v any) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	return json.NewEncoder(w).Encode(v)
}
func decode(r *http.Request, v any) error {
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		return problem(415, "Используйте application/json")
	}
	d := json.NewDecoder(http.MaxBytesReader(nil, r.Body, 64<<10))
	if e := d.Decode(v); e != nil {
		return problem(400, "Некорректный JSON")
	}
	return nil
}
func (a *App) wrap(fn endpoint, auth string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if e := recover(); e != nil {
				slog.Error("handler panic", "path", r.URL.Path)
				http.Error(w, `{"error":"Внутренняя ошибка"}`, 500)
			}
		}()
		w.Header().Set("X-Content-Type-Options", "nosniff")
		if r.Method != "GET" && r.Method != "HEAD" {
			if r.Header.Get("Sec-Fetch-Site") == "cross-site" {
				writeError(w, problem(403, "Недопустимый источник запроса"))
				return
			}
			if origin := r.Header.Get("Origin"); origin != "" {
				allowed := a.appURL
				if allowed == "" {
					scheme := "http"
					if a.secure {
						scheme = "https"
					}
					allowed = scheme + "://" + r.Host
				}
				if strings.TrimSuffix(origin, "/") != strings.TrimSuffix(allowed, "/") {
					writeError(w, problem(403, "Недопустимый источник запроса"))
					return
				}
			}
		}
		if auth != "" {
			cookie := "buddy_session"
			if auth == "admin" {
				cookie = "buddy_admin"
			}
			c, e := r.Cookie(cookie)
			if e != nil {
				writeError(w, problem(401, "Требуется вход"))
				return
			}
			var uid *string
			var admin bool
			e = a.db.QueryRow(r.Context(), "SELECT user_id,admin FROM sessions WHERE token_hash=$1 AND expires_at>now()", hashToken(c.Value)).Scan(&uid, &admin)
			if e != nil || auth == "admin" && !admin || auth != "admin" && (admin || uid == nil) {
				writeError(w, problem(401, "Сессия истекла"))
				return
			}
			if auth != "admin" {
				u, e := a.loadUser(r.Context(), *uid)
				if e != nil {
					writeError(w, problem(401, "Профиль не найден"))
					return
				}
				r = r.WithContext(context.WithValue(r.Context(), userKey, u))
			}
		}
		if e := fn(w, r); e != nil {
			writeError(w, e)
		}
	}
}
func writeError(w http.ResponseWriter, e error) {
	code, msg := 500, "Не удалось выполнить запрос"
	var ae apiError
	if errors.As(e, &ae) {
		code, msg = ae.code, ae.message
	} else if errors.Is(e, pgx.ErrNoRows) {
		code, msg = 404, "Не найдено"
	} else {
		slog.Error("api error", "error", e)
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
func (a *App) routes() http.Handler {
	m := http.NewServeMux()
	add := func(pattern, auth string, fn endpoint) { m.HandleFunc(pattern, a.wrap(fn, auth)) }
	add("GET /health", "", func(w http.ResponseWriter, r *http.Request) error {
		if a.db != nil {
			if e := a.db.Ping(r.Context()); e != nil {
				return problem(503, "Database unavailable")
			}
		}
		return respond(w, map[string]string{"status": "ok"})
	})
	add("GET /api/public/config", "", a.publicConfig)
	add("GET /api/teams", "", func(w http.ResponseWriter, r *http.Request) error {
		x, e := a.teams(r.Context())
		if e != nil {
			return e
		}
		return respond(w, x)
	})
	add("POST /api/auth/demo", "", a.demoLogin)
	add("POST /api/auth/telegram", "", a.telegramLogin)
	add("GET /api/auth/login-link", "", a.loginLink)
	add("POST /api/auth/exchange", "", a.exchange)
	add("POST /api/auth/logout", "", a.logout)
	add("GET /api/me", "user", func(w http.ResponseWriter, r *http.Request) error { return respond(w, current(r)) })
	add("PATCH /api/me", "user", a.patchMe)
	add("DELETE /api/me", "user", a.deleteMe)
	add("GET /api/dashboard", "user", a.dashboardHandler)
	add("GET /api/tasks", "user", a.listTasks)
	add("PATCH /api/tasks/{id}", "user", a.patchTask)
	add("GET /api/knowledge", "user", a.knowledge)
	add("GET /api/knowledge/{id}", "user", a.getArticle)
	add("POST /api/knowledge/{id}/read", "user", a.readArticle)
	add("GET /api/lessons", "user", a.lessons)
	add("GET /api/lessons/{id}/exam", "user", a.exam)
	add("POST /api/lessons/{id}/exam", "user", a.submitExam)
	add("GET /api/chat", "user", a.chatHistory)
	add("POST /api/chat", "user", a.chat)
	add("GET /api/requests", "user", a.listRequests)
	add("POST /api/requests", "user", a.createRequest)
	add("POST /api/requests/{id}/decision", "user", a.decision)
	add("GET /api/meetings", "user", a.listMeetings)
	add("POST /api/meetings/demo", "user", a.demoMeeting)
	add("GET /api/interns", "user", a.listInterns)
	add("GET /api/interns/{id}", "user", a.internDashboard)
	add("POST /api/interns/{id}/tasks", "user", a.assignTask)
	add("POST /api/interns/{id}/meetings", "user", a.assignMeeting)
	add("POST /api/admin/login", "", a.adminLogin)
	add("POST /api/admin/logout", "admin", a.adminLogout)
	add("GET /api/admin/settings", "admin", a.getSettings)
	add("PUT /api/admin/settings", "admin", a.putSettings)
	add("POST /api/admin/test/telegram", "admin", a.testTelegram)
	add("POST /api/admin/test/routerai", "admin", a.testRouterAI)
	add("GET /api/admin/audit", "admin", a.auditList)
	return m
}
func (a *App) newSession(w http.ResponseWriter, r *http.Request, uid string, admin bool) error {
	token := id()
	var owner any
	if uid != "" {
		owner = uid
	}
	expiry := time.Now().Add(7 * 24 * time.Hour)
	if admin {
		expiry = time.Now().Add(8 * time.Hour)
	}
	if _, e := a.db.Exec(r.Context(), "INSERT INTO sessions(token_hash,user_id,admin,expires_at) VALUES($1,$2,$3,$4)", hashToken(token), owner, admin, expiry); e != nil {
		return e
	}
	name := "buddy_session"
	if admin {
		name = "buddy_admin"
	}
	http.SetCookie(w, &http.Cookie{Name: name, Value: token, Path: "/", HttpOnly: true, Secure: a.secure, SameSite: a.sameSite(), Expires: expiry})
	return nil
}
func (a *App) clearSession(w http.ResponseWriter, r *http.Request, admin bool) {
	name := "buddy_session"
	if admin {
		name = "buddy_admin"
	}
	if c, e := r.Cookie(name); e == nil {
		_, _ = a.db.Exec(r.Context(), "DELETE FROM sessions WHERE token_hash=$1", hashToken(c.Value))
	}
	http.SetCookie(w, &http.Cookie{Name: name, Value: "", Path: "/", HttpOnly: true, Secure: a.secure, SameSite: a.sameSite(), MaxAge: -1})
}
func (a *App) publicConfig(w http.ResponseWriter, r *http.Request) error {
	s, _, _, e := a.settings(r.Context())
	if e != nil {
		return e
	}
	return respond(w, map[string]any{"demo_mode": getenv("DEMO_MODE", "true") == "true", "ai_mock_mode": s.AIMockMode, "telegram_username": s.TelegramUsername, "app_url": s.AppURL})
}
func (a *App) demoLogin(w http.ResponseWriter, r *http.Request) error {
	if getenv("DEMO_MODE", "true") != "true" {
		return problem(403, "Демонстрационный вход отключён")
	}
	var in struct {
		Role   string `json:"role"`
		Name   string `json:"name"`
		TeamID string `json:"team_id"`
	}
	if e := decode(r, &in); e != nil {
		return e
	}
	u, e := a.createUser(r.Context(), in.Name, in.Role, in.TeamID, 0, true)
	if e != nil {
		return problem(400, e.Error())
	}
	if e = a.newSession(w, r, u.ID, false); e != nil {
		return e
	}
	return respond(w, u)
}
func (a *App) telegramLogin(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		InitData string `json:"init_data"`
	}
	if e := decode(r, &in); e != nil {
		return e
	}
	s, token, _, e := a.settings(r.Context())
	if e != nil {
		return e
	}
	if !s.TelegramEnabled {
		return problem(503, "Telegram пока не настроен администратором")
	}
	tg, e := verifyTelegram(in.InitData, token, time.Now())
	if e != nil {
		return problem(401, e.Error())
	}
	var uid string
	e = a.db.QueryRow(r.Context(), "SELECT id FROM users WHERE telegram_id=$1", tg.ID).Scan(&uid)
	if errors.Is(e, pgx.ErrNoRows) {
		return problem(409, "Сначала завершите регистрацию в Telegram-боте")
	}
	if e != nil {
		return e
	}
	u, e := a.loadUser(r.Context(), uid)
	if e != nil {
		return e
	}
	if e = a.newSession(w, r, uid, false); e != nil {
		return e
	}
	return respond(w, u)
}
func (a *App) loginLink(w http.ResponseWriter, r *http.Request) error {
	s, _, _, e := a.settings(r.Context())
	if e != nil {
		return e
	}
	if !s.TelegramEnabled || s.TelegramUsername == "" {
		return problem(503, "Telegram пока не настроен администратором")
	}
	token := id()
	_, e = a.db.Exec(r.Context(), "INSERT INTO login_links(token_hash,expires_at) VALUES($1,now()+interval '10 minutes')", hashToken(token))
	if e != nil {
		return e
	}
	return respond(w, map[string]string{"token": token, "url": "https://t.me/" + url.PathEscape(s.TelegramUsername) + "?start=login_" + token})
}
func (a *App) exchange(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		Token string `json:"token"`
	}
	if e := decode(r, &in); e != nil {
		return e
	}
	var uid string
	e := a.db.QueryRow(r.Context(), "UPDATE login_links SET consumed=true WHERE token_hash=$1 AND user_id IS NOT NULL AND expires_at>now() AND NOT consumed RETURNING user_id", hashToken(in.Token)).Scan(&uid)
	if errors.Is(e, pgx.ErrNoRows) {
		var pending bool
		_ = a.db.QueryRow(r.Context(), "SELECT EXISTS(SELECT 1 FROM login_links WHERE token_hash=$1 AND user_id IS NULL AND expires_at>now() AND NOT consumed)", hashToken(in.Token)).Scan(&pending)
		if pending {
			return problem(409, "Ожидаем подтверждение в Telegram")
		}
		return problem(401, "Ссылка недействительна или уже использована")
	}
	if e != nil {
		return e
	}
	u, e := a.loadUser(r.Context(), uid)
	if e != nil {
		return e
	}
	if e = a.newSession(w, r, uid, false); e != nil {
		return e
	}
	return respond(w, u)
}
func (a *App) logout(w http.ResponseWriter, r *http.Request) error {
	a.clearSession(w, r, false)
	return respond(w, map[string]bool{"ok": true})
}
func (a *App) patchMe(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		Notifications   *bool `json:"notifications"`
		ReminderMinutes *int  `json:"reminder_minutes"`
	}
	if e := decode(r, &in); e != nil {
		return e
	}
	u := current(r)
	if in.Notifications != nil {
		u.Notifications = *in.Notifications
	}
	if in.ReminderMinutes != nil {
		if *in.ReminderMinutes < 1 || *in.ReminderMinutes > 10080 {
			return problem(400, "Напоминание: от 1 до 10080 минут")
		}
		u.ReminderMinutes = *in.ReminderMinutes
	}
	_, e := a.db.Exec(r.Context(), "UPDATE users SET notifications=$2,reminder_minutes=$3 WHERE id=$1", u.ID, u.Notifications, u.ReminderMinutes)
	if e != nil {
		return e
	}
	return respond(w, u)
}
func (a *App) deleteMe(w http.ResponseWriter, r *http.Request) error {
	u := current(r)
	if strings.HasPrefix(u.ID, "demo-mentor-") {
		a.clearSession(w, r, false)
		return respond(w, map[string]bool{"ok": true})
	}
	tx, e := a.db.Begin(r.Context())
	if e != nil {
		return e
	}
	defer tx.Rollback(r.Context())
	if u.TelegramID > 0 {
		var pendingToken string
		e = tx.QueryRow(r.Context(), "DELETE FROM bot_states WHERE telegram_id=$1 RETURNING coalesce(body->>'login_token','')", u.TelegramID).Scan(&pendingToken)
		if e != nil && !errors.Is(e, pgx.ErrNoRows) {
			return e
		}
		if pendingToken != "" {
			if _, e = tx.Exec(r.Context(), "DELETE FROM login_links WHERE token_hash=$1", hashToken(pendingToken)); e != nil {
				return e
			}
		}
	}
	if _, e = tx.Exec(r.Context(), "DELETE FROM users WHERE id=$1", u.ID); e != nil {
		return e
	}
	if e = tx.Commit(r.Context()); e != nil {
		return e
	}
	a.clearSession(w, r, false)
	return respond(w, map[string]bool{"ok": true})
}
func (a *App) adminLogin(w http.ResponseWriter, r *http.Request) error {
	host, _, _ := net.SplitHostPort(r.RemoteAddr)
	if getenv("TRUST_PROXY", "false") == "true" {
		if ip := net.ParseIP(strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-For"), ",")[0])); ip != nil {
			host = ip.String()
		}
	}
	if host == "" {
		host = r.RemoteAddr
	}
	a.mu.Lock()
	if a.attempts == nil {
		a.attempts = map[string]loginAttempt{}
	}
	v := a.attempts[host]
	if time.Since(v.since) > 15*time.Minute {
		v = loginAttempt{since: time.Now()}
	}
	if v.count >= 10 {
		a.mu.Unlock()
		return problem(429, "Слишком много попыток. Повторите через 15 минут")
	}
	v.count++
	a.attempts[host] = v
	a.mu.Unlock()
	var in struct {
		Password string `json:"password"`
	}
	if e := decode(r, &in); e != nil {
		return e
	}
	if !equalSecret(in.Password, a.adminPassword) {
		return problem(401, "Неверный пароль")
	}
	a.mu.Lock()
	delete(a.attempts, host)
	a.mu.Unlock()
	if e := a.newSession(w, r, "", true); e != nil {
		return e
	}
	a.audit(r.Context(), "admin.login")
	return respond(w, map[string]bool{"ok": true})
}
func (a *App) adminLogout(w http.ResponseWriter, r *http.Request) error {
	a.clearSession(w, r, true)
	return respond(w, map[string]bool{"ok": true})
}

func (a *App) sameSite() http.SameSite {
	if a.secure {
		return http.SameSiteNoneMode
	}
	return http.SameSiteLaxMode
}
